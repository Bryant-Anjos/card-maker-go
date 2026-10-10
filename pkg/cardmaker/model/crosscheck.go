package model

import (
	"fmt"
	"strings"
)

// ValidateAgainst checks that every card refers to a rarity, an element and a
// variant that the theme actually defines. It does not re-run structural validation;
// call Validate for that.
func (s CardSet) ValidateAgainst(t Theme) error {
	var p problems

	for i, card := range s.Cards {
		prefix := fmt.Sprintf("cards[%d]", i)

		if _, ok := t.Rarity(card.Rarity); !ok {
			p.addf(prefix+".rarity", "unknown rarity %q, theme %q defines: %s",
				card.Rarity, t.Name, rarityIDs(t))
		}
		if card.Element != "" {
			if _, ok := t.Element(card.Element); !ok {
				p.addf(prefix+".element", "unknown element %q, %s", card.Element, elementHint(t))
			}
		}
		if card.Variant != "" {
			if _, ok := t.Variant(card.Variant); !ok {
				p.addf(prefix+".variant", "unknown variant %q, %s", card.Variant, variantHint(t))
			}
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

func variantHint(t Theme) string {
	if len(t.Variants) == 0 {
		return fmt.Sprintf("theme %q has no variants", t.Name)
	}

	ids := make([]string, len(t.Variants))
	for i, v := range t.Variants {
		ids[i] = v.ID
	}

	return fmt.Sprintf("theme %q defines: %s", t.Name, strings.Join(ids, ", "))
}
