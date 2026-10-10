package model

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

// ParseCardSet strictly decodes and validates a cards data file.
func ParseCardSet(r io.Reader) (CardSet, error) {
	return parse[CardSet](r, "card set")
}

// ParseTheme strictly decodes and validates a theme definition.
func ParseTheme(r io.Reader) (Theme, error) {
	return parse[Theme](r, "theme")
}

// ParseManifest strictly decodes and validates a manifest.
func ParseManifest(r io.Reader) (Manifest, error) {
	return parse[Manifest](r, "manifest")
}

type validatable interface {
	Validate() error
}

// parse decodes exactly one JSON document, rejecting unknown fields and
// trailing data, then validates it. Typos in data files fail loudly.
func parse[T validatable](r io.Reader, what string) (T, error) {
	var value T

	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()

	if err := dec.Decode(&value); err != nil {
		return value, fmt.Errorf("decode %s: %w", what, err)
	}
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return value, fmt.Errorf("decode %s: trailing data after JSON document", what)
	}
	if err := value.Validate(); err != nil {
		return value, fmt.Errorf("invalid %s:\n%w", what, err)
	}

	return value, nil
}
