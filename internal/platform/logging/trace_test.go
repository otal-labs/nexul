package logging

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTraceID_FromEmptyContextIsEmpty(t *testing.T) {
	assert.Equal(t, "", TraceIDFromCtx(context.Background()))
}

func TestTraceID_RoundTripsThroughContext(t *testing.T) {
	ctx := CtxWithTraceID(context.Background(), "trace-abc")
	assert.Equal(t, "trace-abc", TraceIDFromCtx(ctx))
}

func TestNewTraceID_IsUUIDv7(t *testing.T) {
	id, err := uuid.Parse(NewTraceID())
	require.NoError(t, err)
	assert.Equal(t, uuid.Version(7), id.Version())
}

func TestTraceID_NotInChildOverwrite(t *testing.T) {
	ctx := CtxWithTraceID(context.Background(), "a")
	ctx = CtxWithTraceID(ctx, "b")
	assert.Equal(t, "b", TraceIDFromCtx(ctx))
}
