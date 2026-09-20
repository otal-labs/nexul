package logging

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func TestFanout_WritesToEveryHandlerAndGatesOnLevel(t *testing.T) {
	var a, b bytes.Buffer
	l := slog.New(fanout{min: slog.LevelInfo, handlers: []slog.Handler{
		slog.NewJSONHandler(&a, nil),
		slog.NewJSONHandler(&b, &slog.HandlerOptions{Level: slog.LevelWarn}),
	}})
	l.Debug("dropped everywhere")
	l.Info("info line")
	l.With("k", "v").WithGroup("g").Warn("warn line", "x", 1)

	if strings.Contains(a.String(), "dropped") || strings.Contains(b.String(), "dropped") {
		t.Fatal("debug record passed the fanout level gate")
	}
	if !strings.Contains(a.String(), "info line") || strings.Contains(b.String(), "info line") {
		t.Fatalf("info should reach a only\na=%s\nb=%s", a.String(), b.String())
	}
	for name, buf := range map[string]*bytes.Buffer{"a": &a, "b": &b} {
		if !strings.Contains(buf.String(), `"warn line"`) || !strings.Contains(buf.String(), `"k":"v"`) || !strings.Contains(buf.String(), `"g":{"x":1}`) {
			t.Fatalf("%s lost the warn record, attrs, or group: %s", name, buf.String())
		}
	}
}

func TestNewWithOTLP_EmptyEndpointIsPlainLogger(t *testing.T) {
	l, shutdown, err := NewWithOTLP(context.Background(), "info", OTLP{})
	if err != nil || l == nil {
		t.Fatalf("err=%v logger=%v", err, l)
	}
	if err := shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown: %v", err)
	}
}

func TestNewWithOTLP_ShipsRecordsWithAuthAndStream(t *testing.T) {
	var posts atomic.Int32
	var gotAuth, gotStream, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		posts.Add(1)
		gotAuth, gotStream, gotPath = r.Header.Get("Authorization"), r.Header.Get("stream-name"), r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	l, shutdown, err := NewWithOTLP(context.Background(), "info", OTLP{
		Endpoint: srv.URL + "/api/default", User: "u@example.com", Token: "tok", Stream: "nexul", Service: "test",
	})
	if err != nil {
		t.Fatal(err)
	}
	l.Info("shipped line", "deploy_id", "d-1")
	if err := shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown: %v", err)
	}

	if posts.Load() == 0 {
		t.Fatal("no OTLP export reached the endpoint after shutdown")
	}
	if gotPath != "/api/default/v1/logs" {
		t.Fatalf("path = %q", gotPath)
	}
	if gotAuth != "Basic dUBleGFtcGxlLmNvbTp0b2s=" {
		t.Fatalf("authorization = %q", gotAuth)
	}
	if gotStream != "nexul" {
		t.Fatalf("stream-name = %q", gotStream)
	}
}
