package imgenc_test

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"math/rand"
	"strings"
	"testing"

	"golang.org/x/image/webp"

	"github.com/Bryant-Anjos/card-maker-go/pkg/cardmaker/imgenc"
)

func solidPNG(t *testing.T, w, h int) []byte {
	t.Helper()

	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
			img.SetNRGBA(x, y, color.NRGBA{R: 200, G: 50, B: 50, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}

	return buf.Bytes()
}

func TestCheckPNGSize(t *testing.T) {
	data := solidPNG(t, 30, 42)

	if err := imgenc.CheckPNGSize(data, 30, 42); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if err := imgenc.CheckPNGSize(data, 60, 84); err == nil || !strings.Contains(err.Error(), "60x84") {
		t.Errorf("error = %v, want a size mismatch mentioning 60x84", err)
	}
	if err := imgenc.CheckPNGSize([]byte("nope"), 1, 1); err == nil {
		t.Error("expected error for invalid png")
	}
}

func TestEncode(t *testing.T) {
	data := solidPNG(t, 30, 42)

	t.Run("png is returned as is", func(t *testing.T) {
		got, err := imgenc.Encode(data, imgenc.PNG, imgenc.Options{})

		if err != nil || !bytes.Equal(got, data) {
			t.Errorf("Encode(png) changed the data or failed: %v", err)
		}
	})

	t.Run("webp decodes with the same size", func(t *testing.T) {
		got, err := imgenc.Encode(data, imgenc.WebP, imgenc.Options{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		cfg, err := webp.DecodeConfig(bytes.NewReader(got))
		if err != nil || cfg.Width != 30 || cfg.Height != 42 {
			t.Errorf("webp config = (%+v, %v)", cfg, err)
		}
	})

	t.Run("unsupported format", func(t *testing.T) {
		if _, err := imgenc.Encode(data, "gif", imgenc.Options{}); err == nil || !strings.Contains(err.Error(), "gif") {
			t.Errorf("error = %v", err)
		}
	})

	t.Run("invalid png for webp", func(t *testing.T) {
		if _, err := imgenc.Encode([]byte("nope"), imgenc.WebP, imgenc.Options{}); err == nil {
			t.Error("expected error")
		}
	})
}

func TestEncodeAll(t *testing.T) {
	data := solidPNG(t, 10, 14)

	got, err := imgenc.EncodeAll(data, []string{imgenc.PNG, imgenc.WebP}, imgenc.Options{})

	if err != nil || len(got) != 2 || got[imgenc.PNG] == nil || got[imgenc.WebP] == nil {
		t.Fatalf("EncodeAll = (%d formats, %v)", len(got), err)
	}
	if _, err := imgenc.EncodeAll(data, []string{"gif"}, imgenc.Options{}); err == nil {
		t.Error("expected error for unsupported format")
	}
}

func TestIsSupported(t *testing.T) {
	if !imgenc.IsSupported("png") || !imgenc.IsSupported("webp") || imgenc.IsSupported("gif") {
		t.Error("IsSupported does not match the supported formats")
	}
}

// noisyCardPNG is a 120x168 image with smooth gradients plus noise, like real
// artwork, so lossy and lossless encoding differ clearly. The four corners are
// fully transparent and everything else is opaque.
func noisyCardPNG(t *testing.T) []byte {
	t.Helper()

	const w, h = 120, 168
	rng := rand.New(rand.NewSource(7))
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
			alpha := uint8(255)
			if x < 6 && y < 6 {
				alpha = 0
			}
			img.SetNRGBA(x, y, color.NRGBA{
				R: uint8(x*2 + rng.Intn(40)),
				G: uint8(y + rng.Intn(40)),
				B: uint8((x+y)/2 + rng.Intn(40)),
				A: alpha,
			})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}

	return buf.Bytes()
}

func decodeWebP(t *testing.T, data []byte) image.Image {
	t.Helper()

	img, err := webp.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("invalid webp: %v", err)
	}

	return img
}

func TestWebPQuality(t *testing.T) {
	data := noisyCardPNG(t)
	original, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}

	lossless, err := imgenc.Encode(data, imgenc.WebP, imgenc.Options{WebPQuality: imgenc.WebPLossless})
	if err != nil {
		t.Fatalf("lossless: %v", err)
	}
	lossy, err := imgenc.Encode(data, imgenc.WebP, imgenc.Options{WebPQuality: 80})
	if err != nil {
		t.Fatalf("lossy: %v", err)
	}
	byDefault, err := imgenc.Encode(data, imgenc.WebP, imgenc.Options{})
	if err != nil {
		t.Fatalf("default: %v", err)
	}
	explicitDefault, err := imgenc.Encode(data, imgenc.WebP, imgenc.Options{WebPQuality: imgenc.DefaultWebPQuality})
	if err != nil {
		t.Fatalf("explicit default: %v", err)
	}

	t.Run("quality 100 is lossless and exact", func(t *testing.T) {
		decoded := decodeWebP(t, lossless)
		b := original.Bounds()
		for y := b.Min.Y; y < b.Max.Y; y++ {
			for x := b.Min.X; x < b.Max.X; x++ {
				r1, g1, b1, a1 := original.At(x, y).RGBA()
				r2, g2, b2, a2 := decoded.At(x, y).RGBA()
				if r1 != r2 || g1 != g2 || b1 != b2 || a1 != a2 {
					t.Fatalf("pixel (%d,%d) differs: lossless output is not exact", x, y)
				}
			}
		}
	})

	t.Run("lower quality gives smaller files", func(t *testing.T) {
		if len(lossy) >= len(lossless) {
			t.Errorf("q80 = %d bytes, lossless = %d bytes: expected q80 to be smaller", len(lossy), len(lossless))
		}
		if len(lossless) >= len(data) {
			t.Logf("lossless webp (%d) is not smaller than png (%d) for this noisy sample", len(lossless), len(data))
		}
	})

	t.Run("zero means the default quality", func(t *testing.T) {
		if !bytes.Equal(byDefault, explicitDefault) {
			t.Error("Options{} and DefaultWebPQuality produced different output")
		}
		if imgenc.DefaultWebPQuality >= imgenc.WebPLossless {
			t.Errorf("default quality %d must be lossy (below %d)", imgenc.DefaultWebPQuality, imgenc.WebPLossless)
		}
	})

	t.Run("lossy keeps the alpha channel exact", func(t *testing.T) {
		decoded := decodeWebP(t, lossy)
		corner := color.NRGBAModel.Convert(decoded.At(2, 2)).(color.NRGBA)
		center := color.NRGBAModel.Convert(decoded.At(60, 84)).(color.NRGBA)
		if corner.A != 0 || center.A != 255 {
			t.Errorf("alpha corner=%d center=%d, want 0 and 255", corner.A, center.A)
		}
	})

	t.Run("lossy output decodes with the original size", func(t *testing.T) {
		if got := decodeWebP(t, lossy).Bounds().Size(); got != image.Pt(120, 168) {
			t.Errorf("size = %v", got)
		}
	})
}

func TestValidateWebPQuality(t *testing.T) {
	for _, ok := range []int{0, 1, 50, 90, 100} {
		if err := imgenc.ValidateWebPQuality(ok); err != nil {
			t.Errorf("quality %d rejected: %v", ok, err)
		}
	}
	for _, bad := range []int{-1, 101, 1000} {
		if err := imgenc.ValidateWebPQuality(bad); err == nil || !strings.Contains(err.Error(), "quality") {
			t.Errorf("quality %d: error = %v, want a quality error", bad, err)
		}
	}

	data := noisyCardPNG(t)
	if _, err := imgenc.Encode(data, imgenc.WebP, imgenc.Options{WebPQuality: 101}); err == nil {
		t.Error("Encode accepted quality 101")
	}
	if _, err := imgenc.EncodeAll(data, []string{imgenc.WebP}, imgenc.Options{WebPQuality: -5}); err == nil {
		t.Error("EncodeAll accepted quality -5")
	}
}
