package auth

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/hostcred"
)

func mustSetupPass(t *testing.T, s *Service) string {
	t.Helper()
	pass, err := s.signSetupPass(s.cfg.Now().Add(setupPassTTL))
	require.NoError(t, err)
	return pass.Token
}

// withSetupPass builds a request carrying a fresh setup pass as its bearer.
func withSetupPass(t *testing.T, s *Service, method, path string, body io.Reader) *http.Request {
	t.Helper()
	req := httptest.NewRequest(method, path, body)
	req.Header.Set("Authorization", "Bearer "+mustSetupPass(t, s))
	return req
}

// seedSetupCode stores code as the live setup code.
func seedSetupCode(t *testing.T, s *Service, code string) {
	t.Helper()
	now := s.cfg.Now()
	require.NoError(t, s.cfg.SetupCodes.ReplaceSetupCode(context.Background(), hostcred.Hash(code), now, now.Add(setupCodeTTL)))
}

func unlock(h http.Handler, addr, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/api/setup/unlock", strings.NewReader(body))
	req.RemoteAddr = addr
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func errorCode(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var env struct {
		Code string `json:"code"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
	return env.Code
}

func TestHandler_UnlockSetup(t *testing.T) {
	tests := []struct {
		name     string
		prepare  func(t *testing.T, s *Service, users *fakeUserStore)
		body     string
		want     int
		wantCode string
	}{
		{"malformed body", func(*testing.T, *Service, *fakeUserStore) {}, `{`, http.StatusBadRequest, "INVALID"},
		{"wrong code", func(*testing.T, *Service, *fakeUserStore) {}, `{"code":"nxs_wrong"}`, http.StatusBadRequest, "invalid_code"},
		{"empty code", func(*testing.T, *Service, *fakeUserStore) {}, `{"code":""}`, http.StatusBadRequest, "invalid_code"},
		{"expired code", func(t *testing.T, s *Service, _ *fakeUserStore) {
			s.cfg.Now = func() time.Time { return time.Unix(1_700_000_000, 0).Add(setupCodeTTL) }
		}, `{"code":"nxs_right"}`, http.StatusBadRequest, "invalid_code"},
		{"setup done once a user exists", func(t *testing.T, _ *Service, users *fakeUserStore) {
			seedOwner(t, users, "u1", "1", "owner")
		}, `{"code":"nxs_right"}`, http.StatusConflict, "setup_done"},
		{"store failure is a 500", func(_ *testing.T, s *Service, _ *fakeUserStore) {
			s.cfg.SetupCodes.(*fakeSetupCodes).err = errors.New("disk gone")
		}, `{"code":"nxs_right"}`, http.StatusInternalServerError, "INTERNAL"},
		{"right code, surrounding space ignored", func(*testing.T, *Service, *fakeUserStore) {}, `{"code":" nxs_right\n"}`, http.StatusOK, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, users, _, _ := newTestHarness(&fakeGitHub{})
			seedSetupCode(t, s, "nxs_right")
			tt.prepare(t, s, users)
			rec := unlock(NewHandler(s).Routes(), "198.51.100.1:4000", tt.body)
			require.Equal(t, tt.want, rec.Code, rec.Body.String())
			if tt.want != http.StatusOK {
				assert.Equal(t, tt.wantCode, errorCode(t, rec))
				return
			}
			var pass SetupPass
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &pass))
			assert.True(t, strings.HasPrefix(pass.Token, setupPassPrefix))
			assert.Equal(t, s.cfg.Now().Add(time.Hour).UTC(), pass.ExpiresAt)
			require.NoError(t, s.verifySetupPass(pass.Token))
		})
	}
}

func TestHandler_UnlockSetup_ThrottlesAfterTenFailures(t *testing.T) {
	s, _, _, _ := newTestHarness(&fakeGitHub{})
	seedSetupCode(t, s, "nxs_right")
	h := NewHandler(s).Routes()
	for range unlockMaxFailures {
		require.Equal(t, http.StatusBadRequest, unlock(h, "198.51.100.1:4000", `{"code":"nxs_wrong"}`).Code)
	}

	assert.Equal(t, http.StatusTooManyRequests, unlock(h, "198.51.100.1:5000", `{"code":"nxs_right"}`).Code, "the right code is refused too while throttled")
	assert.Equal(t, http.StatusOK, unlock(h, "198.51.100.2:4000", `{"code":"nxs_right"}`).Code, "another address is not throttled")

	start := s.cfg.Now()
	s.cfg.Now = func() time.Time { return start.Add(unlockWindow) }
	assert.Equal(t, http.StatusOK, unlock(h, "198.51.100.1:4000", `{"code":"nxs_right"}`).Code, "failures age out of the window")
	assert.Len(t, s.unlocks.failures, 1, "stale addresses are pruned")
}

func TestHandler_UnlockSetup_CodeIsNotConsumed(t *testing.T) {
	s, _, _, _ := newTestHarness(&fakeGitHub{})
	seedSetupCode(t, s, "nxs_right")
	h := NewHandler(s).Routes()
	assert.Equal(t, http.StatusOK, unlock(h, "198.51.100.1:4000", `{"code":"nxs_right"}`).Code)
	assert.Equal(t, http.StatusOK, unlock(h, "198.51.100.1:4000", `{"code":"nxs_right"}`).Code, "the domain handoff reuses the code")
}

func TestClientAddr(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "[2001:db8::1]:443"
	assert.Equal(t, "2001:db8::1", clientAddr(req))
	req.RemoteAddr = "pipe"
	assert.Equal(t, "pipe", clientAddr(req))
}

func TestRequireAuth_SetupPass(t *testing.T) {
	tests := []struct {
		name    string
		method  string
		path    string
		token   func(t *testing.T, s *Service) string
		prepare func(t *testing.T, users *fakeUserStore)
		want    int
	}{
		{"expired pass", http.MethodGet, "/api/dns/zones", func(t *testing.T, s *Service) string {
			pass, err := s.signSetupPass(s.cfg.Now())
			require.NoError(t, err)
			return pass.Token
		}, nil, http.StatusUnauthorized},
		{"tampered pass", http.MethodGet, "/api/dns/zones", func(t *testing.T, s *Service) string {
			return mustSetupPass(t, s) + "x"
		}, nil, http.StatusUnauthorized},
		{"pass without a signature", http.MethodGet, "/api/dns/zones", func(*testing.T, *Service) string {
			return setupPassPrefix + "e30"
		}, nil, http.StatusUnauthorized},
		{"session token does not make a pass", http.MethodGet, "/api/dns/zones", func(t *testing.T, s *Service) string {
			session, err := sign(s, SetupUserID)
			require.NoError(t, err)
			return setupPassPrefix + session
		}, nil, http.StatusUnauthorized},
		{"pass after the first user", http.MethodGet, "/api/dns/zones", mustSetupPass, func(t *testing.T, users *fakeUserStore) {
			seedOwner(t, users, "u1", "1", "owner")
		}, http.StatusUnauthorized},
		{"route off the allowlist", http.MethodGet, "/api/auth/me", mustSetupPass, nil, http.StatusUnauthorized},
		{"mcp is off the allowlist", http.MethodPost, "/mcp", mustSetupPass, nil, http.StatusUnauthorized},
		{"wrong method on an allowlisted path", http.MethodPatch, "/api/machines", mustSetupPass, nil, http.StatusUnauthorized},
		{"other connector", http.MethodPost, "/api/connectors/github/manual", mustSetupPass, nil, http.StatusUnauthorized},
		{"stack writes", http.MethodPost, "/api/services", mustSetupPass, nil, http.StatusUnauthorized},
		{"setup routes", http.MethodPut, "/api/setup/instance-url", mustSetupPass, nil, http.StatusOK},
		{"bootstrap", http.MethodPost, "/api/auth/bootstrap", mustSetupPass, nil, http.StatusOK},
		{"bootstrap verify", http.MethodPost, "/api/auth/bootstrap/verify", mustSetupPass, nil, http.StatusOK},
		{"cloudflare manual save", http.MethodPost, "/api/connectors/cloudflare/manual", mustSetupPass, nil, http.StatusOK},
		{"cloudflare manual verify", http.MethodPost, "/api/connectors/cloudflare/manual/verify", mustSetupPass, nil, http.StatusOK},
		{"connector list", http.MethodGet, "/api/connectors", mustSetupPass, nil, http.StatusOK},
		{"dns tunnel create", http.MethodPost, "/api/dns/tunnels", mustSetupPass, nil, http.StatusOK},
		{"dns instance proxy", http.MethodPost, "/api/dns/instance-proxy", mustSetupPass, nil, http.StatusOK},
		{"machines", http.MethodGet, "/api/machines", mustSetupPass, nil, http.StatusOK},
		{"projects are off the allowlist: first run has none", http.MethodGet, "/api/projects", mustSetupPass, nil, http.StatusUnauthorized},
		{"service deploys", http.MethodGet, "/api/services/s1/deploys", mustSetupPass, nil, http.StatusOK},
		{"deploy log", http.MethodGet, "/api/deploys/d1/log", mustSetupPass, nil, http.StatusOK},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, users, _, _ := newTestHarness(&fakeGitHub{})
			if tt.prepare != nil {
				tt.prepare(t, users)
			}
			var seen *User
			h := s.RequireAuth(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) { seen = UserFromCtx(r.Context()) }))
			req := httptest.NewRequest(tt.method, tt.path, nil)
			req.Header.Set("Authorization", "Bearer "+tt.token(t, s))
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)
			require.Equal(t, tt.want, rec.Code)
			if tt.want != http.StatusOK {
				assert.Nil(t, seen)
				return
			}
			require.NotNil(t, seen)
			assert.Equal(t, SetupUserID, seen.ID)
		})
	}
}

func TestRequireWS_RefusesSetupPass(t *testing.T) {
	s, _, _, _ := newTestHarness(&fakeGitHub{})
	h := s.RequireWS(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/ws/events?token="+mustSetupPass(t, s), nil))
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestService_WriteSetupCode(t *testing.T) {
	s, _, _, _ := newTestHarness(&fakeGitHub{user: ghUser("1", "owner")})
	dir := filepath.Join(t.TempDir(), "enroll")
	s.cfg.EnrollDir = dir
	path := filepath.Join(dir, setupCodeFile)
	ctx := context.Background()

	require.NoError(t, s.WriteSetupCode(ctx))
	first, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(string(first), setupCodePrefix))
	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())
	valid, err := s.setupCodeValid(ctx, string(first), s.cfg.Now())
	require.NoError(t, err)
	assert.True(t, valid)

	require.NoError(t, s.WriteSetupCode(ctx), "a restart rotates the code")
	second, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.NotEqual(t, first, second)
	valid, err = s.setupCodeValid(ctx, string(first), s.cfg.Now())
	require.NoError(t, err)
	assert.False(t, valid, "the old code dies with the rotation")

	_, err = s.Login(ctx, "code")
	require.NoError(t, err)
	assert.NoFileExists(t, path, "the first user removes the code file")
	valid, err = s.setupCodeValid(ctx, string(second), s.cfg.Now())
	require.NoError(t, err)
	assert.False(t, valid, "the first user clears the stored code")

	require.NoError(t, os.WriteFile(path, []byte("nxs_stale"), 0o600))
	require.NoError(t, s.WriteSetupCode(ctx), "a boot with a user present")
	assert.NoFileExists(t, path)
}

func TestService_WriteSetupCode_Errors(t *testing.T) {
	t.Run("store failure", func(t *testing.T) {
		s, _, _, _ := newTestHarness(&fakeGitHub{})
		s.cfg.EnrollDir = t.TempDir()
		s.cfg.SetupCodes.(*fakeSetupCodes).err = errors.New("disk gone")
		require.ErrorContains(t, s.WriteSetupCode(context.Background()), "store setup code")
	})
	t.Run("enroll dir is a file", func(t *testing.T) {
		s, _, _, _ := newTestHarness(&fakeGitHub{})
		file := filepath.Join(t.TempDir(), "file")
		require.NoError(t, os.WriteFile(file, nil, 0o600))
		s.cfg.EnrollDir = filepath.Join(file, "enroll")
		require.ErrorContains(t, s.WriteSetupCode(context.Background()), "create enroll dir")
	})
	t.Run("clear failure once a user exists", func(t *testing.T) {
		s, users, _, _ := newTestHarness(&fakeGitHub{})
		seedOwner(t, users, "u1", "1", "owner")
		s.cfg.SetupCodes.(*fakeSetupCodes).err = errors.New("disk gone")
		require.ErrorContains(t, s.WriteSetupCode(context.Background()), "clear setup codes")
	})
	t.Run("a first user still signs in when clearing fails", func(t *testing.T) {
		s, _, _, _ := newTestHarness(&fakeGitHub{token: "at", user: ghUser("1", "owner")})
		s.cfg.SetupCodes.(*fakeSetupCodes).err = errors.New("disk gone")
		_, err := s.Login(context.Background(), "code")
		require.NoError(t, err)
	})
}

func TestHandler_SetSetupInstanceURL(t *testing.T) {
	nexul := bootstrapStatusServer(t, http.StatusOK, `{"configured":false}`)
	tests := []struct {
		name    string
		local   bool
		url     string
		pass    bool
		prepare func(t *testing.T, users *fakeUserStore)
		want    int
	}{
		{"no pass", false, "https://nexul.example.com", false, nil, http.StatusUnauthorized},
		{"a signed-in owner is not a pass", false, "https://nexul.example.com", false, func(t *testing.T, users *fakeUserStore) {
			seedOwner(t, users, "u1", "1", "owner")
		}, http.StatusUnauthorized},
		{"http refused on a server install", false, nexul.URL, true, nil, http.StatusBadRequest},
		{"not a url", false, "nexul.example.com", true, nil, http.StatusBadRequest},
		{"https that does not answer as Nexul", false, "https://127.0.0.1:1", true, nil, http.StatusBadRequest},
		{"http allowed on a desktop install", true, nexul.URL + "/", true, nil, http.StatusOK},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, users, _, settings := newTestHarness(&fakeGitHub{})
			s.cfg.Local = tt.local
			if tt.prepare != nil {
				tt.prepare(t, users)
			}
			h := s.RequireAuth(NewHandler(s).SetupRoutes())
			body := strings.NewReader(`{"url":"` + tt.url + `"}`)
			req := httptest.NewRequest(http.MethodPut, "/api/setup/instance-url", body)
			if tt.pass {
				req = withSetupPass(t, s, http.MethodPut, "/api/setup/instance-url", body)
			}
			if !tt.pass && tt.prepare != nil {
				req = protectedRequest(t, s, users, http.MethodPut, "/api/setup/instance-url", "u1", `{"url":"`+tt.url+`"}`)
			}
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)
			require.Equal(t, tt.want, rec.Code, rec.Body.String())
			st, err := settings.Get(context.Background())
			require.NoError(t, err)
			if tt.want != http.StatusOK {
				assert.Empty(t, st.InstanceURL, "nothing is stored unless the check passes")
				return
			}
			assert.JSONEq(t, `{"instance_url":"`+nexul.URL+`"}`, rec.Body.String())
			assert.Equal(t, nexul.URL, st.InstanceURL)
		})
	}
}

func TestService_SetSetupInstanceURL_ErrorPaths(t *testing.T) {
	s, users, _, _ := newTestHarness(&fakeGitHub{})
	seedOwner(t, users, "u1", "1", "owner")
	_, err := s.SetSetupInstanceURL(context.Background(), SetupUserID, "https://nexul.example.com")
	require.ErrorIs(t, err, apperrs.ErrConflict, "no user may exist")
}

func TestHandler_Bootstrap_WithStoredInstanceURL(t *testing.T) {
	tests := []struct {
		name string
		body string
		want int
	}{
		{"mismatched instance url", `{"instance_url":"https://other.example.com","client_id":"id","client_secret":"sec","app_slug":"slug"}`, http.StatusBadRequest},
		{"instance url omitted", `{"client_id":"id","client_secret":"sec","app_slug":"slug"}`, http.StatusOK},
		{"matching instance url", `{"instance_url":"https://Nexul.example.com/","client_id":"id","client_secret":"sec","app_slug":"slug"}`, http.StatusOK},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, _, _, settings := newTestHarness(&fakeGitHub{})
			stored, err := settings.Set(context.Background(), "https://nexul.example.com")
			require.NoError(t, err)
			h := s.RequireAuth(NewHandler(s).Routes())
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, withSetupPass(t, s, http.MethodPost, "/api/auth/bootstrap", strings.NewReader(tt.body)))
			require.Equal(t, tt.want, rec.Code, rec.Body.String())
			st, err := settings.Get(context.Background())
			require.NoError(t, err)
			assert.Equal(t, "https://nexul.example.com", st.InstanceURL)
			assert.Equal(t, stored.SettingsVersion, st.SettingsVersion, "bootstrap leaves the stored URL alone")
		})
	}
}

func TestHandler_Bootstrap_RequiresPass(t *testing.T) {
	s, users, _, _ := newTestHarness(&fakeGitHub{})
	h := s.RequireAuth(NewHandler(s).Routes())
	body := `{"instance_url":"https://nexul.example.com","client_id":"id","client_secret":"sec","app_slug":"slug"}`

	for _, path := range []string{"/api/auth/bootstrap", "/api/auth/bootstrap/verify?check=slug", "/api/auth/bootstrap/verify?check=instance_url"} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, path, strings.NewReader(body)))
		assert.Equal(t, http.StatusUnauthorized, rec.Code, path)
	}

	_, _, err := users.UpsertUser(context.Background(), &Identity{UserID: "u1", Provider: ProviderGitHub, ProviderUserID: "1", Login: "someone"})
	require.NoError(t, err)
	_, err = s.Bootstrap(context.Background(), "u1", "https://nexul.example.com", "id", "sec", "slug")
	require.ErrorIs(t, err, apperrs.ErrUnauthorized, "a session is not a setup pass")
	require.ErrorIs(t, s.VerifyBootstrapInstanceURL(context.Background(), "u1", "https://nexul.example.com"), apperrs.ErrUnauthorized)
}

func TestHandler_BootstrapStatus_FirstRunFields(t *testing.T) {
	s, users, _, settings := newTestHarness(&fakeGitHub{})
	s.cfg.Local = true
	h := NewHandler(s).Routes()
	status := func() map[string]any {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/auth/bootstrap-status", nil))
		require.Equal(t, http.StatusOK, rec.Code)
		var body map[string]any
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		return body
	}

	body := status()
	assert.Equal(t, true, body["setup_open"])
	assert.Empty(t, body["instance_url"])
	assert.Equal(t, true, body["local"])

	_, err := settings.Set(context.Background(), "https://nexul.example.com")
	require.NoError(t, err)
	seedOwner(t, users, "u1", "1", "owner")
	body = status()
	assert.Equal(t, false, body["setup_open"])
	assert.Equal(t, "https://nexul.example.com", body["instance_url"])
}

func TestHandler_PublicAddress(t *testing.T) {
	s, users, _, _ := newTestHarness(&fakeGitHub{})
	h := s.RequireAuth(NewHandler(s).SetupRoutes())

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, withSetupPass(t, s, http.MethodGet, "/api/setup/public-address", nil))
	require.Equal(t, http.StatusOK, rec.Code)
	assert.JSONEq(t, `{"ipv4":"203.0.113.7","ipv6":""}`, rec.Body.String())

	seedOwner(t, users, "u1", "1", "owner")
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, protectedRequest(t, s, users, http.MethodGet, "/api/setup/public-address", "u1", ""))
	assert.Equal(t, http.StatusOK, rec.Code, "a signed-in owner may ask too")

	_, _, err := users.UpsertUser(context.Background(), &Identity{UserID: "u2", Provider: ProviderGitHub, ProviderUserID: "2", Login: "member"})
	require.NoError(t, err)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, protectedRequest(t, s, users, http.MethodGet, "/api/setup/public-address", "u2", ""))
	assert.Equal(t, http.StatusForbidden, rec.Code)

	_, err = s.PublicAddress(context.Background(), "")
	require.ErrorIs(t, err, apperrs.ErrUnauthorized)
}

func traceServer(t *testing.T, status int, body string) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv.URL
}

func TestCloudflareTrace(t *testing.T) {
	closed := traceServer(t, http.StatusOK, "")
	unreachable := httptest.NewServer(http.NotFoundHandler())
	unreachable.Close()
	tests := []struct {
		name   string
		v4, v6 string
		want   PublicAddress
	}{
		{"unreachable family is empty", unreachable.URL, unreachable.URL, PublicAddress{}},
		{"bad url is empty", "://", "://", PublicAddress{}},
		{"non-200 is empty", traceServer(t, http.StatusServiceUnavailable, "ip=203.0.113.7"), closed, PublicAddress{}},
		{"no ip line is empty", traceServer(t, http.StatusOK, "fl=1\nh=1.1.1.1\n"), closed, PublicAddress{}},
		{"wrong family is empty", traceServer(t, http.StatusOK, "ip=2001:db8::1\n"), traceServer(t, http.StatusOK, "ip=203.0.113.7\n"), PublicAddress{}},
		{"garbage ip is empty", traceServer(t, http.StatusOK, "ip=not-an-ip\n"), closed, PublicAddress{}},
		{"both families", traceServer(t, http.StatusOK, "fl=1\nip=203.0.113.7\nts=1\n"), traceServer(t, http.StatusOK, "ip=2001:db8::1\n"), PublicAddress{IPv4: "203.0.113.7", IPv6: "2001:db8::1"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			trace := cloudflareTrace{client: &http.Client{Timeout: time.Second}, v4URL: tt.v4, v6URL: tt.v6}
			assert.Equal(t, tt.want, trace.PublicAddress(context.Background()))
		})
	}
	assert.Equal(t, traceURLv4, newCloudflareTrace().v4URL)
}
