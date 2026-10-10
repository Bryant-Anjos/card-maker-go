package main

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"
)

const themesDir = "../../testdata/themes"

func noEnv(string) string { return "" }

func envWith(pairs map[string]string) func(string) string {
	return func(key string) string { return pairs[key] }
}

func runOnce(t *testing.T, getenv func(string) string, args ...string) (code int, stderr string) {
	t.Helper()

	var errOut bytes.Buffer
	code = run(context.Background(), args, getenv, io.Discard, &errOut)

	return code, errOut.String()
}

func TestUsageErrors(t *testing.T) {
	if code, _ := runOnce(t, noEnv, "-nope"); code != 2 {
		t.Errorf("unknown flag: code = %d, want 2", code)
	}
	if code, _ := runOnce(t, noEnv, "-max-concurrent", "x"); code != 2 {
		t.Errorf("bad value: code = %d, want 2", code)
	}
}

func TestStartupFailures(t *testing.T) {
	tests := []struct {
		name string
		args []string
		env  map[string]string
		want string
	}{
		{"missing themes dir", []string{"-renderer", "fake", "-themes", "does-not-exist"}, nil, "does-not-exist"},
		{"unknown renderer", []string{"-renderer", "magic", "-themes", themesDir}, nil, "magic"},
		{"missing browser", []string{"-chrome", "/nope/chrome", "-themes", themesDir}, nil, "/nope/chrome"},
		{"invalid address", []string{"-renderer", "fake", "-themes", themesDir, "-addr", "not an address"}, nil, "listen"},
		{"public address needs a token", []string{"-renderer", "fake", "-themes", themesDir, "-addr", "0.0.0.0:0"}, nil, "CARDMAKER_API_TOKEN"},
		{"all-interfaces shorthand needs a token", []string{"-renderer", "fake", "-themes", themesDir, "-addr", ":0"}, nil, "CARDMAKER_API_TOKEN"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, stderr := runOnce(t, envWith(tt.env), tt.args...)

			if code != 1 || !strings.Contains(stderr, tt.want) {
				t.Errorf("code = %d, stderr = %q, want 1 and %q", code, stderr, tt.want)
			}
		})
	}
}

// lineWriter collects output safely and signals each written line.
type lineWriter struct {
	mu    sync.Mutex
	buf   bytes.Buffer
	lines chan string
}

func (w *lineWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.buf.Write(p)
	for _, line := range strings.Split(strings.TrimRight(string(p), "\n"), "\n") {
		select {
		case w.lines <- line:
		default:
		}
	}

	return len(p), nil
}

func startServer(t *testing.T, env map[string]string, args ...string) (baseURL string, stop func() int) {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	out := &lineWriter{lines: make(chan string, 16)}
	done := make(chan int, 1)
	all := append([]string{"-renderer", "fake", "-themes", themesDir, "-addr", "127.0.0.1:0"}, args...)

	go func() { done <- run(ctx, all, envWith(env), out, io.Discard) }()

	select {
	case line := <-out.lines:
		const prefix = "listening on "
		if !strings.HasPrefix(line, prefix) {
			cancel()
			t.Fatalf("first output line = %q, want %q...", line, prefix)
		}
		baseURL = strings.TrimSpace(strings.TrimPrefix(line, prefix))
	case code := <-done:
		cancel()
		t.Fatalf("server exited early with code %d", code)
	case <-time.After(10 * time.Second):
		cancel()
		t.Fatal("server did not start")
	}

	return baseURL, func() int {
		cancel()
		select {
		case code := <-done:
			return code
		case <-time.After(10 * time.Second):
			t.Fatal("server did not stop after cancel")
			return -1
		}
	}
}

func TestServesAndShutsDownGracefully(t *testing.T) {
	base, stop := startServer(t, nil)

	resp, err := http.Get(base + "/healthz")
	if err != nil {
		t.Fatalf("health check failed: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d", resp.StatusCode)
	}

	if code := stop(); code != 0 {
		t.Errorf("exit code after shutdown = %d, want 0", code)
	}
}

func TestTokenFromEnvironment(t *testing.T) {
	base, stop := startServer(t, map[string]string{"CARDMAKER_API_TOKEN": "tok"})
	defer stop()

	resp, err := http.Get(base + "/v1/themes")
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("without token: status = %d, want 401", resp.StatusCode)
	}

	req, _ := http.NewRequest(http.MethodGet, base+"/v1/themes", nil)
	req.Header.Set("Authorization", "Bearer tok")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("with token: status = %d, want 200", resp.StatusCode)
	}
}

func TestAcceptsNoSandboxFlag(t *testing.T) {
	_, stop := startServer(t, nil, "-no-sandbox")

	if code := stop(); code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
}

func TestIsLoopbackAddress(t *testing.T) {
	tests := map[string]bool{
		"127.0.0.1:8080": true,
		"localhost:8080": true,
		"[::1]:8080":     true,
		"0.0.0.0:8080":   false,
		":8080":          false,
		"192.168.1.5:80": false,
		"example.com:80": false,
		"garbage":        false,
	}

	for addr, want := range tests {
		if got := isLoopbackAddress(addr); got != want {
			t.Errorf("isLoopbackAddress(%q) = %v, want %v", addr, got, want)
		}
	}
}
