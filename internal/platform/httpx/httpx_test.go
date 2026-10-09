package httpx

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

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
		{"rate limited", fmt.Errorf("%w: slow down", apperrs.ErrRateLimited), http.StatusTooManyRequests, "RATE_LIMITED"},
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

type detailedErr struct{ err error }

func (e detailedErr) Error() string     { return e.err.Error() }
func (e detailedErr) Unwrap() error     { return e.err }
func (e detailedErr) ErrorDetails() any { return map[string]string{"computer_id": "c-1"} }

func TestWriteError_Details(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{"detailed domain error", detailedErr{apperrs.ErrInvalid}, `{"message":"invalid","code":"INVALID","details":{"computer_id":"c-1"}}`},
		{"wrapped detailed error", fmt.Errorf("run: %w", detailedErr{apperrs.ErrInvalid}), `{"message":"run: invalid","code":"INVALID","details":{"computer_id":"c-1"}}`},
		{"plain error", apperrs.ErrInvalid, `{"message":"invalid","code":"INVALID"}`},
		{"internal error hides details", detailedErr{errors.New("boom")}, `{"message":"internal error","code":"INTERNAL"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			WriteError(rec, tt.err)
			assert.JSONEq(t, tt.want, rec.Body.String())
		})
	}
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
	WriteJSON(rec, http.StatusOK, map[string]any{"items": []string(nil)})
	assert.Equal(t, `{"items":[]}`, rec.Body.String())
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

func TestWriteError_CodedError_ReportsItsOwnCode(t *testing.T) {
	rec := httptest.NewRecorder()
	WriteError(rec, apperrs.WithCode("invalid_code", fmt.Errorf("%w: enrollment code is unknown, used or expired", apperrs.ErrUnauthorized)))
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.JSONEq(t, `{"message":"unauthorized: enrollment code is unknown, used or expired","code":"invalid_code"}`, rec.Body.String())
}

func TestWriteError_CodedInternalError_KeepsTheCodeHidden(t *testing.T) {
	rec := httptest.NewRecorder()
	WriteError(rec, apperrs.WithCode("secret_detail", errors.New("boom")))
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.JSONEq(t, `{"message":"internal error","code":"INTERNAL"}`, rec.Body.String())
}

func TestClientAddr(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "[2001:db8::1]:443"
	assert.Equal(t, "2001:db8::1", ClientAddr(req))
	req.RemoteAddr = "pipe"
	assert.Equal(t, "pipe", ClientAddr(req))
}

type benchPerson struct {
	Kind  string `json:"kind"`
	Login string `json:"login,omitempty"`
}

type benchRow struct {
	ID         string      `json:"id"`
	ProjectID  string      `json:"project_id"`
	Title      string      `json:"title"`
	Body       string      `json:"body"`
	Status     string      `json:"status"`
	Number     int         `json:"number"`
	Reporter   benchPerson `json:"reporter"`
	CreatedAt  time.Time   `json:"created_at"`
	FinishedAt *time.Time  `json:"finished_at,omitempty"`
	Labels     []string    `json:"labels"`
	Watchers   []string    `json:"watchers"`
}

// largeList is a list response the size of a busy board: 1,000 rows, two in three without labels.
func largeList() map[string]any {
	now := time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)
	rows := make([]*benchRow, 1000)
	for i := range rows {
		rows[i] = &benchRow{
			ID: fmt.Sprintf("t-%d", i), ProjectID: "p-1", Title: fmt.Sprintf("Fix the thing %d", i), Body: "a body",
			Status: "open", Number: i + 1, Reporter: benchPerson{Kind: "user", Login: "sam"}, CreatedAt: now,
			Watchers: []string{"lena"},
		}
		if i%3 == 0 {
			rows[i].Labels = []string{"bug", "ui"}
			rows[i].FinishedAt = &now
		}
	}
	return map[string]any{"tickets": rows, "total": len(rows)}
}

type discardWriter struct{ h http.Header }

func (d discardWriter) Header() http.Header       { return d.h }
func (discardWriter) Write(b []byte) (int, error) { return len(b), nil }
func (discardWriter) WriteHeader(int)             {}

func BenchmarkWriteJSON_LargeList(b *testing.B) {
	v := largeList()
	w := discardWriter{h: http.Header{}}
	b.ReportAllocs()
	for b.Loop() {
		WriteJSON(w, http.StatusOK, v)
	}
}
