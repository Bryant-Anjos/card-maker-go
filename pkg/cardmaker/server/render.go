package server

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/Bryant-Anjos/card-maker-go/pkg/cardmaker/i18n"
	"github.com/Bryant-Anjos/card-maker-go/pkg/cardmaker/imgenc"
	"github.com/Bryant-Anjos/card-maker-go/pkg/cardmaker/model"
	"github.com/Bryant-Anjos/card-maker-go/pkg/cardmaker/render"
)

const (
	defaultLocale = "pt-BR"
	retryAfterSec = 2
)

// output is the requested size and format of an image.
type output struct {
	scale  int
	format string
}

func (s *Server) parseOutput(q url.Values) (output, *apiError) {
	out := output{scale: 1, format: imgenc.PNG}

	if raw := q.Get("scale"); raw != "" {
		scale, err := strconv.Atoi(raw)
		if err != nil || scale < 1 || scale > s.cfg.MaxScale {
			return output{}, badRequest(fmt.Sprintf("scale must be a whole number from 1 to %d", s.cfg.MaxScale))
		}
		out.scale = scale
	}
	if raw := q.Get("format"); raw != "" {
		if !imgenc.IsSupported(raw) {
			return output{}, badRequest(fmt.Sprintf("unsupported format %q, use png or webp", raw))
		}
		out.format = raw
	}

	return out, nil
}

// acquire waits for a free render slot. It gives up with 503 after
// QueueTimeout, so a flood of requests cannot pile up browser tabs.
func (s *Server) acquire(ctx context.Context) (release func(), err error) {
	timer := time.NewTimer(s.cfg.QueueTimeout)
	defer timer.Stop()

	select {
	case s.sem <- struct{}{}:
		return func() { <-s.sem }, nil
	case <-timer.C:
		return nil, &apiError{status: http.StatusServiceUnavailable,
			message: "the server is busy, retry shortly", retryAfter: retryAfterSec}
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// handleRenderCard renders one card from a multipart request: the part "card"
// (JSON card data) and the part "art" (the artwork file).
func (s *Server) handleRenderCard(w http.ResponseWriter, r *http.Request) error {
	q := r.URL.Query()

	th, apiErr := s.themeNamed(q.Get("theme"))
	if apiErr != nil {
		return apiErr
	}
	out, apiErr := s.parseOutput(q)
	if apiErr != nil {
		return apiErr
	}
	locale := q.Get("locale")
	if locale == "" {
		locale = defaultLocale
	}
	resolver, err := i18n.NewResolver(locale, firstNonEmpty(q.Get("fallback"), locale))
	if err != nil {
		return badRequest(err.Error())
	}

	card, art, apiErr := s.readUpload(w, r)
	if apiErr != nil {
		return apiErr
	}

	set := model.CardSet{SchemaVersion: model.SchemaVersion, Cards: []model.Card{card}}
	if err := set.Validate(); err != nil {
		return badRequest(oneLine(err))
	}
	if err := set.ValidateAgainst(th.Theme); err != nil {
		return badRequest(oneLine(err))
	}
	resolved, err := resolver.Card(card)
	if err != nil {
		return badRequest(err.Error())
	}

	release, err := s.acquire(r.Context())
	if err != nil {
		return err
	}
	defer release()

	pngData, err := s.cfg.Renderer.RenderCard(r.Context(), render.CardRequest{
		Theme: th, Card: resolved, Art: art, Scale: out.scale,
	})

	return s.respondImage(w, pngData, err, th.Size, out)
}

// handleBack renders the card back of a theme.
func (s *Server) handleBack(w http.ResponseWriter, r *http.Request) error {
	th, apiErr := s.themeNamed(r.PathValue("name"))
	if apiErr != nil {
		return apiErr
	}
	if th.Back == "" {
		return notFound(fmt.Sprintf("theme %q has no back", r.PathValue("name")))
	}
	out, apiErr := s.parseOutput(r.URL.Query())
	if apiErr != nil {
		return apiErr
	}

	release, err := s.acquire(r.Context())
	if err != nil {
		return err
	}
	defer release()

	pngData, err := s.cfg.Renderer.RenderBack(r.Context(), render.BackRequest{Theme: th, Scale: out.scale})

	return s.respondImage(w, pngData, err, th.Size, out)
}

func (s *Server) respondImage(w http.ResponseWriter, pngData []byte, renderErr error, size model.Size, out output) error {
	if renderErr != nil {
		return fmt.Errorf("render: %w", renderErr)
	}
	if err := imgenc.CheckPNGSize(pngData, size.Width*out.scale, size.Height*out.scale); err != nil {
		return err
	}
	encoded, err := imgenc.Encode(pngData, out.format)
	if err != nil {
		return err
	}

	writeImage(w, encoded, out.format)

	return nil
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}

	return ""
}
