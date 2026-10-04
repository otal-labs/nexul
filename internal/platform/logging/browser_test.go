package logging

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func postBrowserLogs(t *testing.T, body string) (*httptest.ResponseRecorder, string) {
	t.Helper()
	var out bytes.Buffer
	h := BrowserHandler(slog.New(slog.NewJSONHandler(&out, nil)), func(*http.Request) string { return "u1" })
	req := httptest.NewRequest(http.MethodPost, "/api/logs", strings.NewReader(body))
	req.Header.Set("User-Agent", "vitest")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec, out.String()
}

func TestBrowserHandler_RelaysEachRecordWithSourceAndUser(t *testing.T) {
	rec, out := postBrowserLogs(t, `{"records":[
		{"level":"error","message":"boom","url":"/stacks/1","attrs":{"stack":"Error: boom"}},
		{"level":"warn","message":"ws dropped malformed frame","url":"/"}
	]}`)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 2 {
		t.Fatalf("want 2 log lines, got %d: %s", len(lines), out)
	}
	for _, want := range []string{`"level":"ERROR"`, `"msg":"boom"`, `"source":"web"`, `"user_id":"u1"`, `"user_agent":"vitest"`, `"url":"/stacks/1"`, `"stack":"Error: boom"`} {
		if !strings.Contains(lines[0], want) {
			t.Fatalf("first line lacks %s: %s", want, lines[0])
		}
	}
	if !strings.Contains(lines[1], `"level":"WARN"`) {
		t.Fatalf("second line kept the wrong level: %s", lines[1])
	}
}

func TestBrowserHandler_RejectsMalformedEmptyAndOversizedBatches(t *testing.T) {
	for name, body := range map[string]string{
		"malformed": `{"records":`,
		"empty":     `{"records":[]}`,
		"oversized": `{"records":[` + strings.Repeat(`{"level":"error","message":"x"},`, maxBrowserBatch) + `{"level":"error","message":"x"}]}`,
	} {
		rec, out := postBrowserLogs(t, body)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("%s: status=%d body=%s", name, rec.Code, rec.Body.String())
		}
		if out != "" {
			t.Fatalf("%s: rejected batch still logged: %s", name, out)
		}
	}
}

func TestBrowserHandler_RedactsTokensInMessageURLAndAttrs(t *testing.T) {
	session := "ses_" + strings.Repeat("a", 43)
	rec, out := postBrowserLogs(t, `{"records":[
		{"level":"error","message":"ws connection error {\"url\":\"wss://n/ws/events?token=`+session+`\"}","url":"/chat?token=abc","attrs":{"socket":"wss://n/ws/x?token=nxr_1","line":3}}
	]}`)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	for _, leaked := range []string{session, "token=abc", "nxr_1"} {
		if strings.Contains(out, leaked) {
			t.Fatalf("logged %s: %s", leaked, out)
		}
	}
	if !strings.Contains(out, `"line":3`) {
		t.Fatalf("non-string attrs were dropped: %s", out)
	}
}
