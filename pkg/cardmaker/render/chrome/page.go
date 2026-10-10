package chrome

import (
	"bytes"
	"fmt"
	"html/template"
	"io/fs"

	"github.com/Bryant-Anjos/card-maker-go/pkg/cardmaker/i18n"
	"github.com/Bryant-Anjos/card-maker-go/pkg/cardmaker/model"
	"github.com/Bryant-Anjos/card-maker-go/pkg/cardmaker/theme"
)

// Reserved names served next to the theme files. Themes must not use them.
const (
	pagePath = "__page.html"
	artPath  = "__art"
)

// CardData is the value a theme's card template is executed with.
//
//	{{.Card.Name}} {{.Card.Description}} {{printf "%03d" .Card.Number}}
//	{{.Rarity.ID}} {{.Rarity.Color}} {{inline .Rarity.Frame}}
//	{{if .Element}}{{inline .Element.Icon}}{{end}}
//	<img src="{{.ArtURL}}"> <link rel="stylesheet" href="{{.Stylesheet}}">
type CardData struct {
	Card       i18n.ResolvedCard // texts already resolved for the output locale
	Rarity     model.Rarity      // the card's rarity as defined by the theme
	Element    *model.Element    // nil when the card has no element
	Size       model.Size        // card size in CSS pixels
	ArtURL     string            // URL of the card artwork
	Stylesheet string            // theme stylesheet, relative to the theme root
}

// BackData is the value a theme's back template is executed with.
type BackData struct {
	Size       model.Size
	Stylesheet string
}

func cardPage(th theme.Theme, card i18n.ResolvedCard) ([]byte, error) {
	rarity, ok := th.Rarity(card.Rarity)
	if !ok {
		return nil, fmt.Errorf("card %q: unknown rarity %q", card.ID, card.Rarity)
	}

	var element *model.Element
	if card.Element != "" {
		e, ok := th.Element(card.Element)
		if !ok {
			return nil, fmt.Errorf("card %q: unknown element %q", card.ID, card.Element)
		}
		element = &e
	}

	template, ok := th.TemplateFor(card.Variant)
	if !ok {
		return nil, fmt.Errorf("card %q: unknown variant %q", card.ID, card.Variant)
	}

	data := CardData{
		Card:       card,
		Rarity:     rarity,
		Element:    element,
		Size:       th.Size,
		ArtURL:     artPath,
		Stylesheet: th.Stylesheet,
	}

	return executeTemplate(th.FS, template, data)
}

func backPage(th theme.Theme) ([]byte, error) {
	if th.Back == "" {
		return nil, fmt.Errorf("theme %q has no back template", th.Name)
	}

	return executeTemplate(th.FS, th.Back, BackData{Size: th.Size, Stylesheet: th.Stylesheet})
}

func executeTemplate(fsys fs.FS, name string, data any) ([]byte, error) {
	source, err := fs.ReadFile(fsys, name)
	if err != nil {
		return nil, fmt.Errorf("read template %q: %w", name, err)
	}

	tmpl, err := template.New(name).Funcs(templateFuncs(fsys)).Parse(string(source))
	if err != nil {
		return nil, fmt.Errorf("parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("execute template: %w", err)
	}

	return buf.Bytes(), nil
}

// templateFuncs are the helpers available to theme templates.
func templateFuncs(fsys fs.FS) template.FuncMap {
	return template.FuncMap{
		// inline embeds a theme file (typically an SVG) verbatim, so it can be
		// styled with the page CSS. Theme files are trusted content.
		"inline": func(path string) (template.HTML, error) {
			if !fs.ValidPath(path) {
				return "", fmt.Errorf("inline %q: path must stay inside the theme", path)
			}
			content, err := fs.ReadFile(fsys, path)
			if err != nil {
				return "", fmt.Errorf("inline %q: %w", path, err)
			}

			return template.HTML(content), nil //nolint:gosec // theme files are trusted
		},
	}
}
