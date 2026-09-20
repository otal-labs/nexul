package logging

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"

	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
)

// OTLP says where to ship logs besides stderr. An empty Endpoint disables shipping.
type OTLP struct {
	// Endpoint is the OTLP/HTTP base URL; "/v1/logs" is appended.
	Endpoint string
	// User and Token become a basic-auth header; both empty sends no header.
	User  string
	Token string
	// Stream is the destination stream name (OpenObserve's stream-name header).
	Stream string
	// Service is the service.name resource attribute.
	Service string
}

// Shutdown flushes buffered records and releases the exporter; a no-op when shipping is off.
type Shutdown func(ctx context.Context) error

func newOTLPHandler(ctx context.Context, o OTLP) (slog.Handler, Shutdown, error) {
	headers := map[string]string{"stream-name": o.Stream}
	if o.User != "" || o.Token != "" {
		headers["Authorization"] = "Basic " + base64.StdEncoding.EncodeToString([]byte(o.User+":"+o.Token))
	}
	exp, err := otlploghttp.New(ctx,
		otlploghttp.WithEndpointURL(o.Endpoint+"/v1/logs"),
		otlploghttp.WithHeaders(headers),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("otlp log exporter: %w", err)
	}
	res, err := resource.Merge(resource.Default(), resource.NewSchemaless(attribute.String("service.name", o.Service)))
	if err != nil {
		return nil, nil, fmt.Errorf("otlp resource: %w", err)
	}
	provider := sdklog.NewLoggerProvider(
		sdklog.WithProcessor(sdklog.NewBatchProcessor(exp)),
		sdklog.WithResource(res),
	)
	return otelslog.NewHandler(o.Service, otelslog.WithLoggerProvider(provider)), provider.Shutdown, nil
}

// fanout writes each record to every handler; min is the shared level gate because the OTel bridge has none of its own.
type fanout struct {
	min      slog.Leveler
	handlers []slog.Handler
}

func (f fanout) Enabled(_ context.Context, l slog.Level) bool {
	return l >= f.min.Level()
}

func (f fanout) Handle(ctx context.Context, r slog.Record) error {
	var errs []error
	for _, h := range f.handlers {
		if !h.Enabled(ctx, r.Level) {
			continue
		}
		errs = append(errs, h.Handle(ctx, r.Clone()))
	}
	return errors.Join(errs...)
}

func (f fanout) WithAttrs(attrs []slog.Attr) slog.Handler {
	return f.mapHandlers(func(h slog.Handler) slog.Handler { return h.WithAttrs(attrs) })
}

func (f fanout) WithGroup(name string) slog.Handler {
	return f.mapHandlers(func(h slog.Handler) slog.Handler { return h.WithGroup(name) })
}

func (f fanout) mapHandlers(fn func(slog.Handler) slog.Handler) slog.Handler {
	out := make([]slog.Handler, len(f.handlers))
	for i, h := range f.handlers {
		out[i] = fn(h)
	}
	return fanout{min: f.min, handlers: out}
}
