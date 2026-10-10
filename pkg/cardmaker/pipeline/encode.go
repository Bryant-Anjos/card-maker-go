package pipeline

import (
	"bytes"
	"fmt"
	"image"
	"image/png"

	"github.com/HugoSmits86/nativewebp"
)

// checkPNGSize verifies that a renderer honored the size contract.
func checkPNGSize(data []byte, width, height int) error {
	cfg, err := png.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("renderer did not return a valid png: %w", err)
	}
	if cfg.Width != width || cfg.Height != height {
		return fmt.Errorf("renderer returned a %dx%d image, expected %dx%d", cfg.Width, cfg.Height, width, height)
	}

	return nil
}

// encodeAll converts the renderer's PNG into every requested format. The
// result maps format to file content.
func encodeAll(pngData []byte, formats []string) (map[string][]byte, error) {
	out := make(map[string][]byte, len(formats))

	var decoded image.Image
	for _, format := range formats {
		switch format {
		case FormatPNG:
			out[format] = pngData
		case FormatWebP:
			if decoded == nil {
				img, err := png.Decode(bytes.NewReader(pngData))
				if err != nil {
					return nil, fmt.Errorf("decode png: %w", err)
				}
				decoded = img
			}

			var buf bytes.Buffer
			if err := nativewebp.Encode(&buf, decoded, nil); err != nil {
				return nil, fmt.Errorf("encode webp: %w", err)
			}
			out[format] = buf.Bytes()
		default:
			return nil, fmt.Errorf("unsupported format %q", format)
		}
	}

	return out, nil
}
