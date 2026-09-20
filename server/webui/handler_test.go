package webui

import (
	"io/fs"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
)

func newAssets() fs.FS {
	return fstest.MapFS{
		"index.html":    {Data: []byte("<html>root</html>")},
		"assets/app.js": {Data: []byte("console.log(1)")},
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
		{"api nested", "/api/tickets/1"},
		{"ws", "/ws"},
		{"ws nested", "/ws/runner"},
		{"mcp", "/mcp"},
		{"mcp nested", "/mcp/some/tool"},
		{"auth", "/auth"},
		{"auth nested", "/auth/github"},
		{"hooks", "/hooks"},
		{"hooks nested", "/hooks/github"},
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
		{"deep route", "/a/b/c"},
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
