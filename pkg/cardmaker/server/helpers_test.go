package server_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"image"
	"image/color"
	"image/png"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"testing"

	"github.com/Bryant-Anjos/card-maker-go/pkg/cardmaker/render"
	"github.com/Bryant-Anjos/card-maker-go/pkg/cardmaker/server"
)

const themesDir = "../../../testdata/themes"

const goodCard = `{"id":"lunaris","number":32,"rarity":"rare","element":"dark",` +
	`"name":{"pt-BR":"Lunaris","en":"Lunaris EN"},"description":{"pt-BR":"Brilha."}}`

// envelope is the JSON shape of every non-image response.
type envelope struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data"`
	Error   *string         `json:"error"`
}

func newServer(t *testing.T, mutate ...func(*server.Config)) *httptest.Server {
	t.Helper()

	themes, err := server.LoadThemes(themesDir)
	if err != nil {
		t.Fatalf("LoadThemes: %v", err)
	}
	cfg := server.Config{
		Themes:   themes,
		Renderer: render.NewFake(),
		Logger:   slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
	for _, m := range mutate {
		m(&cfg)
	}

	s, err := server.New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	ts := httptest.NewServer(s.Handler())
	t.Cleanup(ts.Close)

	return ts
}

func artPNG(t *testing.T, w, h int) []byte {
	t.Helper()

	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
			img.SetNRGBA(x, y, color.NRGBA{R: 10, G: 120, B: 200, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}

	return buf.Bytes()
}

// multipartBody builds the body of a render request. A nil card or art omits
// that part. The card is sent as a plain field, like curl -F does.
func multipartBody(t *testing.T, card string, art []byte) (body *bytes.Buffer, contentType string) {
	t.Helper()

	return buildMultipart(t, card, false, art)
}

// buildMultipart builds a render request body. With cardAsFile the card is
// sent as a file part (with a filename), which is what browsers and fetch
// produce when the value is a Blob.
func buildMultipart(t *testing.T, card string, cardAsFile bool, art []byte) (body *bytes.Buffer, contentType string) {
	t.Helper()

	body = &bytes.Buffer{}
	w := multipart.NewWriter(body)
	if card != "" {
		h := textproto.MIMEHeader{}
		disposition := `form-data; name="card"`
		if cardAsFile {
			disposition += `; filename="blob"`
		}
		h.Set("Content-Disposition", disposition)
		h.Set("Content-Type", "application/json")
		part, _ := w.CreatePart(h)
		_, _ = part.Write([]byte(card))
	}
	if art != nil {
		part, _ := w.CreateFormFile("art", "art.png")
		_, _ = part.Write(art)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}

	return body, w.FormDataContentType()
}

func postRender(t *testing.T, ts *httptest.Server, query, card string, art []byte, headers ...string) *http.Response {
	t.Helper()

	body, contentType := multipartBody(t, card, art)
	req, err := http.NewRequest(http.MethodPost, ts.URL+"/v1/render?"+query, body)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", contentType)
	for i := 0; i+1 < len(headers); i += 2 {
		req.Header.Set(headers[i], headers[i+1])
	}

	return do(t, req)
}

func get(t *testing.T, ts *httptest.Server, path string, headers ...string) *http.Response {
	t.Helper()

	req, err := http.NewRequest(http.MethodGet, ts.URL+path, nil)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i+1 < len(headers); i += 2 {
		req.Header.Set(headers[i], headers[i+1])
	}

	return do(t, req)
}

func do(t *testing.T, req *http.Request) *http.Response {
	t.Helper()

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })

	return resp
}

func readEnvelope(t *testing.T, resp *http.Response) envelope {
	t.Helper()

	var env envelope
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		t.Fatalf("response is not a JSON envelope: %v", err)
	}

	return env
}

func readBody(t *testing.T, resp *http.Response) []byte {
	t.Helper()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	return data
}

// blockingRenderer holds every card render until released, to test limits.
type blockingRenderer struct {
	*render.Fake
	started chan struct{}
	release chan struct{}
}

func (b *blockingRenderer) RenderCard(ctx context.Context, req render.CardRequest) ([]byte, error) {
	b.started <- struct{}{}
	select {
	case <-b.release:
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	return b.Fake.RenderCard(ctx, req)
}

type failingRenderer struct{ *render.Fake }

func (failingRenderer) RenderCard(context.Context, render.CardRequest) ([]byte, error) {
	return nil, errors.New("secret internal detail: /home/user/chrome crashed")
}
