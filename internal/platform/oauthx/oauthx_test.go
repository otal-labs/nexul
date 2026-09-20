package oauthx

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPostForm_DecodesJSONAndReturnsStatus(t *testing.T) {
	var gotContentType, gotAccept, gotAuthUser, gotAuthPass string
	var gotForm url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotContentType = r.Header.Get("Content-Type")
		gotAccept = r.Header.Get("Accept")
		gotAuthUser, gotAuthPass, _ = r.BasicAuth()
		require.NoError(t, r.ParseForm())
		gotForm = r.Form
		_, _ = w.Write([]byte(`{"access_token":"at"}`))
	}))
	defer srv.Close()

	var out struct {
		AccessToken string `json:"access_token"`
	}
	status, err := PostForm(context.Background(), srv.Client(), srv.URL,
		url.Values{"code": {"c1"}}, "user", "pass", "application/json", &out)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, status)
	assert.Equal(t, "at", out.AccessToken)
	assert.Equal(t, "application/x-www-form-urlencoded", gotContentType)
	assert.Equal(t, "application/json", gotAccept)
	assert.Equal(t, "user", gotAuthUser)
	assert.Equal(t, "pass", gotAuthPass)
	assert.Equal(t, "c1", gotForm.Get("code"))
}

func TestPostForm_NoBasicAuthWhenUserEmpty(t *testing.T) {
	var sawAuth bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _, sawAuth = r.BasicAuth()
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	var out map[string]any
	_, err := PostForm(context.Background(), srv.Client(), srv.URL, url.Values{}, "", "", "", &out)
	require.NoError(t, err)
	assert.False(t, sawAuth)
}

func TestPostForm_ReturnsStatusOnErrorResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"invalid_grant"}`))
	}))
	defer srv.Close()

	var out struct {
		Error string `json:"error"`
	}
	status, err := PostForm(context.Background(), srv.Client(), srv.URL, url.Values{}, "", "", "", &out)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, status)
	assert.Equal(t, "invalid_grant", out.Error)
}

func TestPostForm_InvalidJSONIsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`not json`))
	}))
	defer srv.Close()

	var out map[string]any
	_, err := PostForm(context.Background(), srv.Client(), srv.URL, url.Values{}, "", "", "", &out)
	require.Error(t, err)
}

func TestPostForm_NetworkErrorIsError(t *testing.T) {
	var out map[string]any
	_, err := PostForm(context.Background(), http.DefaultClient, "http://127.0.0.1:1/token", url.Values{}, "", "", "", &out)
	require.Error(t, err)
}
