package pairing

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/harness"
	"github.com/otal-labs/nexul/internal/platform/crypto"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
)

const day = 24 * time.Hour

// runnerFixture is alice's laptop, reached through its connected runner, whose T3 Code mints "pair-tok".
type runnerFixture struct {
	svc     *Service
	repo    *fakeRepo
	exch    *fakeExchanger
	runners *fakeRunners
	bus     *fakeBus
}

func newRunnerFixture(t *testing.T, laptop Computer) *runnerFixture {
	t.Helper()
	repo := newFakeRepo()
	laptop.ID, laptop.UserID, laptop.Kind = "c-laptop", "u-alice", harness.KindT3Code
	repo.computers[laptop.ID] = laptop
	exch := &fakeExchanger{result: harness.PairResult{BearerToken: "bearer", ExpiresIn: 30 * day}, version: "0.0.34"}
	runners := &fakeRunners{runners: map[string]ComputerRunner{laptop.ID: {Connected: true}}, token: "pair-tok"}
	bus := &fakeBus{}
	svc := NewService(Config{Repo: repo, Harnesses: registry(exch), EncryptionKey: testEncKey, Tokens: newFakeTokens(), Runners: runners,
		Bus: bus, Now: func() time.Time { return testNow }})
	return &runnerFixture{svc: svc, repo: repo, exch: exch, runners: runners, bus: bus}
}

func factsEvent(t *testing.T, state string) eventbus.Event {
	t.Helper()
	raw, err := json.Marshal(map[string]any{
		"runner_id": "r-laptop", "computer_id": "c-laptop", "user_id": "u-alice", "members_only": true,
		"facts": map[string]any{"hostname": "alice-laptop", "t3": map[string]any{"state": state, "port": 3773, "version": "0.0.34"}},
	})
	require.NoError(t, err)
	return eventbus.Event{Topic: "runner.facts_reported", Payload: raw}
}

// TestHandleFactsReported_PairsThroughTheRunnerWhenTheSessionEndsWithinSevenDays: a facts report pairs a computer with
// no session, or one ending within 7 days, through its runner; a session with longer to run, or a T3 Code that is not
// answering, asks the computer for nothing.
func TestHandleFactsReported_PairsThroughTheRunnerWhenTheSessionEndsWithinSevenDays(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		expires  time.Time
		state    string
		wantPair bool
	}{
		{"never paired", time.Time{}, "answering", true},
		{"session ends in 6 days", testNow.Add(6 * day), "answering", true},
		{"session ends in 8 days", testNow.Add(8 * day), "answering", false},
		{"T3 Code not running", time.Time{}, "not_running", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := newRunnerFixture(t, Computer{TokenExpiresAt: tt.expires})

			require.NoError(t, f.svc.HandleFactsReported(t.Context(), factsEvent(t, tt.state)))

			stored, err := f.repo.GetComputer(t.Context(), "u-alice", "c-laptop")
			require.NoError(t, err)
			if !tt.wantPair {
				assert.Zero(t, f.runners.tokens, "no token is minted")
				assert.Equal(t, tt.expires, stored.TokenExpiresAt)
				return
			}
			assert.Equal(t, 1, f.runners.tokens)
			assert.Equal(t, "http://c-laptop.nexul-computer.invalid", f.exch.pairedURL, "exchanged through the runner")
			assert.Equal(t, "pair-tok", f.exch.pairedToken)
			assert.Equal(t, testNow.Add(30*day), stored.TokenExpiresAt)
			assert.Equal(t, "alice-laptop", stored.Name, "an unnamed computer takes the hostname it reported")
			plain, err := crypto.Decrypt(testEncKey, stored.BearerToken)
			require.NoError(t, err)
			assert.Equal(t, "bearer", string(plain), "the bearer is stored encrypted")
		})
	}
}

