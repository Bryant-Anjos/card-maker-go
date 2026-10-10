package server_test

import (
	"bytes"
	"encoding/json"
	"image"
	"image/png"
	"net/http"
	"strings"
	"testing"
	"time"

	"golang.org/x/image/webp"

	"github.com/Bryant-Anjos/card-maker-go/pkg/cardmaker/render"
	"github.com/Bryant-Anjos/card-maker-go/pkg/cardmaker/server"
)

func TestHealthz(t *testing.T) {
	ts := newServer(t)

	resp := get(t, ts, "/healthz")

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	if env := readEnvelope(t, resp); !env.Success {
		t.Errorf("envelope = %+v", env)
	}
}

func TestListThemes(t *testing.T) {
	ts := newServer(t)

	resp := get(t, ts, "/v1/themes")

	env := readEnvelope(t, resp)
	var themes []struct {
		Name     string `json:"name"`
		Size     struct{ Width, Height int }
		Rarities []string `json:"rarities"`
		Elements []string `json:"elements"`
		Variants []string `json:"variants"`
		HasBack  bool     `json:"hasBack"`
	}
	if err := json.Unmarshal(env.Data, &themes); err != nil || !env.Success {
		t.Fatalf("data = %s, err = %v", env.Data, err)
	}
	if len(themes) != 1 || themes[0].Name != "basic" {
		t.Fatalf("themes = %+v", themes)
	}
	got := themes[0]
	if got.Size.Width != 75 || len(got.Rarities) != 5 || got.Rarities[0] != "common" ||
		len(got.Elements) != 7 || len(got.Variants) != 1 || got.Variants[0] != "fullart" || !got.HasBack {
		t.Errorf("theme summary = %+v", got)
	}
}

func TestRenderCard(t *testing.T) {
	ts := newServer(t)

	tests := []struct {
		name       string
		query      string
		wantType   string
		wantW      int
		wantH      int
		decodeWebP bool
	}{
		{"defaults to png at 1x", "theme=basic", "image/png", 75, 105, false},
		{"scale 2", "theme=basic&scale=2", "image/png", 150, 210, false},
		{"webp", "theme=basic&format=webp", "image/webp", 75, 105, true},
		{"explicit locale", "theme=basic&locale=en&fallback=pt-BR", "image/png", 75, 105, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := postRender(t, ts, tt.query, goodCard, artPNG(t, 8, 8))

			if resp.StatusCode != http.StatusOK {
				t.Fatalf("status = %d, body = %s", resp.StatusCode, readBody(t, resp))
			}
			if got := resp.Header.Get("Content-Type"); got != tt.wantType {
				t.Errorf("content type = %q, want %q", got, tt.wantType)
			}
			if resp.Header.Get("X-Content-Type-Options") != "nosniff" {
				t.Error("missing X-Content-Type-Options: nosniff")
			}
			data := readBody(t, resp)

			var size image.Point
			if tt.decodeWebP {
				cfg, err := webp.DecodeConfig(bytes.NewReader(data))
				if err != nil {
					t.Fatalf("invalid webp: %v", err)
				}
				size = image.Pt(cfg.Width, cfg.Height)
			} else {
				cfg, err := png.DecodeConfig(bytes.NewReader(data))
				if err != nil {
					t.Fatalf("invalid png: %v", err)
				}
				size = image.Pt(cfg.Width, cfg.Height)
			}
			if size != image.Pt(tt.wantW, tt.wantH) {
				t.Errorf("size = %v, want %dx%d", size, tt.wantW, tt.wantH)
			}
		})
	}
}

// Browsers and fetch send a Blob as a file part, so the card must be accepted
// both as a plain field and as a file.
func TestRenderCardAcceptsCardAsFilePart(t *testing.T) {
	ts := newServer(t)
	body, contentType := buildMultipart(t, goodCard, true, artPNG(t, 8, 8))
	req, err := http.NewRequest(http.MethodPost, ts.URL+"/v1/render?theme=basic", body)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", contentType)

	resp := do(t, req)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, body = %s", resp.StatusCode, readBody(t, resp))
	}
}

