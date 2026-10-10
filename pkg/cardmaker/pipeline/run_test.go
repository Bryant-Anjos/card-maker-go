package pipeline_test

import (
	"context"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/image/webp"

	"github.com/Bryant-Anjos/card-maker-go/pkg/cardmaker/model"
	"github.com/Bryant-Anjos/card-maker-go/pkg/cardmaker/pipeline"
	"github.com/Bryant-Anjos/card-maker-go/pkg/cardmaker/render"
)

var cardIDs = []string{"vulpin", "cervallis", "lunaris", "aureon", "drakarion"}

func TestRunWritesImagesAndManifest(t *testing.T) {
	ws := newWorkspace(t)
	fake := render.NewFake()

	result := mustRun(t, ws.options(), fake)

	file, err := os.Open(filepath.Join(ws.localeDir(), "manifest.json"))
	if err != nil {
		t.Fatalf("manifest missing: %v", err)
	}
	defer file.Close()
	manifest, err := model.ParseManifest(file)
	if err != nil {
		t.Fatalf("manifest invalid: %v", err)
	}

	if manifest.Theme != "basic" || manifest.Locale != "pt-BR" || manifest.Size != (model.Size{Width: 75, Height: 105}) {
		t.Errorf("manifest header = %+v", manifest)
	}
	if len(manifest.Cards) != len(cardIDs) {
		t.Fatalf("manifest has %d cards, want %d", len(manifest.Cards), len(cardIDs))
	}
	for i, card := range manifest.Cards {
		if card.ID != cardIDs[i] {
			t.Errorf("card %d = %q, want %q (input order)", i, card.ID, cardIDs[i])
		}
		if len(card.Images) != 4 {
			t.Errorf("%s has %d images, want 4 (2 scales x 2 formats)", card.ID, len(card.Images))
		}
		for _, img := range card.Images {
			if _, err := os.Stat(filepath.Join(ws.localeDir(), filepath.FromSlash(img.Path))); err != nil {
				t.Errorf("%s: listed file missing: %v", card.ID, err)
			}
		}
	}
	if manifest.Cards[0].Name != "Vulpin" || manifest.Cards[0].Number != 1 {
		t.Errorf("first card = %+v", manifest.Cards[0])
	}
	if len(manifest.Back) != 4 {
		t.Errorf("back has %d images, want 4", len(manifest.Back))
	}
	if result.Rendered != 6*2 || result.Skipped != 0 {
		t.Errorf("result = %+v, want 12 rendered (5 cards + back, 2 scales) and 0 skipped", result)
	}
}

func TestRunImageFilesAreCorrect(t *testing.T) {
	ws := newWorkspace(t)
	mustRun(t, ws.options(), render.NewFake())

	t.Run("png at 1x", func(t *testing.T) {
		cfg := decodePNGConfig(t, filepath.Join(ws.localeDir(), "cards", "vulpin.png"))
		if cfg.Width != 75 || cfg.Height != 105 {
			t.Errorf("size = %dx%d", cfg.Width, cfg.Height)
		}
	})

	t.Run("png at 2x uses @2x suffix", func(t *testing.T) {
		cfg := decodePNGConfig(t, filepath.Join(ws.localeDir(), "cards", "vulpin@2x.png"))
		if cfg.Width != 150 || cfg.Height != 210 {
			t.Errorf("size = %dx%d", cfg.Width, cfg.Height)
		}
	})

	t.Run("webp decodes with right size", func(t *testing.T) {
		file, err := os.Open(filepath.Join(ws.localeDir(), "cards", "lunaris@2x.webp"))
		if err != nil {
			t.Fatal(err)
		}
		defer file.Close()
		cfg, err := webp.DecodeConfig(file)
		if err != nil {
			t.Fatalf("invalid webp: %v", err)
		}
		if cfg.Width != 150 || cfg.Height != 210 {
			t.Errorf("size = %dx%d", cfg.Width, cfg.Height)
		}
	})

	t.Run("back", func(t *testing.T) {
		if cfg := decodePNGConfig(t, filepath.Join(ws.localeDir(), "back.png")); cfg.Width != 75 {
			t.Errorf("width = %d", cfg.Width)
		}
	})
}

