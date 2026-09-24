package cloudflare

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// stubAPI answers each Cloudflare path with a status; unlisted paths succeed with a realistic body.
func stubAPI(t *testing.T, deny map[string]int, tokenStatus string) *string {
	t.Helper()
	var seenAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenAuth = r.Header.Get("Authorization")
		for prefix, code := range deny {
			if strings.HasPrefix(r.URL.Path, "/"+prefix) {
				w.WriteHeader(code)
				_, _ = w.Write([]byte(`{"success":false,"errors":[{"code":10000,"message":"Authentication error"}]}`))
				return
			}
		}
		switch {
		case r.Method == http.MethodPost:
			// Allowed but malformed: Cloudflare validates only after authorising.
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"success":false,"errors":[{"code":1004,"message":"DNS Validation Error"}]}`))
		case strings.HasSuffix(r.URL.Path, "/user/tokens/verify"):
			_, _ = w.Write([]byte(`{"success":true,"result":{"status":"` + tokenStatus + `"}}`))
		case strings.HasSuffix(r.URL.Path, "/zones"):
			_, _ = w.Write([]byte(`{"success":true,"result":[{"id":"z1","name":"example.com","account":{"id":"acct-1"}}]}`))
		default:
			_, _ = w.Write([]byte(`{"success":true,"result":[]}`))
		}
	}))
	t.Cleanup(srv.Close)
	prev := apiBaseURL
	apiBaseURL = srv.URL
	t.Cleanup(func() { apiBaseURL = prev })
	return &seenAuth
}

func TestTokenVerifier_FullPermissionsPass(t *testing.T) {
	auth := stubAPI(t, nil, "active")
	err := NewTokenVerifier(nil).Verify(context.Background(), map[string]string{"api_token": "tok-1"})
	require.NoError(t, err)
	assert.Equal(t, "Bearer tok-1", *auth)
}

func TestTokenVerifier_RejectedTokenIsInvalid(t *testing.T) {
	stubAPI(t, map[string]int{"user/tokens/verify": http.StatusUnauthorized}, "active")
	err := NewTokenVerifier(nil).Verify(context.Background(), map[string]string{"api_token": "bad"})
	require.Error(t, err)
	assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	assert.Contains(t, err.Error(), "rejected the token")
}

func TestTokenVerifier_InactiveTokenIsInvalid(t *testing.T) {
	stubAPI(t, nil, "disabled")
	err := NewTokenVerifier(nil).Verify(context.Background(), map[string]string{"api_token": "tok-2"})
	require.Error(t, err)
	assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	assert.Contains(t, err.Error(), "disabled")
}

func TestTokenVerifier_NamesTheMissingPermission(t *testing.T) {
	tests := []struct {
		name   string
		deny   string
		expect string
	}{
		{"zone read", "zones", "Zone → Zone: Read"},
		{"dns edit", "zones/z1/dns_records", "Zone → DNS: Edit"},
		{"tunnel edit", "accounts/acct-1/cfd_tunnel", "Account → Cloudflare Tunnel: Edit"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stubAPI(t, map[string]int{tt.deny: http.StatusForbidden}, "active")
			err := NewTokenVerifier(nil).Verify(context.Background(), map[string]string{"api_token": "tok"})
			require.Error(t, err)
			assert.True(t, errors.Is(err, apperrs.ErrInvalid))
			assert.Contains(t, err.Error(), tt.expect)
		})
	}
}

func TestTokenVerifier_VerifyCheck_IsolatesOnePermission(t *testing.T) {
	stubAPI(t, map[string]int{"accounts/acct-1/cfd_tunnel": http.StatusForbidden}, "active")
	v := NewTokenVerifier(nil)
	fields := map[string]string{"api_token": "tok"}
	_, err := v.VerifyCheck(context.Background(), fields, "token")
	require.NoError(t, err)
	_, err = v.VerifyCheck(context.Background(), fields, "zone_read")
	require.NoError(t, err)
	detail, err := v.VerifyCheck(context.Background(), fields, "dns_edit")
	require.NoError(t, err)
	assert.Equal(t, "Can edit DNS on example.com", detail)
	_, err = v.VerifyCheck(context.Background(), fields, "tunnel_edit")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Cloudflare Tunnel: Edit")
}

// A read-only tunnel token passes GETs; only the write probe tells Edit from Read.
func TestTokenVerifier_ProbesWithAWriteNotARead(t *testing.T) {
	var methods []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/access/") {
			methods = append(methods, r.Method+" "+r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"success":false,"errors":[{"code":12135,"message":"access.api.error.not_found"}]}`))
			return
		}
		if strings.Contains(r.URL.Path, "cfd_tunnel") || strings.Contains(r.URL.Path, "dns_records") {
			methods = append(methods, r.Method+" "+r.URL.Path)
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"success":false,"errors":[{"code":1004,"message":"validation"}]}`))
			return
		}
		if strings.HasSuffix(r.URL.Path, "/zones") {
			_, _ = w.Write([]byte(`{"success":true,"result":[{"id":"z1","name":"example.com","account":{"id":"acct-1"}}]}`))
			return
		}
		_, _ = w.Write([]byte(`{"success":true,"result":{"status":"active"}}`))
	}))
	t.Cleanup(srv.Close)
	prev := apiBaseURL
	apiBaseURL = srv.URL
	t.Cleanup(func() { apiBaseURL = prev })

	require.NoError(t, NewTokenVerifier(nil).Verify(context.Background(), map[string]string{"api_token": "tok"}))
	assert.Equal(t, []string{"POST /zones/z1/dns_records", "POST /accounts/acct-1/cfd_tunnel"}, methods,
		"the advisory Access checks never gate saving")

	methods = nil
	fields := map[string]string{"api_token": "tok"}
	_, err := NewTokenVerifier(nil).VerifyCheck(context.Background(), fields, "access_apps_edit")
	require.NoError(t, err)
	_, err = NewTokenVerifier(nil).VerifyCheck(context.Background(), fields, "access_tokens_edit")
	require.NoError(t, err)
	assert.Equal(t, []string{"DELETE /accounts/acct-1/access/apps/" + nilUUID, "DELETE /accounts/acct-1/access/service_tokens/" + nilUUID}, methods,
		"the Access probes delete an object that cannot exist, so verifying never mints a token")
}

func TestTokenVerifier_ProviderOutageIsNotInvalid(t *testing.T) {
	stubAPI(t, map[string]int{"zones/z1/dns_records": http.StatusBadGateway}, "active")
	err := NewTokenVerifier(nil).Verify(context.Background(), map[string]string{"api_token": "tok"})
	require.Error(t, err)
	assert.False(t, errors.Is(err, apperrs.ErrInvalid))
}

func TestTokenVerifier_Verify_IgnoresAccessPermissions(t *testing.T) {
	stubAPI(t, map[string]int{"accounts/acct-1/access": http.StatusForbidden}, "active")
	require.NoError(t, NewTokenVerifier(nil).Verify(context.Background(), map[string]string{"api_token": "tok"}))
}

func TestTokenVerifier_AccessChecks(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   string
		expect string
	}{
		{"zero trust disabled", http.StatusNotFound, `{"success":false,"errors":[{"code":9999,"message":"Unable to find your Access organization"}]}`, "Zero Trust is not enabled"},
		{"denied on 200", http.StatusOK, `{"success":false,"errors":[{"code":10000,"message":"Authentication error"}]}`, "the token is missing Account → Access"},
		{"outage", http.StatusBadGateway, `<html>bad gateway</html>`, "status 502"},
		{"allowed", http.StatusNotFound, `{"success":false,"errors":[{"code":12135,"message":"access.api.error.not_found"}]}`, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if strings.Contains(r.URL.Path, "/access/") {
					w.WriteHeader(tt.status)
					_, _ = w.Write([]byte(tt.body))
					return
				}
				_, _ = w.Write([]byte(`{"success":true,"result":[{"id":"z1","name":"example.com","account":{"id":"acct-1"}}]}`))
			}))
			t.Cleanup(srv.Close)
			prev := apiBaseURL
			apiBaseURL = srv.URL
			t.Cleanup(func() { apiBaseURL = prev })

			fields := map[string]string{"api_token": "tok"}
			for _, key := range []string{"access_apps_edit", "access_tokens_edit"} {
				_, err := NewTokenVerifier(nil).VerifyCheck(t.Context(), fields, key)
				if tt.expect == "" {
					require.NoError(t, err)
					continue
				}
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expect)
			}
		})
	}
}

func TestTokenVerifier_UnreachableCloudflare(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	srv.Close()
	prev := apiBaseURL
	apiBaseURL = srv.URL
	t.Cleanup(func() { apiBaseURL = prev })
	_, err := NewTokenVerifier(nil).VerifyCheck(t.Context(), map[string]string{"api_token": "tok"}, "token")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "reach cloudflare")
}

// stubTwoZones lists a.com and b.com and answers each zone's DNS write probe with its status, 400 meaning allowed.
func stubTwoZones(t *testing.T, dnsStatus map[string]int) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/zones"):
			_, _ = w.Write([]byte(`{"success":true,"result":[{"id":"za","name":"a.com","account":{"id":"acct-1"}},{"id":"zb","name":"b.com","account":{"id":"acct-1"}}]}`))
		case strings.HasSuffix(r.URL.Path, "/dns_records"):
			zone := strings.Split(r.URL.Path, "/")[2]
			w.WriteHeader(dnsStatus[zone])
			_, _ = w.Write([]byte(`{"success":false,"errors":[{"code":10000,"message":"probe"}]}`))
		case r.Method == http.MethodPost:
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"success":false,"errors":[{"code":1004,"message":"validation"}]}`))
		default:
			_, _ = w.Write([]byte(`{"success":true,"result":{"status":"active"}}`))
		}
	}))
	t.Cleanup(srv.Close)
	prev := apiBaseURL
	apiBaseURL = srv.URL
	t.Cleanup(func() { apiBaseURL = prev })
}

