package apperrors

import (
	std "errors"
	"testing"
)

func TestRetryable_WrapsErrRetryable(t *testing.T) {
	cause := std.New("boom")
	err := Retryable(cause)
	if !std.Is(err, ErrRetryable) {
		t.Fatalf("Retryable(%v): want ErrRetryable in chain, got %v", cause, err)
	}
	if !std.Is(err, cause) {
		t.Fatalf("Retryable(%v): want cause preserved, got %v", cause, err)
	}
}

func TestFatal_WrapsErrFatal(t *testing.T) {
	cause := std.New("boom")
	err := Fatal(cause)
	if !std.Is(err, ErrFatal) {
		t.Fatalf("Fatal(%v): want ErrFatal in chain, got %v", cause, err)
	}
	if !std.Is(err, cause) {
		t.Fatalf("Fatal(%v): want cause preserved, got %v", cause, err)
	}
}