// TestPairComputer_ThroughTheRunner_FailureLeavesTheComputerAndShowsWhy: a failed mint or exchange changes nothing on
// the computer and its row says why, until a pairing through the runner succeeds.
func TestPairComputer_ThroughTheRunner_FailureLeavesTheComputerAndShowsWhy(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		spoil func(f *runnerFixture)
		why   string
	}{
		{"runner not connected", func(f *runnerFixture) { f.runners.runners["c-laptop"] = ComputerRunner{} }, "Laptop isn't connected to Nexul"},
		{"mint refused on the computer", func(f *runnerFixture) {
			f.runners.tokenErr = errors.New("t3 auth pairing create failed (exit status 1): no database")
		},
			"T3 Code on Laptop made no pairing token: t3 auth pairing create failed (exit status 1): no database"},
		{"exchange refused", func(f *runnerFixture) { f.exch.exchangeErr = errBoom }, "the harness refused this token"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			before := Computer{Name: "Laptop", ServerURL: runnerAddress("c-laptop"), TokenExpiresAt: testNow.Add(2 * day), BearerToken: "sealed-old", HarnessVersion: "0.0.33"}
			f := newRunnerFixture(t, before)
			tt.spoil(f)

			_, err := f.svc.PairComputer(t.Context(), "u-alice", "c-laptop", "")

			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.why)
			stored, err := f.repo.GetComputer(t.Context(), "u-alice", "c-laptop")
			require.NoError(t, err)
			before.ID, before.UserID, before.Kind = "c-laptop", "u-alice", harness.KindT3Code
			assert.Equal(t, before, *stored, "the computer is as it was")
			assert.Empty(t, f.repo.outbox)
			listed, err := f.svc.ListComputers(t.Context(), "u-alice")
			require.NoError(t, err)
			assert.Contains(t, listed[0].PairError, tt.why, "the row shows the reason")
			assert.Equal(t, []PairFailedEvent{{ComputerID: "c-laptop", UserID: "u-alice", MembersOnly: true}}, f.bus.pairFailed,
				"its owner's open views hear to read the reason")

			*f = *newRunnerFixture(t, before)
			f.svc.pairFailures["c-laptop"] = "an older failure"
			_, err = f.svc.PairComputer(t.Context(), "u-alice", "c-laptop", "")
			require.NoError(t, err)
			listed, err = f.svc.ListComputers(t.Context(), "u-alice")
			require.NoError(t, err)
			assert.Empty(t, listed[0].PairError, "a pairing that works clears it")
			assert.Empty(t, f.bus.pairFailed)
		})
	}
}

// recordingHarness lists providers for every computer but the offline one, recording which computers it was asked about.
type recordingHarness struct {
	*fakeExchanger
	mu      sync.Mutex
	offline string
	asked   []string
}

func (h *recordingHarness) ListProviders(_ context.Context, s harness.Session) ([]harness.Provider, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.asked = append(h.asked, s.ComputerID)
	if s.ComputerID == h.offline {
		return nil, apperrs.Retryable(errors.New("the runner of computer c-laptop is not connected"))
	}
	return []harness.Provider{{ID: "codex", Driver: "codex", Name: "Codex"}}, nil
}

// TestResolveTarget_OfflineComputerIsNeverSwapped: with a laptop and a desktop, a run aimed at the laptop while it is
// offline fails at once naming it and why, whether a project link, the defaults or the run's own pick aimed it there,
// and nothing is asked of the desktop.
func TestResolveTarget_OfflineComputerIsNeverSwapped(t *testing.T) {
	t.Parallel()
	repo := newFakeRepo()
	h := &recordingHarness{fakeExchanger: &fakeExchanger{}, offline: "c-laptop"}
	runners := &fakeRunners{runners: map[string]ComputerRunner{"c-laptop": {}, "c-desktop": {Connected: true}}}
	svc := NewService(Config{Repo: repo, Harnesses: harness.Registry{harness.KindT3Code: h}, EncryptionKey: testEncKey, Tokens: newFakeTokens(),
		Projects: fakeProjects{}, Runners: runners, Now: func() time.Time { return testNow }})
	sealed, err := crypto.Encrypt(testEncKey, []byte("bearer"))
	require.NoError(t, err)
	for id, name := range map[string]string{"c-laptop": "Laptop", "c-desktop": "Desktop"} {
		repo.computers[id] = Computer{ID: id, UserID: "u-alice", Kind: harness.KindT3Code, Name: name, ServerURL: runnerAddress(id),
			BearerToken: sealed, TokenExpiresAt: testNow.Add(20 * day), SetupConfirmedAt: &testNow}
		repo.setups[id] = []ProviderSetup{{Provider: "codex", ConfirmedAt: &testNow}}
	}
	ctx := t.Context()
	_, err = svc.SetProjectLink(ctx, "u-alice", "p-acme", ProjectLink{ComputerID: "c-laptop", HarnessProjectID: "t3-acme"})
	require.NoError(t, err)

	runs := map[string]func() (*ResolvedTarget, error){
		"project link": func() (*ResolvedTarget, error) { return svc.ResolveTarget(ctx, "u-alice", "p-acme") },
		"run's own pick": func() (*ResolvedTarget, error) {
			return svc.ResolveTargetOverride(ctx, "u-alice", "p-acme", "c-laptop", "", "", nil)
		},
		"defaults": func() (*ResolvedTarget, error) {
			_, err := svc.SetDefaults(ctx, "u-alice", Defaults{DefaultComputerID: "c-laptop", FallbackProjectID: "t3-acme"})
			require.NoError(t, err)
			return svc.ResolveTarget(ctx, "u-alice", "")
		},
	}
	for _, name := range []string{"project link", "run's own pick", "defaults"} {
		target, err := runs[name]()

		assert.Nil(t, target, name)
		var nc *NotConfiguredError
		require.ErrorAs(t, err, &nc, name)
		assert.Equal(t, ReasonOffline, nc.Reason, name)
		assert.Equal(t, "Laptop is offline: it isn't connected to Nexul.", err.Error(), name)
	}
	assert.False(t, slices.Contains(h.asked, "c-desktop"), "nothing ran on the desktop: %v", h.asked)
}
