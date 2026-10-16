package logging

import (
	"context"
	"io"
	"log/slog"
	"os"
)

type ctxKey struct{}

func New(level string) *slog.Logger {
	return slog.New(stderrHandler(level))
}

// NewTee is New plus one JSON line per record written to w, at the same level.
func NewTee(level string, w io.Writer) *slog.Logger {
	l := parseLevel(level)
	return slog.New(fanout{min: l, handlers: []slog.Handler{stderrHandler(level), slog.NewJSONHandler(w, &slog.HandlerOptions{Level: l})}})
}

// NewWithQueue is New plus every record at level or above queued for the server to relay.
func NewWithQueue(level string, q *Queue) *slog.Logger {
	return slog.New(fanout{min: parseLevel(level), handlers: []slog.Handler{stderrHandler(level), q.Handler(level)}})
}

// NewWithOTLP is New plus shipping every record to o.Endpoint, and the Relay that writes records senders shipped to
// that endpoint alone. With an empty endpoint it is exactly New, and the Relay is nil.
func NewWithOTLP(ctx context.Context, level string, o OTLP) (*slog.Logger, *Relay, Shutdown, error) {
	stderr := stderrHandler(level)
	if o.Endpoint == "" {
		return slog.New(stderr), nil, func(context.Context) error { return nil }, nil
	}
	otlp, shutdown, err := newOTLPHandler(ctx, o)
	if err != nil {
		return nil, nil, nil, err
	}
	return slog.New(fanout{min: parseLevel(level), handlers: []slog.Handler{stderr, otlp}}), NewRelay(otlp), shutdown, nil
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
