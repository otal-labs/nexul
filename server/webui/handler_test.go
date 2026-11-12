package webui

import (
	"bytes"
	"context"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newAssets() fs.FS {
	return fstest.MapFS{
		"index.html":         {Data: []byte("<html>root</html>")},
		"assets/app.js":      {Data: []byte("console.log(1)")},
		"assets/inter.woff2": {Data: []byte("font")},
	}
}

func TestHandler_NoEmbed_Returns404(t *testing.T) {
	h := Handler(nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("code = %d, want 404", rec.Code)
	}
}

func TestHandler_ReservedPrefix_Returns404(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{"api", "/api"},
		{"ws", "/ws"},
		{"mcp", "/mcp"},
		{"auth", "/auth"},
		{"hooks", "/hooks"},
		{"openobserve", "/openobserve"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := Handler(newAssets())
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tt.path, nil))
			if rec.Code != http.StatusNotFound {
				t.Fatalf("path %q: code = %d, want 404", tt.path, rec.Code)
			}
		})
	}
}

func TestHandler_ExistingFile_ServesContent(t *testing.T) {
	h := Handler(newAssets())
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/assets/app.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("code = %d, want 200", rec.Code)
	}
	if rec.Body.String() != "console.log(1)" {
		t.Fatalf("body = %q, want asset content", rec.Body.String())
	}
}

func TestHandler_UnknownRoute_FallsBackToIndex(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{"root", "/"},
		{"client route", "/docs/guide"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := Handler(newAssets())
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tt.path, nil))
			if rec.Code != http.StatusOK {
				t.Fatalf("path %q: code = %d, want 200", tt.path, rec.Code)
			}
			if rec.Body.String() != "<html>root</html>" {
				t.Fatalf("path %q: body = %q, want index.html", tt.path, rec.Body.String())
			}
		})
	}
}

func TestHandler_CacheHeaders(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{"fingerprinted asset is cached for a year", "/assets/app.js", "public, max-age=31536000, immutable"},
		{"index is revalidated on every load", "/", "no-cache"},
		{"client route serves the revalidated index", "/docs/guide", "no-cache"},
		{"index by name serves the revalidated index", "/index.html", "no-cache"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			Handler(newAssets()).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tt.path, nil))
			require.Equal(t, http.StatusOK, rec.Code)
			assert.Equal(t, tt.want, rec.Header().Get("Cache-Control"))
		})
	}
}

func TestHandler_Font_IsReadableFromAnOpaqueOrigin(t *testing.T) {
	h := Handler(newAssets())
	tests := []struct {
		path string
		want string
	}{
		{"/assets/inter.woff2", "*"},
		{"/assets/app.js", ""},
	}
	for _, tt := range tests {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tt.path, nil))
		assert.Equal(t, tt.want, rec.Header().Get("Access-Control-Allow-Origin"), tt.path)
	}
}

func TestHandler_Transform_RewritesTheIndexTheSPAFallsBackTo(t *testing.T) {
	upper := func(_ context.Context, index []byte) []byte { return bytes.ToUpper(index) }
	h := Handler(newAssets(), upper)

	for _, path := range []string{"/", "/board/abc"} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		require.Equal(t, http.StatusOK, rec.Code, path)
		assert.Equal(t, "<HTML>ROOT</HTML>", rec.Body.String(), path)
		assert.Equal(t, "no-cache", rec.Header().Get("Cache-Control"))
		assert.Contains(t, rec.Header().Get("Content-Type"), "text/html")
	}

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/assets/app.js", nil))
	assert.Equal(t, "console.log(1)", rec.Body.String(), "a real file is served as it is")

	rec = httptest.NewRecorder()
	Handler(newAssets(), func(_ context.Context, index []byte) []byte { return index }).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	assert.Equal(t, "<html>root</html>", rec.Body.String(), "a transform that changes nothing leaves the file server's answer")
}
