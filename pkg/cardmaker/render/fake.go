package render

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"strconv"
	"sync/atomic"
)

// backColor is the flat color of the fake card back.
var backColor = color.NRGBA{R: 0x1b, G: 0x25, B: 0x3d, A: 0xff}

// Fake is a Renderer that paints flat rectangles with the rarity color. It
// needs no browser, which makes it ideal for tests and for exercising the
// pipeline before a real theme exists.
type Fake struct {
	cardCalls atomic.Int64
	backCalls atomic.Int64
}

// NewFake returns a ready-to-use Fake renderer.
func NewFake() *Fake {
	return &Fake{}
}

// Version implements Renderer.
func (f *Fake) Version() string { return "fake-1" }

// CardCalls reports how many cards were rendered so far.
func (f *Fake) CardCalls() int { return int(f.cardCalls.Load()) }

// BackCalls reports how many backs were rendered so far.
func (f *Fake) BackCalls() int { return int(f.backCalls.Load()) }

// RenderCard implements Renderer.
func (f *Fake) RenderCard(ctx context.Context, req CardRequest) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	rarity, ok := req.Theme.Rarity(req.Card.Rarity)
	if !ok {
		return nil, fmt.Errorf("card %q: unknown rarity %q", req.Card.ID, req.Card.Rarity)
	}
	fill, err := parseHexColor(rarity.Color)
	if err != nil {
		return nil, err
	}

	f.cardCalls.Add(1)

	return solidPNG(req.Theme.Size.Width, req.Theme.Size.Height, req.Scale, fill)
}

// RenderBack implements Renderer.
func (f *Fake) RenderBack(ctx context.Context, req BackRequest) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	f.backCalls.Add(1)

	return solidPNG(req.Theme.Size.Width, req.Theme.Size.Height, req.Scale, backColor)
}

func solidPNG(width, height, scale int, fill color.NRGBA) ([]byte, error) {
	if scale < 1 {
		return nil, fmt.Errorf("scale must be at least 1, got %d", scale)
	}

	img := image.NewNRGBA(image.Rect(0, 0, width*scale, height*scale))
	draw.Draw(img, img.Bounds(), image.NewUniform(fill), image.Point{}, draw.Src)

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, fmt.Errorf("encode png: %w", err)
	}

	return buf.Bytes(), nil
}

func parseHexColor(hex string) (color.NRGBA, error) {
	if len(hex) != 7 || hex[0] != '#' {
		return color.NRGBA{}, fmt.Errorf("invalid color %q", hex)
	}

	value, err := strconv.ParseUint(hex[1:], 16, 32)
	if err != nil {
		return color.NRGBA{}, fmt.Errorf("invalid color %q: %w", hex, err)
	}

	return color.NRGBA{R: uint8(value >> 16), G: uint8(value >> 8), B: uint8(value), A: 0xff}, nil
}
