package chrome

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/Bryant-Anjos/card-maker-go/pkg/cardmaker/i18n"
	"github.com/Bryant-Anjos/card-maker-go/pkg/cardmaker/model"
	"github.com/Bryant-Anjos/card-maker-go/pkg/cardmaker/theme"
)

func pageTheme(cardTemplate string) theme.Theme {
	return theme.Theme{
		Theme: model.Theme{
			SchemaVersion: 1, Name: "t", Size: model.Size{Width: 750, Height: 1050},
			Template: "card.html.tmpl", Stylesheet: "style.css", Back: "back.html.tmpl",
			Rarities: []model.Rarity{{ID: "rare", Color: "#3b82f6", Frame: "frames/rare.svg"}},
			Elements: []model.Element{{ID: "fire", Icon: "icons/fire.svg"}},
			Variants: []model.Variant{{ID: "fullart", Template: "fullart.html.tmpl"}},
		},
		FS: fstest.MapFS{
			"fullart.html.tmpl": {Data: []byte(`full:{{.Card.Variant}}:{{.Card.Name}}`)},
			"card.html.tmpl":    {Data: []byte(cardTemplate)},
			"back.html.tmpl":    {Data: []byte(`<div class="back" style="width:{{.Size.Width}}px">{{.Stylesheet}}</div>`)},
			"style.css":         {Data: []byte("body{}")},
			"frames/rare.svg":   {Data: []byte(`<svg id="frame"/>`)},
			"icons/fire.svg":    {Data: []byte(`<svg id="fire"/>`)},
		},
	}
}

func resolvedCard() i18n.ResolvedCard {
	return i18n.ResolvedCard{ID: "lunaris", Number: 32, Rarity: "rare", Element: "fire",
		Name: "Lunaris <script>", Description: "Brilha.", Art: "art/lunaris.png"}
}

func TestCardPageExposesTemplateData(t *testing.T) {
	tmpl := `{{.Card.ID}}|{{printf "%03d" .Card.Number}}|{{.Card.Name}}|{{.Card.Description}}|` +
		`{{.Rarity.ID}}|{{.Rarity.Color}}|{{.Element.ID}}|{{.Size.Width}}x{{.Size.Height}}|` +
		`{{.ArtURL}}|{{.Stylesheet}}`

	page, err := cardPage(pageTheme(tmpl), resolvedCard())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "lunaris|032|Lunaris &lt;script&gt;|Brilha.|rare|#3b82f6|fire|750x1050|" + artPath + "|style.css"
	if string(page) != want {
		t.Errorf("page =\n%s\nwant\n%s", page, want)
	}
}

func TestCardPageInlinesThemeFiles(t *testing.T) {
	page, err := cardPage(pageTheme(`{{inline .Rarity.Frame}}{{if .Element}}{{inline .Element.Icon}}{{end}}`), resolvedCard())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(page) != `<svg id="frame"/><svg id="fire"/>` {
		t.Errorf("page = %s", page)
	}
}

func TestCardPageWithoutElement(t *testing.T) {
	card := resolvedCard()
	card.Element = ""

	page, err := cardPage(pageTheme(`[{{if .Element}}has{{else}}none{{end}}]`), card)

	if err != nil || string(page) != "[none]" {
		t.Errorf("page = %q, err = %v", page, err)
	}
}

func TestCardPageErrors(t *testing.T) {
	tests := []struct {
		name    string
		tmpl    string
		mutate  func(th *theme.Theme, c *i18n.ResolvedCard)
		wantErr string
	}{
		{"template syntax error", `{{.Card.Name`, nil, "card.html.tmpl"},
		{"unknown field", `{{.Card.Nope}}`, nil, "Nope"},
		{"inline outside theme", `{{inline "../secret"}}`, nil, "../secret"},
		{"inline missing file", `{{inline "icons/none.svg"}}`, nil, "icons/none.svg"},
		{"unknown rarity", `x`, func(_ *theme.Theme, c *i18n.ResolvedCard) { c.Rarity = "mythic" }, "mythic"},
		{"unknown element", `x`, func(_ *theme.Theme, c *i18n.ResolvedCard) { c.Element = "void" }, "void"},
		{"unknown variant", `x`, func(_ *theme.Theme, c *i18n.ResolvedCard) { c.Variant = "etched" }, "etched"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			th, card := pageTheme(tt.tmpl), resolvedCard()
			if tt.mutate != nil {
				tt.mutate(&th, &card)
			}

			_, err := cardPage(th, card)

			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("error = %v, want it to contain %q", err, tt.wantErr)
			}
		})
	}
}

func TestBackPage(t *testing.T) {
	page, err := backPage(pageTheme(""))

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(page) != `<div class="back" style="width:750px">style.css</div>` {
		t.Errorf("page = %s", page)
	}

	th := pageTheme("")
	th.Back = ""
	if _, err := backPage(th); err == nil {
		t.Error("expected error for theme without back")
	}
}

func TestCardPageUsesVariantTemplate(t *testing.T) {
	card := resolvedCard()
	card.Variant = "fullart"

	page, err := cardPage(pageTheme(`standard`), card)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(page) != "full:fullart:Lunaris &lt;script&gt;" {
		t.Errorf("page = %q, want the fullart template", page)
	}
}
