package runner

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

// fakeSettingsReader is a test double for SettingsReader.
type fakeSettingsReader struct {
	url string
	err error
}

func (f *fakeSettingsReader) GetInstanceURL(context.Context) (string, error) {
	return f.url, f.err
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
