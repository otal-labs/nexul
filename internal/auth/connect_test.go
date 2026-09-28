package auth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

var connectCodeShape = regexp.MustCompile(`^[0-9A-HJKMNP-TV-Z]{4}-[0-9A-HJKMNP-TV-Z]{4}-[0-9A-HJKMNP-TV-Z]{4}$`)

func newConnectHarness(t *testing.T) (*Service, *fakeConnectCodes, string) {
	t.Helper()
	s, _, u := newSessionHarness(t)
	codes := newFakeConnectCodes()
	s.cfg.ConnectCodes = codes
	return s, codes, u
}

func TestNormalizeConnectCode(t *testing.T) {
	tests := []struct {
		name, in, want string
	}{
		{"as issued", "7Q4F-K92M-3XYZ", "7Q4FK92M3XYZ"},
		{"lowercase without dashes", "7q4fk92m3xyz", "7Q4FK92M3XYZ"},
		{"spaces", "7q4f k92m 3xyz", "7Q4FK92M3XYZ"},
		{"look-alikes fold to crockford", "oQ4F-Ki2l-3XYZ", "0Q4FK1213XYZ"},
		{"empty", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, normalizeConnectCode(tt.in))
		})
	}
}

func TestIssueConnectCode(t *testing.T) {
	s, codes, u := newConnectHarness(t)

	t.Run("store errors first", func(t *testing.T) {
		codes.err = errors.New("disk gone")
		_, err := s.IssueConnectCode(context.Background(), u, "https://nexul.example.com")
		require.Error(t, err)
		require.NotErrorIs(t, err, apperrs.ErrInvalid)
		codes.err = nil
	})

	t.Run("unconfigured store refuses", func(t *testing.T) {
		unwired := newTestService(&fakeGitHub{}, newFakeUserStore(), newFakeAllowlist(), newFakeSettings())
		_, err := unwired.IssueConnectCode(context.Background(), u, "")
		require.ErrorIs(t, err, apperrs.ErrUnauthorized)
	})

	code, err := s.IssueConnectCode(context.Background(), u, "https://nexul.example.com")
	require.NoError(t, err)
	assert.Regexp(t, connectCodeShape, code.Code, "twelve crockford characters as XXXX-XXXX-XXXX")
	assert.Equal(t, "https://nexul.example.com", code.Host)
	assert.Equal(t, s.cfg.Now().UTC().Add(connectCodeTTL), code.ExpiresAt)
	assert.Equal(t, 1, codes.replace)

	newer, err := s.IssueConnectCode(context.Background(), u, "https://nexul.example.com")
	require.NoError(t, err)
	assert.NotEqual(t, code.Code, newer.Code)
	_, err = s.ExchangeConnectCode(context.Background(), "203.0.113.5", code.Code, ConnectDevice{Model: "Pixel 8"})
	require.ErrorIs(t, err, apperrs.ErrInvalid, "a newer code invalidates the last")
	_, err = s.ExchangeConnectCode(context.Background(), "203.0.113.5", newer.Code, ConnectDevice{Model: "Pixel 8"})
	require.NoError(t, err)
}

