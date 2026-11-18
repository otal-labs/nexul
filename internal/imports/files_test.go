package imports

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// tlsFetcher reads from srv; srv's own host is the only one it is allowed unless a test says otherwise.
func tlsFetcher(t *testing.T, h http.Handler) (*Fetcher, *httptest.Server, string) {
	t.Helper()
	srv := httptest.NewTLSServer(h)
	t.Cleanup(srv.Close)
	u, err := url.Parse(srv.URL)
	require.NoError(t, err)
	return &Fetcher{client: srv.Client()}, srv, u.Hostname()
}

func TestFetch_ReadsAnAllowedFileWithoutTheCredential(t *testing.T) {
	var auth string
	f, srv, host := tlsFetcher(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth = r.Header.Get("Authorization") + r.Header.Get("Cookie")
		_, _ = w.Write([]byte("png bytes")) // test server: write errors are irrelevant
	}))
	name, data, err := f.Fetch(t.Context(), srv.URL+"/assets/shot.png", []string{host}, nil)
	require.NoError(t, err)
	assert.Equal(t, "shot.png", name)
	assert.Equal(t, []byte("png bytes"), data)
	assert.Empty(t, auth)
}

func TestFetch_SendsTheHeaderTheSourceSignedWith(t *testing.T) {
	var auth string
	f, srv, host := tlsFetcher(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth = r.Header.Get("Authorization")
		_, _ = w.Write([]byte("bytes")) // test server: write errors are irrelevant
	}))
	_, _, err := f.Fetch(t.Context(), srv.URL+"/download/a.pdf", []string{host}, http.Header{"Authorization": {"OAuth k"}})
	require.NoError(t, err)
	assert.Equal(t, "OAuth k", auth)
}

func TestFetch_RefusesWhatIsNotTheSourcesToGive(t *testing.T) {
	f, srv, host := tlsFetcher(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/hop" {
			http.Redirect(w, r, "https://169.254.169.254/latest/meta-data", http.StatusFound)
			return
		}
		if r.URL.Path == "/signin" {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write([]byte("<!doctype html><title>Sign in</title>")) // test server: write errors are irrelevant
			return
		}
		if r.URL.Path == "/unlabelled-page" {
			w.Header().Set("Content-Type", "application/octet-stream")
			_, _ = w.Write([]byte("<!DOCTYPE html><html><body>Sign in to GitHub</body></html>")) // test server: write errors are irrelevant
			return
		}
		if r.URL.Path == "/big" {
			_, _ = w.Write(bytes.Repeat([]byte("x"), MaxFileSize+1)) // test server: write errors are irrelevant
			return
		}
		http.NotFound(w, r)
	}))
	tests := []struct {
		name  string
		url   string
		hosts []string
		want  error
	}{
		{"a host outside the list", srv.URL + "/a.png", []string{"github.com"}, errHost},
		{"plain http", "http://" + host + "/a.png", []string{host}, errHost},
		{"credentials in the URL", "https://user:pass@" + host + "/a.png", []string{host}, errHost},
		{"a redirect off the list", srv.URL + "/hop", []string{host}, errHost},
		{"a file over the cap", srv.URL + "/big", []string{host}, ErrTooLarge},
		{"a sign-in page where the file was", srv.URL + "/signin", []string{host}, errPage},
		{"a page served as bytes", srv.URL + "/unlabelled-page", []string{host}, errPage},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := f.Fetch(t.Context(), tt.url, tt.hosts, nil)
			assert.ErrorIs(t, err, tt.want)
		})
	}
	_, _, err := f.Fetch(t.Context(), srv.URL+"/missing", []string{host}, nil)
	assert.ErrorContains(t, err, "404")
}

func TestRewriteLinks_PointsEachCopiedFileAtItsAttachment(t *testing.T) {
	text := "![a](https://x/a.png) and ![b](https://x/a.png.big) and https://x/kept.zip"
	got := rewriteLinks(text, map[string]string{"https://x/a.png": "att-1", "https://x/a.png.big": "att-2"})
	assert.Equal(t, "![a](/api/attachments/att-1) and ![b](/api/attachments/att-2) and https://x/kept.zip", got)
}
