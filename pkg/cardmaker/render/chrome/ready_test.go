package chrome_test

import (
	"context"
	"image/color"
	"testing"
	"testing/fstest"
	"time"

	"github.com/Bryant-Anjos/card-maker-go/pkg/cardmaker/i18n"
	"github.com/Bryant-Anjos/card-maker-go/pkg/cardmaker/render"
	"github.com/Bryant-Anjos/card-maker-go/pkg/cardmaker/render/chrome"
)

// readyTheme is a theme whose card template defines window.cardReady, the way a
// theme does when it has asynchronous work (such as fitting text) to finish
// before the screenshot.
func readyTheme(script string) func() render.CardRequest {
	return func() render.CardRequest {
		th := testTheme()
		th.FS = fstest.MapFS{
			"card.html.tmpl": {Data: []byte(`<!doctype html><html><head>
<link rel="stylesheet" href="{{.Stylesheet}}"><script>` + script + `</script></head><body></body></html>`)},
			"style.css": {Data: []byte(`html, body { margin: 0; width: 100%; height: 100%; }`)},
		}

		return render.CardRequest{Theme: th, Card: i18n.ResolvedCard{ID: "a", Rarity: "rare", Name: "A"}, Scale: 1}
	}
}

func TestRendererWaitsForCardReady(t *testing.T) {
	r := newRenderer(t)
	// The page turns red only after 300 ms: a screenshot taken without waiting
	// for cardReady would still be transparent.
	req := readyTheme(`window.cardReady = new Promise((resolve) => setTimeout(() => {
		document.body.style.background = 'rgb(255, 0, 0)';
		resolve();
	}, 300));`)()

	data, err := r.RenderCard(context.Background(), req)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := nrgba(decode(t, data), 20, 28); !near(got, color.NRGBA{R: 255, A: 255}) {
		t.Errorf("pixel = %v, want red: the screenshot was taken before cardReady resolved", got)
	}
}

func TestRendererWorksWithoutCardReady(t *testing.T) {
	r := newRenderer(t)

	// The plain test theme defines no cardReady; rendering must not depend on it.
	if _, err := r.RenderCard(context.Background(), cardRequest(t, 1)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRendererFailsWhenCardReadyRejects(t *testing.T) {
	r := newRenderer(t)
	req := readyTheme(`window.cardReady = Promise.reject(new Error('fit exploded'));`)()

	_, err := r.RenderCard(context.Background(), req)

	if err == nil {
		t.Fatal("expected an error when the theme's cardReady rejects")
	}
}

func TestRendererTimesOutWhenCardReadyNeverResolves(t *testing.T) {
	if _, err := chrome.FindBrowser(""); err != nil {
		t.Skipf("no browser available: %v", err)
	}
	r, err := chrome.New(chrome.Options{Timeout: 700 * time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = r.Close() })
	req := readyTheme(`window.cardReady = new Promise(() => {});`)()

	start := time.Now()
	_, err = r.RenderCard(context.Background(), req)

	if err == nil {
		t.Fatal("expected a timeout error")
	}
	if elapsed := time.Since(start); elapsed > 10*time.Second {
		t.Errorf("took %v to give up, want it to respect the %v timeout", elapsed, 700*time.Millisecond)
	}
}
