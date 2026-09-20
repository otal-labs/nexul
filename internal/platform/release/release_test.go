package release

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// fakeGitHub serves a "latest release" and a "list releases" endpoint plus an asset body, recording every
// Authorization header it saw.
func fakeGitHub(t *testing.T, latest ghRelease, list []ghRelease, assetBody string) (srv *httptest.Server, authHeaders *[]string) {
	t.Helper()
	authHeaders = &[]string{}
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/"+repo+"/releases/latest", func(w http.ResponseWriter, r *http.Request) {
		*authHeaders = append(*authHeaders, r.Header.Get("Authorization"))
		require.NoError(t, jsonEncode(w, latest))
	})
	mux.HandleFunc("/repos/"+repo+"/releases", func(w http.ResponseWriter, r *http.Request) {
		*authHeaders = append(*authHeaders, r.Header.Get("Authorization"))
		require.NoError(t, jsonEncode(w, list))
	})
	mux.HandleFunc("/repos/"+repo+"/releases/tags/", func(w http.ResponseWriter, r *http.Request) {
		*authHeaders = append(*authHeaders, r.Header.Get("Authorization"))
		require.NoError(t, jsonEncode(w, latest))
	})
	mux.HandleFunc("/assets/binary", func(w http.ResponseWriter, r *http.Request) {
		*authHeaders = append(*authHeaders, r.Header.Get("Authorization"))
		_, _ = w.Write([]byte(assetBody))
	})
	srv = httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv, authHeaders
}

func jsonEncode(w http.ResponseWriter, v any) error {
	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(v)
}

func TestNew_Defaults(t *testing.T) {
	c := New(Config{})
	assert.Equal(t, "https://api.github.com", c.cfg.APIBase)
	require.NotNil(t, c.cfg.HTTP)
	assert.Equal(t, 2*time.Minute, c.cfg.HTTP.Timeout)
}

func TestClient_Latest_Stable(t *testing.T) {
	srv, headers := fakeGitHub(t, ghRelease{TagName: "v0.2.0", HTMLURL: "https://github.com/x/releases/tag/v0.2.0"}, nil, "")
	c := New(Config{APIBase: srv.URL, TokenSource: func(context.Context) (string, error) { return "tok", nil }})

	rel, err := c.Latest(context.Background(), "stable")
	require.NoError(t, err)
	assert.Equal(t, "v0.2.0", rel.Tag)
	assert.Equal(t, "https://github.com/x/releases/tag/v0.2.0", rel.URL)
	for _, h := range *headers {
		assert.Equal(t, "Bearer tok", h)
	}
}

func TestClient_Latest_Dev_BehavesAsStable(t *testing.T) {
	srv, _ := fakeGitHub(t, ghRelease{TagName: "v0.2.0"}, nil, "")
	c := New(Config{APIBase: srv.URL})

	rel, err := c.Latest(context.Background(), "dev")
	require.NoError(t, err)
	assert.Equal(t, "v0.2.0", rel.Tag)
}

func TestClient_Latest_Beta_PicksNewestNonDraftBetaTag(t *testing.T) {
	list := []ghRelease{
		{TagName: "v0.2.0", Draft: false},
		{TagName: "v0.2.0-beta-330", Draft: true},
		{TagName: "v0.2.0-beta-320", Draft: false},
		{TagName: "v0.2.0-beta-300", Draft: false},
	}
	srv, _ := fakeGitHub(t, ghRelease{}, list, "")
	c := New(Config{APIBase: srv.URL})

	rel, err := c.Latest(context.Background(), "beta")
	require.NoError(t, err)
	assert.Equal(t, "v0.2.0-beta-320", rel.Tag)
}

func TestClient_Latest_Beta_NoneFound(t *testing.T) {
	srv, _ := fakeGitHub(t, ghRelease{}, []ghRelease{{TagName: "v0.2.0"}}, "")
	c := New(Config{APIBase: srv.URL})

	_, err := c.Latest(context.Background(), "beta")
	require.Error(t, err)
	assert.ErrorIs(t, err, apperrs.ErrNotFound)
}