func decodePNGConfig(t *testing.T, path string) (cfg struct{ Width, Height int }) {
	t.Helper()

	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	c, err := png.DecodeConfig(file)
	if err != nil {
		t.Fatalf("%s: %v", path, err)
	}

	return struct{ Width, Height int }{c.Width, c.Height}
}

func TestRunCaching(t *testing.T) {
	t.Run("second run renders nothing", func(t *testing.T) {
		ws := newWorkspace(t)
		fake := render.NewFake()
		mustRun(t, ws.options(), fake)
		before := fake.CardCalls() + fake.BackCalls()

		result := mustRun(t, ws.options(), fake)

		if got := fake.CardCalls() + fake.BackCalls(); got != before {
			t.Errorf("renderer called %d more times on a cached run", got-before)
		}
		if result.Rendered != 0 || result.Skipped != 12 {
			t.Errorf("result = %+v, want 0 rendered and 12 skipped", result)
		}
	})

	t.Run("changed art re-renders only that card", func(t *testing.T) {
		ws := newWorkspace(t)
		fake := render.NewFake()
		mustRun(t, ws.options(), fake)
		before := fake.CardCalls()

		writeSolidPNG(t, ws.path("art/lunaris.png"), color.NRGBA{R: 1, G: 2, B: 3, A: 255})
		mustRun(t, ws.options(), fake)

		if got := fake.CardCalls() - before; got != 2 {
			t.Errorf("rendered %d card images, want 2 (one card, two scales)", got)
		}
	})

	t.Run("changed text re-renders only that card", func(t *testing.T) {
		ws := newWorkspace(t)
		fake := render.NewFake()
		mustRun(t, ws.options(), fake)
		before := fake.CardCalls()

		replaceInFile(t, ws.cardsFile, "Sua pelagem brilha", "Sua pelagem reluz")
		mustRun(t, ws.options(), fake)

		if got := fake.CardCalls() - before; got != 2 {
			t.Errorf("rendered %d card images, want 2", got)
		}
	})

	t.Run("changed theme re-renders everything", func(t *testing.T) {
		ws := newWorkspace(t)
		fake := render.NewFake()
		mustRun(t, ws.options(), fake)
		cardsBefore, backBefore := fake.CardCalls(), fake.BackCalls()

		replaceInFile(t, filepath.Join(ws.themeDir, "style.css"), "margin: 0", "margin: 1px")
		mustRun(t, ws.options(), fake)

		if got := fake.CardCalls() - cardsBefore; got != 10 {
			t.Errorf("rendered %d card images, want 10", got)
		}
		if got := fake.BackCalls() - backBefore; got != 2 {
			t.Errorf("rendered %d back images, want 2", got)
		}
	})

	t.Run("changed output options re-render", func(t *testing.T) {
		ws := newWorkspace(t)
		fake := render.NewFake()
		opts := ws.options()
		opts.Scales = []int{1}
		mustRun(t, opts, fake)
		before := fake.CardCalls()

		opts.Scales = []int{1, 2}
		mustRun(t, opts, fake)

		if got := fake.CardCalls() - before; got != 10 {
			t.Errorf("rendered %d card images, want 10 (both scales for every card)", got)
		}
	})

	t.Run("deleted output is regenerated", func(t *testing.T) {
		ws := newWorkspace(t)
		fake := render.NewFake()
		mustRun(t, ws.options(), fake)
		before := fake.CardCalls()

		if err := os.Remove(filepath.Join(ws.localeDir(), "cards", "aureon@2x.webp")); err != nil {
			t.Fatal(err)
		}
		mustRun(t, ws.options(), fake)

		if got := fake.CardCalls() - before; got != 2 {
			t.Errorf("rendered %d card images, want 2", got)
		}
	})

	t.Run("force ignores the cache", func(t *testing.T) {
		ws := newWorkspace(t)
		fake := render.NewFake()
		mustRun(t, ws.options(), fake)
		before := fake.CardCalls()

		opts := ws.options()
		opts.Force = true
		mustRun(t, opts, fake)

		if got := fake.CardCalls() - before; got != 10 {
			t.Errorf("rendered %d card images, want 10", got)
		}
	})

	t.Run("corrupt cache file is tolerated", func(t *testing.T) {
		ws := newWorkspace(t)
		fake := render.NewFake()
		mustRun(t, ws.options(), fake)
		if err := os.WriteFile(filepath.Join(ws.localeDir(), ".cache.json"), []byte("{not json"), 0o644); err != nil {
			t.Fatal(err)
		}

		result := mustRun(t, ws.options(), fake)

		if result.Rendered != 12 {
			t.Errorf("rendered %d, want everything rebuilt", result.Rendered)
		}
	})
}

