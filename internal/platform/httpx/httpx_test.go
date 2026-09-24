package httpx

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

func TestWriteError_MapsSentinelsToC2(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{"not found", apperrs.ErrNotFound, http.StatusNotFound, "NOT_FOUND"},
		{"conflict", apperrs.ErrConflict, http.StatusConflict, "CONFLICT"},
		{"unauthorized", apperrs.ErrUnauthorized, http.StatusUnauthorized, "UNAUTHORIZED"},
		{"forbidden", apperrs.ErrForbidden, http.StatusForbidden, "FORBIDDEN"},
		{"invalid", apperrs.ErrInvalid, http.StatusBadRequest, "INVALID"},
		{"retryable", apperrs.Retryable(errors.New("boom")), http.StatusServiceUnavailable, "RETRYABLE"},
		{"fatal", apperrs.Fatal(errors.New("boom")), http.StatusUnprocessableEntity, "FATAL"},
		{"unknown", errors.New("boom"), http.StatusInternalServerError, "INTERNAL"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			WriteError(rec, tt.err)
			assert.Equal(t, tt.wantStatus, rec.Code)

			var env Envelope
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
			assert.Equal(t, tt.wantCode, env.Code)
			assert.NotEmpty(t, env.Message)
		})
	}
}

func TestWriteFieldError_KeysTheMessageUnderTheField(t *testing.T) {
	rec := httptest.NewRecorder()
	WriteFieldError(rec, fmt.Errorf("%w: token refused", apperrs.ErrInvalid), "token")
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.JSONEq(t, `{"message":"invalid: token refused","code":"INVALID","errors":{"token":["invalid: token refused"]}}`, rec.Body.String())
}

func TestWriteError_OmitsFieldErrors(t *testing.T) {
	rec := httptest.NewRecorder()
	WriteError(rec, apperrs.ErrInvalid)
	assert.NotContains(t, rec.Body.String(), "errors")
}

func TestWriteError_UnknownErrorDoesNotLeakMessage(t *testing.T) {
	rec := httptest.NewRecorder()
	WriteError(rec, errors.New("secret internal detail"))
	var env Envelope
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
	assert.Equal(t, "internal error", env.Message)
}

func TestWriteError_WrappedSentinelMapsToNotFound(t *testing.T) {
	rec := httptest.NewRecorder()
	WriteError(rec, fmt.Errorf("get doc xyz: %w", apperrs.ErrNotFound))
	assert.Equal(t, http.StatusNotFound, rec.Code)
	var env Envelope
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
	assert.Equal(t, "NOT_FOUND", env.Code)
}

func TestWriteJSON_SetsContentTypeAndBody(t *testing.T) {
	rec := httptest.NewRecorder()
	WriteJSON(rec, http.StatusCreated, map[string]string{"a": "b"})
	assert.Equal(t, http.StatusCreated, rec.Code)
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))
	assert.JSONEq(t, `{"a":"b"}`, rec.Body.String())
}

func TestWriteJSON_NilSliceMarshalsAsEmptyArray(t *testing.T) {
	rec := httptest.NewRecorder()
	var nilSlice []string
	WriteJSON(rec, http.StatusOK, nilSlice)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.JSONEq(t, `[]`, rec.Body.String())

	rec = httptest.NewRecorder()
	WriteJSON(rec, http.StatusOK, []string{"x"})
	assert.JSONEq(t, `["x"]`, rec.Body.String())
}

func TestWriteJSON_NilSlicesInsideStructsMarshalAsEmptyArrays(t *testing.T) {
	type links struct {
		PRs      []string `json:"prs"`
		Branches []string `json:"branches"`
		Names    []string `json:"names"`
		EmptyPtr []string `json:"ptr"`
	}

	rec := httptest.NewRecorder()
	WriteJSON(rec, http.StatusOK, links{Names: []string{"a"}})
	assert.JSONEq(t, `{"prs":[],"branches":[],"names":["a"],"ptr":[]}`, rec.Body.String())

	// nested inside a wrapper and inside slice elements
	rec = httptest.NewRecorder()
	WriteJSON(rec, http.StatusOK, struct {
		Items []links `json:"items"`
	}{Items: []links{{}}})
	assert.JSONEq(t, `{"items":[{"prs":[],"branches":[],"names":[],"ptr":[]}]}`, rec.Body.String())
}

func TestDecodeJSON_ValidBody(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"title":"hi"}`))
	var v map[string]string
	require.NoError(t, DecodeJSON(r, &v))
	assert.Equal(t, "hi", v["title"])
}

func TestDecodeJSON_MalformedBody_ErrInvalid(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"title":`))
	var v map[string]string
	err := DecodeJSON(r, &v)
	require.Error(t, err)
	assert.True(t, errors.Is(err, apperrs.ErrInvalid))
}
