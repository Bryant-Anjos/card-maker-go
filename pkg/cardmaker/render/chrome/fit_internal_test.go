package chrome

import (
	"bytes"
	"image"
	"image/png"
	"os"
	"strings"
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
	// Clipped is Overflow measured with the real, fractional widths. scrollWidth
	// and clientWidth are whole pixels, so a text wider than its box by less than
	// a pixel is not Overflow, yet the browser still cuts it with an ellipsis.
	Clipped bool `json:"clipped"`
}

// descriptionMetrics is what the browser reports about the card description
// after the page finished loading and fitting.
type descriptionMetrics struct {
	Size     float64 `json:"size"`     // computed font size in px
	Overflow bool    `json:"overflow"` // text taller than its box (cut off without an ellipsis)
	Clamp    string  `json:"clamp"`    // -webkit-line-clamp once the text no longer fits at the minimum
	Lines    float64 `json:"lines"`    // lines the text takes (scrollHeight / line height), clamp aside
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
		const range = document.createRange();
		range.selectNodeContents(el);
		return {
			size: parseFloat(getComputedStyle(el).fontSize),
			overflow: el.scrollWidth > el.clientWidth,
			clipped: range.getBoundingClientRect().width > el.getBoundingClientRect().width + 0.01,
		};
	})()`))
	if err != nil {
		t.Fatalf("measure name: %v", err)
	}

	return metrics
}

// measureDescription renders the card page and reads how the description was
// fitted into its box.
func measureDescription(t *testing.T, r *Renderer, th theme.Theme, card i18n.ResolvedCard) descriptionMetrics {
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

	metrics, err := chromedp.Run(ctx, chromedp.Evaluate[descriptionMetrics](`(() => {
		const el = document.querySelector('.description');
		const cs = getComputedStyle(el);
		return {
			size: parseFloat(cs.fontSize),
			overflow: el.scrollHeight > parseFloat(cs.maxHeight) + 3,
			clamp: el.style.webkitLineClamp || '',
			lines: el.scrollHeight / parseFloat(cs.lineHeight),
		};
	})()`))
	if err != nil {
		t.Fatalf("measure description: %v", err)
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
	r, err := New(Options{NoSandbox: NoSandboxFromEnv(os.Getenv)})
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

func TestDefaultThemeFitsLongDescriptions(t *testing.T) {
	if _, err := FindBrowser(""); err != nil {
		t.Skipf("no browser available: %v", err)
	}
	r, err := New(Options{NoSandbox: NoSandboxFromEnv(os.Getenv)})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = r.Close() })

	th, err := theme.LoadDir(defaultThemeDir)
	if err != nil {
		t.Fatalf("load default theme: %v", err)
	}

	const baseSize, minSize = 39.0, 26.0
	short := "Pequena criatura curiosa."
	// About 4 lines at the base size: it no longer fits in the 3 lines the box
	// holds, but does fit once the text shrinks.
	long := "Habita regiões verdes e é conhecido por sua grande agilidade entre as árvores, " +
		"guiando viajantes perdidos até a clareira mais próxima antes do anoitecer."
	// Far more than any size can hold.
	huge := strings.Repeat("Texto muito longo que não cabe de jeito nenhum na caixa da descrição. ", 8)

	for _, variant := range []string{"", "fullart"} {
		name := "standard"
		if variant != "" {
			name = variant
		}
		t.Run(name, func(t *testing.T) {
			card := func(text string) i18n.ResolvedCard {
				return i18n.ResolvedCard{ID: "x", Number: 7, Rarity: "rare", Element: "dark", Variant: variant, Name: "Vulpin", Description: text}
			}

			t.Run("a short description keeps the base size", func(t *testing.T) {
				m := measureDescription(t, r, th, card(short))
				if m.Size != baseSize || m.Overflow || m.Clamp != "" {
					t.Errorf("metrics = %+v, want the untouched base size", m)
				}
			})
			t.Run("a long description shrinks until it fits", func(t *testing.T) {
				m := measureDescription(t, r, th, card(long))
				if m.Size >= baseSize || m.Size < minSize || m.Overflow || m.Clamp != "" {
					t.Errorf("metrics = %+v, want a size between %v and %v that fits with no clamp", m, minSize, baseSize)
				}
			})
			t.Run("a huge description stops at the minimum and ends with an ellipsis", func(t *testing.T) {
				m := measureDescription(t, r, th, card(huge))
				// With the clamp on, scrollHeight still reports the whole text, so
				// only the size and the clamp say what happened.
				if m.Size != minSize || m.Clamp == "" {
					t.Errorf("metrics = %+v, want the minimum size %v with a line clamp (ellipsis)", m, minSize)
				}
			})
		})
	}
}

// A name that could still shrink must never be cut: the fit loop used to compare
// whole pixels, so a name that overflowed its box by a fraction of a pixel kept
// that size and the browser replaced its last letters with an ellipsis
// ("Serpente Mari..."). Short names landed on that boundary at random.
func TestDefaultThemeNeverClipsANameThatCouldShrink(t *testing.T) {
	if _, err := FindBrowser(""); err != nil {
		t.Skipf("no browser available: %v", err)
	}
	r, err := New(Options{NoSandbox: NoSandboxFromEnv(os.Getenv)})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = r.Close() })

	th, err := theme.LoadDir(defaultThemeDir)
	if err != nil {
		t.Fatalf("load default theme: %v", err)
	}

	const minSize = 28.0

	// "Serpente Marinha" is cut at 59 px by the whole-pixel comparison; the others
	// are controls that always fit.
	names := []string{"Serpente Marinha", "Vulpin das Brumas", "Guardião do Vale", "Lobo da Montanha"}
	for _, variant := range []string{"", "fullart"} {
		for _, name := range names {
			t.Run(variant+"/"+name, func(t *testing.T) {
				card := i18n.ResolvedCard{ID: "x", Number: 7, Rarity: "epic", Element: "nature", Variant: variant, Name: name}

				m := measureName(t, r, th, card)
				if m.Clipped && m.Size > minSize {
					t.Errorf("metrics = %+v: the name is cut with an ellipsis although it could shrink to %v", m, minSize)
				}
			})
		}
	}
}