func TestRunDeterministicAcrossConcurrency(t *testing.T) {
	read := func(concurrency int) string {
		ws := newWorkspace(t)
		opts := ws.options()
		opts.Concurrency = concurrency
		mustRun(t, opts, render.NewFake())
		data, err := os.ReadFile(filepath.Join(ws.localeDir(), "manifest.json"))
		if err != nil {
			t.Fatal(err)
		}

		return string(data)
	}

	if serial, parallel := read(1), read(8); serial != parallel {
		t.Error("manifest differs between concurrency 1 and 8")
	}
}

func TestRunCardVariant(t *testing.T) {
	ws := newWorkspace(t)
	fake := render.NewFake()
	mustRun(t, ws.options(), fake)
	before := fake.CardCalls()

	replaceInFile(t, ws.cardsFile, `"rarity": "rare",`, `"rarity": "rare", "variant": "fullart",`)
	result := mustRun(t, ws.options(), fake)

	if got := result.Manifest.Cards[2]; got.ID != "lunaris" || got.Variant != "fullart" {
		t.Errorf("manifest card = %+v, want lunaris with variant fullart", got)
	}
	if result.Manifest.Cards[0].Variant != "" {
		t.Errorf("standard card has variant %q", result.Manifest.Cards[0].Variant)
	}
	if got := fake.CardCalls() - before; got != 2 {
		t.Errorf("rendered %d card images, want 2 (only the card whose variant changed)", got)
	}
}

func TestRunThemeWithoutBack(t *testing.T) {
	ws := newWorkspace(t)
	replaceInFile(t, filepath.Join(ws.themeDir, "theme.json"), `"back": "back.html.tmpl",`, "")
	fake := render.NewFake()

	mustRun(t, ws.options(), fake)

	if fake.BackCalls() != 0 {
		t.Errorf("back rendered %d times, want 0", fake.BackCalls())
	}
	if _, err := os.Stat(filepath.Join(ws.localeDir(), "back.png")); err == nil {
		t.Error("back.png should not exist")
	}
}

func TestRunFailsBeforeRenderingOnBadInput(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(t *testing.T, ws workspace)
		wantErr []string
	}{
		{"unknown rarity", func(t *testing.T, ws workspace) {
			replaceInFile(t, ws.cardsFile, `"rarity": "rare"`, `"rarity": "mythic"`)
		}, []string{"cards[2].rarity", "mythic"}},
		{"unknown variant", func(t *testing.T, ws workspace) {
			replaceInFile(t, ws.cardsFile, `"rarity": "rare",`, `"rarity": "rare", "variant": "etched",`)
		}, []string{"cards[2].variant", "etched"}},
		{"unknown element", func(t *testing.T, ws workspace) {
			replaceInFile(t, ws.cardsFile, `"element": "fire"`, `"element": "void"`)
		}, []string{"cards[4].element", "void"}},
		{"missing art file", func(t *testing.T, ws workspace) {
			if err := os.Remove(ws.path("art/aureon.png")); err != nil {
				t.Fatal(err)
			}
		}, []string{"aureon", "art/aureon.png"}},
		{"art is not an image", func(t *testing.T, ws workspace) {
			if err := os.WriteFile(ws.path("art/vulpin.png"), []byte("not an image"), 0o644); err != nil {
				t.Fatal(err)
			}
		}, []string{"vulpin", "art/vulpin.png", "image"}},
		{"locale without translation", func(t *testing.T, ws workspace) {}, []string{"no name for locale"}},
		{"malformed cards file", func(t *testing.T, ws workspace) {
			if err := os.WriteFile(ws.cardsFile, []byte("{"), 0o644); err != nil {
				t.Fatal(err)
			}
		}, []string{"decode"}},
		{"theme file missing", func(t *testing.T, ws workspace) {
			if err := os.Remove(filepath.Join(ws.themeDir, "style.css")); err != nil {
				t.Fatal(err)
			}
		}, []string{"stylesheet"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ws := newWorkspace(t)
			tt.mutate(t, ws)
			opts := ws.options()
			if tt.name == "locale without translation" {
				opts.Locale, opts.Fallback = "fr", "de"
			}
			fake := render.NewFake()

			_, err := pipeline.Run(context.Background(), opts, fake)

			if err == nil {
				t.Fatal("expected error, got nil")
			}
			for _, want := range tt.wantErr {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("error %q does not contain %q", err, want)
				}
			}
			if fake.CardCalls() != 0 || fake.BackCalls() != 0 {
				t.Error("renderer must not run when inputs are invalid")
			}
		})
	}
}

