// Package apperrors defines the cross-cutting sentinel errors; named so callers can also import "errors".
package apperrors

import (
	std "errors"
	"fmt"
)

var (
	ErrNotFound     = std.New("not found")
	ErrConflict     = std.New("conflict")
	ErrUnauthorized = std.New("unauthorized")
	ErrForbidden    = std.New("forbidden")
	ErrInvalid      = std.New("invalid")
	ErrRetryable    = std.New("retryable")
	ErrFatal        = std.New("fatal")
)

// Retryable wraps err so errors.Is(err, ErrRetryable) holds. Event handlers
// return it for transient failures the bus middleware should retry.
func Retryable(err error) error {
	return fmt.Errorf("%w: %w", ErrRetryable, err)
}

// Fatal wraps err so errors.Is(err, ErrFatal) holds. Event handlers return it
// for permanent failures the bus middleware should dead-letter immediately.
func Fatal(err error) error {
	return fmt.Errorf("%w: %w", ErrFatal, err)
}

// Coded is an error carrying a stable machine code adapters report instead of the sentinel's generic one.
type Coded struct {
	Code string
	Err  error
}

func (e *Coded) Error() string { return e.Err.Error() }
func (e *Coded) Unwrap() error { return e.Err }

// WithCode wraps err, which still matches its sentinel, with a machine code such as "invalid_code".
func WithCode(code string, err error) error {
	return &Coded{Code: code, Err: err}
}
