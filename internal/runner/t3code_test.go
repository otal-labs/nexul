package runner

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
)

// personalClient is a personal runner whose T3 Code lives in home and whose t3 command is t3 ("" for none), connected
// to serve.
func personalClient(t *testing.T, home, t3 string, logs *bytes.Buffer, serve func(ctx context.Context, conn *websocket.Conn)) {
	t.Helper()
	c := newTestClient(wsURL(wsTestServer(t, serve)), &fakeExecutor{})
	c.cfg.Personal, c.cfg.T3Home = true, home
	c.findT3 = func(string) string { return t3 }
	if logs != nil {
		c.log = slog.New(slog.NewTextHandler(logs, nil))
	}
	cancel, done := runClient(t, c)
	t.Cleanup(func() {
		cancel()
		<-done
	})
}

// firstFacts is the first facts frame a personal runner sends once connected.
func firstFacts(t *testing.T, home, t3 string) Facts {
	t.Helper()
	got := make(chan Facts, 1)
	personalClient(t, home, t3, nil, func(ctx context.Context, conn *websocket.Conn) {
		for {
			var f Frame
			if err := wsjson.Read(ctx, conn, &f); err != nil {
				return
			}
			if f.Type == FrameFacts {
				got <- *f.Facts
				return
			}
		}
	})
	select {
	case f := <-got:
		return f
	case <-time.After(5 * time.Second):
		t.Fatal("no facts within 5s")
		return Facts{}
	}
}

func TestPersonalRunner_ReportsT3CodeOnConnect(t *testing.T) {
	t3 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/.well-known/t3/environment" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(`{"serverVersion":"0.0.34","orchestrationProtocolVersion":2}`))
	}))
	t.Cleanup(t3.Close)
	u, err := url.Parse(t3.URL)
	require.NoError(t, err)
	port, err := strconv.Atoi(u.Port())
	require.NoError(t, err)
	hostname, err := os.Hostname()
	require.NoError(t, err)

	tests := []struct {
		name string
		home string
		t3   string
		want T3Facts
	}{
		{"answering on loopback", writeRuntimeFile(t, `{"host":"127.0.0.1","port":`+u.Port()+`}`), "", T3Facts{State: T3Answering, Port: port, Version: "0.0.34"}},
		{"bound to a LAN address", writeRuntimeFile(t, `{"host":"192.168.1.20","port":`+u.Port()+`}`), "/usr/bin/t3", T3Facts{State: T3NotLoopback}},
		{"installed, nothing answers", t.TempDir(), "/usr/bin/t3", T3Facts{State: T3NotRunning}},
		{"not installed", t.TempDir(), "", T3Facts{State: T3Missing}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, Facts{Hostname: hostname, T3: tt.want}, firstFacts(t, tt.home, tt.t3))
		})
	}
}

// fakeT3 writes a t3 command that checks it was asked for a pairing token in home, then runs body.
func fakeT3(t *testing.T, home, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "t3")
	script := "#!/bin/sh\n" +
		`[ "$*" = "auth pairing create --json --ttl 5m --label Nexul" ] || { echo "unexpected args: $*" >&2; exit 3; }` + "\n" +
		`[ "$T3CODE_HOME" = "` + home + `" ] || { echo "unexpected home: $T3CODE_HOME" >&2; exit 3; }` + "\n" + body + "\n"
	require.NoError(t, os.WriteFile(path, []byte(script), 0o700))
	return path
}

// pairTokenAnswer asks a personal runner for a pairing token and returns its answer and what it logged.
func pairTokenAnswer(t *testing.T, home, t3 string) (Frame, string) {
	t.Helper()
	answer := make(chan Frame, 1)
	var logs bytes.Buffer
	personalClient(t, home, t3, &logs, func(ctx context.Context, conn *websocket.Conn) {
		_ = wsjson.Write(ctx, conn, Frame{Type: FrameT3PairTokenRequest, ID: "p1"})
		for {
			var f Frame
			if err := wsjson.Read(ctx, conn, &f); err != nil {
				return
			}
			if f.Type == FrameT3PairToken {
				answer <- f
				return
			}
		}
	})
	select {
	case f := <-answer:
		return f, logs.String()
	case <-time.After(10 * time.Second):
		t.Fatal("no t3_pair_token within 10s")
		return Frame{}, ""
	}
}

func TestPersonalRunner_MintsAPairingTokenWithT3CodesOwnCommand(t *testing.T) {
	home := t.TempDir()

	t.Run("minted", func(t *testing.T) {
		got, logs := pairTokenAnswer(t, home, fakeT3(t, home, `echo '{"id":"pc-1","credential":"secret-pair-tok","label":"Nexul","scopes":[],"expiresAt":"2026-10-10T12:05:00Z"}'`))
		assert.Equal(t, Frame{Type: FrameT3PairToken, ID: "p1", Token: "secret-pair-tok"}, got)
		assert.NotContains(t, logs, "secret-pair-tok", "the token is never logged")
	})
	t.Run("refused by T3 Code", func(t *testing.T) {
		got, _ := pairTokenAnswer(t, home, fakeT3(t, home, "echo 'starting' >&2\necho 'no database at this home' >&2\nexit 1"))
		assert.Equal(t, Frame{Type: FrameT3PairToken, ID: "p1", Error: "t3 auth pairing create failed (exit status 1): no database at this home"}, got)
	})
	t.Run("printed no token", func(t *testing.T) {
		got, _ := pairTokenAnswer(t, home, fakeT3(t, home, "echo 'Pairing URL: http://127.0.0.1:3773/pair#token=abc'"))
		assert.Equal(t, "t3 auth pairing create printed no token", got.Error)
	})
	t.Run("not installed", func(t *testing.T) {
		got, _ := pairTokenAnswer(t, home, "")
		assert.Equal(t, "T3 Code's t3 command is not installed on this computer", got.Error)
	})
}

