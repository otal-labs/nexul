package logging

import (
	"context"

	"github.com/google/uuid"
)

type traceKey struct{}

// NewTraceID returns a time-sortable UUIDv7, the trace_id format every entry
// point stamps. Falls back to a random v4 if the PRNG fails.
func NewTraceID() string {
	id, err := uuid.NewV7()
	if err != nil {
		return uuid.NewString()
	}
	return id.String()
}

// TraceIDFromCtx returns the trace_id carried in ctx, or "" if none was set.
func TraceIDFromCtx(ctx context.Context) string {
	if v, ok := ctx.Value(traceKey{}).(string); ok {
		return v
	}
	return ""
}

// CtxWithTraceID returns a child context carrying the trace_id.
func CtxWithTraceID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, traceKey{}, id)
}
