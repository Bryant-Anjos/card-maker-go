// Package server exposes the card renderer over HTTP, so tools can render a
// single card on demand. It is a thin layer over the same core the CLI uses.
package server

import (
	"crypto/sha256"
	"crypto/subtle"
	"errors"
	"log/slog"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"github.com/Bryant-Anjos/card-maker-go/pkg/cardmaker/render"
	"github.com/Bryant-Anjos/card-maker-go/pkg/cardmaker/theme"
)

// Defaults applied by New to zero Config fields.
const (
	DefaultMaxConcurrent  = 4
	DefaultQueueTimeout   = 10 * time.Second
	DefaultMaxUploadBytes = 20 << 20
	DefaultMaxScale       = 3
	DefaultMaxArtPixels   = 40_000_000
)

// Config configures a Server. Themes and Renderer are required.
type Config struct {
	Themes   map[string]theme.Theme // themes by name, see LoadThemes
	Renderer render.Renderer
	Logger   *slog.Logger // default: slog.Default()

	Token          string        // when set, requests must send "Authorization: Bearer <token>"
	MaxConcurrent  int           // renders in parallel
	QueueTimeout   time.Duration // how long a request waits for a free slot before getting 503
	MaxUploadBytes int64         // largest accepted request body
	MaxScale       int           // largest accepted scale
	MaxArtPixels   int64         // largest accepted artwork, in pixels
}

// Server is an http.Handler provider for the card API.
type Server struct {
	cfg Config
	sem chan struct{}
}

// New validates the configuration, fills in defaults and returns a Server.
func New(cfg Config) (*Server, error) {
	if cfg.Renderer == nil {
		return nil, errors.New("server: a renderer is required")
	}
	if len(cfg.Themes) == 0 {
		return nil, errors.New("server: at least one theme is required")
	}
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}
	if cfg.MaxConcurrent <= 0 {
		cfg.MaxConcurrent = DefaultMaxConcurrent
	}
	if cfg.QueueTimeout <= 0 {
		cfg.QueueTimeout = DefaultQueueTimeout
	}
	if cfg.MaxUploadBytes <= 0 {
		cfg.MaxUploadBytes = DefaultMaxUploadBytes
	}
	if cfg.MaxScale <= 0 {
		cfg.MaxScale = DefaultMaxScale
	}
	if cfg.MaxArtPixels <= 0 {
		cfg.MaxArtPixels = DefaultMaxArtPixels
	}

	return &Server{cfg: cfg, sem: make(chan struct{}, cfg.MaxConcurrent)}, nil
}

// Handler returns the HTTP handler with all routes and middleware.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeData(w, map[string]string{"status": "ok"})
	})
	mux.Handle("GET /v1/themes", s.protected(func(w http.ResponseWriter, _ *http.Request) error {
		writeData(w, s.listThemes())
		return nil
	}))
	mux.Handle("GET /v1/themes/{name}/back", s.protected(s.handleBack))
	mux.Handle("POST /v1/render", s.protected(s.handleRenderCard))

	return s.recoverPanics(s.logRequests(securityHeaders(mux)))
}

// handlerFunc is a handler that reports failures by returning an error.
type handlerFunc func(w http.ResponseWriter, r *http.Request) error

// protected applies authentication and turns returned errors into responses:
// *apiError values are shown to the client, anything else becomes a generic
// 500 (details are only logged).
func (s *Server) protected(h handlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !s.authorized(r) {
			w.Header().Set("WWW-Authenticate", "Bearer")
			writeAPIError(w, &apiError{status: http.StatusUnauthorized, message: "missing or invalid bearer token"})
			return
		}

		err := h(w, r)
		var apiErr *apiError
		switch {
		case err == nil:
		case errors.As(err, &apiErr):
			writeAPIError(w, apiErr)
		case isClientGone(err):
			s.cfg.Logger.Debug("client went away", "path", r.URL.Path)
		default:
			s.cfg.Logger.Error("request failed", "path", r.URL.Path, "error", err)
			writeAPIError(w, &apiError{status: http.StatusInternalServerError, message: "internal error while rendering"})
		}
	})
}

func (s *Server) authorized(r *http.Request) bool {
	if s.cfg.Token == "" {
		return true
	}
	given, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
	if !ok {
		return false
	}

	// Comparing fixed-size digests keeps the comparison constant-time and
	// independent of the token length.
	want, got := sha256.Sum256([]byte(s.cfg.Token)), sha256.Sum256([]byte(given))

	return subtle.ConstantTimeCompare(want[:], got[:]) == 1
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		next.ServeHTTP(w, r)
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (s *Server) logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(rec, r)

		s.cfg.Logger.Info("request", "method", r.Method, "path", r.URL.Path,
			"status", rec.status, "duration", time.Since(start).Round(time.Millisecond))
	})
}

func (s *Server) recoverPanics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if v := recover(); v != nil {
				s.cfg.Logger.Error("panic in handler", "panic", v, "stack", string(debug.Stack()))
				writeAPIError(w, &apiError{status: http.StatusInternalServerError, message: "internal error"})
			}
		}()
		next.ServeHTTP(w, r)
	})
}