func TestClient_Latest_CachedPerChannel(t *testing.T) {
	var requests int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		require.NoError(t, jsonEncode(w, ghRelease{TagName: "v0.2.0"}))
	}))
	t.Cleanup(srv.Close)
	c := New(Config{APIBase: srv.URL})

	_, err := c.Latest(context.Background(), "stable")
	require.NoError(t, err)
	_, err = c.Latest(context.Background(), "stable")
	require.NoError(t, err)
	assert.Equal(t, 1, requests, "second call within 5 minutes must be served from cache")
}

func TestClient_Latest_CacheExpiresAfterFiveMinutes(t *testing.T) {
	var requests int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		require.NoError(t, jsonEncode(w, ghRelease{TagName: "v0.2.0"}))
	}))
	t.Cleanup(srv.Close)
	c := New(Config{APIBase: srv.URL})
	now := time.Now()
	c.now = func() time.Time { return now }

	_, err := c.Latest(context.Background(), "stable")
	require.NoError(t, err)

	now = now.Add(6 * time.Minute)
	_, err = c.Latest(context.Background(), "stable")
	require.NoError(t, err)
	assert.Equal(t, 2, requests, "an expired cache entry must re-fetch")
}

func TestClient_Latest_ErrorsAreNeverCached(t *testing.T) {
	var requests int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if requests == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		require.NoError(t, jsonEncode(w, ghRelease{TagName: "v0.2.0"}))
	}))
	t.Cleanup(srv.Close)
	c := New(Config{APIBase: srv.URL})

	_, err := c.Latest(context.Background(), "stable")
	require.Error(t, err)

	rel, err := c.Latest(context.Background(), "stable")
	require.NoError(t, err)
	assert.Equal(t, "v0.2.0", rel.Tag)
	assert.Equal(t, 2, requests)
}

func TestClient_Latest_DifferentChannelsCacheIndependently(t *testing.T) {
	var latestReqs, listReqs int
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/"+repo+"/releases/latest", func(w http.ResponseWriter, r *http.Request) {
		latestReqs++
		require.NoError(t, jsonEncode(w, ghRelease{TagName: "v0.2.0"}))
	})
	mux.HandleFunc("/repos/"+repo+"/releases", func(w http.ResponseWriter, r *http.Request) {
		listReqs++
		require.NoError(t, jsonEncode(w, []ghRelease{{TagName: "v0.2.0-beta-320"}}))
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	c := New(Config{APIBase: srv.URL})

	_, err := c.Latest(context.Background(), "stable")
	require.NoError(t, err)
	_, err = c.Latest(context.Background(), "beta")
	require.NoError(t, err)
	assert.Equal(t, 1, latestReqs)
	assert.Equal(t, 1, listReqs)
}

func TestClient_Latest_404ReportsMissingRelease(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(srv.Close)
	c := New(Config{APIBase: srv.URL})

	_, err := c.Latest(context.Background(), "stable")
	require.Error(t, err)
	assert.ErrorIs(t, err, apperrs.ErrNotFound)
	assert.Contains(t, err.Error(), "release or asset not found")
	assert.NotContains(t, err.Error(), "GitHub App")
}

func TestClient_Latest_NonNotFoundStatusIsRetryable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	t.Cleanup(srv.Close)
	c := New(Config{APIBase: srv.URL})

	_, err := c.Latest(context.Background(), "stable")
	require.Error(t, err)
	assert.ErrorIs(t, err, apperrs.ErrRetryable)
}

func TestClient_Latest_NetworkError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	addr := srv.URL
	srv.Close()
	c := New(Config{APIBase: addr})

	_, err := c.Latest(context.Background(), "stable")
	require.Error(t, err)
}

func TestClient_Latest_MalformedJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("not json"))
	}))
	t.Cleanup(srv.Close)
	c := New(Config{APIBase: srv.URL})

	_, err := c.Latest(context.Background(), "stable")
	require.Error(t, err)
}

