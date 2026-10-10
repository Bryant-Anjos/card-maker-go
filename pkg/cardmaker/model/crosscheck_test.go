package model_test

import (
	"strings"
	"testing"

	"github.com/Bryant-Anjos/card-maker-go/pkg/cardmaker/model"
)

func TestCardSetValidateAgainstTheme(t *testing.T) {
	theme := validTheme() // rarities: common, rare; elements: fire

	cardWith := func(id, rarity, element string) model.Card {
		c := validCard()
		c.ID, c.Rarity, c.Element = id, rarity, element
		return c
	}

	tests := []struct {
		name    string
		cards   []model.Card
		wantErr []string
	}{
		{"all known", []model.Card{cardWith("a", "common", "fire"), cardWith("b", "rare", "")}, nil},
		{"unknown rarity", []model.Card{cardWith("a", "mythic", "fire")}, []string{"cards[0].rarity", "mythic", "common, rare"}},
		{"unknown element", []model.Card{cardWith("a", "common", "void")}, []string{"cards[0].element", "void", "fire"}},
		{"reports every card", []model.Card{cardWith("a", "x", ""), cardWith("b", "y", "")}, []string{"cards[0].rarity", "cards[1].rarity"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			set := model.CardSet{SchemaVersion: 1, Cards: tt.cards}

			err := set.ValidateAgainst(theme)

			if len(tt.wantErr) == 0 {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error containing %v, got nil", tt.wantErr)
			}
			for _, want := range tt.wantErr {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("error %q does not contain %q", err, want)
				}
			}
		})
	}
}

func TestValidateAgainstThemeWithoutElements(t *testing.T) {
	theme := validTheme()
	theme.Elements = nil
	card := validCard()
	card.Element = "fire"
	set := model.CardSet{SchemaVersion: 1, Cards: []model.Card{card}}

	err := set.ValidateAgainst(theme)

	if err == nil || !strings.Contains(err.Error(), "cards[0].element") || !strings.Contains(err.Error(), "no elements") {
		t.Fatalf("error = %v, want it to flag the element and say the theme has no elements", err)
	}
}
