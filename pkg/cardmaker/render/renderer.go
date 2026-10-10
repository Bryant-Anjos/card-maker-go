// Package render defines how a card becomes an image. Renderers are
// interchangeable: the pipeline only depends on the Renderer interface.
package render

import (
	"context"

	"github.com/Bryant-Anjos/card-maker-go/pkg/cardmaker/i18n"
	"github.com/Bryant-Anjos/card-maker-go/pkg/cardmaker/theme"
)

// CardRequest asks for one card at one scale. Art holds the raw bytes of the
// card's artwork file.
type CardRequest struct {
	Theme theme.Theme
	Card  i18n.ResolvedCard
	Art   []byte
	Scale int
}

// BackRequest asks for the card back at one scale.
type BackRequest struct {
	Theme theme.Theme
	Scale int
}

// Renderer turns requests into PNG images. Implementations must return an
// image whose pixel size is the theme size multiplied by the request scale.
type Renderer interface {
	// Version identifies the renderer implementation. It is part of the
	// pipeline cache key, so bump it whenever output changes.
	Version() string
	RenderCard(ctx context.Context, req CardRequest) ([]byte, error)
	RenderBack(ctx context.Context, req BackRequest) ([]byte, error)
}
