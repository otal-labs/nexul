package inprocess

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
)

func TestRecover_PanicsBecomeRetryableErrors(t *testing.T) {
	mw := Recover()
	tests := []struct {
		name      string
		panicWith any
	}{
		{"string panic", "kaboom"},
		{"error panic", errors.New("kaboom")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := mw(func(ctx context.Context, ev eventbus.Event) error {
				panic(tt.panicWith)
			})
			err := h(context.Background(), eventbus.Event{ID: "e1"})
			require.Error(t, err)
			require.ErrorIs(t, err, apperrs.ErrRetryable)
		})
	}
}

func TestRecover_NoPanic_PassesThrough(t *testing.T) {
	mw := Recover()
	h := mw(func(ctx context.Context, ev eventbus.Event) error {
		return nil
	})
	require.NoError(t, h(context.Background(), eventbus.Event{ID: "e1"}))
}

func TestRecover_HandlerError_PassesThroughUnchanged(t *testing.T) {
	mw := Recover()
	h := mw(func(ctx context.Context, ev eventbus.Event) error {
		return apperrs.Fatal(errBoom)
	})
	err := h(context.Background(), eventbus.Event{ID: "e1"})
	require.ErrorIs(t, err, apperrs.ErrFatal)
	assert.NotErrorIs(t, err, apperrs.ErrRetryable)
}
