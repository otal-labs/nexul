package main

import (
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/otal-labs/nexul/internal/platform/httpx"
)

// mountLogsProxy serves the local OpenObserve at /openobserve/ outside Nexul auth; OpenObserve keeps its own login.
func mountLogsProxy(mux *httpx.ServeMux, target *url.URL, logger *slog.Logger) {
	if target == nil {
		return
	}
	mux.Handle("/openobserve/", logsProxy(target, logger))
}

func logsProxy(target *url.URL, logger *slog.Logger) http.Handler {
	return &httputil.ReverseProxy{
		Rewrite: func(r *httputil.ProxyRequest) {
			r.SetURL(target)
			// Keep the public Host so anything OpenObserve derives from it points back through this proxy.
			r.Out.Host = r.In.Host
			r.SetXForwarded()
		},
		FlushInterval: -1,
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			logger.Warn("logs proxy upstream unavailable", "path", r.URL.Path, "error", err)
			w.WriteHeader(http.StatusBadGateway)
		},
	}
}
