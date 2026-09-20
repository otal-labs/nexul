package mcp

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

func TestMapError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want int
	}{
		{"not found", apperrs.ErrNotFound, CodeNotFound},
		{"unauthorized", apperrs.ErrUnauthorized, CodeUnauthorized},
		{"invalid", apperrs.ErrInvalid, CodeInvalidParams},
		{"wrapped not found", fmt.Errorf("get doc: %w", apperrs.ErrNotFound), CodeNotFound},
		{"conflict falls to internal", apperrs.ErrConflict, CodeInternalError},
		{"retryable falls to internal", apperrs.Retryable(errors.New("boom")), CodeInternalError},
		{"generic falls to internal", errors.New("boom"), CodeInternalError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, mapError(tt.err).Code)
		})
	}
}
