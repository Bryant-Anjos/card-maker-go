package chrome

import (
	"bytes"
	"image"
	"image/png"
	"testing"

	"github.com/chromedp/chromedp"

	"github.com/Bryant-Anjos/card-maker-go/pkg/cardmaker/i18n"
	"github.com/Bryant-Anjos/card-maker-go/pkg/cardmaker/theme"
)

const defaultThemeDir = "../../../../themes/default"

// nameMetrics is what the browser reports about the card name after the page
// finished loading and fitting.
type nameMetrics struct {
	Size     float64 `json:"size"`     // computed font size in px
	Overflow bool    `json:"overflow"` // text wider than its box (shown with an ellipsis)
}

// measureName renders the default theme page of a card in the browser and
// reads the final font size of the card name.
func measureName(t *testing.T, r *Renderer, th theme.Theme, card i18n.ResolvedCard) nameMetrics {
	t.Helper()

	html, err := cardPage(th, card)
	if err != nil {
		t.Fatalf("cardPage: %v", err)
	}
	url, release := r.server.register(pageContent{html: html, art: tinyPNG(t), fsys: th.FS})
	defer release()

	ctx, cancel := chromedp.NewContext(r.browserCtx)
	defer cancel()

	if err := r.loadPage(ctx, url, th.Size, 1); err != nil {
		t.Fatalf("load page: %v", err)
	}

	metrics, err := chromedp.Run(ctx, chromedp.Evaluate[nameMetrics](`(() => {
		const el = document.querySelector('.name');
		return { size: parseFloat(getComputedStyle(el).fontSize), overflow: el.scrollWidth > el.clientWidth };
	})()`))
	if err != nil {
		t.Fatalf("measure name: %v", err)
	}

	return metrics
}

func tinyPNG(t *testing.T) []byte {
	t.Helper()

	var buf bytes.Buffer
	if err := png.Encode(&buf, image.NewNRGBA(image.Rect(0, 0, 4, 4))); err != nil {
		t.Fatal(err)
	}

	return buf.Bytes()
}

func TestDefaultThemeFitsLongNames(t *testing.T) {
	if _, err := FindBrowser(""); err != nil {
		t.Skipf("no browser available: %v", err)
	}
	r, err := New(Options{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = r.Close() })

	th, err := theme.LoadDir(defaultThemeDir)
	if err != nil {
		t.Fatalf("load default theme: %v", err)
	}

	const baseSize, minSize = 62.0, 28.0

	tests := []struct {
		name    string
		text    string
		variant string
		check   func(t *testing.T, m nameMetrics)
	}{
		{"short name keeps the base size", "Vulpin", "", func(t *testing.T, m nameMetrics) {
			if m.Size != baseSize || m.Overflow {
				t.Errorf("metrics = %+v, want the untouched base size", m)
			}
		}},
		{"36 characters still fit, at about the minimum", "Drakarion, Senhor das Cinzas Eternas", "", func(t *testing.T, m nameMetrics) {
			if m.Size < minSize || m.Size > minSize+4 || m.Overflow {
				t.Errorf("metrics = %+v, want a size just above %v that fits", m, minSize)
			}
		}},
		{"long name shrinks to fit", "Leviatã das Marés Profundas", "", func(t *testing.T, m nameMetrics) {
			if m.Size >= baseSize || m.Size < minSize || m.Overflow {
				t.Errorf("metrics = %+v, want a size between %v and %v that fits", m, minSize, baseSize)
			}
		}},
		{"very long name stops at the minimum and is ellipsized", "Drakarion, Senhor das Cinzas Eternas e Esquecidas do Mundo", "", func(t *testing.T, m nameMetrics) {
			if m.Size != minSize || !m.Overflow {
				t.Errorf("metrics = %+v, want the minimum size %v with an ellipsis", m, minSize)
			}
		}},
		{"full art variant fits too", "Leviatã das Marés Profundas", "fullart", func(t *testing.T, m nameMetrics) {
			if m.Size >= baseSize || m.Size < minSize || m.Overflow {
				t.Errorf("metrics = %+v, want a size between %v and %v that fits", m, minSize, baseSize)
			}
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			card := i18n.ResolvedCard{ID: "x", Number: 7, Rarity: "rare", Element: "dark", Variant: tt.variant, Name: tt.text}

			tt.check(t, measureName(t, r, th, card))
		})
	}

}
