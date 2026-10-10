// Package imgenc converts the PNG produced by a renderer into the output
// formats of a build and checks that renderers honored their size contract.
//
// WebP is encoded with libwebp compiled to WebAssembly and translated to Go,
// so it needs neither cgo nor a system library.
package imgenc

import (
	"bytes"
	"fmt"
	"image"
	"image/png"

	"github.com/gen2brain/webp"
)

// Supported output formats.
const (
	PNG  = "png"
	WebP = "webp"
)

// WebP quality levels.
const (
	// DefaultWebPQuality is used when Options.WebPQuality is zero. At this level
	// the output is visually indistinguishable from the PNG (PSNR above 40 dB on
	// the default theme) at about a tenth of the size.
	DefaultWebPQuality = 90

	// WebPLossless as a quality selects lossless WebP, which is pixel-exact but
	// much larger and slower to encode.
	WebPLossless = 100

	minWebPQuality = 1
	webPMethod     = 4 // libwebp effort, 0 (fast) to 6 (slow); 4 is its default
)

// Options tunes the encoders.
type Options struct {
	// WebPQuality is 1-99 for lossy WebP or WebPLossless (100). Zero means
	// DefaultWebPQuality. The alpha channel is always kept exact.
	WebPQuality int
}

// ValidateWebPQuality reports whether q is usable as Options.WebPQuality.
func ValidateWebPQuality(q int) error {
	if q != 0 && (q < minWebPQuality || q > WebPLossless) {
		return fmt.Errorf("webp quality must be from %d to %d (or 0 for the default), got %d",
			minWebPQuality, WebPLossless, q)
	}

	return nil
}

func (o Options) webPQuality() int {
	if o.WebPQuality == 0 {
		return DefaultWebPQuality
	}

	return o.WebPQuality
}

// IsSupported reports whether format is an output format.
func IsSupported(format string) bool {
	return format == PNG || format == WebP
}

// CheckPNGSize verifies that data is a PNG of exactly width x height pixels.
func CheckPNGSize(data []byte, width, height int) error {
	cfg, err := png.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("renderer did not return a valid png: %w", err)
	}
	if cfg.Width != width || cfg.Height != height {
		return fmt.Errorf("renderer returned a %dx%d image, expected %dx%d", cfg.Width, cfg.Height, width, height)
	}

	return nil
}

// Encode converts a PNG into format. PNG input is returned unchanged.
func Encode(pngData []byte, format string, opts Options) ([]byte, error) {
	if err := ValidateWebPQuality(opts.WebPQuality); err != nil {
		return nil, err
	}

	switch format {
	case PNG:
		return pngData, nil
	case WebP:
		img, err := png.Decode(bytes.NewReader(pngData))
		if err != nil {
			return nil, fmt.Errorf("decode png: %w", err)
		}

		return encodeWebP(img, opts.webPQuality())
	default:
		return nil, fmt.Errorf("unsupported format %q", format)
	}
}

// EncodeAll converts a PNG into every requested format, decoding it only once.
// The result maps format to file content.
func EncodeAll(pngData []byte, formats []string, opts Options) (map[string][]byte, error) {
	if err := ValidateWebPQuality(opts.WebPQuality); err != nil {
		return nil, err
	}
	out := make(map[string][]byte, len(formats))

	var decoded image.Image
	for _, format := range formats {
		switch format {
		case PNG:
			out[format] = pngData
		case WebP:
			if decoded == nil {
				img, err := png.Decode(bytes.NewReader(pngData))
				if err != nil {
					return nil, fmt.Errorf("decode png: %w", err)
				}
				decoded = img
			}
			data, err := encodeWebP(decoded, opts.webPQuality())
			if err != nil {
				return nil, err
			}
			out[format] = data
		default:
			return nil, fmt.Errorf("unsupported format %q", format)
		}
	}

	return out, nil
}

func encodeWebP(img image.Image, quality int) ([]byte, error) {
	opts := webp.Options{Quality: quality, Method: webPMethod}
	if quality == WebPLossless {
		opts.Lossless = true
	}

	var buf bytes.Buffer
	if err := webp.Encode(&buf, img, opts); err != nil {
		return nil, fmt.Errorf("encode webp: %w", err)
	}

	return buf.Bytes(), nil
}
