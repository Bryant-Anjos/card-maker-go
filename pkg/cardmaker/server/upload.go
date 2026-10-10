package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg" // register decoders to check uploaded artwork
	_ "image/png"
	"io"
	"net/http"
	"strings"

	_ "golang.org/x/image/webp"

	"github.com/Bryant-Anjos/card-maker-go/pkg/cardmaker/model"
)

// uploadedArt is the name given to the uploaded artwork inside the card data.
const uploadedArt = "upload"

// multipartMemory is how much of a multipart body is kept in memory; larger
// parts spill to temporary files (the body itself is capped by MaxUploadBytes).
const multipartMemory = 1 << 20

// readUpload parses the multipart body of a render request into a validated
// card and its artwork bytes.
func (s *Server) readUpload(w http.ResponseWriter, r *http.Request) (model.Card, []byte, *apiError) {
	r.Body = http.MaxBytesReader(w, r.Body, s.cfg.MaxUploadBytes)

	if err := r.ParseMultipartForm(multipartMemory); err != nil {
		return model.Card{}, nil, uploadError(err)
	}
	defer func() { _ = r.MultipartForm.RemoveAll() }()

	card, apiErr := decodeCard(r)
	if apiErr != nil {
		return model.Card{}, nil, apiErr
	}
	art, apiErr := s.readArt(r)
	if apiErr != nil {
		return model.Card{}, nil, apiErr
	}

	return card, art, nil
}

func uploadError(err error) *apiError {
	var tooLarge *http.MaxBytesError
	switch {
	case errors.As(err, &tooLarge):
		return &apiError{status: http.StatusRequestEntityTooLarge,
			message: fmt.Sprintf("request is larger than %d bytes", tooLarge.Limit)}
	case errors.Is(err, http.ErrNotMultipart):
		return badRequest("request must be multipart/form-data with the parts 'card' (JSON) and 'art' (image)")
	default:
		return badRequest("invalid multipart request: " + err.Error())
	}
}

// maxCardJSONBytes bounds the card data part; real cards are a few hundred bytes.
const maxCardJSONBytes = 1 << 20

// cardJSON returns the "card" part, sent either as a plain field (curl -F) or
// as a file (what browsers and fetch send for a Blob).
func cardJSON(r *http.Request) (string, *apiError) {
	if values := r.MultipartForm.Value["card"]; len(values) > 0 {
		return values[0], nil
	}

	files := r.MultipartForm.File["card"]
	if len(files) == 0 {
		return "", badRequest("missing 'card' part with the card data as JSON")
	}
	file, err := files[0].Open()
	if err != nil {
		return "", badRequest("card: cannot read the uploaded part")
	}
	defer file.Close()

	data, err := io.ReadAll(io.LimitReader(file, maxCardJSONBytes+1))
	if err != nil || len(data) > maxCardJSONBytes {
		return "", badRequest("card: the part is unreadable or larger than 1 MiB")
	}

	return string(data), nil
}

func decodeCard(r *http.Request) (model.Card, *apiError) {
	raw, apiErr := cardJSON(r)
	if apiErr != nil {
		return model.Card{}, apiErr
	}

	var card model.Card
	dec := json.NewDecoder(strings.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&card); err != nil {
		return model.Card{}, badRequest("card: invalid JSON: " + err.Error())
	}
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return model.Card{}, badRequest("card: unexpected data after the JSON document")
	}

	// The artwork comes from the 'art' part, whatever the card says.
	card.Art = uploadedArt

	return card, nil
}

func (s *Server) readArt(r *http.Request) ([]byte, *apiError) {
	files := r.MultipartForm.File["art"]
	if len(files) == 0 {
		return nil, badRequest("missing 'art' file part with the card artwork")
	}

	file, err := files[0].Open()
	if err != nil {
		return nil, badRequest("art: cannot read the uploaded file")
	}
	defer file.Close()

	art, err := io.ReadAll(file)
	if err != nil {
		return nil, badRequest("art: cannot read the uploaded file")
	}

	cfg, _, err := image.DecodeConfig(bytes.NewReader(art))
	if err != nil {
		return nil, badRequest("art: not a supported image (png, jpeg or webp)")
	}
	if pixels := int64(cfg.Width) * int64(cfg.Height); pixels > s.cfg.MaxArtPixels {
		return nil, badRequest(fmt.Sprintf("art: %d pixels exceed the limit of %d pixels", pixels, s.cfg.MaxArtPixels))
	}

	return art, nil
}