func TestClient_ByTag(t *testing.T) {
	srv, _ := fakeGitHub(t, ghRelease{TagName: "v0.1.9"}, nil, "")
	c := New(Config{APIBase: srv.URL})

	rel, err := c.ByTag(context.Background(), "v0.1.9")
	require.NoError(t, err)
	assert.Equal(t, "v0.1.9", rel.Tag)
}

func TestClient_Download(t *testing.T) {
	t.Run("streams the matching asset and sends the token", func(t *testing.T) {
		srv, headers := fakeGitHub(t, ghRelease{}, nil, "binary-bytes")
		c := New(Config{APIBase: srv.URL, TokenSource: func(context.Context) (string, error) { return "tok", nil }})
		rel := &Release{Assets: []Asset{{Name: "bin", URL: srv.URL + "/assets/binary", Size: int64(len("binary-bytes"))}}}

		body, size, err := c.Download(context.Background(), rel, "bin")
		require.NoError(t, err)
		defer func() { require.NoError(t, body.Close()) }()
		assert.Equal(t, int64(len("binary-bytes")), size)
		data, err := io.ReadAll(body)
		require.NoError(t, err)
		assert.Equal(t, "binary-bytes", string(data))
		for _, h := range *headers {
			assert.Equal(t, "Bearer tok", h)
		}
	})

	t.Run("asset missing from the release", func(t *testing.T) {
		c := New(Config{})
		rel := &Release{Assets: []Asset{{Name: "other"}}}

		_, _, err := c.Download(context.Background(), rel, "bin")
		require.Error(t, err)
		assert.ErrorIs(t, err, apperrs.ErrNotFound)
	})

	t.Run("error status on the asset fetch", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusForbidden)
		}))
		t.Cleanup(srv.Close)
		c := New(Config{})
		rel := &Release{Assets: []Asset{{Name: "bin", URL: srv.URL}}}

		_, _, err := c.Download(context.Background(), rel, "bin")
		require.Error(t, err)
		assert.ErrorIs(t, err, apperrs.ErrRetryable)
	})
}

func TestClient_Checksums(t *testing.T) {
	t.Run("parses sha256 lines keyed by name", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = fmt.Fprint(w, "aaa111  nexul-runner-linux-amd64\nbbb222  nexul-server-linux-amd64.tar.gz\n")
		}))
		t.Cleanup(srv.Close)
		c := New(Config{})
		rel := &Release{Assets: []Asset{{Name: "checksums.txt", URL: srv.URL}}}

		sums, err := c.Checksums(context.Background(), rel)
		require.NoError(t, err)
		assert.Equal(t, "aaa111", sums["nexul-runner-linux-amd64"])
		assert.Equal(t, "bbb222", sums["nexul-server-linux-amd64.tar.gz"])
	})

	t.Run("no checksums.txt asset yields an empty map, no error", func(t *testing.T) {
		c := New(Config{})
		rel := &Release{Assets: []Asset{{Name: "other"}}}

		sums, err := c.Checksums(context.Background(), rel)
		require.NoError(t, err)
		assert.Empty(t, sums)
	})

	t.Run("fetch error propagates", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		t.Cleanup(srv.Close)
		c := New(Config{})
		rel := &Release{Assets: []Asset{{Name: "checksums.txt", URL: srv.URL}}}

		_, err := c.Checksums(context.Background(), rel)
		require.Error(t, err)
	})
}

func TestClient_Token(t *testing.T) {
	tests := []struct {
		name string
		cfg  Config
		want string
	}{
		{"connector token", Config{TokenSource: func(context.Context) (string, error) { return "conn", nil }}, "conn"},
		{"source error falls back to unauthenticated", Config{TokenSource: func(context.Context) (string, error) { return "", errors.New("not connected") }}, ""},
		{"nothing configured", Config{}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := New(tt.cfg)
			assert.Equal(t, tt.want, c.token(context.Background()))
		})
	}
}
