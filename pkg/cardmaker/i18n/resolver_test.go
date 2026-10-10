package i18n_test

import (
	"strings"
	"testing"

	"github.com/Bryant-Anjos/card-maker-go/pkg/cardmaker/i18n"
	"github.com/Bryant-Anjos/card-maker-go/pkg/cardmaker/model"
)

func card(id string, name, description model.LocalizedText) model.Card {
	return model.Card{
		ID: id, Number: 7, Rarity: "rare", Element: "fire",
		Name: name, Description: description, Art: "art/" + id + ".png",
	}
}

func TestNewResolver(t *testing.T) {
	if _, err := i18n.NewResolver("pt-BR", "pt-BR"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := i18n.NewResolver("english", "en"); err == nil {
		t.Error("expected error for invalid locale")
	}
	if _, err := i18n.NewResolver("en", "??"); err == nil {
		t.Error("expected error for invalid fallback")
	}
}

func TestResolverCard(t *testing.T) {
	names := model.LocalizedText{"pt-BR": "Vulpin", "en": "Vulpine"}
	descriptions := model.LocalizedText{"pt-BR": "Curiosa."}

	tests := []struct {
		name     string
		locale   string
		card     model.Card
		wantName string
		wantDesc string
		wantErr  string
	}{
		{"exact locale", "en", card("a", names, descriptions), "Vulpine", "Curiosa.", ""},
		{"falls back for description", "en", card("a", names, descriptions), "Vulpine", "Curiosa.", ""},
		{"regional falls back to language", "en-US", card("a", names, nil), "Vulpine", "", ""},
		{"default locale", "pt-BR", card("a", names, descriptions), "Vulpin", "Curiosa.", ""},
		{"missing name is an error", "fr", card("a", model.LocalizedText{"de": "x"}, nil), "", "", `card "a"`},
		{"description is optional", "pt-BR", card("a", names, nil), "Vulpin", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolver, err := i18n.NewResolver(tt.locale, "pt-BR")
			if err != nil {
				t.Fatal(err)
			}

			got, err := resolver.Card(tt.card)

			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) || !strings.Contains(err.Error(), tt.locale) {
					t.Fatalf("error = %v, want it to mention %q and locale %q", err, tt.wantErr, tt.locale)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.Name != tt.wantName || got.Description != tt.wantDesc {
				t.Errorf("got (%q, %q), want (%q, %q)", got.Name, got.Description, tt.wantName, tt.wantDesc)
			}
		})
	}
}

func TestResolverCardKeepsNonTextFields(t *testing.T) {
	resolver, _ := i18n.NewResolver("pt-BR", "pt-BR")

	input := card("a", model.LocalizedText{"pt-BR": "A"}, nil)
	input.Variant = "fullart"

	got, err := resolver.Card(input)

	if err != nil {
		t.Fatal(err)
	}
	want := i18n.ResolvedCard{ID: "a", Number: 7, Rarity: "rare", Element: "fire", Variant: "fullart", Name: "A", Art: "art/a.png"}
	if got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestResolverCardsReportsEveryFailure(t *testing.T) {
	resolver, _ := i18n.NewResolver("pt-BR", "pt-BR")
	cards := []model.Card{
		card("ok", model.LocalizedText{"pt-BR": "Ok"}, nil),
		card("bad1", model.LocalizedText{"en": "x"}, nil),
		card("bad2", model.LocalizedText{"en": "y"}, nil),
	}

	got, err := resolver.Cards(cards)

	if err == nil {
		t.Fatal("expected error")
	}
	for _, id := range []string{"bad1", "bad2"} {
		if !strings.Contains(err.Error(), id) {
			t.Errorf("error %q does not mention %q", err, id)
		}
	}
	if got != nil {
		t.Errorf("expected nil result on error, got %v", got)
	}

	got, err = resolver.Cards(cards[:1])
	if err != nil || len(got) != 1 || got[0].Name != "Ok" {
		t.Errorf("Cards(valid) = (%v, %v)", got, err)
	}
}
