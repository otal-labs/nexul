package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

func bootstrapStatusServer(t *testing.T, status int, body string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/auth/bootstrap-status" {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestService_VerifyInstanceURL(t *testing.T) {
	s, _, _, _ := newTestHarness(&fakeGitHub{})

	t.Run("passes when the url answers as a Nexul bootstrap-status", func(t *testing.T) {
		srv := bootstrapStatusServer(t, http.StatusOK, `{"configured":false}`)
		require.NoError(t, s.VerifyInstanceURL(context.Background(), srv.URL+"/"))
	})

	t.Run("fails when something else answers", func(t *testing.T) {
		srv := bootstrapStatusServer(t, http.StatusOK, `<html>welcome to nginx</html>`)
		err := s.VerifyInstanceURL(context.Background(), srv.URL)
		require.ErrorIs(t, err, apperrs.ErrInvalid)
		assert.Contains(t, err.Error(), "not as a Nexul server")
	})

	t.Run("fails when nothing answers", func(t *testing.T) {
		srv := bootstrapStatusServer(t, http.StatusOK, `{}`)
		srv.Close()
		err := s.VerifyInstanceURL(context.Background(), srv.URL)
		require.ErrorIs(t, err, apperrs.ErrInvalid)
		assert.Contains(t, err.Error(), "could not reach")
	})

	t.Run("rejects a relative url before dialing", func(t *testing.T) {
		require.ErrorIs(t, s.VerifyInstanceURL(context.Background(), "deploy.example.com"), apperrs.ErrInvalid)
	})
}

func TestHandler_BootstrapVerify_InstanceURL(t *testing.T) {
	s, _, _, _ := newTestHarness(&fakeGitHub{})
	h := NewHandler(s).Routes()
	srv := bootstrapStatusServer(t, http.StatusOK, `{"configured":false}`)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/auth/bootstrap/verify?check=instance_url",
		strings.NewReader(`{"instance_url":"`+srv.URL+`"}`)))
	assert.Equal(t, http.StatusNoContent, rec.Code)
}

func TestHandler_CallbackGET_RedirectsToInstanceURL(t *testing.T) {
	s, _, _, settings := newTestHarness(&fakeGitHub{user: ghUser("1", "onik97")})
	_, err := settings.Set(context.Background(), "https://deploy.example.com/")
	require.NoError(t, err)
	h := NewHandler(s).Routes()

	// Plain http request host, as seen behind a TLS-terminating proxy: the stored instance URL wins.
	req := httptest.NewRequest(http.MethodGet, "http://server:8080/auth/callback?code=c&state=st8", nil)
	req.AddCookie(&http.Cookie{Name: stateCookie, Value: "st8"})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusFound, rec.Code)
	assert.True(t, strings.HasPrefix(rec.Header().Get("Location"), "https://deploy.example.com/login?token="), rec.Header().Get("Location"))
}
