package main

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/harness"
	"github.com/otal-labs/nexul/internal/pairing"
	"github.com/otal-labs/nexul/internal/platform/crypto"
	"github.com/otal-labs/nexul/internal/runner"
	"github.com/otal-labs/nexul/internal/t3rpc/t3rpctest"
)

// listedComputer is a computer as GET /api/pairing/computers shows it.
type listedComputer struct {
	ID             string       `json:"id"`
	Kind           harness.Kind `json:"kind"`
	ServerURL      string       `json:"server_url"`
	TokenExpiresAt time.Time    `json:"token_expires_at"`
	PairError      string       `json:"pair_error"`
}

func listComputer(t *testing.T, c privacyCast, id string) listedComputer {
	t.Helper()
	rec := callAPI(t, c.h, http.MethodGet, "/api/pairing/computers", c.alice, "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var got struct {
		Computers []listedComputer `json:"computers"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	for _, lc := range got.Computers {
		if lc.ID == id {
			return lc
		}
	}
	t.Fatalf("computer %s is not listed", id)
	return listedComputer{}
}

// startPersonalRunner enrolls alice's computer with its command's token and runs its runner against srv, reaching the
// T3 Code at t3URL.
func startPersonalRunner(t *testing.T, c privacyCast, srv *httptest.Server, enrollToken, t3URL string) {
	t.Helper()
	rec := callAPI(t, c.h, http.MethodPost, "/api/runners/enroll", "", `{"token":"`+enrollToken+`","machine":"alice-laptop","os":"linux","arch":"amd64"}`)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	var enrolled struct {
		Name       string `json:"name"`
		Credential string `json:"credential"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &enrolled))

	u, err := url.Parse(t3URL)
	require.NoError(t, err)
	home := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(home, "userdata"), 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(home, "userdata", "server-runtime.json"), []byte(`{"host":"127.0.0.1","port":`+u.Port()+`}`), 0o600))
	base := "ws" + strings.TrimPrefix(srv.URL, "http")
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	personal := runner.NewClient(runner.ClientConfig{
		URL: base + "/ws/runner", StreamURL: base + "/api/runners/streams", Credential: enrolled.Credential, Name: enrolled.Name,
		Logger: logger, Executor: runner.NewShellExecutor(nil, runner.ExecutorConfig{}, logger), Personal: true, T3Home: home,
	})
	ctx, stop := context.WithCancel(context.Background())
	stopped := make(chan error, 1)
	go func() { stopped <- personal.Run(ctx) }()
	t.Cleanup(func() {
		stop()
		<-stopped
	})
}

// TestIntegration_PersonalRunnerPairsT3CodeWithNoHumanInput: once alice's runner connects and reports T3 Code answering,
// the server mints a token through it and pairs over it, landing on the kind T3 Code speaks; a computer already on
// protocol 2 whose T3 Code answers protocol 1 is refused through the runner as before (ADR 0113), and pairing it
// again now over HTTP or MCP says the same.
func TestIntegration_PersonalRunnerPairsT3CodeWithNoHumanInput(t *testing.T) {
	bin := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(bin, "t3"), []byte("#!/bin/sh\necho '{\"id\":\"pc-1\",\"credential\":\"pair-token\"}'\n"), 0o700))
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))

	tests := []struct {
		name     string
		protocol int
		stored   harness.Kind
		want     harness.Kind
		refusal  string
	}{
		{name: "protocol 1", protocol: 1, stored: harness.KindT3Code, want: harness.KindT3Code},
		{name: "protocol 2", protocol: 2, stored: harness.KindT3Code, want: harness.KindT3CodeV2},
		{name: "protocol-2 computer answering protocol 1", protocol: 1, stored: harness.KindT3CodeV2, want: harness.KindT3CodeV2,
			refusal: "went back to its old orchestrator; Nexul only moves forward. Update T3 Code there."},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t3 := t3rpctest.New(t)
			t3.Protocol = tt.protocol
			c, db := newPrivacyCast(t)
			srv := httptest.NewServer(c.h)
			t.Cleanup(srv.Close)
			id, enrollToken := addComputer(t, c, c.alice, "")
			_, err := db.Exec(`UPDATE pairing_computers SET kind = ? WHERE id = ?`, string(tt.stored), id)
			require.NoError(t, err)

			startPersonalRunner(t, c, srv, enrollToken, t3.URL)

			var got listedComputer
			require.Eventually(t, func() bool {
				got = listComputer(t, c, id)
				return !got.TokenExpiresAt.IsZero() || got.PairError != ""
			}, 10*time.Second, 20*time.Millisecond, "paired, or refused, with no one asking")
			assert.Equal(t, tt.want, got.Kind)
			if tt.refusal != "" {
				assert.True(t, got.TokenExpiresAt.IsZero(), "the computer is as it was")
				assert.Contains(t, got.PairError, tt.refusal, "the row shows why")
				assert.Zero(t, t3.Exchanges.Load(), "the one-time token was never spent")
				rec := callAPI(t, c.h, http.MethodPost, "/api/pairing/computers/"+id+"/pair", c.alice, "")
				assert.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
				assert.Contains(t, rec.Body.String(), tt.refusal)
				assert.Contains(t, callToolResult(t, c.h, c.alice, "computer_pair", `{"id":"`+id+`"}`), tt.refusal)
				return
			}
			assert.Empty(t, got.PairError)
			assert.Equal(t, "http://"+id+pairing.ComputerHostSuffix, got.ServerURL, "reached through its runner from now on")
			stored, err := c.store.Pairing.GetComputer(t.Context(), "u-alice", id)
			require.NoError(t, err)
			assert.NotContains(t, stored.BearerToken, t3.Session().BearerToken, "the bearer is stored encrypted")
			plain, err := crypto.Decrypt([]byte("0123456789abcdef0123456789abcdef"), stored.BearerToken)
			require.NoError(t, err)
			assert.Equal(t, t3.Session().BearerToken, string(plain))
		})
	}
}
