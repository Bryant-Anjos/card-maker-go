package model_test

import (
	"strings"
	"testing"

	"github.com/Bryant-Anjos/card-maker-go/pkg/cardmaker/model"
)

func validCard() model.Card {
	return model.Card{
		ID:          "vulpin",
		Number:      1,
		Rarity:      "common",
		Element:     "beast",
		Name:        model.LocalizedText{"pt-BR": "Vulpin"},
		Description: model.LocalizedText{"pt-BR": "Pequena criatura curiosa."},
		Art:         "art/vulpin.png",
	}
}

func TestCardValidate(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(c *model.Card)
		wantErr string
	}{
		{"valid card", func(c *model.Card) {}, ""},
		{"valid without element and description", func(c *model.Card) { c.Element = ""; c.Description = nil }, ""},
		{"empty id", func(c *model.Card) { c.ID = "" }, "id"},
		{"id with path separator", func(c *model.Card) { c.ID = "../evil" }, "id"},
		{"id with uppercase", func(c *model.Card) { c.ID = "Vulpin" }, "id"},
		{"negative number", func(c *model.Card) { c.Number = -1 }, "number"},
		{"empty rarity", func(c *model.Card) { c.Rarity = "" }, "rarity"},
		{"missing name", func(c *model.Card) { c.Name = nil }, "name"},
		{"blank name text", func(c *model.Card) { c.Name = model.LocalizedText{"pt-BR": "  "} }, "name"},
		{"invalid locale code", func(c *model.Card) { c.Name = model.LocalizedText{"português": "Vulpin"} }, "locale"},
		{"empty art", func(c *model.Card) { c.Art = "" }, "art"},
		{"absolute art path", func(c *model.Card) { c.Art = "/etc/passwd" }, "art"},
		{"art path with dot segment", func(c *model.Card) { c.Art = "art/./vulpin.png" }, "art"},
		{"art path with empty segment", func(c *model.Card) { c.Art = "art//vulpin.png" }, "art"},
		{"art path with trailing slash", func(c *model.Card) { c.Art = "art/" }, "art"},
		{"art path with backslash", func(c *model.Card) { c.Art = `art\vulpin.png` }, "art"},
		{"art path escaping root", func(c *model.Card) { c.Art = "../secret.png" }, "art"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			card := validCard()
			tt.mutate(&card)

			err := card.Validate()

			assertErr(t, err, tt.wantErr)
		})
	}
}

func TestCardValidateReportsAllProblems(t *testing.T) {
	card := model.Card{}

	err := card.Validate()

	if err == nil {
		t.Fatal("expected errors, got nil")
	}
	for _, field := range []string{"id", "rarity", "name", "art"} {
		if !strings.Contains(err.Error(), field) {
			t.Errorf("error %q does not mention %q", err, field)
		}
	}
}

func TestCardSetValidate(t *testing.T) {
	other := validCard()
	other.ID = "cervallis"

	tests := []struct {
		name    string
		set     model.CardSet
		wantErr string
	}{
		{"valid", model.CardSet{SchemaVersion: 1, Cards: []model.Card{validCard(), other}}, ""},
		{"wrong schema version", model.CardSet{SchemaVersion: 2, Cards: []model.Card{validCard()}}, "schemaVersion"},
		{"no cards", model.CardSet{SchemaVersion: 1}, "cards"},
		{"duplicate ids", model.CardSet{SchemaVersion: 1, Cards: []model.Card{validCard(), validCard()}}, "duplicate"},
		{"invalid card is indexed", model.CardSet{SchemaVersion: 1, Cards: []model.Card{validCard(), {}}}, "cards[1]"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertErr(t, tt.set.Validate(), tt.wantErr)
		})
	}
}

func TestParseCardSet(t *testing.T) {
	t.Run("valid json", func(t *testing.T) {
		src := `{"schemaVersion":1,"cards":[{"id":"vulpin","number":1,"rarity":"common",
			"name":{"pt-BR":"Vulpin"},"art":"art/vulpin.png"}]}`

		set, err := model.ParseCardSet(strings.NewReader(src))

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got := set.Cards[0].Name["pt-BR"]; got != "Vulpin" {
			t.Errorf("name = %q, want Vulpin", got)
		}
	})

	t.Run("unknown field is rejected", func(t *testing.T) {
		src := `{"schemaVersion":1,"cards":[],"surprise":true}`

		_, err := model.ParseCardSet(strings.NewReader(src))

		assertErr(t, err, "surprise")
	})

	t.Run("malformed json", func(t *testing.T) {
		_, err := model.ParseCardSet(strings.NewReader(`{`))

		assertErr(t, err, "decode")
	})

	t.Run("trailing data is rejected", func(t *testing.T) {
		src := `{"schemaVersion":1,"cards":[{"id":"a","rarity":"c","name":{"pt-BR":"A"},"art":"a.png"}]} {}`

		_, err := model.ParseCardSet(strings.NewReader(src))

		assertErr(t, err, "trailing")
	})

	t.Run("semantic errors surface", func(t *testing.T) {
		src := `{"schemaVersion":1,"cards":[{"id":"","rarity":"c","name":{"pt-BR":"A"},"art":"a.png"}]}`

		_, err := model.ParseCardSet(strings.NewReader(src))

		assertErr(t, err, "cards[0].id")
	})
}

func TestLocalizedTextResolve(t *testing.T) {
	text := model.LocalizedText{"pt-BR": "Olá", "en": "Hello"}

	tests := []struct {
		name     string
		text     model.LocalizedText
		locale   string
		fallback string
		want     string
		wantOK   bool
	}{
		{"exact locale", text, "en", "pt-BR", "Hello", true},
		{"falls back to default", text, "fr", "pt-BR", "Olá", true},
		{"language-only falls back from regional", text, "en-US", "pt-BR", "Hello", true},
		{"nothing matches", text, "fr", "de", "", false},
		{"nil text", nil, "pt-BR", "pt-BR", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := tt.text.Resolve(tt.locale, tt.fallback)

			if got != tt.want || ok != tt.wantOK {
				t.Errorf("Resolve(%q,%q) = (%q,%v), want (%q,%v)", tt.locale, tt.fallback, got, ok, tt.want, tt.wantOK)
			}
		})
	}
}

func assertErr(t *testing.T, err error, wantSubstr string) {
	t.Helper()

	if wantSubstr == "" {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		return
	}
	if err == nil {
		t.Fatalf("expected error containing %q, got nil", wantSubstr)
	}
	if !strings.Contains(err.Error(), wantSubstr) {
		t.Fatalf("error %q does not contain %q", err, wantSubstr)
	}
}

func TestParseCardSetAcceptsSchemaHint(t *testing.T) {
	src := `{"$schema":"../schema/cards.schema.json","schemaVersion":1,"cards":[
		{"id":"a","rarity":"c","name":{"pt-BR":"A"},"art":"a.png"}]}`

	set, err := model.ParseCardSet(strings.NewReader(src))

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if set.Schema != "../schema/cards.schema.json" {
		t.Errorf("schema = %q", set.Schema)
	}
}
