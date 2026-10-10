package model

import "fmt"

// Size is a width and height in pixels.
type Size struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

// Rarity describes one rarity tier a theme can render.
type Rarity struct {
	ID    string        `json:"id"`
	Label LocalizedText `json:"label,omitempty"`
	Color string        `json:"color"`
	Frame string        `json:"frame"`
}

// Element describes one element/category icon a theme can render.
type Element struct {
	ID    string        `json:"id"`
	Label LocalizedText `json:"label,omitempty"`
	Icon  string        `json:"icon"`
}

// Variant is an alternative card layout, such as full art, with its own
// template. Cards opt into it with their variant field.
type Variant struct {
	ID       string        `json:"id"`
	Label    LocalizedText `json:"label,omitempty"`
	Template string        `json:"template"`
}

// Theme is the visual definition of a card set. All paths are relative to the
// theme directory. Rarities and elements are data, not code, so any game can
// define its own.
type Theme struct {
	Schema        string    `json:"$schema,omitempty"` // editor hint, ignored
	SchemaVersion int       `json:"schemaVersion"`
	Name          string    `json:"name"`
	Size          Size      `json:"size"`
	Template      string    `json:"template"`
	Stylesheet    string    `json:"stylesheet"`
	Back          string    `json:"back,omitempty"`
	Rarities      []Rarity  `json:"rarities"`
	Elements      []Element `json:"elements,omitempty"`
	Variants      []Variant `json:"variants,omitempty"`
}

// Rarity looks up a rarity by id.
func (t Theme) Rarity(id string) (Rarity, bool) {
	for _, r := range t.Rarities {
		if r.ID == id {
			return r, true
		}
	}

	return Rarity{}, false
}

// Element looks up an element by id.
func (t Theme) Element(id string) (Element, bool) {
	for _, e := range t.Elements {
		if e.ID == id {
			return e, true
		}
	}

	return Element{}, false
}

// Variant looks up a variant by id.
func (t Theme) Variant(id string) (Variant, bool) {
	for _, v := range t.Variants {
		if v.ID == id {
			return v, true
		}
	}

	return Variant{}, false
}

// TemplateFor returns the card template of a variant; the empty variant is
// the standard layout.
func (t Theme) TemplateFor(variant string) (string, bool) {
	if variant == "" {
		return t.Template, true
	}

	v, ok := t.Variant(variant)

	return v.Template, ok
}

// Validate reports every structural problem of the theme.
func (t Theme) Validate() error {
	var p problems

	checkSchemaVersion(&p, t.SchemaVersion)
	if t.Name == "" {
		p.addf("name", "is required")
	}
	if t.Size.Width <= 0 || t.Size.Height <= 0 {
		p.addf("size", "width and height must be positive, got %dx%d", t.Size.Width, t.Size.Height)
	}
	checkRelativePath(&p, "template", t.Template)
	checkRelativePath(&p, "stylesheet", t.Stylesheet)
	if t.Back != "" {
		checkRelativePath(&p, "back", t.Back)
	}

	p = append(p, t.raritiesProblems()...)
	p = append(p, t.elementsProblems()...)
	p = append(p, t.variantsProblems()...)

	return p.err()
}

func (t Theme) raritiesProblems() problems {
	var p problems

	if len(t.Rarities) == 0 {
		p.addf("rarities", "at least one rarity is required")
	}

	seen := make(map[string]bool, len(t.Rarities))
	for i, r := range t.Rarities {
		prefix := fmt.Sprintf("rarities[%d]", i)

		checkIdentifier(&p, prefix+".id", r.ID)
		if seen[r.ID] {
			p.addf(prefix+".id", "duplicate id %q", r.ID)
		}
		seen[r.ID] = true

		if !hexColorPattern.MatchString(r.Color) {
			p.addf(prefix+".color", "%q must be a #rrggbb hex color", r.Color)
		}
		checkRelativePath(&p, prefix+".frame", r.Frame)
		p = append(p, r.Label.validate(prefix+".label", false)...)
	}

	return p
}

func (t Theme) elementsProblems() problems {
	var p problems

	seen := make(map[string]bool, len(t.Elements))
	for i, e := range t.Elements {
		prefix := fmt.Sprintf("elements[%d]", i)

		checkIdentifier(&p, prefix+".id", e.ID)
		if seen[e.ID] {
			p.addf(prefix+".id", "duplicate id %q", e.ID)
		}
		seen[e.ID] = true

		checkRelativePath(&p, prefix+".icon", e.Icon)
		p = append(p, e.Label.validate(prefix+".label", false)...)
	}

	return p
}

func (t Theme) variantsProblems() problems {
	var p problems

	seen := make(map[string]bool, len(t.Variants))
	for i, v := range t.Variants {
		prefix := fmt.Sprintf("variants[%d]", i)

		checkIdentifier(&p, prefix+".id", v.ID)
		if seen[v.ID] {
			p.addf(prefix+".id", "duplicate id %q", v.ID)
		}
		seen[v.ID] = true

		checkRelativePath(&p, prefix+".template", v.Template)
		p = append(p, v.Label.validate(prefix+".label", false)...)
	}

	return p
}