func TestExchangeConnectCode(t *testing.T) {
	s, codes, u := newConnectHarness(t)
	issue := func(t *testing.T) string {
		t.Helper()
		code, err := s.IssueConnectCode(context.Background(), u, "")
		require.NoError(t, err)
		return code.Code
	}

	t.Run("wrong code is the one generic error and counts", func(t *testing.T) {
		_, err := s.ExchangeConnectCode(context.Background(), "203.0.113.1", "", ConnectDevice{})
		require.ErrorIs(t, err, apperrs.ErrInvalid)
		var coded *apperrs.Coded
		require.ErrorAs(t, err, &coded)
		assert.Equal(t, "invalid_code", coded.Code)
		assert.Len(t, s.exchanges.failures["203.0.113.1"], 1)
	})

	t.Run("store errors are not the generic answer", func(t *testing.T) {
		codes.err = errors.New("disk gone")
		_, err := s.ExchangeConnectCode(context.Background(), "203.0.113.2", "AAAA-AAAA-AAAA", ConnectDevice{})
		require.Error(t, err)
		require.NotErrorIs(t, err, apperrs.ErrInvalid)
		codes.err = nil
	})

	t.Run("a code works exactly once, in any spelling", func(t *testing.T) {
		code := issue(t)
		typed := strings.ToLower(strings.ReplaceAll(code, "-", ""))
		token, err := s.ExchangeConnectCode(context.Background(), "203.0.113.3", typed, ConnectDevice{Model: "Pixel 8", OS: "Android 15", AppVersion: "1.0"})
		require.NoError(t, err)
		require.True(t, strings.HasPrefix(token, sessionPrefix))
		_, err = s.ExchangeConnectCode(context.Background(), "203.0.113.3", code, ConnectDevice{Model: "Pixel 8"})
		require.ErrorIs(t, err, apperrs.ErrInvalid)

		_, ses, err := s.AuthenticateSession(context.Background(), token, "")
		require.NoError(t, err)
		assert.Equal(t, ClientPhone, ses.Client)
		assert.Equal(t, "Android", ses.Platform)
		assert.Equal(t, "Pixel 8", ses.Label)
		assert.Equal(t, "203.0.113.3", ses.IP)
		assert.Equal(t, s.cfg.Now().Add(phoneSessionTTL), ses.ExpiresAt)
	})

	t.Run("a nameless phone is still labelled", func(t *testing.T) {
		token, err := s.ExchangeConnectCode(context.Background(), "203.0.113.4", issue(t), ConnectDevice{Model: "  "})
		require.NoError(t, err)
		_, ses, err := s.AuthenticateSession(context.Background(), token, "")
		require.NoError(t, err)
		assert.Equal(t, "Phone", ses.Label)
	})

	t.Run("never after two minutes", func(t *testing.T) {
		code := issue(t)
		start := s.cfg.Now()
		s.cfg.Now = func() time.Time { return start.Add(connectCodeTTL) }
		defer func() { s.cfg.Now = func() time.Time { return start } }()
		_, err := s.ExchangeConnectCode(context.Background(), "203.0.113.5", code, ConnectDevice{})
		require.ErrorIs(t, err, apperrs.ErrInvalid)
	})

	t.Run("the sixth wrong code from one address is throttled", func(t *testing.T) {
		for range exchangeMaxFailures {
			_, err := s.ExchangeConnectCode(context.Background(), "203.0.113.6", "AAAA-AAAA-AAAA", ConnectDevice{})
			require.ErrorIs(t, err, apperrs.ErrInvalid)
		}
		code := issue(t)
		_, err := s.ExchangeConnectCode(context.Background(), "203.0.113.6", code, ConnectDevice{})
		require.ErrorIs(t, err, apperrs.ErrRateLimited, "the right code is refused too while throttled")
		_, err = s.ExchangeConnectCode(context.Background(), "203.0.113.7", code, ConnectDevice{})
		require.NoError(t, err, "another address is not throttled, and the code survived the refused attempt")

		start := s.cfg.Now()
		s.cfg.Now = func() time.Time { return start.Add(exchangeWindow) }
		defer func() { s.cfg.Now = func() time.Time { return start } }()
		_, err = s.ExchangeConnectCode(context.Background(), "203.0.113.6", "AAAA-AAAA-AAAA", ConnectDevice{})
		require.ErrorIs(t, err, apperrs.ErrInvalid, "failures age out of the window")
	})

	t.Run("unconfigured store refuses", func(t *testing.T) {
		unwired := newTestService(&fakeGitHub{}, newFakeUserStore(), newFakeAllowlist(), newFakeSettings())
		_, err := unwired.ExchangeConnectCode(context.Background(), "203.0.113.8", "AAAA-AAAA-AAAA", ConnectDevice{})
		require.ErrorIs(t, err, apperrs.ErrUnauthorized)
	})
}

