package githubapp

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// stub answers /apps/{slug} with appStatus+appBody and /applications/{id}/token with tokenStatus, recording the basic auth it saw.
func stub(t *testing.T, appStatus int, appBody string, tokenStatus int) (*httptest.Server, *string) {
	t.Helper()
	var seenAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.WriteHeader(appStatus)
			_, _ = w.Write([]byte(appBody))
			return
		}
		user, pass, _ := r.BasicAuth()
		seenAuth = user + ":" + pass
		w.WriteHeader(tokenStatus)
	}))
	t.Cleanup(srv.Close)
	return srv, &seenAuth
}

func TestVerify_ValidAppPasses(t *testing.T) {
	srv, auth := stub(t, http.StatusOK, `{"client_id":"Iv1.abc"}`, http.StatusNotFound)
	err := Verify(context.Background(), srv.Client(), srv.URL, "Iv1.abc", "s3cret", "my-app")
	require.NoError(t, err)
	assert.Equal(t, "Iv1.abc:s3cret", *auth)
}

func TestVerify_UnknownSlugIsInvalid(t *testing.T) {
	srv, _ := stub(t, http.StatusNotFound, `{"message":"Not Found"}`, http.StatusNotFound)
	err := Verify(context.Background(), srv.Client(), srv.URL, "Iv1.abc", "s3cret", "nope")
	require.Error(t, err)
	assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	assert.Contains(t, err.Error(), `slug "nope"`)
}

func TestVerify_SlugBelongsToAnotherApp(t *testing.T) {
	srv, _ := stub(t, http.StatusOK, `{"client_id":"Iv1.other"}`, http.StatusNotFound)
	err := Verify(context.Background(), srv.Client(), srv.URL, "Iv1.abc", "s3cret", "my-app")
	require.Error(t, err)
	assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	assert.Contains(t, err.Error(), "Iv1.other")
}

func TestVerify_BadSecretIsInvalid(t *testing.T) {
	srv, _ := stub(t, http.StatusOK, `{"client_id":"Iv1.abc"}`, http.StatusUnauthorized)
	err := Verify(context.Background(), srv.Client(), srv.URL, "Iv1.abc", "wrong", "my-app")
	require.Error(t, err)
	assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	assert.Contains(t, err.Error(), "client secret")
}

func TestVerify_GitHubOutageIsNotInvalid(t *testing.T) {
	srv, _ := stub(t, http.StatusBadGateway, ``, http.StatusBadGateway)
	err := Verify(context.Background(), srv.Client(), srv.URL, "Iv1.abc", "s3cret", "my-app")
	require.Error(t, err)
	assert.False(t, errors.Is(err, apperrs.ErrInvalid))
}

func TestVerifyCheck_SlugOnlySkipsTheSecret(t *testing.T) {
	srv, auth := stub(t, http.StatusOK, `{"client_id":"Iv1.abc"}`, http.StatusUnauthorized)
	err := VerifyCheck(context.Background(), srv.Client(), srv.URL, "slug", "Iv1.abc", "wrong", "my-app")
	require.NoError(t, err)
	assert.Empty(t, *auth, "the slug check must not touch the token endpoint")
}

func TestVerifyCheck_SecretOnlySkipsTheSlug(t *testing.T) {
	srv, _ := stub(t, http.StatusNotFound, `{"message":"Not Found"}`, http.StatusNotFound)
	err := VerifyCheck(context.Background(), srv.Client(), srv.URL, "secret", "Iv1.abc", "s3cret", "nope")
	require.NoError(t, err)
}

func TestVerifyCheck_UnknownIsInvalid(t *testing.T) {
	srv, _ := stub(t, http.StatusOK, `{"client_id":"Iv1.abc"}`, http.StatusNotFound)
	err := VerifyCheck(context.Background(), srv.Client(), srv.URL, "nope", "Iv1.abc", "s3cret", "my-app")
	require.Error(t, err)
	assert.True(t, errors.Is(err, apperrs.ErrInvalid))
}
