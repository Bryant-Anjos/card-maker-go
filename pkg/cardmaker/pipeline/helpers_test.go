package pipeline_test

import (
	"context"
	"errors"
	"image"
	"image/color"
	"image/png"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Bryant-Anjos/card-maker-go/pkg/cardmaker/pipeline"
	"github.com/Bryant-Anjos/card-maker-go/pkg/cardmaker/render"
)

const fixturesRoot = "../../../testdata"

// workspace is a writable copy of the sample cards and theme.
type workspace struct {
	cardsFile string
	themeDir  string
	outDir    string
}

func newWorkspace(t *testing.T) workspace {
	t.Helper()

	root := t.TempDir()
	copyTree(t, filepath.Join(fixturesRoot, "cards"), filepath.Join(root, "cards"))
	copyTree(t, filepath.Join(fixturesRoot, "themes", "basic"), filepath.Join(root, "theme"))

	return workspace{
		cardsFile: filepath.Join(root, "cards", "cards.json"),
		themeDir:  filepath.Join(root, "theme"),
		outDir:    filepath.Join(root, "out"),
	}
}

func (w workspace) options() pipeline.Options {
	return pipeline.Options{
		CardsFile: w.cardsFile,
		ThemeDir:  w.themeDir,
		OutDir:    w.outDir,
		Locale:    "pt-BR",
		Fallback:  "pt-BR",
		Scales:    []int{1, 2},
		Formats:   []string{"png", "webp"},
	}
}

// localeDir is where the pipeline writes everything for pt-BR.
func (w workspace) localeDir() string { return filepath.Join(w.outDir, "pt-BR") }

func (w workspace) path(rel string) string {
	return filepath.Join(filepath.Dir(w.cardsFile), filepath.FromSlash(rel))
}

func copyTree(t *testing.T, from, to string) {
	t.Helper()

	err := filepath.WalkDir(from, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(from, path)
		target := filepath.Join(to, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		return os.WriteFile(target, data, 0o644)
	})
	if err != nil {
		t.Fatalf("copy %s: %v", from, err)
	}
}

func mustRun(t *testing.T, opts pipeline.Options, r render.Renderer) pipeline.Result {
	t.Helper()

	result, err := pipeline.Run(context.Background(), opts, r)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	return result
}

func replaceInFile(t *testing.T, path, old, replacement string) {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), old) {
		t.Fatalf("%s does not contain %q", path, old)
	}
	updated := strings.Replace(string(data), old, replacement, 1)
	if err := os.WriteFile(path, []byte(updated), 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeSolidPNG(t *testing.T, path string, c color.NRGBA) {
	t.Helper()

	img := image.NewNRGBA(image.Rect(0, 0, 4, 4))
	for i := range img.Pix {
		img.Pix[i] = []uint8{c.R, c.G, c.B, c.A}[i%4]
	}
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if err := png.Encode(file, img); err != nil {
		t.Fatal(err)
	}
}

// wrongSizeRenderer wraps Fake but always renders at scale 1, violating the
// Renderer size contract whenever a larger scale is requested.
type wrongSizeRenderer struct{ *render.Fake }

func (r wrongSizeRenderer) RenderCard(ctx context.Context, req render.CardRequest) ([]byte, error) {
	req.Scale = 1
	return r.Fake.RenderCard(ctx, req)
}

type failingRenderer struct{ *render.Fake }

func (failingRenderer) RenderCard(context.Context, render.CardRequest) ([]byte, error) {
	return nil, errors.New("browser exploded")
}
