package render_test

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"testing"
	"testing/fstest"

	"github.com/Bryant-Anjos/card-maker-go/pkg/cardmaker/i18n"
	"github.com/Bryant-Anjos/card-maker-go/pkg/cardmaker/model"
	"github.com/Bryant-Anjos/card-maker-go/pkg/cardmaker/render"
	"github.com/Bryant-Anjos/card-maker-go/pkg/cardmaker/theme"
)

func testTheme() theme.Theme {
	return theme.Theme{
		Theme: model.Theme{
			SchemaVersion: 1, Name: "t", Size: model.Size{Width: 30, Height: 42},
			Rarities: []model.Rarity{{ID: "rare", Color: "#3b82f6", Frame: "f.svg"}},
		},
		FS: fstest.MapFS{},
	}
}

func decode(t *testing.T, data []byte) image.Image {
	t.Helper()

	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("not a valid png: %v", err)
	}

	return img
}

var _ render.Renderer = (*render.Fake)(nil)

func TestFakeRenderCard(t *testing.T) {
	fake := render.NewFake()
	req := render.CardRequest{Theme: testTheme(), Card: i18n.ResolvedCard{ID: "a", Rarity: "rare"}, Scale: 2}

	data, err := fake.RenderCard(context.Background(), req)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	img := decode(t, data)
	if got := img.Bounds().Size(); got != (image.Point{X: 60, Y: 84}) {
		t.Errorf("size = %v, want 60x84 (theme size x scale)", got)
	}
	want := color.NRGBA{R: 0x3b, G: 0x82, B: 0xf6, A: 0xff}
	if got := color.NRGBAModel.Convert(img.At(5, 5)); got != want {
		t.Errorf("pixel = %v, want rarity color %v", got, want)
	}
}

func TestFakeRenderCardErrors(t *testing.T) {
	fake := render.NewFake()
	good := render.CardRequest{Theme: testTheme(), Card: i18n.ResolvedCard{ID: "a", Rarity: "rare"}, Scale: 1}

	t.Run("unknown rarity", func(t *testing.T) {
		req := good
		req.Card.Rarity = "mythic"

		if _, err := fake.RenderCard(context.Background(), req); err == nil {
			t.Error("expected error")
		}
	})

	t.Run("invalid scale", func(t *testing.T) {
		req := good
		req.Scale = 0

		if _, err := fake.RenderCard(context.Background(), req); err == nil {
			t.Error("expected error")
		}
	})

	t.Run("cancelled context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		if _, err := fake.RenderCard(ctx, good); err == nil {
			t.Error("expected error")
		}
	})
}

func TestFakeRenderBackAndCalls(t *testing.T) {
	fake := render.NewFake()

	data, err := fake.RenderBack(context.Background(), render.BackRequest{Theme: testTheme(), Scale: 1})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := decode(t, data).Bounds().Size(); got != (image.Point{X: 30, Y: 42}) {
		t.Errorf("size = %v", got)
	}
	if _, err := fake.RenderCard(context.Background(),
		render.CardRequest{Theme: testTheme(), Card: i18n.ResolvedCard{ID: "a", Rarity: "rare"}, Scale: 1}); err != nil {
		t.Fatal(err)
	}
	if fake.CardCalls() != 1 || fake.BackCalls() != 1 {
		t.Errorf("calls = card %d, back %d, want 1 and 1", fake.CardCalls(), fake.BackCalls())
	}
	if fake.Version() == "" {
		t.Error("Version must not be empty: it is part of the cache key")
	}
}