func TestRunRendererProblems(t *testing.T) {
	t.Run("renderer error is reported", func(t *testing.T) {
		ws := newWorkspace(t)

		_, err := pipeline.Run(context.Background(), ws.options(), failingRenderer{render.NewFake()})

		if err == nil || !strings.Contains(err.Error(), "browser exploded") {
			t.Fatalf("error = %v, want it to carry the renderer error", err)
		}
	})

	t.Run("wrong image size is rejected", func(t *testing.T) {
		ws := newWorkspace(t)

		_, err := pipeline.Run(context.Background(), ws.options(), wrongSizeRenderer{render.NewFake()})

		if err == nil || !strings.Contains(err.Error(), "150x210") {
			t.Fatalf("error = %v, want a size mismatch mentioning 150x210", err)
		}
	})

	t.Run("cancelled context", func(t *testing.T) {
		ws := newWorkspace(t)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		if _, err := pipeline.Run(ctx, ws.options(), render.NewFake()); err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestValidateDoesNotRender(t *testing.T) {
	ws := newWorkspace(t)

	count, err := pipeline.Validate(ws.options())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != len(cardIDs) {
		t.Errorf("count = %d, want %d", count, len(cardIDs))
	}
	if _, err := os.Stat(ws.outDir); err == nil {
		t.Error("Validate must not create the output directory")
	}

	replaceInFile(t, ws.cardsFile, `"rarity": "rare"`, `"rarity": "mythic"`)
	if _, err := pipeline.Validate(ws.options()); err == nil || !strings.Contains(err.Error(), "mythic") {
		t.Errorf("error = %v, want it to flag the unknown rarity", err)
	}
}

func TestOptionsValidation(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(o *pipeline.Options)
		wantErr string
	}{
		{"missing cards file", func(o *pipeline.Options) { o.CardsFile = "" }, "cards"},
		{"missing theme dir", func(o *pipeline.Options) { o.ThemeDir = "" }, "theme"},
		{"missing out dir", func(o *pipeline.Options) { o.OutDir = "" }, "out"},
		{"invalid locale", func(o *pipeline.Options) { o.Locale = "português" }, "locale"},
		{"zero scale", func(o *pipeline.Options) { o.Scales = []int{0} }, "scale"},
		{"unsupported format", func(o *pipeline.Options) { o.Formats = []string{"gif"} }, "format"},
		{"negative concurrency", func(o *pipeline.Options) { o.Concurrency = -1 }, "concurrency"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ws := newWorkspace(t)
			opts := ws.options()
			tt.mutate(&opts)

			_, err := pipeline.Run(context.Background(), opts, render.NewFake())

			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("error = %v, want it to mention %q", err, tt.wantErr)
			}
		})
	}

	t.Run("defaults apply when scales and formats are empty", func(t *testing.T) {
		ws := newWorkspace(t)
		opts := ws.options()
		opts.Scales, opts.Formats = nil, nil

		result := mustRun(t, opts, render.NewFake())

		if len(result.Manifest.Cards[0].Images) != 2 {
			t.Errorf("default images = %d, want 2 (1x and 2x png)", len(result.Manifest.Cards[0].Images))
		}
	})
}
