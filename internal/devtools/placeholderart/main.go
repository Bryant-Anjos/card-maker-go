// Command placeholderart generates the placeholder artwork used by the sample
// card data in testdata/. The images are flat gradients: they exist only to
// exercise the pipeline, not to look good.
package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
)

const side = 600

var palette = map[string]color.NRGBA{
	"vulpin":    {R: 0xd9, G: 0x9a, B: 0x5b, A: 0xff},
	"cervallis": {R: 0x4c, G: 0x9a, B: 0x5f, A: 0xff},
	"lunaris":   {R: 0x3b, G: 0x5f, B: 0xc4, A: 0xff},
	"aureon":    {R: 0x8a, G: 0x55, B: 0xc9, A: 0xff},
	"drakarion": {R: 0xd0, G: 0x8a, B: 0x2e, A: 0xff},
}

func main() {
	dir := "testdata/cards/art"
	if len(os.Args) > 1 {
		dir = os.Args[1]
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		fail(err)
	}

	for name, base := range palette {
		if err := writeGradient(filepath.Join(dir, name+".png"), base); err != nil {
			fail(err)
		}
	}
}

func writeGradient(path string, base color.NRGBA) error {
	img := image.NewNRGBA(image.Rect(0, 0, side, side))
	for y := range side {
		shade := uint8(255 - y*140/side)
		row := color.NRGBA{R: scale(base.R, shade), G: scale(base.G, shade), B: scale(base.B, shade), A: 0xff}
		for x := range side {
			img.SetNRGBA(x, y, row)
		}
	}

	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	return png.Encode(file, img)
}

func scale(channel, shade uint8) uint8 {
	return uint8(int(channel) * int(shade) / 255)
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "placeholderart:", err)
	os.Exit(1)
}
