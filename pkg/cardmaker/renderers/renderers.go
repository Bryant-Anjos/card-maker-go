// Package renderers selects a render.Renderer by name, so every command
// offers the same choices.
package renderers

import (
	"fmt"

	"github.com/Bryant-Anjos/card-maker-go/pkg/cardmaker/render"
	"github.com/Bryant-Anjos/card-maker-go/pkg/cardmaker/render/chrome"
)

// Renderer names.
const (
	Chrome = "chrome" // real rendering in a headless Chromium-based browser
	Fake   = "fake"   // flat colored rectangles, no browser needed
)

// Open starts the named renderer and returns it with a function that releases
// its resources (the browser, for chrome). browserPath only applies to chrome;
// when empty the browser is looked up as described by chrome.FindBrowser.
func Open(name, browserPath string) (render.Renderer, func() error, error) {
	switch name {
	case Fake:
		return render.NewFake(), func() error { return nil }, nil
	case Chrome:
		r, err := chrome.New(chrome.Options{BrowserPath: browserPath})
		if err != nil {
			return nil, nil, err
		}

		return r, r.Close, nil
	default:
		return nil, nil, fmt.Errorf("unknown renderer %q, use %s or %s", name, Chrome, Fake)
	}
}
