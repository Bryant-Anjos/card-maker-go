// Package pipeline builds a full card collection: it validates the inputs,
// renders every card through a render.Renderer, skips work that is already up
// to date and writes the images plus a manifest.json.
package pipeline

import (
	"errors"
	"fmt"
	"slices"

	"github.com/Bryant-Anjos/card-maker-go/pkg/cardmaker/i18n"
	"github.com/Bryant-Anjos/card-maker-go/pkg/cardmaker/imgenc"
)

const defaultConcurrency = 4

// Options configures a build. CardsFile, ThemeDir, OutDir and Locale are
// required; everything else has a default.
type Options struct {
	CardsFile string // cards data file; art paths are relative to its directory
	ThemeDir  string // theme directory containing theme.json
	OutDir    string // root output directory; files go to OutDir/<locale>/
	Locale    string // output locale, for example "pt-BR"
	Fallback  string // locale used when a text has no translation; defaults to Locale

	Scales      []int    // image scales, default [1, 2]
	Formats     []string // image formats, default ["png"]
	Concurrency int      // parallel cards, default 4
	Force       bool     // ignore the cache and render everything
}

// normalized validates the options and fills in defaults. It never mutates
// the receiver.
func (o Options) normalized() (Options, error) {
	var errs []error

	if o.CardsFile == "" {
		errs = append(errs, errors.New("cards file is required"))
	}
	if o.ThemeDir == "" {
		errs = append(errs, errors.New("theme directory is required"))
	}
	if o.Fallback == "" {
		o.Fallback = o.Locale
	}
	if _, err := i18n.NewResolver(o.Locale, o.Fallback); err != nil {
		errs = append(errs, err)
	}

	scales, err := normalizeScales(o.Scales)
	errs = append(errs, err)
	formats, err := normalizeFormats(o.Formats)
	errs = append(errs, err)

	switch {
	case o.Concurrency < 0:
		errs = append(errs, fmt.Errorf("concurrency must not be negative, got %d", o.Concurrency))
	case o.Concurrency == 0:
		o.Concurrency = defaultConcurrency
	}

	if err := errors.Join(errs...); err != nil {
		return Options{}, err
	}

	o.Scales, o.Formats = scales, formats

	return o, nil
}

func normalizeScales(scales []int) ([]int, error) {
	if len(scales) == 0 {
		return []int{1, 2}, nil
	}

	out := slices.Clone(scales)
	for _, scale := range out {
		if scale < 1 {
			return nil, fmt.Errorf("scale must be at least 1, got %d", scale)
		}
	}
	slices.Sort(out)

	return slices.Compact(out), nil
}

func normalizeFormats(formats []string) ([]string, error) {
	if len(formats) == 0 {
		return []string{imgenc.PNG}, nil
	}

	var out []string
	for _, format := range formats {
		if !imgenc.IsSupported(format) {
			return nil, fmt.Errorf("unsupported format %q, use %s or %s", format, imgenc.PNG, imgenc.WebP)
		}
		if !slices.Contains(out, format) {
			out = append(out, format)
		}
	}

	return out, nil
}