func TestRenderCardVariant(t *testing.T) {
	ts := newServer(t)
	card := strings.Replace(goodCard, `"rarity":"rare",`, `"rarity":"rare","variant":"fullart",`, 1)

	resp := postRender(t, ts, "theme=basic", card, artPNG(t, 8, 8))

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, body = %s", resp.StatusCode, readBody(t, resp))
	}
}

func TestRenderCardRejectsBadRequests(t *testing.T) {
	ts := newServer(t, func(c *server.Config) { c.MaxArtPixels = 100 })
	art := artPNG(t, 8, 8)

	tests := []struct {
		name       string
		query      string
		card       string
		art        []byte
		wantStatus int
		wantError  string
	}{
		{"missing theme", "", goodCard, art, 400, "theme"},
		{"unknown theme", "theme=nope", goodCard, art, 404, "nope"},
		{"invalid scale", "theme=basic&scale=abc", goodCard, art, 400, "scale"},
		{"scale zero", "theme=basic&scale=0", goodCard, art, 400, "scale"},
		{"scale above the limit", "theme=basic&scale=4", goodCard, art, 400, "scale"},
		{"unsupported format", "theme=basic&format=gif", goodCard, art, 400, "format"},
		{"invalid locale", "theme=basic&locale=português", goodCard, art, 400, "locale"},
		{"missing card part", "theme=basic", "", art, 400, "card"},
		{"missing art part", "theme=basic", goodCard, nil, 400, "art"},
		{"card is not json", "theme=basic", "{", art, 400, "card"},
		{"unknown card field", "theme=basic", `{"id":"a","rarity":"rare","bogus":1,"name":{"pt-BR":"A"}}`, art, 400, "bogus"},
		{"unknown rarity", "theme=basic", strings.Replace(goodCard, `"rare"`, `"mythic"`, 1), art, 400, "mythic"},
		{"unknown variant", "theme=basic", strings.Replace(goodCard, `"rarity":"rare",`, `"rarity":"rare","variant":"etched",`, 1), art, 400, "etched"},
		{"invalid card id", "theme=basic", strings.Replace(goodCard, `"lunaris"`, `"../evil"`, 1), art, 400, "id"},
		{"no name for locale", "theme=basic&locale=fr&fallback=de", goodCard, art, 400, "no name"},
		{"art is not an image", "theme=basic", goodCard, []byte("not an image"), 400, "image"},
		{"art has too many pixels", "theme=basic", goodCard, artPNG(t, 20, 20), 400, "pixels"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := postRender(t, ts, tt.query, tt.card, tt.art)

			if resp.StatusCode != tt.wantStatus {
				t.Fatalf("status = %d, want %d (body %s)", resp.StatusCode, tt.wantStatus, readBody(t, resp))
			}
			env := readEnvelope(t, resp)
			if env.Success || env.Error == nil || !strings.Contains(*env.Error, tt.wantError) {
				t.Errorf("envelope = %+v, want an error containing %q", env, tt.wantError)
			}
		})
	}
}

func TestRenderCardUploadTooLarge(t *testing.T) {
	ts := newServer(t, func(c *server.Config) { c.MaxUploadBytes = 2048 })

	resp := postRender(t, ts, "theme=basic", goodCard, bytes.Repeat([]byte("x"), 10_000))

	if resp.StatusCode != http.StatusRequestEntityTooLarge {
		t.Errorf("status = %d, want 413", resp.StatusCode)
	}
}

func TestRenderCardWrongMethod(t *testing.T) {
	ts := newServer(t)

	resp := get(t, ts, "/v1/render?theme=basic")

	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", resp.StatusCode)
	}
}

func TestRendererFailureDoesNotLeakDetails(t *testing.T) {
	ts := newServer(t, func(c *server.Config) { c.Renderer = failingRenderer{render.NewFake()} })

	resp := postRender(t, ts, "theme=basic", goodCard, artPNG(t, 8, 8))

	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", resp.StatusCode)
	}
	body := string(readBody(t, resp))
	if strings.Contains(body, "secret") || strings.Contains(body, "/home/user") {
		t.Errorf("response leaks internal details: %s", body)
	}
}

