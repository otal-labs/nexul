package logging

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTraceID_ContextCarriesTheLatestOne(t *testing.T) {
	assert.Equal(t, "", TraceIDFromCtx(context.Background()))
	ctx := CtxWithTraceID(context.Background(), "a")
	assert.Equal(t, "a", TraceIDFromCtx(ctx))
	assert.Equal(t, "b", TraceIDFromCtx(CtxWithTraceID(ctx, "b")))
}

func TestNewTraceID_IsUUIDv7(t *testing.T) {
	id, err := uuid.Parse(NewTraceID())
	require.NoError(t, err)
	assert.Equal(t, uuid.Version(7), id.Version())
}
