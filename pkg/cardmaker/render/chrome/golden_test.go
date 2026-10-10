package chrome_test

import (
	"context"
	"flag"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Bryant-Anjos/card-maker-go/pkg/cardmaker/pipeline"
)

// Run with -update to regenerate the golden images after an intentional
// visual change, then review them before committing.
var update = flag.Bool("update", false, "rewrite the golden images of the default theme")

const (
	repoRoot  = "../../../.."
	goldenDir = repoRoot + "/testdata/golden/default"

	// rendererFile records which renderer (and browser) produced the goldens.
	rendererFile = goldenDir + "/RENDERER"

	// A browser renders the same page pixel-identically every time, so with
	// the browser that produced the goldens the comparison is exact (beyond a
	// rounding margin): even a single changed digit fails.
	maxChannelDiffStrict = 2

	// Other browser builds differ in text antialiasing (measured mean ~0.3),
	// so only a mean tolerance applies: layout, color and asset changes still
	// fail, tiny local changes may not.
	maxMeanDiffTolerant = 1.0
)

var goldenFiles = []string{
	"cards/vulpin.png", "cards/cervallis.png", "cards/lunaris.png",
	"cards/aureon.png", "cards/drakarion.png", "back.png",
}

// TestDefaultThemeGolden renders the sample cards with the default theme and
// compares them with the reviewed golden images.
func TestDefaultThemeGolden(t *testing.T) {
	r := newRenderer(t)
	out := t.TempDir()

	_, err := pipeline.Run(context.Background(), pipeline.Options{
		CardsFile: repoRoot + "/testdata/cards/cards.json",
		ThemeDir:  repoRoot + "/themes/default",
		OutDir:    out,
		Locale:    "pt-BR",
		Scales:    []int{1},
	}, r)
	if err != nil {
		t.Fatalf("build: %v", err)
	}

	if *update {
		if err := os.WriteFile(rendererFile, []byte(r.Version()+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	strict := goldenRenderer(t) == r.Version()
	if !strict {
		t.Logf("goldens were made by %q, this is %q: comparing with tolerance only", goldenRenderer(t), r.Version())
	}

	for _, name := range goldenFiles {
		t.Run(name, func(t *testing.T) {
			actualPath := filepath.Join(out, "pt-BR", filepath.FromSlash(name))
			goldenPath := filepath.Join(goldenDir, filepath.FromSlash(name))

			if *update {
				copyFile(t, actualPath, goldenPath)
				return
			}

			stats := compare(t, readPNG(t, goldenPath), readPNG(t, actualPath))
			switch {
			case strict && stats.changedPixels > 0:
				t.Errorf("%d pixels differ from the golden; actual image kept at %s (run with -update if intended)",
					stats.changedPixels, keep(t, actualPath, name))
			case !strict && stats.meanDiff > maxMeanDiffTolerant:
				t.Errorf("mean difference %.2f exceeds %.2f; actual image kept at %s (run with -update if intended)",
					stats.meanDiff, maxMeanDiffTolerant, keep(t, actualPath, name))
			}
		})
	}
}

func goldenRenderer(t *testing.T) string {
	t.Helper()

	data, err := os.ReadFile(rendererFile)
	if err != nil {
		return ""
	}

	return strings.TrimSpace(string(data))
}

func readPNG(t *testing.T, path string) image.Image {
	t.Helper()

	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("open %s: %v (run with -update to create goldens)", path, err)
	}
	defer file.Close()

	img, err := png.Decode(file)
	if err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}

	return img
}

type diffStats struct {
	meanDiff      float64 // mean absolute difference per RGBA channel, 0-255
	changedPixels int     // pixels with a channel off by more than maxChannelDiffStrict
}

func compare(t *testing.T, want, got image.Image) diffStats {
	t.Helper()

	if want.Bounds() != got.Bounds() {
		t.Fatalf("size = %v, want %v", got.Bounds().Size(), want.Bounds().Size())
	}

	var total uint64
	changed := 0
	b := want.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r1, g1, b1, a1 := want.At(x, y).RGBA()
			r2, g2, b2, a2 := got.At(x, y).RGBA()
			diffs := [4]uint64{absDiff(r1, r2), absDiff(g1, g2), absDiff(b1, b2), absDiff(a1, a2)}

			worst := uint64(0)
			for _, d := range diffs {
				total += d
				worst = max(worst, d)
			}
			if worst/257 > maxChannelDiffStrict { // RGBA() is 16-bit
				changed++
			}
		}
	}

	channels := float64(b.Dx() * b.Dy() * 4)

	return diffStats{meanDiff: float64(total) / channels / 257, changedPixels: changed}
}

func absDiff(a, b uint32) uint64 {
	if a > b {
		return uint64(a - b)
	}

	return uint64(b - a)
}

func copyFile(t *testing.T, from, to string) {
	t.Helper()

	data, err := os.ReadFile(from)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(to), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(to, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

// keep copies a failing image somewhere that survives the test run.
func keep(t *testing.T, path, name string) string {
	t.Helper()

	dir, err := os.MkdirTemp("", "cardmaker-golden-*")
	if err != nil {
		return path
	}
	target := filepath.Join(dir, filepath.Base(name))
	copyFile(t, path, target)

	return target
}