func TestBackImage(t *testing.T) {
	ts := newServer(t)

	resp := get(t, ts, "/v1/themes/basic/back?scale=2")

	if resp.StatusCode != http.StatusOK || resp.Header.Get("Content-Type") != "image/png" {
		t.Fatalf("status = %d, type = %q", resp.StatusCode, resp.Header.Get("Content-Type"))
	}
	cfg, err := png.DecodeConfig(bytes.NewReader(readBody(t, resp)))
	if err != nil || cfg.Width != 150 || cfg.Height != 210 {
		t.Errorf("config = (%+v, %v)", cfg, err)
	}

	if got := get(t, ts, "/v1/themes/nope/back").StatusCode; got != http.StatusNotFound {
		t.Errorf("unknown theme: status = %d, want 404", got)
	}
}

func TestAuthentication(t *testing.T) {
	ts := newServer(t, func(c *server.Config) { c.Token = "s3cret-token" })

	tests := []struct {
		name    string
		path    string
		headers []string
		want    int
	}{
		{"health needs no token", "/healthz", nil, http.StatusOK},
		{"missing token", "/v1/themes", nil, http.StatusUnauthorized},
		{"wrong token", "/v1/themes", []string{"Authorization", "Bearer nope"}, http.StatusUnauthorized},
		{"wrong scheme", "/v1/themes", []string{"Authorization", "Basic s3cret-token"}, http.StatusUnauthorized},
		{"right token", "/v1/themes", []string{"Authorization", "Bearer s3cret-token"}, http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := get(t, ts, tt.path, tt.headers...)

			if resp.StatusCode != tt.want {
				t.Errorf("status = %d, want %d", resp.StatusCode, tt.want)
			}
			if tt.want == http.StatusUnauthorized && resp.Header.Get("WWW-Authenticate") == "" {
				t.Error("missing WWW-Authenticate header")
			}
		})
	}

	resp := postRender(t, ts, "theme=basic", goodCard, artPNG(t, 8, 8))
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("render without token: status = %d, want 401", resp.StatusCode)
	}
}

func TestBusyServerAnswers503(t *testing.T) {
	blocker := &blockingRenderer{
		Fake:    render.NewFake(),
		started: make(chan struct{}, 4),
		release: make(chan struct{}),
	}
	ts := newServer(t, func(c *server.Config) {
		c.Renderer = blocker
		c.MaxConcurrent = 1
		c.QueueTimeout = 50 * time.Millisecond
	})

	first := make(chan int, 1)
	go func() { first <- postRender(t, ts, "theme=basic", goodCard, artPNG(t, 8, 8)).StatusCode }()
	<-blocker.started // the only slot is now taken

	resp := postRender(t, ts, "theme=basic", goodCard, artPNG(t, 8, 8))

	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503", resp.StatusCode)
	}
	if resp.Header.Get("Retry-After") == "" {
		t.Error("missing Retry-After header")
	}

	close(blocker.release)
	if got := <-first; got != http.StatusOK {
		t.Errorf("first request status = %d, want 200", got)
	}
}

func TestNewValidatesConfig(t *testing.T) {
	themes, err := server.LoadThemes(themesDir)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := server.New(server.Config{Themes: themes}); err == nil {
		t.Error("expected an error without a renderer")
	}
	if _, err := server.New(server.Config{Renderer: render.NewFake()}); err == nil {
		t.Error("expected an error without themes")
	}
}

func TestLoadThemes(t *testing.T) {
	if _, err := server.LoadThemes("does-not-exist"); err == nil {
		t.Error("expected an error for a missing directory")
	}
	if _, err := server.LoadThemes(t.TempDir()); err == nil || !strings.Contains(err.Error(), "no themes") {
		t.Errorf("error = %v, want 'no themes found'", err)
	}
}
