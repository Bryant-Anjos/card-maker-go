package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/Bryant-Anjos/card-maker-go/pkg/cardmaker/render/chrome"
	"github.com/Bryant-Anjos/card-maker-go/pkg/cardmaker/renderers"
	"github.com/Bryant-Anjos/card-maker-go/pkg/cardmaker/server"
)

// Exit codes.
const (
	exitOK      = 0
	exitFailure = 1
	exitUsage   = 2
)

// envToken names the environment variable holding the API token. It is read
// from the environment, not a flag, so it never shows up in process listings.
const envToken = "CARDMAKER_API_TOKEN"

const (
	shutdownTimeout   = 15 * time.Second
	readHeaderTimeout = 10 * time.Second
	readTimeout       = 60 * time.Second
	writeTimeout      = 120 * time.Second
	idleTimeout       = 60 * time.Second
)

// run starts the server and blocks until ctx is cancelled. It is separate from
// main so it can be tested.
func run(ctx context.Context, args []string, getenv func(string) string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("cardmaker-server", flag.ContinueOnError)
	fs.SetOutput(stderr)
	addr := fs.String("addr", "127.0.0.1:8080", "address to listen on")
	themesDir := fs.String("themes", "themes", "directory whose subdirectories are themes")
	rendererName := fs.String("renderer", renderers.Chrome, "renderer: chrome or fake")
	browserPath := fs.String("chrome", "", "browser executable (default: $"+chrome.EnvBrowserPath+", then PATH)")
	maxConcurrent := fs.Int("max-concurrent", server.DefaultMaxConcurrent, "renders in parallel")
	queueTimeout := fs.Duration("queue-timeout", server.DefaultQueueTimeout, "how long a request waits for a free slot before getting 503")
	maxUploadMB := fs.Int64("max-upload-mb", server.DefaultMaxUploadBytes>>20, "largest accepted request body, in MiB")
	allowNoAuth := fs.Bool("allow-no-auth", false, "allow a non-loopback address without "+envToken)

	if err := fs.Parse(args); err != nil {
		return exitUsage
	}

	token := getenv(envToken)
	if token == "" && !isLoopbackAddress(*addr) && !*allowNoAuth {
		fmt.Fprintf(stderr, "refusing to listen on %q without authentication: set %s, bind to a loopback address or pass -allow-no-auth\n",
			*addr, envToken)
		return exitFailure
	}

	themes, err := server.LoadThemes(*themesDir)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return exitFailure
	}

	renderer, closeRenderer, err := renderers.Open(*rendererName, *browserPath)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return exitFailure
	}
	defer func() {
		if err := closeRenderer(); err != nil {
			fmt.Fprintln(stderr, "close renderer:", err)
		}
	}()

	logger := slog.New(slog.NewTextHandler(stderr, nil))
	api, err := server.New(server.Config{
		Themes:         themes,
		Renderer:       renderer,
		Logger:         logger,
		Token:          token,
		MaxConcurrent:  *maxConcurrent,
		QueueTimeout:   *queueTimeout,
		MaxUploadBytes: *maxUploadMB << 20,
	})
	if err != nil {
		fmt.Fprintln(stderr, err)
		return exitFailure
	}

	listener, err := net.Listen("tcp", *addr)
	if err != nil {
		fmt.Fprintln(stderr, "listen:", err)
		return exitFailure
	}

	httpServer := &http.Server{
		Handler:           api.Handler(),
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
	}
	serveErr := make(chan error, 1)
	go func() { serveErr <- httpServer.Serve(listener) }()

	fmt.Fprintf(stdout, "listening on http://%s\n", listener.Addr())

	select {
	case err := <-serveErr:
		fmt.Fprintln(stderr, "server stopped:", err)
		return exitFailure
	case <-ctx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil && !errors.Is(err, http.ErrServerClosed) {
		fmt.Fprintln(stderr, "shutdown:", err)
		return exitFailure
	}

	return exitOK
}

// isLoopbackAddress reports whether a listen address only accepts local
// connections. An empty host (":8080") listens on every interface.
func isLoopbackAddress(addr string) bool {
	host, _, err := net.SplitHostPort(addr)
	if err != nil || host == "" {
		return false
	}
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)

	return ip != nil && ip.IsLoopback()
}
