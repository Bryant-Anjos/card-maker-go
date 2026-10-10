package model

import "strings"

// LocalizedText maps a locale code (for example "pt-BR") to a text.
type LocalizedText map[string]string

// Resolve returns the text for locale, trying in order: the exact locale, its
// language-only form ("en-US" -> "en"), then the same two steps for fallback.
// The boolean is false when no non-blank text was found.
func (t LocalizedText) Resolve(locale, fallback string) (string, bool) {
	for _, candidate := range candidates(locale, fallback) {
		if text := strings.TrimSpace(t[candidate]); text != "" {
			return t[candidate], true
		}
	}

	return "", false
}

func candidates(locales ...string) []string {
	var out []string

	for _, locale := range locales {
		out = append(out, locale)
		if language, _, found := strings.Cut(locale, "-"); found {
			out = append(out, language)
		}
	}

	return out
}

func (t LocalizedText) validate(field string, required bool) problems {
	var p problems

	hasText := false
	for locale, text := range t {
		checkLocale(&p, field, locale)
		if strings.TrimSpace(text) != "" {
			hasText = true
		}
	}
	if required && !hasText {
		p.addf(field, "at least one non-empty text is required")
	}

	return p
}
