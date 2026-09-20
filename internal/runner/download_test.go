package runner

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/release"
	"github.com/otal-labs/nexul/internal/platform/version"
)

const testReleaseRepo = "otal-labs/nexul"

// fakeGitHub stands in for the GitHub release API: it serves the "latest release" and "release by tag" lookups
// (one asset, nexul-runner-linux-amd64, plus checksums.txt) and the asset bytes.
func fakeGitHub(t *testing.T, tag, assetBody string) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	respond := func(w http.ResponseWriter, r *http.Request) {
		type ghAsset struct {
			Name string `json:"name"`
			URL  string `json:"url"`
			Size int64  `json:"size"`
		}
		type ghRelease struct {
			TagName string    `json:"tag_name"`
			Assets  []ghAsset `json:"assets"`
		}
		rel := ghRelease{
			TagName: tag,
			Assets: []ghAsset{
				{Name: "nexul-runner-linux-amd64", URL: "http://" + r.Host + "/assets/nexul-runner-linux-amd64", Size: int64(len(assetBody))},
				{Name: "checksums.txt", URL: "http://" + r.Host + "/assets/checksums.txt", Size: 40},
			},
		}
		require.NoError(t, json.NewEncoder(w).Encode(rel))
	}
	mux.HandleFunc("/repos/"+testReleaseRepo+"/releases/latest", respond)
	mux.HandleFunc("/repos/"+testReleaseRepo+"/releases/tags/", respond)
	mux.HandleFunc("/assets/nexul-runner-linux-amd64", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(assetBody))
	})
	mux.HandleFunc("/assets/checksums.txt", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("deadbeef  nexul-runner-linux-amd64\n"))
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func newDownloadService(apiBase string) *Service {
	client := release.New(release.Config{APIBase: apiBase})
	return NewService(newFakeRunnerRepo(), &fakeDispatch{}).WithInstall(InstallConfig{Release: client})
}

// withVersion sets version.Version for the duration of the test and restores it on cleanup.
func withVersion(t *testing.T, v string) {
	t.Helper()
	orig := version.Version
	t.Cleanup(func() { version.Version = orig })
	version.Version = v
}

