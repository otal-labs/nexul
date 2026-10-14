// Package apperrors defines the cross-cutting sentinel errors; named so callers can also import "errors".
package apperrors

import (
	std "errors"
	"fmt"
	"strings"
)

var (
	ErrNotFound     = std.New("not found")
	ErrConflict     = std.New("conflict")
	ErrUnauthorized = std.New("unauthorized")
	ErrForbidden    = std.New("forbidden")
	ErrInvalid      = std.New("invalid")
	ErrRetryable    = std.New("retryable")
	ErrFatal        = std.New("fatal")
	ErrRateLimited  = std.New("rate limited")
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

// InUseError refuses removing a thing while others still depend on it and names them; it matches ErrConflict,
// so every adapter reports it as a conflict with this one message.
type InUseError struct {
	// Thing is what was asked to go, as it reads in a sentence, such as `tunnel "edge"`.
	Thing string
	// Users are what depend on it, each as it reads in a list, such as "app.example.com".
	Users []string
	// Fix says what to do first, such as "edit or remove them first".
	Fix string
}

func (e *InUseError) Error() string {
	return fmt.Sprintf("%s is still used by %s; %s", e.Thing, strings.Join(e.Users, ", "), e.Fix)
}

func (e *InUseError) Unwrap() error { return ErrConflict }