func TestTokenVerifier_DNSEdit_NamesEveryZone(t *testing.T) {
	tests := []struct {
		name        string
		dnsStatus   map[string]int
		wantDetail  string
		wantErr     string
		wantInvalid bool
	}{
		{"every zone editable", map[string]int{"za": 400, "zb": 400}, "Can edit DNS on a.com, b.com", "", false},
		{"one zone read-only", map[string]int{"za": 400, "zb": 403}, "Can edit DNS on a.com; read-only on b.com", "", false},
		{"no zone editable", map[string]int{"za": 403, "zb": 401}, "", "Zone → DNS: Edit on a.com, b.com", true},
		{"outage on one zone", map[string]int{"za": 400, "zb": 502}, "", "status 502", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stubTwoZones(t, tt.dnsStatus)
			fields := map[string]string{"api_token": "tok"}
			detail, err := NewTokenVerifier(nil).VerifyCheck(t.Context(), fields, "dns_edit")
			saveErr := NewTokenVerifier(nil).Verify(t.Context(), fields)
			if tt.wantErr == "" {
				require.NoError(t, err)
				require.NoError(t, saveErr, "saving needs one editable zone, not all of them")
				assert.Equal(t, tt.wantDetail, detail)
				return
			}
			require.Error(t, err)
			require.Error(t, saveErr)
			assert.Contains(t, err.Error(), tt.wantErr)
			assert.Equal(t, tt.wantInvalid, errors.Is(err, apperrs.ErrInvalid))
		})
	}
}