func TestHandler_ConnectCodes(t *testing.T) {
	s, users, _, settings := newTestHarness(&fakeGitHub{user: ghUser("1", "onik97")})
	seedOwner(t, users, "u1", "1", "onik97")
	s.cfg.PATs = newFakePATStore()
	s.cfg.ConnectCodes = newFakeConnectCodes()
	protected := protectedHandler(s, users)
	public := NewHandler(s).Routes()

	issue := func(token string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/api/auth/connect-codes", nil)
		req.Host = "192.0.2.10:8080"
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		protected.ServeHTTP(rec, req)
		return rec
	}
	exchange := func(addr, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/api/auth/connect-codes/exchange", strings.NewReader(body))
		req.RemoteAddr = addr
		rec := httptest.NewRecorder()
		public.ServeHTTP(rec, req)
		return rec
	}

	current, err := sign(s, "u1")
	require.NoError(t, err)
	patRaw, _, err := s.MintPAT(context.Background(), "u1", "ci")
	require.NoError(t, err)

	t.Run("a personal access token cannot issue a code", func(t *testing.T) {
		assert.Equal(t, http.StatusForbidden, issue(patRaw).Code)
	})

	t.Run("host falls back to the request when no instance URL is set", func(t *testing.T) {
		rec := issue(current)
		require.Equal(t, http.StatusOK, rec.Code)
		var code ConnectCode
		require.NoError(t, decodeJSON(rec, &code))
		assert.Equal(t, "http://192.0.2.10:8080", code.Host)
	})

	_, err = settings.Set(context.Background(), "https://nexul.example.com/")
	require.NoError(t, err)
	rec := issue(current)
	require.Equal(t, http.StatusOK, rec.Code)
	var code ConnectCode
	require.NoError(t, decodeJSON(rec, &code))
	assert.Equal(t, "https://nexul.example.com", code.Host)
	assert.Regexp(t, connectCodeShape, code.Code)

	t.Run("bad body", func(t *testing.T) {
		assert.Equal(t, http.StatusBadRequest, exchange("198.51.100.1:4000", "{").Code)
	})

	t.Run("exchange returns a phone session and the server version", func(t *testing.T) {
		rec := exchange("198.51.100.1:4000", `{"code":"`+code.Code+`","device":{"model":"Pixel 8","os":"Android 15","app_version":"1.0"}}`)
		require.Equal(t, http.StatusOK, rec.Code)
		var body struct {
			Token         string `json:"token"`
			ServerVersion string `json:"server_version"`
		}
		require.NoError(t, decodeJSON(rec, &body))
		assert.True(t, strings.HasPrefix(body.Token, sessionPrefix))
		assert.Equal(t, "dev", body.ServerVersion)
		rec = exchange("198.51.100.1:4000", `{"code":"`+code.Code+`","device":{"model":"Pixel 8"}}`)
		assert.Equal(t, http.StatusBadRequest, rec.Code, "a code works once")
		assert.Equal(t, "invalid_code", errorCode(t, rec))
	})

	t.Run("the sixth wrong code from one address gets 429", func(t *testing.T) {
		for range exchangeMaxFailures {
			assert.Equal(t, http.StatusBadRequest, exchange("198.51.100.2:4000", `{"code":"AAAA-AAAA-AAAA"}`).Code)
		}
		assert.Equal(t, http.StatusTooManyRequests, exchange("198.51.100.2:5000", `{"code":"AAAA-AAAA-AAAA"}`).Code)
		assert.Equal(t, http.StatusBadRequest, exchange("198.51.100.3:4000", `{"code":"AAAA-AAAA-AAAA"}`).Code, "another address is not throttled")
	})
}
