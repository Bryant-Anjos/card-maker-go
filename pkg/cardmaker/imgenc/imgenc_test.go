package imgenc_test

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
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
		got, err := imgenc.Encode(data, imgenc.PNG)

		if err != nil || !bytes.Equal(got, data) {
			t.Errorf("Encode(png) changed the data or failed: %v", err)
		}
	})

	t.Run("webp decodes with the same size", func(t *testing.T) {
		got, err := imgenc.Encode(data, imgenc.WebP)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		cfg, err := webp.DecodeConfig(bytes.NewReader(got))
		if err != nil || cfg.Width != 30 || cfg.Height != 42 {
			t.Errorf("webp config = (%+v, %v)", cfg, err)
		}
	})

	t.Run("unsupported format", func(t *testing.T) {
		if _, err := imgenc.Encode(data, "gif"); err == nil || !strings.Contains(err.Error(), "gif") {
			t.Errorf("error = %v", err)
		}
	})

	t.Run("invalid png for webp", func(t *testing.T) {
		if _, err := imgenc.Encode([]byte("nope"), imgenc.WebP); err == nil {
			t.Error("expected error")
		}
	})
}

func TestEncodeAll(t *testing.T) {
	data := solidPNG(t, 10, 14)

	got, err := imgenc.EncodeAll(data, []string{imgenc.PNG, imgenc.WebP})

	if err != nil || len(got) != 2 || got[imgenc.PNG] == nil || got[imgenc.WebP] == nil {
		t.Fatalf("EncodeAll = (%d formats, %v)", len(got), err)
	}
	if _, err := imgenc.EncodeAll(data, []string{"gif"}); err == nil {
		t.Error("expected error for unsupported format")
	}
}

func TestIsSupported(t *testing.T) {
	if !imgenc.IsSupported("png") || !imgenc.IsSupported("webp") || imgenc.IsSupported("gif") {
		t.Error("IsSupported does not match the supported formats")
	}
}
