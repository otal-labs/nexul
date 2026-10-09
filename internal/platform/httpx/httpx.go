// Package httpx provides the JSON/error adapter helpers shared by the domain HTTP gateways (ADR 0019).
package httpx

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/jsonx"
)

// Envelope is the error body every adapter returns.
type Envelope struct {
	Message string `json:"message"`
	Code    string `json:"code"`
	// Errors keys the message under the input that caused it, so a form can show it on that field.
	Errors map[string][]string `json:"errors,omitempty"`
	// Details is the error's structured side when it has one (DetailedError), so a client never parses the message.
	Details any `json:"details,omitempty"`
}

// DetailedError is a domain error that carries structured fields for the envelope's details.
type DetailedError interface {
	ErrorDetails() any
}

// WriteJSON encodes v through jsonx, so a nil slice reaches the browser as [] rather than null.
func WriteJSON(w http.ResponseWriter, status int, v any) {
	b, err := jsonx.Marshal(v)
	if err != nil {
		http.Error(w, "encode response", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(b)
}

// WriteError maps a domain error to its status + envelope and writes it.
func WriteError(w http.ResponseWriter, err error) {
	status, code, message := mapError(err)
	WriteJSON(w, status, Envelope{Message: message, Code: code, Details: details(status, err)})
}

// WriteFieldError is WriteError with the message also keyed under the input field that caused it.
func WriteFieldError(w http.ResponseWriter, err error, field string) {
	status, code, message := mapError(err)
	WriteJSON(w, status, Envelope{Message: message, Code: code, Errors: map[string][]string{field: {message}}, Details: details(status, err)})
}

// details stays empty on a 500, whose message is hidden too.
func details(status int, err error) any {
	var d DetailedError
	if status == http.StatusInternalServerError || !errors.As(err, &d) {
		return nil
	}
	return d.ErrorDetails()
}

// DecodeJSON reads a JSON request body, rejecting malformed bodies as
// ErrInvalid (HTTP 400).
func DecodeJSON(r *http.Request, v any) error {
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		return fmt.Errorf("%w: invalid json body", apperrs.ErrInvalid)
	}
	return nil
}

func mapError(err error) (status int, code, message string) {
	status, code, message = mapSentinel(err)
	var coded *apperrs.Coded
	if status != http.StatusInternalServerError && errors.As(err, &coded) {
		code = coded.Code
	}
	return status, code, message
}

func mapSentinel(err error) (status int, code, message string) {
	switch {
	case errors.Is(err, apperrs.ErrNotFound):
		return http.StatusNotFound, "NOT_FOUND", err.Error()
	case errors.Is(err, apperrs.ErrConflict):
		return http.StatusConflict, "CONFLICT", err.Error()
	case errors.Is(err, apperrs.ErrUnauthorized):
		return http.StatusUnauthorized, "UNAUTHORIZED", err.Error()
	case errors.Is(err, apperrs.ErrForbidden):
		return http.StatusForbidden, "FORBIDDEN", err.Error()
	case errors.Is(err, apperrs.ErrInvalid):
		return http.StatusBadRequest, "INVALID", err.Error()
	case errors.Is(err, apperrs.ErrRateLimited):
		return http.StatusTooManyRequests, "RATE_LIMITED", err.Error()
	case errors.Is(err, apperrs.ErrRetryable):
		return http.StatusServiceUnavailable, "RETRYABLE", err.Error()
	case errors.Is(err, apperrs.ErrFatal):
		return http.StatusUnprocessableEntity, "FATAL", err.Error()
	default:
		return http.StatusInternalServerError, "INTERNAL", "internal error"
	}
}

// ClientAddr is the peer's IP, the one client address the server trusts.
// ponytail: callers behind the instance's own proxy share one address; trust a forwarded header if that bites.
func ClientAddr(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