func TestAssetName(t *testing.T) {
	tests := []struct {
		name   string
		target string
		want   string
		wantOK bool
	}{
		{"linux amd64", "linux-amd64", "nexul-runner-linux-amd64", true},
		{"linux arm64", "linux-arm64", "nexul-runner-linux-arm64", true},
		{"darwin amd64", "darwin-amd64", "nexul-runner-darwin-amd64", true},
		{"darwin arm64", "darwin-arm64", "nexul-runner-darwin-arm64", true},
		{"windows amd64 gets .exe", "windows-amd64", "nexul-runner-windows-amd64.exe", true},
		{"unknown target", "windows-arm64", "", false},
		{"empty target", "", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := AssetName(tt.target)
			assert.Equal(t, tt.wantOK, ok)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestService_Download(t *testing.T) {
	t.Run("dev build resolves the channel's latest and streams the asset with its checksum", func(t *testing.T) {
		withVersion(t, "dev")
		srv := fakeGitHub(t, "v0.2.0-beta-330", "binary-bytes")
		svc := newDownloadService(srv.URL)

		asset, err := svc.Download(context.Background(), "linux-amd64", "")
		require.NoError(t, err)
		defer func() { require.NoError(t, asset.Body.Close()) }()

		assert.Equal(t, "nexul-runner-linux-amd64", asset.Name)
		assert.Equal(t, int64(len("binary-bytes")), asset.Size)
		assert.Equal(t, "deadbeef", asset.Sha256)
		body, err := io.ReadAll(asset.Body)
		require.NoError(t, err)
		assert.Equal(t, "binary-bytes", string(body))
	})

	t.Run("release build resolves its own version", func(t *testing.T) {
		withVersion(t, "v0.2.0")
		srv := fakeGitHub(t, "v0.2.0", "binary-bytes")
		svc := newDownloadService(srv.URL)

		asset, err := svc.Download(context.Background(), "linux-amd64", "")
		require.NoError(t, err)
		require.NoError(t, asset.Body.Close())
		assert.Equal(t, "deadbeef", asset.Sha256)
	})

	t.Run("an explicit tag is used regardless of the server's own version", func(t *testing.T) {
		withVersion(t, "v0.2.0")
		srv := fakeGitHub(t, "v0.1.5", "binary-bytes")
		svc := newDownloadService(srv.URL)

		asset, err := svc.Download(context.Background(), "linux-amd64", "v0.1.5")
		require.NoError(t, err)
		require.NoError(t, asset.Body.Close())
	})

	t.Run("unknown target is rejected before any github call", func(t *testing.T) {
		svc := newDownloadService("http://127.0.0.1:1")
		_, err := svc.Download(context.Background(), "windows-arm64", "")
		require.Error(t, err)
		assert.ErrorIs(t, err, apperrs.ErrInvalid)
	})

	t.Run("asset missing from the release", func(t *testing.T) {
		withVersion(t, "dev")
		srv := fakeGitHub(t, "v0.2.0-beta-330", "binary-bytes")
		svc := newDownloadService(srv.URL)

		_, err := svc.Download(context.Background(), "darwin-arm64", "")
		require.Error(t, err)
		assert.ErrorIs(t, err, apperrs.ErrNotFound)
	})

	t.Run("404 from github hints at the release token", func(t *testing.T) {
		withVersion(t, "dev")
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		t.Cleanup(srv.Close)
		svc := newDownloadService(srv.URL)

		_, err := svc.Download(context.Background(), "linux-amd64", "")
		require.Error(t, err)
		assert.ErrorIs(t, err, apperrs.ErrNotFound)
		assert.Contains(t, err.Error(), "connect GitHub")
	})
}

func TestHTTPHandler_Download(t *testing.T) {
	t.Run("401 on a missing authorization header", func(t *testing.T) {
		withVersion(t, "dev")
		gh := fakeGitHub(t, "v0.2.0-beta-001", "binary-bytes")
		repo := newFakeRunnerRepo()
		client := release.New(release.Config{APIBase: gh.URL})
		svc := NewService(repo, &fakeDispatch{}).WithInstall(InstallConfig{Release: client})
		srv := httptest.NewServer(NewHTTPHandler(svc).PublicRoutes())
		defer srv.Close()

		resp, err := http.Get(srv.URL + "/api/runners/download/linux-amd64")
		require.NoError(t, err)
		defer func() { require.NoError(t, resp.Body.Close()) }()
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("401 on the wrong secret", func(t *testing.T) {
		gh := fakeGitHub(t, "v0.2.0-beta-001", "binary-bytes")
		repo := newFakeRunnerRepo()
		client := release.New(release.Config{APIBase: gh.URL})
		svc := NewService(repo, &fakeDispatch{}).WithInstall(InstallConfig{Release: client})
		srv := httptest.NewServer(NewHTTPHandler(svc).PublicRoutes())
		defer srv.Close()

		req, err := http.NewRequest(http.MethodGet, srv.URL+"/api/runners/download/linux-amd64", nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer nope")
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer func() { require.NoError(t, resp.Body.Close()) }()
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("500 when the secret lookup fails", func(t *testing.T) {
		repo := newFakeRunnerRepo()
		repo.secretErr = assert.AnError
		svc := NewService(repo, &fakeDispatch{})
		srv := httptest.NewServer(NewHTTPHandler(svc).PublicRoutes())
		defer srv.Close()

		req, err := http.NewRequest(http.MethodGet, srv.URL+"/api/runners/download/linux-amd64", nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer whatever")
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer func() { require.NoError(t, resp.Body.Close()) }()
		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	})

	t.Run("400 on an unsupported target", func(t *testing.T) {
		gh := fakeGitHub(t, "v0.2.0-beta-001", "binary-bytes")
		repo := newFakeRunnerRepo()
		client := release.New(release.Config{APIBase: gh.URL})
		svc := NewService(repo, &fakeDispatch{}).WithInstall(InstallConfig{Release: client})
		srv := httptest.NewServer(NewHTTPHandler(svc).PublicRoutes())
		defer srv.Close()

		req, err := http.NewRequest(http.MethodGet, srv.URL+"/api/runners/download/windows-arm64", nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer s3cr3t")
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer func() { require.NoError(t, resp.Body.Close()) }()
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("200 streams the binary with the checksum header", func(t *testing.T) {
		withVersion(t, "dev")
		gh := fakeGitHub(t, "v0.2.0-beta-001", "binary-bytes")
		repo := newFakeRunnerRepo()
		client := release.New(release.Config{APIBase: gh.URL})
		svc := NewService(repo, &fakeDispatch{}).WithInstall(InstallConfig{Release: client})
		srv := httptest.NewServer(NewHTTPHandler(svc).PublicRoutes())
		defer srv.Close()

		req, err := http.NewRequest(http.MethodGet, srv.URL+"/api/runners/download/linux-amd64", nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer s3cr3t")
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer func() { require.NoError(t, resp.Body.Close()) }()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "application/octet-stream", resp.Header.Get("Content-Type"))
		assert.Equal(t, `attachment; filename="nexul-runner-linux-amd64"`, resp.Header.Get("Content-Disposition"))
		assert.Equal(t, strconv.Itoa(len("binary-bytes")), resp.Header.Get("Content-Length"))
		assert.Equal(t, "deadbeef", resp.Header.Get("X-Checksum-Sha256"))
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.Equal(t, "binary-bytes", string(body))
	})

	t.Run("explicit ?version= is forwarded to the release lookup", func(t *testing.T) {
		gh := fakeGitHub(t, "v0.1.5", "binary-bytes")
		repo := newFakeRunnerRepo()
		client := release.New(release.Config{APIBase: gh.URL})
		svc := NewService(repo, &fakeDispatch{}).WithInstall(InstallConfig{Release: client})
		srv := httptest.NewServer(NewHTTPHandler(svc).PublicRoutes())
		defer srv.Close()

		req, err := http.NewRequest(http.MethodGet, srv.URL+"/api/runners/download/linux-amd64?version=v0.1.5", nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer s3cr3t")
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer func() { require.NoError(t, resp.Body.Close()) }()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("404 from github propagates as a not-found status", func(t *testing.T) {
		withVersion(t, "dev")
		gh := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		t.Cleanup(gh.Close)
		repo := newFakeRunnerRepo()
		client := release.New(release.Config{APIBase: gh.URL})
		svc := NewService(repo, &fakeDispatch{}).WithInstall(InstallConfig{Release: client})
		srv := httptest.NewServer(NewHTTPHandler(svc).PublicRoutes())
		defer srv.Close()

		req, err := http.NewRequest(http.MethodGet, srv.URL+"/api/runners/download/linux-amd64", nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer s3cr3t")
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer func() { require.NoError(t, resp.Body.Close()) }()
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})
}

func TestService_LatestVersion(t *testing.T) {
	t.Run("release build returns its own version without calling github", func(t *testing.T) {
		withVersion(t, "v0.2.0")
		svc := newDownloadService("http://127.0.0.1:1")

		got, err := svc.LatestVersion(context.Background())
		require.NoError(t, err)
		assert.Equal(t, "v0.2.0", got)
	})

	t.Run("dev build returns the channel's latest tag", func(t *testing.T) {
		withVersion(t, "dev")
		srv := fakeGitHub(t, "v0.2.0-beta-330", "")
		svc := newDownloadService(srv.URL)

		got, err := svc.LatestVersion(context.Background())
		require.NoError(t, err)
		assert.Equal(t, "v0.2.0-beta-330", got)
	})

	t.Run("upstream error surfaces", func(t *testing.T) {
		withVersion(t, "dev")
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		t.Cleanup(srv.Close)
		svc := newDownloadService(srv.URL)

		_, err := svc.LatestVersion(context.Background())
		require.Error(t, err)
	})
}

func TestHTTPHandler_LatestVersion(t *testing.T) {
	t.Run("200 returns the release tag", func(t *testing.T) {
		withVersion(t, "dev")
		gh := fakeGitHub(t, "v0.2.0-beta-001", "binary-bytes")
		repo := newFakeRunnerRepo()
		client := release.New(release.Config{APIBase: gh.URL})
		svc := NewService(repo, &fakeDispatch{}).WithInstall(InstallConfig{Release: client})
		srv := httptest.NewServer(NewHTTPHandler(svc).Routes())
		defer srv.Close()

		resp, err := http.Get(srv.URL + "/api/runners/latest-version")
		require.NoError(t, err)
		defer func() { require.NoError(t, resp.Body.Close()) }()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var body map[string]string
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
		assert.Equal(t, "v0.2.0-beta-001", body["version"])
	})

	t.Run("upstream error surfaces as an error response", func(t *testing.T) {
		withVersion(t, "dev")
		gh := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		t.Cleanup(gh.Close)
		repo := newFakeRunnerRepo()
		client := release.New(release.Config{APIBase: gh.URL})
		svc := NewService(repo, &fakeDispatch{}).WithInstall(InstallConfig{Release: client})
		srv := httptest.NewServer(NewHTTPHandler(svc).Routes())
		defer srv.Close()

		resp, err := http.Get(srv.URL + "/api/runners/latest-version")
		require.NoError(t, err)
		defer func() { require.NoError(t, resp.Body.Close()) }()
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})
}

func TestService_ReleaseClient(t *testing.T) {
	client := release.New(release.Config{})
	svc := NewService(newFakeRunnerRepo(), &fakeDispatch{}).WithInstall(InstallConfig{Release: client})
	assert.Same(t, client, svc.ReleaseClient())
}
