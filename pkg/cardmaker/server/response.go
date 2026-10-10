package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
)

// envelope is the JSON shape of every non-image response: a success flag, the
// payload (null on error) and an error message (null on success).
type envelope struct {
	Success bool    `json:"success"`
	Data    any     `json:"data"`
	Error   *string `json:"error"`
}

// apiError is an error with an HTTP status that is safe to show to clients.
type apiError struct {
	status     int
	message    string
	retryAfter int // seconds; sets the Retry-After header when positive
}

func (e *apiError) Error() string { return e.message }

func badRequest(message string) *apiError {
	return &apiError{status: http.StatusBadRequest, message: message}
}

func notFound(message string) *apiError {
	return &apiError{status: http.StatusNotFound, message: message}
}

// oneLine flattens a multi-line validation error for a JSON message.
func oneLine(err error) string {
	return strings.ReplaceAll(err.Error(), "\n", "; ")
}

func writeJSON(w http.ResponseWriter, status int, body envelope) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeData(w http.ResponseWriter, data any) {
	writeJSON(w, http.StatusOK, envelope{Success: true, Data: data})
}

func writeAPIError(w http.ResponseWriter, e *apiError) {
	if e.retryAfter > 0 {
		w.Header().Set("Retry-After", strconv.Itoa(e.retryAfter))
	}
	message := e.message
	writeJSON(w, e.status, envelope{Error: &message})
}

func writeImage(w http.ResponseWriter, data []byte, format string) {
	w.Header().Set("Content-Type", "image/"+format)
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

// isClientGone reports errors caused by the client disconnecting, which need
// no response.
func isClientGone(err error) bool {
	return errors.Is(err, context.Canceled)
}
