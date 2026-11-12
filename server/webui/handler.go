// Package webui serves the built web frontend from the server binary in single-binary mode.
package webui

import (
	"bytes"
	"context"
	"io/fs"
	"net/http"
	"path"
	"strings"
)

// reservedPrefixes are owned by other handlers and must never fall back to the SPA's index.html.
var reservedPrefixes = []string{"/api", "/ws", "/mcp", "/auth", "/hooks", "/openobserve"}

// IndexTransform rewrites the SPA's index.html for one request, such as to put an instance's own name in the page head.
type IndexTransform func(ctx context.Context, index []byte) []byte

// Handler serves the embedded SPA with client-side routing fallback, or a 404 build hint when assets is nil. Each
// transform rewrites the index the SPA falls back to.
func Handler(assets fs.FS, transforms ...IndexTransform) http.Handler {
	if assets == nil {
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "web assets not embedded; build with -tags embed", http.StatusNotFound)
		})
	}
	fileServer := http.FileServer(http.FS(assets))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clean := path.Clean(r.URL.Path)
		if isReserved(clean) {
			http.NotFound(w, r)
			return
		}
		p := strings.TrimPrefix(clean, "/")
		if p != "" && p != "index.html" {
			if _, err := fs.Stat(assets, p); err == nil {
				// Vite fingerprints every file under assets/, so a cached copy can never be stale.
				if strings.HasPrefix(p, "assets/") {
					w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
				}
				// A doc's HTML page runs on an opaque origin and loads the app's fonts from here.
				if strings.HasPrefix(p, "assets/") && strings.HasSuffix(p, ".woff2") {
					w.Header().Set("Access-Control-Allow-Origin", "*")
				}
				fileServer.ServeHTTP(w, r)
				return
			}
		}
		w.Header().Set("Cache-Control", "no-cache")
		if len(transforms) > 0 && serveTransformed(r.Context(), w, assets, transforms) {
			return
		}
		index := r.Clone(r.Context())
		index.URL.Path = "/"
		fileServer.ServeHTTP(w, index)
	})
}

func isReserved(p string) bool {
	for _, prefix := range reservedPrefixes {
		if strings.HasPrefix(p, prefix) {
			return true
		}
	}
	return false
}

// serveTransformed writes the index through the transforms; false leaves the plain file to the caller.
func serveTransformed(ctx context.Context, w http.ResponseWriter, assets fs.FS, transforms []IndexTransform) bool {
	raw, err := fs.ReadFile(assets, "index.html")
	if err != nil {
		return false
	}
	out := raw
	for _, t := range transforms {
		out = t(ctx, out)
	}
	if bytes.Equal(out, raw) {
		return false
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(out)
	return true
}
