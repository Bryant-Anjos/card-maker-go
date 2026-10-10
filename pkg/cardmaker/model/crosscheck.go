package model

import (
	"fmt"
	"strings"
)

// ValidateAgainst checks that every card refers to a rarity and an element
// that the theme actually defines. It does not re-run structural validation;
// call Validate for that.
func (s CardSet) ValidateAgainst(t Theme) error {
	var p problems

	for i, card := range s.Cards {
		prefix := fmt.Sprintf("cards[%d]", i)

		if _, ok := t.Rarity(card.Rarity); !ok {
			p.addf(prefix+".rarity", "unknown rarity %q, theme %q defines: %s",
				card.Rarity, t.Name, rarityIDs(t))
		}
		if card.Element == "" {
			continue
		}
		if _, ok := t.Element(card.Element); !ok {
			p.addf(prefix+".element", "unknown element %q, %s", card.Element, elementHint(t))
		}
	}

	return p.err()
}

func rarityIDs(t Theme) string {
	ids := make([]string, len(t.Rarities))
	for i, r := range t.Rarities {
		ids[i] = r.ID
	}

	return strings.Join(ids, ", ")
}

func elementHint(t Theme) string {
	if len(t.Elements) == 0 {
		return fmt.Sprintf("theme %q has no elements", t.Name)
	}

	ids := make([]string, len(t.Elements))
	for i, e := range t.Elements {
		ids[i] = e.ID
	}

	return fmt.Sprintf("theme %q defines: %s", t.Name, strings.Join(ids, ", "))
}
