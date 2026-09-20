package dns

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

func TestRecordInput_Validate(t *testing.T) {
	valid := RecordInput{Type: RecordA, Name: "api", Content: "1.2.3.4", TTL: 300}
	tests := []struct {
		name    string
		mutate  func(*RecordInput)
		wantErr bool
	}{
		{"valid A record", func(r *RecordInput) {}, false},
		{"valid TXT", func(r *RecordInput) { r.Type = RecordTXT }, false},
		{"unsupported type", func(r *RecordInput) { r.Type = "SRV" }, true},
		{"missing name", func(r *RecordInput) { r.Name = " " }, true},
		{"missing content", func(r *RecordInput) { r.Content = "" }, true},
		{"negative ttl", func(r *RecordInput) { r.TTL = -1 }, true},
		{"zero ttl ok (provider default)", func(r *RecordInput) { r.TTL = 0 }, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := valid
			tt.mutate(&in)
			err := in.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.True(t, errors.Is(err, apperrs.ErrInvalid))
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestServiceHostnameInput_Validate(t *testing.T) {
	valid := ServiceHostnameInput{
		Service: "api", Hostname: "api.example.com", ZoneID: "z1", Zone: "example.com",
		Type: RecordA, Target: "1.2.3.4",
	}
	tests := []struct {
		name    string
		mutate  func(*ServiceHostnameInput)
		wantErr bool
	}{
		{"valid", func(in *ServiceHostnameInput) {}, false},
		{"missing service", func(in *ServiceHostnameInput) { in.Service = "" }, true},
		{"missing hostname", func(in *ServiceHostnameInput) { in.Hostname = " " }, true},
		{"missing zone id", func(in *ServiceHostnameInput) { in.ZoneID = "" }, true},
		{"missing zone name", func(in *ServiceHostnameInput) { in.Zone = "" }, true},
		{"unsupported type", func(in *ServiceHostnameInput) { in.Type = "MX" }, true},
		{"missing target", func(in *ServiceHostnameInput) { in.Target = "" }, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := valid
			tt.mutate(&in)
			err := in.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.True(t, errors.Is(err, apperrs.ErrInvalid))
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestRecordNameFor(t *testing.T) {
	tests := []struct {
		name     string
		hostname string
		zone     string
		want     string
		wantErr  bool
	}{
		{"subdomain under zone", "api.example.com", "example.com", "api", false},
		{"apex hostname", "example.com", "example.com", "@", false},
		{"deep subdomain", "a.b.example.com", "example.com", "a.b", false},
		{"zone not a suffix is a mismatch", "deploy.other.com", "example.com", "", true},
		{"case-insensitive", "API.EXAMPLE.COM", "Example.com", "api", false},
		{"trailing dots trimmed", "api.example.com.", "example.com", "api", false},
		{"empty zone", "api.example.com", "", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := recordNameFor(tt.hostname, tt.zone)
			if tt.wantErr {
				require.Error(t, err)
				assert.True(t, errors.Is(err, apperrs.ErrInvalid))
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
