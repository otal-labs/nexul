package mcp

import (
	"context"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestNew_RegistersTheWholeRegistry builds the real endpoint, since the SDK rejects a malformed tool or resource URI
// by panicking at startup rather than in any handler test.
func TestNew_RegistersTheWholeRegistry(t *testing.T) {
	assert.NotPanics(t, func() {
		New(RegistryOptions{Logger: slog.Default(), InstanceURL: func(context.Context) (string, error) { return testInstanceURL, nil }})
	})
}
