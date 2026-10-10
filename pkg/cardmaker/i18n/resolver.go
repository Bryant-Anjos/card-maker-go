// Package i18n resolves the localized texts of cards for one output locale.
package i18n

import (
	"errors"
	"fmt"

	"github.com/Bryant-Anjos/card-maker-go/pkg/cardmaker/model"
)

// ResolvedCard is a card whose texts were resolved for a single locale. It is
// what renderers consume, so they never deal with localization themselves.
type ResolvedCard struct {
	ID          string
	Number      int
	Rarity      string
	Element     string
	Name        string
	Description string
	Art         string
}

// Resolver picks card texts for Locale, falling back to Fallback when a text
// has no translation. Regional locales fall back to their language first
// ("en-US" -> "en").
type Resolver struct {
	locale   string
	fallback string
}

// NewResolver validates both locale codes and returns a Resolver.
func NewResolver(locale, fallback string) (Resolver, error) {
	for _, code := range []struct{ field, value string }{{"locale", locale}, {"fallback", fallback}} {
		if !model.IsValidLocale(code.value) {
			return Resolver{}, fmt.Errorf("invalid %s code %q", code.field, code.value)
		}
	}

	return Resolver{locale: locale, fallback: fallback}, nil
}

// Card resolves one card. The name is required; the description is optional.
func (r Resolver) Card(c model.Card) (ResolvedCard, error) {
	name, ok := c.Name.Resolve(r.locale, r.fallback)
	if !ok {
		return ResolvedCard{}, fmt.Errorf("card %q: no name for locale %q (fallback %q)", c.ID, r.locale, r.fallback)
	}
	description, _ := c.Description.Resolve(r.locale, r.fallback)

	return ResolvedCard{
		ID:          c.ID,
		Number:      c.Number,
		Rarity:      c.Rarity,
		Element:     c.Element,
		Name:        name,
		Description: description,
		Art:         c.Art,
	}, nil
}

// Cards resolves every card, reporting all failures together. It returns nil
// cards when any card fails.
func (r Resolver) Cards(cards []model.Card) ([]ResolvedCard, error) {
	resolved := make([]ResolvedCard, 0, len(cards))

	var errs []error
	for _, c := range cards {
		rc, err := r.Card(c)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		resolved = append(resolved, rc)
	}
	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}

	return resolved, nil
}
