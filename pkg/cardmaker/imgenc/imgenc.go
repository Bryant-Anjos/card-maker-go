// Package imgenc converts the PNG produced by a renderer into the output
// formats of a build and checks that renderers honored their size contract.
package imgenc

import (
	"bytes"
	"fmt"
	"image"
	"image/png"

	"github.com/HugoSmits86/nativewebp"
)

// Supported output formats.
const (
	PNG  = "png"
	WebP = "webp"
)

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

// Encode converts a PNG into format. PNG input is returned unchanged; WebP is
// lossless (the encoder is pure Go and has no lossy mode).
func Encode(pngData []byte, format string) ([]byte, error) {
	switch format {
	case PNG:
		return pngData, nil
	case WebP:
		img, err := png.Decode(bytes.NewReader(pngData))
		if err != nil {
			return nil, fmt.Errorf("decode png: %w", err)
		}

		return encodeWebP(img)
	default:
		return nil, fmt.Errorf("unsupported format %q", format)
	}
}

// EncodeAll converts a PNG into every requested format, decoding it only once.
// The result maps format to file content.
func EncodeAll(pngData []byte, formats []string) (map[string][]byte, error) {
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
			data, err := encodeWebP(decoded)
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

func encodeWebP(img image.Image) ([]byte, error) {
	var buf bytes.Buffer
	if err := nativewebp.Encode(&buf, img, nil); err != nil {
		return nil, fmt.Errorf("encode webp: %w", err)
	}

	return buf.Bytes(), nil
}
