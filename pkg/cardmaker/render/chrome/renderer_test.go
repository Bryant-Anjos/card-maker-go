package chrome_test

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"os"
	"strings"
	"sync"
	"testing"
	"testing/fstest"

	"github.com/Bryant-Anjos/card-maker-go/pkg/cardmaker/i18n"
	"github.com/Bryant-Anjos/card-maker-go/pkg/cardmaker/model"
	"github.com/Bryant-Anjos/card-maker-go/pkg/cardmaker/render"
	"github.com/Bryant-Anjos/card-maker-go/pkg/cardmaker/render/chrome"
	"github.com/Bryant-Anjos/card-maker-go/pkg/cardmaker/theme"
)

var _ render.Renderer = (*chrome.Renderer)(nil)

// The card template paints the top half with a color from the stylesheet
// (proving theme files are served) and the bottom half with the artwork,
// inside a transparent margin.
const cardTemplate = `<!doctype html><html><head><link rel="stylesheet" href="{{.Stylesheet}}"></head>
<body><div class="card"><div class="top"></div><img class="art" src="{{.ArtURL}}"></div></body></html>`

const backTemplate = `<!doctype html><html><head><link rel="stylesheet" href="{{.Stylesheet}}"></head>
<body><div class="card back"></div></body></html>`

const stylesheet = `html, body { margin: 0; background: transparent; }
.card { position: absolute; inset: 4px; }
.top { height: 50%; background: rgb(0, 128, 255); }
.art { display: block; width: 100%; height: 50%; object-fit: cover; }
.back { background: rgb(255, 200, 0); }`

func testTheme() theme.Theme {
	return theme.Theme{
		Theme: model.Theme{
			SchemaVersion: 1, Name: "t", Size: model.Size{Width: 40, Height: 56},
			Template: "card.html.tmpl", Stylesheet: "style.css", Back: "back.html.tmpl",
			Rarities: []model.Rarity{{ID: "rare", Color: "#3b82f6", Frame: "card.html.tmpl"}},
		},
		FS: fstest.MapFS{
			"card.html.tmpl": {Data: []byte(cardTemplate)},
			"back.html.tmpl": {Data: []byte(backTemplate)},
			"style.css":      {Data: []byte(stylesheet)},
		},
	}
}

func redPNG(t *testing.T) []byte {
	t.Helper()

	img := image.NewNRGBA(image.Rect(0, 0, 8, 8))
	for i := 0; i < len(img.Pix); i += 4 {
		copy(img.Pix[i:], []uint8{255, 0, 0, 255})
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}

	return buf.Bytes()
}

func newRenderer(t *testing.T) *chrome.Renderer {
	t.Helper()

	if _, err := chrome.FindBrowser(""); err != nil {
		t.Skipf("no browser available: %v", err)
	}
	r, err := chrome.New(chrome.Options{NoSandbox: chrome.NoSandboxFromEnv(os.Getenv)})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { _ = r.Close() })

	return r
}

func cardRequest(t *testing.T, scale int) render.CardRequest {
	return render.CardRequest{
		Theme: testTheme(),
		Card:  i18n.ResolvedCard{ID: "a", Rarity: "rare", Name: "A"},
		Art:   redPNG(t),
		Scale: scale,
	}
}

func decode(t *testing.T, data []byte) image.Image {
	t.Helper()

	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("invalid png: %v", err)
	}

	return img
}

func nrgba(img image.Image, x, y int) color.NRGBA {
	return color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA)
}

func near(a, b color.NRGBA) bool {
	diff := func(x, y uint8) int {
		if x > y {
			return int(x - y)
		}
		return int(y - x)
	}

	return diff(a.R, b.R) <= 8 && diff(a.G, b.G) <= 8 && diff(a.B, b.B) <= 8 && diff(a.A, b.A) <= 8
}

func TestRenderCard(t *testing.T) {
	r := newRenderer(t)

	for _, scale := range []int{1, 2} {
		data, err := r.RenderCard(context.Background(), cardRequest(t, scale))
		if err != nil {
			t.Fatalf("scale %d: %v", scale, err)
		}
		img := decode(t, data)

		if got, want := img.Bounds().Size(), image.Pt(40*scale, 56*scale); got != want {
			t.Errorf("scale %d: size = %v, want %v", scale, got, want)
		}
		if got := nrgba(img, 0, 0); got.A != 0 {
			t.Errorf("scale %d: margin pixel = %v, want transparent", scale, got)
		}
		if got := nrgba(img, 20*scale, 12*scale); !near(got, color.NRGBA{0, 128, 255, 255}) {
			t.Errorf("scale %d: top pixel = %v, want the stylesheet color", scale, got)
		}
		if got := nrgba(img, 20*scale, 44*scale); !near(got, color.NRGBA{255, 0, 0, 255}) {
			t.Errorf("scale %d: art pixel = %v, want the artwork color", scale, got)
		}
	}
}

func TestRenderBack(t *testing.T) {
	r := newRenderer(t)

	data, err := r.RenderBack(context.Background(), render.BackRequest{Theme: testTheme(), Scale: 1})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	img := decode(t, data)
	if got := nrgba(img, 20, 28); !near(got, color.NRGBA{255, 200, 0, 255}) {
		t.Errorf("back pixel = %v", got)
	}
}

func TestRenderConcurrently(t *testing.T) {
	r := newRenderer(t)

	var wg sync.WaitGroup
	errs := make(chan error, 6)
	for range 6 {
		wg.Go(func() {
			_, err := r.RenderCard(context.Background(), cardRequest(t, 1))
			errs <- err
		})
	}
	wg.Wait()
	close(errs)

	for err := range errs {
		if err != nil {
			t.Errorf("concurrent render failed: %v", err)
		}
	}
}

func TestRenderErrors(t *testing.T) {
	r := newRenderer(t)

	t.Run("template error", func(t *testing.T) {
		req := cardRequest(t, 1)
		req.Theme.FS = fstest.MapFS{"card.html.tmpl": {Data: []byte("{{.Nope}}")}}

		_, err := r.RenderCard(context.Background(), req)

		if err == nil || !strings.Contains(err.Error(), "Nope") {
			t.Errorf("error = %v", err)
		}
	})

	t.Run("invalid scale", func(t *testing.T) {
		if _, err := r.RenderCard(context.Background(), cardRequest(t, 0)); err == nil {
			t.Error("expected error")
		}
	})

	t.Run("cancelled context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		if _, err := r.RenderCard(ctx, cardRequest(t, 1)); err == nil {
			t.Error("expected error")
		}
	})
}

func TestVersionAndClose(t *testing.T) {
	r := newRenderer(t)

	if !strings.HasPrefix(r.Version(), "chrome-") || len(r.Version()) <= len("chrome-") {
		t.Errorf("Version = %q, want it to include the browser version", r.Version())
	}
	if err := r.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if _, err := r.RenderCard(context.Background(), cardRequest(t, 1)); err == nil {
		t.Error("expected error after Close")
	}
}

func TestNewWithMissingBrowser(t *testing.T) {
	if _, err := chrome.New(chrome.Options{BrowserPath: "/nope/chrome"}); err == nil {
		t.Error("expected error")
	}
}
