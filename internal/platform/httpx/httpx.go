// Package httpx provides the JSON/error adapter helpers shared by the domain HTTP gateways (ADR 0019).
package httpx

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// Envelope is the error body every adapter returns.
type Envelope struct {
	Message string `json:"message"`
	Code    string `json:"code"`
	// Errors keys the message under the input that caused it, so a form can show it on that field.
	Errors map[string][]string `json:"errors,omitempty"`
}

// WriteJSON marshals a nil slice as [] instead of null, since the frontend expects arrays.
func WriteJSON(w http.ResponseWriter, status int, v any) {
	v = normalizeNilSlice(v)
	b, err := json.Marshal(v)
	if err != nil {
		http.Error(w, "encode response", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(b)
}

func normalizeNilSlice(v any) any {
	rv := reflect.ValueOf(v)
	if !rv.IsValid() {
		return v
	}
	// Structs/slices/maps arrive by value (unaddressable), so copy them into
	// an addressable slot before walking — reflect requires settable fields.
	if rv.Kind() == reflect.Struct || rv.Kind() == reflect.Slice || rv.Kind() == reflect.Map {
		ptr := reflect.New(rv.Type())
		ptr.Elem().Set(rv)
		rv = ptr.Elem()
	}
	return normalizeNilSlicesValue(rv).Interface()
}

// normalizeNilSlicesValue walks a value and replaces every nil slice with an empty one, including nested fields.
func normalizeNilSlicesValue(rv reflect.Value) reflect.Value {
	switch rv.Kind() {
	case reflect.Slice:
		return normalizeNilSliceKind(rv)
	case reflect.Struct:
		return normalizeNilSliceStruct(rv)
	case reflect.Map:
		return normalizeNilSliceMap(rv)
	default:
		return rv
	}
}

func normalizeNilSliceKind(rv reflect.Value) reflect.Value {
	if rv.IsNil() {
		return reflect.MakeSlice(rv.Type(), 0, 0)
	}
	for i := 0; i < rv.Len(); i++ {
		item := rv.Index(i)
		if !item.CanInterface() || !item.CanSet() {
			continue
		}
		item.Set(normalizeNilSlicesValue(item))
	}
	return rv
}

func normalizeNilSliceStruct(rv reflect.Value) reflect.Value {
	for i := 0; i < rv.NumField(); i++ {
		f := rv.Field(i)
		if !f.CanInterface() || !f.CanSet() {
			continue
		}
		f.Set(normalizeNilSlicesValue(f))
	}
	return rv
}

func normalizeNilSliceMap(rv reflect.Value) reflect.Value {
	if rv.IsNil() {
		return rv
	}
	for _, key := range rv.MapKeys() {
		item := rv.MapIndex(key)
		if !item.CanInterface() {
			continue
		}
		rv.SetMapIndex(key, normalizeNilSlicesValue(item))
	}
	return rv
}

// WriteError maps a domain error to its status + envelope and writes it.
func WriteError(w http.ResponseWriter, err error) {
	status, code, message := mapError(err)
	WriteJSON(w, status, Envelope{Message: message, Code: code})
}

// WriteFieldError is WriteError with the message also keyed under the input field that caused it.
func WriteFieldError(w http.ResponseWriter, err error, field string) {
	status, code, message := mapError(err)
	WriteJSON(w, status, Envelope{Message: message, Code: code, Errors: map[string][]string{field: {message}}})
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
	case errors.Is(err, apperrs.ErrRetryable):
		return http.StatusServiceUnavailable, "RETRYABLE", err.Error()
	case errors.Is(err, apperrs.ErrFatal):
		return http.StatusUnprocessableEntity, "FATAL", err.Error()
	default:
		return http.StatusInternalServerError, "INTERNAL", "internal error"
	}
}
