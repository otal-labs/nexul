// Package webui serves the built web frontend from the server binary in single-binary mode.
package webui

import (
	"io/fs"
	"net/http"
	"path"
	"strings"
)

// reservedPrefixes are owned by other handlers and must never fall back to the SPA's index.html.
var reservedPrefixes = []string{"/api", "/ws", "/mcp", "/auth", "/hooks"}

// Handler serves the embedded SPA with client-side routing fallback, or a 404 build hint when assets is nil.
func Handler(assets fs.FS) http.Handler {
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
		if p != "" {
			if _, err := fs.Stat(assets, p); err == nil {
				fileServer.ServeHTTP(w, r)
				return
			}
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
