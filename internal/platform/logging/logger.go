package logging

import (
	"context"
	"log/slog"
	"os"
)

type ctxKey struct{}

func New(level string) *slog.Logger {
	return slog.New(stderrHandler(level))
}

// NewWithOTLP is New plus shipping every record to o.Endpoint; with an empty endpoint it is exactly New.
func NewWithOTLP(ctx context.Context, level string, o OTLP) (*slog.Logger, Shutdown, error) {
	stderr := stderrHandler(level)
	if o.Endpoint == "" {
		return slog.New(stderr), func(context.Context) error { return nil }, nil
	}
	otlp, shutdown, err := newOTLPHandler(ctx, o)
	if err != nil {
		return nil, nil, err
	}
	return slog.New(fanout{min: parseLevel(level), handlers: []slog.Handler{stderr, otlp}}), shutdown, nil
}

func stderrHandler(level string) slog.Handler {
	return slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: parseLevel(level)})
}

func parseLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func FromCtx(ctx context.Context) *slog.Logger {
	if l, ok := ctx.Value(ctxKey{}).(*slog.Logger); ok {
		return l
	}
	return slog.Default()
}

func CtxWithLogger(ctx context.Context, l *slog.Logger) context.Context {
	return context.WithValue(ctx, ctxKey{}, l)
}
