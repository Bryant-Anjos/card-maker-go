package chrome_test

import (
	"image"
	"image/color"
	"testing"
)

// flat returns a w x h image filled with c.
func flat(w, h int, c color.NRGBA) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
			img.SetNRGBA(x, y, c)
		}
	}

	return img
}

func shifted(img *image.NRGBA, c color.NRGBA, points ...image.Point) *image.NRGBA {
	out := image.NewNRGBA(img.Bounds())
	copy(out.Pix, img.Pix)
	for _, p := range points {
		out.SetNRGBA(p.X, p.Y, c)
	}

	return out
}

// The strict comparison must ignore the rendering noise a loaded machine
// produces (a few pixels off by a few levels) yet still catch any real change.
func TestStrictComparisonTolerance(t *testing.T) {
	const w, h = 400, 560
	base := flat(w, h, color.NRGBA{R: 100, G: 100, B: 100, A: 255})

	scattered := func(n int, c color.NRGBA) *image.NRGBA {
		var pts []image.Point
		for i := range n {
			pts = append(pts, image.Pt((i*37)%w, (i*53)%h))
		}
		return shifted(base, c, pts...)
	}

	tests := []struct {
		name     string
		actual   image.Image
		wantFail bool
	}{
		{"identical", base, false},
		{"noise seen under load: 45 pixels off by 3 levels", scattered(45, color.NRGBA{R: 103, G: 97, B: 100, A: 255}), false},
		{"noise: 400 pixels off by 6 levels", scattered(400, color.NRGBA{R: 106, G: 100, B: 100, A: 255}), false},
		{"a changed glyph: 300 pixels off by 120 levels", scattered(300, color.NRGBA{R: 220, G: 220, B: 220, A: 255}), true},
		{"a single pixel off by 40 levels", scattered(1, color.NRGBA{R: 140, G: 100, B: 100, A: 255}), true},
		{"a subtle global color change of 5 levels", flat(w, h, color.NRGBA{R: 105, G: 100, B: 100, A: 255}), true},
		{"a subtle global change of 3 levels in every channel", flat(w, h, color.NRGBA{R: 103, G: 103, B: 103, A: 255}), true},
		{"alpha change", flat(w, h, color.NRGBA{R: 100, G: 100, B: 100, A: 200}), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stats := compare(t, base, tt.actual)

			if got := stats.exceedsStrict(); got != tt.wantFail {
				t.Errorf("exceedsStrict() = %v, want %v (changedPixels=%d meanDiff=%.4f)",
					got, tt.wantFail, stats.changedPixels, stats.meanDiff)
			}
		})
	}
}
