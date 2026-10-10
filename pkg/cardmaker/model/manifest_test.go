package model_test

import (
	"strings"
	"testing"

	"github.com/Bryant-Anjos/card-maker-go/pkg/cardmaker/model"
)

func validManifest() model.Manifest {
	return model.Manifest{
		SchemaVersion: 1,
		Theme:         "default",
		Locale:        "pt-BR",
		Size:          model.Size{Width: 750, Height: 1050},
		Back:          &model.Image{Scale: 1, Format: "png", Path: "back.png", Width: 750, Height: 1050},
		Cards: []model.ManifestCard{
			{
				ID: "vulpin", Number: 1, Rarity: "common", Element: "beast",
				Name: "Vulpin", Description: "Pequena criatura.",
				Images: []model.Image{
					{Scale: 1, Format: "png", Path: "cards/vulpin.png", Width: 750, Height: 1050},
					{Scale: 2, Format: "webp", Path: "cards/vulpin@2x.webp", Width: 1500, Height: 2100},
				},
			},
		},
	}
}

func TestManifestValidate(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(m *model.Manifest)
		wantErr string
	}{
		{"valid", func(m *model.Manifest) {}, ""},
		{"valid without back", func(m *model.Manifest) { m.Back = nil }, ""},
		{"wrong schema version", func(m *model.Manifest) { m.SchemaVersion = 0 }, "schemaVersion"},
		{"empty theme", func(m *model.Manifest) { m.Theme = "" }, "theme"},
		{"invalid locale", func(m *model.Manifest) { m.Locale = "x y" }, "locale"},
		{"zero size", func(m *model.Manifest) { m.Size = model.Size{} }, "size"},
		{"no cards", func(m *model.Manifest) { m.Cards = nil }, "cards"},
		{"duplicate card", func(m *model.Manifest) { m.Cards = append(m.Cards, m.Cards[0]) }, "duplicate"},
		{"card without images", func(m *model.Manifest) { m.Cards[0].Images = nil }, "images"},
		{"unsupported format", func(m *model.Manifest) { m.Cards[0].Images[0].Format = "gif" }, "format"},
		{"zero scale", func(m *model.Manifest) { m.Cards[0].Images[0].Scale = 0 }, "scale"},
		{"image path escapes", func(m *model.Manifest) { m.Cards[0].Images[0].Path = "../x.png" }, "path"},
		{"zero image width", func(m *model.Manifest) { m.Cards[0].Images[1].Width = 0 }, "width"},
		{"invalid back", func(m *model.Manifest) { m.Back.Path = "" }, "back"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manifest := validManifest()
			tt.mutate(&manifest)

			assertErr(t, manifest.Validate(), tt.wantErr)
		})
	}
}

func TestParseManifest(t *testing.T) {
	src := `{"schemaVersion":1,"theme":"default","locale":"pt-BR","size":{"width":750,"height":1050},
		"cards":[{"id":"a","rarity":"common","name":"A",
		"images":[{"scale":1,"format":"png","path":"cards/a.png","width":750,"height":1050}]}]}`

	m, err := model.ParseManifest(strings.NewReader(src))

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Cards[0].ID != "a" {
		t.Errorf("id = %q", m.Cards[0].ID)
	}
}
