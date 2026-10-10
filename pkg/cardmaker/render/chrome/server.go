package chrome

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// pageContent is everything the browser may fetch while rendering one image:
// the generated HTML, the artwork and the files of the theme.
type pageContent struct {
	html []byte
	art  []byte
	fsys fs.FS
}

// server serves pages to the headless browser. It listens only on loopback,
// on a random port, and every URL carries a random token, so other local
// processes cannot guess its paths.
type server struct {
	listener net.Listener
	http     *http.Server
	token    string

	nextID atomic.Uint64
	pages  sync.Map // id -> pageContent
}

func newServer() (*server, error) {
	tokenBytes := make([]byte, 16)
	if _, err := rand.Read(tokenBytes); err != nil {
		return nil, fmt.Errorf("generate token: %w", err)
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("listen on loopback: %w", err)
	}

	s := &server{listener: listener, token: hex.EncodeToString(tokenBytes)}
	s.http = &http.Server{Handler: s, ReadHeaderTimeout: 10 * time.Second}

	go func() { _ = s.http.Serve(listener) }()

	return s, nil
}

// register makes a page available and returns its URL plus a function that
// removes it again.
func (s *server) register(p pageContent) (url string, release func()) {
	id := strconv.FormatUint(s.nextID.Add(1), 10)
	s.pages.Store(id, p)

	url = fmt.Sprintf("http://%s/%s/%s/%s", s.listener.Addr(), s.token, id, pagePath)

	return url, func() { s.pages.Delete(id) }
}

func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")

	rest, ok := strings.CutPrefix(r.URL.Path, "/"+s.token+"/")
	if !ok || r.Method != http.MethodGet {
		http.NotFound(w, r)
		return
	}
	id, name, _ := strings.Cut(rest, "/")

	value, ok := s.pages.Load(id)
	if !ok {
		http.NotFound(w, r)
		return
	}
	p := value.(pageContent)

	switch {
	case name == pagePath:
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(p.html)
	case name == artPath:
		w.Header().Set("Content-Type", http.DetectContentType(p.art))
		http.ServeContent(w, r, artPath, time.Time{}, bytes.NewReader(p.art))
	case fs.ValidPath(name):
		http.ServeFileFS(w, r, p.fsys, name)
	default:
		http.NotFound(w, r)
	}
}

func (s *server) close() error {
	err := s.http.Close()
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}

	return err
}
