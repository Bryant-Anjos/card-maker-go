package model

import "fmt"

// Card is the input data for a single card. Rarity and Element are identifiers
// that must exist in the theme used to render it.
type Card struct {
	ID          string        `json:"id"`
	Number      int           `json:"number,omitempty"`
	Rarity      string        `json:"rarity"`
	Element     string        `json:"element,omitempty"`
	Name        LocalizedText `json:"name"`
	Description LocalizedText `json:"description,omitempty"`
	Art         string        `json:"art"`
}

// CardSet is a versioned collection of cards, the root of a cards data file.
type CardSet struct {
	SchemaVersion int    `json:"schemaVersion"`
	Cards         []Card `json:"cards"`
}

func (c Card) problems() problems {
	var p problems

	checkIdentifier(&p, "id", c.ID)
	if c.Number < 0 {
		p.addf("number", "must not be negative, got %d", c.Number)
	}
	checkIdentifier(&p, "rarity", c.Rarity)
	if c.Element != "" {
		checkIdentifier(&p, "element", c.Element)
	}
	p = append(p, c.Name.validate("name", true)...)
	p = append(p, c.Description.validate("description", false)...)
	checkRelativePath(&p, "art", c.Art)

	return p
}

// Validate reports every structural problem of the card.
func (c Card) Validate() error {
	return c.problems().err()
}

// Validate reports every structural problem of the set, including duplicate ids.
func (s CardSet) Validate() error {
	var p problems

	checkSchemaVersion(&p, s.SchemaVersion)
	if len(s.Cards) == 0 {
		p.addf("cards", "at least one card is required")
	}

	seen := make(map[string]bool, len(s.Cards))
	for i, card := range s.Cards {
		prefix := fmt.Sprintf("cards[%d]", i)
		p.addNested(prefix, card.problems())

		if seen[card.ID] {
			p.addf(prefix+".id", "duplicate id %q", card.ID)
		}
		seen[card.ID] = true
	}

	return p.err()
}