// t3Fixture is a runner handler on a real HTTP server with alice's laptop runner and bob's desktop runner.
type t3Fixture struct {
	h                  *Handler
	bus                *fakeBus
	repo               *fakeRunnerRepo
	alice, bob, deploy string
	srv                *httptest.Server
}

func newT3Fixture(t *testing.T) *t3Fixture {
	t.Helper()
	repo := newFakeRunnerRepo()
	bus := newFakeBus()
	h := newTestHandler(bus, repo)
	srv := httptest.NewServer(streamMux(h))
	t.Cleanup(srv.Close)
	return &t3Fixture{h: h, bus: bus, repo: repo, srv: srv, deploy: repo.enrolled("r-edge", "edge", "m1"),
		alice: repo.personal("r-alice", "u-alice", "c-laptop"), bob: repo.personal("r-bob", "u-bob", "c-desktop")}
}

func (f *t3Fixture) control(t *testing.T, credential string) (<-chan Frame, *websocket.Conn) {
	t.Helper()
	return (&streamFixture{h: f.h, srv: f.srv}).control(t, credential)
}

func TestPairingToken_OnlyTheRunnerAskedAnswers(t *testing.T) {
	f := newT3Fixture(t)
	aliceFrames, alice := f.control(t, f.alice)
	_, bob := f.control(t, f.bob)
	ctx := t.Context()

	type result struct {
		token string
		err   error
	}
	ask := func() <-chan result {
		out := make(chan result, 1)
		go func() {
			token, err := f.h.PairingToken(ctx, "c-laptop")
			out <- result{token, err}
		}()
		return out
	}

	got := ask()
	req := nextFrame(t, aliceFrames)
	require.Equal(t, FrameT3PairTokenRequest, req.Type)
	require.NoError(t, wsjson.Write(ctx, bob, Frame{Type: FrameT3PairToken, ID: req.ID, Token: "bobs-token"}))
	require.NoError(t, wsjson.Write(ctx, alice, Frame{Type: FrameT3PairToken, ID: req.ID, Token: "alices-token"}))
	assert.Equal(t, result{token: "alices-token"}, <-got, "another runner's answer to the id is dropped")

	got = ask()
	req = nextFrame(t, aliceFrames)
	require.NoError(t, wsjson.Write(ctx, alice, Frame{Type: FrameT3PairToken, ID: req.ID, Error: "T3 Code's t3 command is not installed on this computer"}))
	r := <-got
	require.ErrorIs(t, r.err, apperrs.ErrRetryable)
	assert.Contains(t, r.err.Error(), "T3 Code's t3 command is not installed on this computer")
}

// TestFacts_ReachPairingOnlyFromAPersonalRunner: a report is published for the computer and owner its runner was
// enrolled for, whatever the frame says, and a deploy runner's is dropped.
func TestFacts_ReachPairingOnlyFromAPersonalRunner(t *testing.T) {
	f := newT3Fixture(t)
	_, deploy := f.control(t, f.deploy)
	_, alice := f.control(t, f.alice)
	facts := &Facts{Hostname: "alice-laptop", T3: T3Facts{State: T3Answering, Port: 3773, Version: "0.0.34"}}
	require.NoError(t, wsjson.Write(t.Context(), deploy, Frame{Type: FrameFacts, Facts: &Facts{Hostname: "edge", T3: T3Facts{State: T3Missing}}}))
	require.NoError(t, wsjson.Write(t.Context(), deploy, Frame{Type: FrameHeartbeat, TS: time.Now().Unix()}))
	eventually(t, 2*time.Second, func() bool { return f.repo.heartbeatCount() == 1 })
	require.NoError(t, wsjson.Write(t.Context(), alice, Frame{Type: FrameFacts, Facts: facts}))

	eventually(t, 2*time.Second, func() bool { return len(f.bus.topicEvents(TopicFactsReported)) > 0 })
	require.Len(t, f.bus.topicEvents(TopicFactsReported), 1, "the deploy runner's report, read before alice's, was dropped")
	var got FactsReportedEvent
	require.NoError(t, json.Unmarshal(f.bus.topicEvents(TopicFactsReported)[0].Payload, &got))
	assert.Equal(t, FactsReportedEvent{RunnerID: "r-alice", ComputerID: "c-laptop", UserID: "u-alice", Facts: *facts, MembersOnly: true}, got)
	assert.True(t, eventbus.MembersOnly(f.bus.topicEvents(TopicFactsReported)[0].Payload), "integrations and automations skip it")
}
