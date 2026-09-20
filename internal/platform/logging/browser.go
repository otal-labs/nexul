package logging

import (
	"fmt"
	"log/slog"
	"net/http"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/httpx"
)

const (
	maxBrowserBatch = 50
	maxBrowserBody  = 64 << 10
)

// BrowserRecord is one log line the web UI captured (console.error/warn, uncaught errors, unhandled rejections).
type BrowserRecord struct {
	Level   string         `json:"level"`
	Message string         `json:"message"`
	URL     string         `json:"url"`
	Attrs   map[string]any `json:"attrs,omitempty"`
}

// BrowserHandler relays batches the web UI posts through the server's logger, so browser errors land in the same sinks
// (stderr, OTLP) as server logs without exposing the ingest token to the browser.
func BrowserHandler(logger *slog.Logger, userID func(*http.Request) string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxBrowserBody)
		var body struct {
			Records []BrowserRecord `json:"records"`
		}
		if err := httpx.DecodeJSON(r, &body); err != nil {
			httpx.WriteError(w, err)
			return
		}
		if len(body.Records) == 0 || len(body.Records) > maxBrowserBatch {
			httpx.WriteError(w, fmt.Errorf("%w: records must hold 1..%d entries", apperrs.ErrInvalid, maxBrowserBatch))
			return
		}
		l := logger.With("source", "web", "user_id", userID(r), "user_agent", r.UserAgent())
		for _, rec := range body.Records {
			l.LogAttrs(r.Context(), browserLevel(rec.Level), rec.Message, slog.String("url", rec.URL), slog.Any("attrs", rec.Attrs))
		}
		w.WriteHeader(http.StatusNoContent)
	})
}

// browserLevel maps the console method name; anything unknown is treated as an error so nothing is silently downgraded.
func browserLevel(level string) slog.Level {
	switch level {
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	default:
		return slog.LevelError
	}
}
