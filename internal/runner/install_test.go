package runner

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeSettingsReader is a test double for SettingsReader.
type fakeSettingsReader struct {
	url string
	err error
}

func (f *fakeSettingsReader) GetInstanceURL(context.Context) (string, error) {
	return f.url, f.err
}

func TestInstallWSURL(t *testing.T) {
	tests := []struct {
		name        string
		instanceURL string
		requestHost string
		requestTLS  bool
		want        string
	}{
		{
			name:        "https instance url gives wss on the same origin",
			instanceURL: "https://app.example.com",
			requestHost: "server:8080",
			want:        "wss://app.example.com/ws/runner",
		},
		{
			name:        "http instance url gives ws",
			instanceURL: "http://app.example.com",
			want:        "ws://app.example.com/ws/runner",
		},
		{
			name:        "instance url's own port is kept",
			instanceURL: "http://localhost:5173",
			want:        "ws://localhost:5173/ws/runner",
		},
		{
			name:        "no instance url falls back to request host over TLS",
			requestHost: "app.local:8443",
			requestTLS:  true,
			want:        "wss://app.local:8443/ws/runner",
		},
		{
			name:        "no instance url falls back to request host without TLS",
			requestHost: "app.local",
			want:        "ws://app.local/ws/runner",
		},
		{
			name:        "unparseable instance url falls back to request host",
			instanceURL: "://not-a-url",
			requestHost: "app.local",
			want:        "ws://app.local/ws/runner",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := InstallWSURL(tt.instanceURL, tt.requestHost, tt.requestTLS)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestInstallDownloadURL(t *testing.T) {
	tests := []struct {
		name        string
		instanceURL string
		requestHost string
		requestTLS  bool
		want        string
	}{
		{
			name:        "instance url's scheme and host are used as-is, including its port",
			instanceURL: "https://app.example.com:9443",
			requestHost: "ignored.example",
			want:        "https://app.example.com:9443/api/runners/download",
		},
		{
			name:        "instance url with no explicit port",
			instanceURL: "https://app.example.com",
			want:        "https://app.example.com/api/runners/download",
		},
		{
			name:        "no instance url falls back to the request host over TLS",
			requestHost: "app.local:8443",
			requestTLS:  true,
			want:        "https://app.local:8443/api/runners/download",
		},
		{
			name:        "no instance url falls back to the request host without TLS",
			requestHost: "app.local",
			want:        "http://app.local/api/runners/download",
		},
		{
			name:        "unparseable instance url falls back to the request host",
			instanceURL: "://not-a-url",
			requestHost: "app.local",
			want:        "http://app.local/api/runners/download",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := InstallDownloadURL(tt.instanceURL, tt.requestHost, tt.requestTLS)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestService_Install(t *testing.T) {
	t.Run("derives the ws url and download url from the instance url and returns the repo secret", func(t *testing.T) {
		repo := newFakeRunnerRepo()
		svc := NewService(repo, &fakeDispatch{}).WithInstall(InstallConfig{
			Settings: &fakeSettingsReader{url: "https://app.example.com"},
		})
		info, err := svc.Install(context.Background(), "ignored.example", false)
		require.NoError(t, err)
		assert.Equal(t, "wss://app.example.com/ws/runner", info.WSURL)
		assert.Equal(t, "s3cr3t", info.Secret)
		assert.Equal(t, "https://app.example.com/api/runners/download", info.DownloadURL)
	})

	t.Run("no Settings wired falls back to the request host", func(t *testing.T) {
		repo := newFakeRunnerRepo()
		svc := NewService(repo, &fakeDispatch{}).WithInstall(InstallConfig{})
		info, err := svc.Install(context.Background(), "app.local", false)
		require.NoError(t, err)
		assert.Equal(t, "ws://app.local/ws/runner", info.WSURL)
		assert.Equal(t, "http://app.local/api/runners/download", info.DownloadURL)
	})

	t.Run("settings error is wrapped", func(t *testing.T) {
		repo := newFakeRunnerRepo()
		svc := NewService(repo, &fakeDispatch{}).WithInstall(InstallConfig{
			Settings: &fakeSettingsReader{err: assert.AnError},
		})
		_, err := svc.Install(context.Background(), "app.local", false)
		require.Error(t, err)
	})

	t.Run("repo secret error is wrapped", func(t *testing.T) {
		repo := newFakeRunnerRepo()
		repo.secretErr = assert.AnError
		svc := NewService(repo, &fakeDispatch{})
		_, err := svc.Install(context.Background(), "app.local", false)
		require.Error(t, err)
	})
}

func TestHTTPHandler_Install(t *testing.T) {
	t.Run("200 with the ws url, secret, and download url", func(t *testing.T) {
		repo := newFakeRunnerRepo()
		svc := NewService(repo, &fakeDispatch{}).WithInstall(InstallConfig{})
		srv := httptest.NewServer(NewHTTPHandler(svc).Routes())
		defer srv.Close()

		resp, err := http.Get(srv.URL + "/api/runners/install")
		require.NoError(t, err)
		defer func() { require.NoError(t, resp.Body.Close()) }()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var info InstallInfo
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&info))
		assert.NotEmpty(t, info.WSURL)
		assert.Equal(t, "s3cr3t", info.Secret)
		assert.Contains(t, info.DownloadURL, "/api/runners/download")
	})

	t.Run("service error propagates as an error status", func(t *testing.T) {
		repo := newFakeRunnerRepo()
		repo.secretErr = assert.AnError
		svc := NewService(repo, &fakeDispatch{})
		srv := httptest.NewServer(NewHTTPHandler(svc).Routes())
		defer srv.Close()

		resp, err := http.Get(srv.URL + "/api/runners/install")
		require.NoError(t, err)
		defer func() { require.NoError(t, resp.Body.Close()) }()
		assert.NotEqual(t, http.StatusOK, resp.StatusCode)
	})
}
