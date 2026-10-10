package model_test

import (
	"strings"
	"testing"

	"github.com/Bryant-Anjos/card-maker-go/pkg/cardmaker/model"
)

func validTheme() model.Theme {
	return model.Theme{
		SchemaVersion: 1,
		Name:          "default",
		Size:          model.Size{Width: 750, Height: 1050},
		Template:      "card.html.tmpl",
		Stylesheet:    "style.css",
		Back:          "back.html.tmpl",
		Rarities: []model.Rarity{
			{ID: "common", Color: "#8a8f98", Frame: "frames/common.svg", Label: model.LocalizedText{"pt-BR": "Comum"}},
			{ID: "rare", Color: "#3b82f6", Frame: "frames/rare.svg", Label: model.LocalizedText{"pt-BR": "Raro"}},
		},
		Elements: []model.Element{
			{ID: "fire", Icon: "icons/fire.svg", Label: model.LocalizedText{"pt-BR": "Fogo"}},
		},
		Variants: []model.Variant{
			{ID: "fullart", Template: "fullart.html.tmpl", Label: model.LocalizedText{"pt-BR": "Arte completa"}},
		},
	}
}

func TestThemeValidate(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(th *model.Theme)
		wantErr string
	}{
		{"valid", func(th *model.Theme) {}, ""},
		{"valid without elements and back", func(th *model.Theme) { th.Elements = nil; th.Back = "" }, ""},
		{"wrong schema version", func(th *model.Theme) { th.SchemaVersion = 9 }, "schemaVersion"},
		{"empty name", func(th *model.Theme) { th.Name = "" }, "name"},
		{"zero width", func(th *model.Theme) { th.Size.Width = 0 }, "size"},
		{"negative height", func(th *model.Theme) { th.Size.Height = -5 }, "size"},
		{"missing template", func(th *model.Theme) { th.Template = "" }, "template"},
		{"template escapes theme dir", func(th *model.Theme) { th.Template = "../x.tmpl" }, "template"},
		{"stylesheet absolute", func(th *model.Theme) { th.Stylesheet = "/etc/x.css" }, "stylesheet"},
		{"no rarities", func(th *model.Theme) { th.Rarities = nil }, "rarities"},
		{"duplicate rarity", func(th *model.Theme) { th.Rarities[1].ID = "common" }, "duplicate"},
		{"bad rarity id", func(th *model.Theme) { th.Rarities[0].ID = "Com um" }, "rarities[0].id"},
		{"bad rarity color", func(th *model.Theme) { th.Rarities[0].Color = "gray" }, "color"},
		{"short hex color", func(th *model.Theme) { th.Rarities[0].Color = "#fff" }, "color"},
		{"rarity frame escapes", func(th *model.Theme) { th.Rarities[0].Frame = "../../x.svg" }, "frame"},
		{"duplicate element", func(th *model.Theme) { th.Elements = append(th.Elements, th.Elements[0]) }, "duplicate"},
		{"element icon missing", func(th *model.Theme) { th.Elements[0].Icon = "" }, "icon"},
		{"valid without variants", func(th *model.Theme) { th.Variants = nil }, ""},
		{"bad variant id", func(th *model.Theme) { th.Variants[0].ID = "Full Art" }, "variants[0].id"},
		{"duplicate variant", func(th *model.Theme) { th.Variants = append(th.Variants, th.Variants[0]) }, "duplicate"},
		{"variant template escapes", func(th *model.Theme) { th.Variants[0].Template = "../x.tmpl" }, "variants[0].template"},
		{"variant template missing", func(th *model.Theme) { th.Variants[0].Template = "" }, "variants[0].template"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			theme := validTheme()
			tt.mutate(&theme)

			assertErr(t, theme.Validate(), tt.wantErr)
		})
	}
}

func TestThemeLookups(t *testing.T) {
	theme := validTheme()

	if r, ok := theme.Rarity("rare"); !ok || r.Color != "#3b82f6" {
		t.Errorf("Rarity(rare) = (%+v,%v)", r, ok)
	}
	if _, ok := theme.Rarity("mythic"); ok {
		t.Error("Rarity(mythic) should not exist")
	}
	if e, ok := theme.Element("fire"); !ok || e.Icon != "icons/fire.svg" {
		t.Errorf("Element(fire) = (%+v,%v)", e, ok)
	}
	if _, ok := theme.Element("void"); ok {
		t.Error("Element(void) should not exist")
	}
}

func TestThemeTemplateFor(t *testing.T) {
	theme := validTheme()

	tests := []struct {
		variant string
		want    string
		wantOK  bool
	}{
		{"", "card.html.tmpl", true},
		{"fullart", "fullart.html.tmpl", true},
		{"etched", "", false},
	}

	for _, tt := range tests {
		got, ok := theme.TemplateFor(tt.variant)
		if got != tt.want || ok != tt.wantOK {
			t.Errorf("TemplateFor(%q) = (%q, %v), want (%q, %v)", tt.variant, got, ok, tt.want, tt.wantOK)
		}
	}
}

func TestParseTheme(t *testing.T) {
	src := `{"schemaVersion":1,"name":"default","size":{"width":750,"height":1050},
		"template":"card.html.tmpl","stylesheet":"style.css",
		"rarities":[{"id":"common","color":"#8a8f98","frame":"frames/common.svg"}]}`

	theme, err := model.ParseTheme(strings.NewReader(src))

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if theme.Size.Width != 750 {
		t.Errorf("width = %d", theme.Size.Width)
	}

	_, err = model.ParseTheme(strings.NewReader(`{"schemaVersion":1,"nope":1}`))
	assertErr(t, err, "nope")
}
