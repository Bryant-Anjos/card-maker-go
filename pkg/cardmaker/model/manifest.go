package model

import "fmt"

// supportedFormats lists the image formats a manifest may reference.
var supportedFormats = map[string]bool{"png": true, "webp": true}

// Image is one rendered file of a card (or of the card back).
type Image struct {
	Scale  int    `json:"scale"`
	Format string `json:"format"`
	Path   string `json:"path"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

// ManifestCard is a rendered card with its text already resolved for the
// manifest locale, so consumers never need to know about localization.
type ManifestCard struct {
	ID          string  `json:"id"`
	Number      int     `json:"number,omitempty"`
	Rarity      string  `json:"rarity"`
	Element     string  `json:"element,omitempty"`
	Name        string  `json:"name"`
	Description string  `json:"description,omitempty"`
	Images      []Image `json:"images"`
}

// Manifest is the output contract consumed by every client platform. Image
// paths are relative to the directory that contains the manifest.
type Manifest struct {
	SchemaVersion int            `json:"schemaVersion"`
	Theme         string         `json:"theme"`
	Locale        string         `json:"locale"`
	Size          Size           `json:"size"`
	Back          []Image        `json:"back,omitempty"`
	Cards         []ManifestCard `json:"cards"`
}

func (i Image) problems() problems {
	var p problems

	checkPositive(&p, "scale", i.Scale)
	if !supportedFormats[i.Format] {
		p.addf("format", "unsupported format %q", i.Format)
	}
	checkRelativePath(&p, "path", i.Path)
	checkPositive(&p, "width", i.Width)
	checkPositive(&p, "height", i.Height)

	return p
}

// Validate reports every structural problem of the manifest.
func (m Manifest) Validate() error {
	var p problems

	checkSchemaVersion(&p, m.SchemaVersion)
	if m.Theme == "" {
		p.addf("theme", "is required")
	}
	checkLocale(&p, "locale", m.Locale)
	if m.Size.Width <= 0 || m.Size.Height <= 0 {
		p.addf("size", "width and height must be positive, got %dx%d", m.Size.Width, m.Size.Height)
	}
	for i, img := range m.Back {
		p.addNested(fmt.Sprintf("back[%d]", i), img.problems())
	}
	if len(m.Cards) == 0 {
		p.addf("cards", "at least one card is required")
	}

	seen := make(map[string]bool, len(m.Cards))
	for i, card := range m.Cards {
		prefix := fmt.Sprintf("cards[%d]", i)

		if seen[card.ID] {
			p.addf(prefix+".id", "duplicate id %q", card.ID)
		}
		seen[card.ID] = true

		p.addNested(prefix, card.problems())
	}

	return p.err()
}

func (c ManifestCard) problems() problems {
	var p problems

	checkIdentifier(&p, "id", c.ID)
	checkIdentifier(&p, "rarity", c.Rarity)
	if c.Name == "" {
		p.addf("name", "is required")
	}
	if len(c.Images) == 0 {
		p.addf("images", "at least one image is required")
	}
	for i, img := range c.Images {
		p.addNested(fmt.Sprintf("images[%d]", i), img.problems())
	}

	return p
}
