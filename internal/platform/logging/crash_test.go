package logging

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func postCrashReports(h http.Handler, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/api/browser-reports", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/reports+json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestCrashReportHandler_RejectsMalformedAndOversizedUploads(t *testing.T) {
	for name, body := range map[string]string{
		"malformed":  `[{"type":`,
		"not a list": `{"type":"crash"}`,
		"oversized":  `[` + strings.Repeat(`{"type":"crash"},`, maxCrashReports) + `{"type":"crash"}]`,
	} {
		var out bytes.Buffer
		rec := postCrashReports(CrashReportHandler(slog.New(slog.NewJSONHandler(&out, nil))), body)
		assert.Equal(t, http.StatusBadRequest, rec.Code, name)
		assert.Empty(t, out.String(), name)
	}
}

func TestCrashReportHandler_RefusesUploadsPastTheSharedRate(t *testing.T) {
	var out bytes.Buffer
	h := CrashReportHandler(slog.New(slog.NewJSONHandler(&out, nil)))
	for range crashBurst {
		require.Equal(t, http.StatusNoContent, postCrashReports(h, `[]`).Code)
	}
	assert.Equal(t, http.StatusTooManyRequests, postCrashReports(h, `[{"type":"crash","body":{"reason":"oom"}}]`).Code)
	assert.Empty(t, out.String())
}

func TestCrashReportHandler_LogsEachCrashWithItsStackAndSkipsOtherReports(t *testing.T) {
	var out bytes.Buffer
	rec := postCrashReports(CrashReportHandler(slog.New(slog.NewJSONHandler(&out, nil))), `[
		{"type":"deprecation","url":"https://example.com/","body":{"id":"x"}},
		{"type":"crash","url":"https://example.com/acme/docs/1?token=dep_secret","user_agent":"Chrome/155",
		 "body":{"reason":"unresponsive","stack":"at findDiffStart (app.js:2:10)","is_top_level":true,"page_visibility":"visible"}}
	]`)
	require.Equal(t, http.StatusNoContent, rec.Code)

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	require.Len(t, lines, 1)
	var line map[string]any
	require.NoError(t, json.Unmarshal([]byte(lines[0]), &line))
	assert.Equal(t, "ERROR", line["level"])
	assert.Equal(t, "browser tab crashed (unresponsive)", line["msg"])
	assert.Equal(t, "web", line["source"])
	assert.Equal(t, "/acme/docs/1", line["url"])
	assert.Equal(t, "Chrome/155", line["user_agent"])
	assert.Equal(t, map[string]any{
		"reason": "unresponsive", "stack": "at findDiffStart (app.js:2:10)", "is_top_level": true, "page_visibility": "visible",
	}, line["attrs"])
}
