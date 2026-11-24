package logging

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/url"

	"golang.org/x/time/rate"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/httpx"
	"github.com/otal-labs/nexul/internal/platform/redact"
)

const (
	maxCrashReports = 20
	maxCrashStack   = 8 << 10
	crashesPerSec   = 1
	crashBurst      = 20
)

// crashReport is one entry of a Reporting API upload; only the crash type is read.
type crashReport struct {
	Type      string `json:"type"`
	URL       string `json:"url"`
	UserAgent string `json:"user_agent"`
	Body      struct {
		Reason         string `json:"reason"`
		Stack          string `json:"stack"`
		IsTopLevel     *bool  `json:"is_top_level"`
		PageVisibility string `json:"page_visibility"`
	} `json:"body"`
}

// CrashReportHandler logs the crash reports a browser uploads for a tab it killed, under source=web. The browser sends
// them without the page's credentials, so the route is public: one shared rate limit bounds it, and a line names no user.
func CrashReportHandler(logger *slog.Logger) http.Handler {
	limit := rate.NewLimiter(crashesPerSec, crashBurst)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !limit.Allow() {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, maxBrowserBody)
		var reports []crashReport
		if err := httpx.DecodeJSON(r, &reports); err != nil {
			httpx.WriteError(w, err)
			return
		}
		if len(reports) > maxCrashReports {
			httpx.WriteError(w, fmt.Errorf("%w: at most %d reports", apperrs.ErrInvalid, maxCrashReports))
			return
		}
		l := logger.With("source", "web")
		for _, rep := range reports {
			if rep.Type != "crash" {
				continue
			}
			attrs := map[string]any{"reason": rep.Body.Reason, "page_visibility": rep.Body.PageVisibility}
			if rep.Body.IsTopLevel != nil {
				attrs["is_top_level"] = *rep.Body.IsTopLevel
			}
			if rep.Body.Stack != "" {
				attrs["stack"] = redact.Tokens(truncate(rep.Body.Stack, maxCrashStack))
			}
			l.LogAttrs(r.Context(), slog.LevelError, "browser tab crashed ("+rep.Body.Reason+")",
				slog.String("url", redact.Tokens(pathOf(rep.URL))), slog.String("user_agent", rep.UserAgent), slog.Any("attrs", attrs))
		}
		w.WriteHeader(http.StatusNoContent)
	})
}

// pathOf keeps a report URL's path, as a browser log line's url does; its query may hold anything.
func pathOf(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	return u.Path
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
