package pairing

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/harness"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
)

const wentBack = "T3 Code on Onik's laptop went back to its old orchestrator; Nexul only moves forward. Update T3 Code there."

// twoKinds is a service whose computers pair through v1 and, once moved, v2; both answer Pair with their own kind.
func twoKinds(t *testing.T) (*Service, *fakeRepo, *fakeExchanger, *fakeExchanger) {
	t.Helper()
	repo := newFakeRepo()
	v1 := &fakeExchanger{result: harness.PairResult{BearerToken: "b1", ExpiresIn: time.Hour, Kind: harness.KindT3Code}, version: "0.0.45"}
	v2 := &fakeExchanger{result: harness.PairResult{BearerToken: "b2", ExpiresIn: time.Hour, Kind: harness.KindT3CodeV2}, version: "0.0.46-nightly.20261003.2632"}
	svc := NewService(Config{
		Repo: repo, EncryptionKey: testEncKey, Tokens: newFakeTokens(), Projects: fakeProjects{},
		Harnesses: harness.Registry{harness.KindT3Code: v1, harness.KindT3CodeV2: v2},
		Now:       func() time.Time { return time.Date(2026, 10, 3, 12, 0, 0, 0, time.UTC) },
	})
	return svc, repo, v1, v2
}

func topics(evts []eventbus.OutboxEvent) []string {
	out := []string{}
	for _, e := range evts {
		out = append(out, e.Topic)
	}
	return out
}

func TestSwitchHarness_Failures_LeaveTheComputerWhereItWas(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		to      harness.Kind
		breakIt func(repo *fakeRepo, v2 *fakeExchanger)
		wantIs  error
	}{
		{"a kind nothing moves to", harness.KindT3Code, func(*fakeRepo, *fakeExchanger) {}, apperrs.ErrInvalid},
		{"an unknown kind", "opencode2", func(*fakeRepo, *fakeExchanger) {}, apperrs.ErrInvalid},
		{"the new kind's version probe fails", harness.KindT3CodeV2, func(_ *fakeRepo, v2 *fakeExchanger) { v2.versionErr = apperrs.Retryable(errBoom) }, errBoom},
		{"the store fails", harness.KindT3CodeV2, func(repo *fakeRepo, _ *fakeExchanger) { repo.saveErr = errBoom }, errBoom},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			svc, repo, _, v2 := twoKinds(t)
			c, err := svc.Pair(t.Context(), "u1", harness.KindT3Code, "Onik's laptop", "https://h.example.com", "tok")
			require.NoError(t, err)
			before := len(repo.outbox)
			tt.breakIt(repo, v2)

			err = svc.SwitchHarness(t.Context(), c.Session(), tt.to)
			require.ErrorIs(t, err, tt.wantIs)
			repo.saveErr = nil
			stored, err := repo.GetComputer(t.Context(), "u1", c.ID)
			require.NoError(t, err)
			assert.Equal(t, harness.KindT3Code, stored.Kind)
			assert.Equal(t, "0.0.45", stored.HarnessVersion)
			assert.Len(t, repo.outbox, before, "no event for a switch that did not happen")
		})
	}
}

func TestSwitchHarness_MovesTheComputerOnce_WithTheNewVersion(t *testing.T) {
	t.Parallel()
	svc, repo, _, _ := twoKinds(t)
	c, err := svc.Pair(t.Context(), "u1", harness.KindT3Code, "Onik's laptop", "https://h.example.com", "tok")
	require.NoError(t, err)
	var changed []string
	svc.changed = func(userID string) { changed = append(changed, userID) }

	require.NoError(t, svc.SwitchHarness(t.Context(), c.Session(), harness.KindT3CodeV2))
	require.NoError(t, svc.SwitchHarness(t.Context(), c.Session(), harness.KindT3CodeV2), "a computer already moved is a no-op")

	stored, err := repo.GetComputer(t.Context(), "u1", c.ID)
	require.NoError(t, err)
	assert.Equal(t, harness.KindT3CodeV2, stored.Kind)
	assert.Equal(t, "0.0.46-nightly.20261003.2632", stored.HarnessVersion, "re-read from the descriptor, so the next turn sees no drift")
	assert.Equal(t, []string{TopicComputerPaired, TopicHarnessSwitched}, topics(repo.outbox))
	assert.Equal(t, HarnessSwitchedEvent{
		ComputerID: c.ID, UserID: "u1", FromKind: harness.KindT3Code, ToKind: harness.KindT3CodeV2, HarnessVersion: "0.0.46-nightly.20261003.2632",
	}, repo.outbox[1].Payload)
	assert.Equal(t, []string{"u1"}, changed, "presence hears of the move once, so it holds the computer on its new kind")
}

func TestPair_KindOnlyMovesForward(t *testing.T) {
	t.Parallel()

	t.Run("a re-pair that lands on a lower kind is refused", func(t *testing.T) {
		t.Parallel()
		svc, repo, v1, v2 := twoKinds(t)
		v1.result.Kind = harness.KindT3CodeV2
		c, err := svc.Pair(t.Context(), "u1", harness.KindT3Code, "Onik's laptop", "https://h.example.com", "tok")
		require.NoError(t, err)
		require.Equal(t, harness.KindT3CodeV2, c.Kind)

		v2.result.Kind = harness.KindT3Code
		_, err = svc.Repair(t.Context(), "u1", c.ID, "Onik's laptop", "https://h.example.com", "tok2")
		require.ErrorIs(t, err, apperrs.ErrConflict)
		stored, err := repo.GetComputer(t.Context(), "u1", c.ID)
		require.NoError(t, err)
		assert.Equal(t, harness.KindT3CodeV2, stored.Kind)
	})

	t.Run("a fresh pairing stores the kind it landed on", func(t *testing.T) {
		t.Parallel()
		svc, repo, v1, _ := twoKinds(t)
		v1.result.Kind = harness.KindT3CodeV2
		c, err := svc.Pair(t.Context(), "u1", harness.KindT3Code, "Onik's laptop", "https://h.example.com", "tok")
		require.NoError(t, err)
		assert.Equal(t, harness.KindT3CodeV2, c.Kind)
		assert.Equal(t, []string{TopicComputerPaired}, topics(repo.outbox), "a new computer never switched")
	})

	t.Run("a re-pair that lands on a higher kind raises it and says so", func(t *testing.T) {
		t.Parallel()
		svc, repo, v1, _ := twoKinds(t)
		c, err := svc.Pair(t.Context(), "u1", harness.KindT3Code, "Onik's laptop", "https://h.example.com", "tok")
		require.NoError(t, err)

		v1.result.Kind, v1.version = harness.KindT3CodeV2, "0.0.46-nightly.20261003.2632"
		updated, err := svc.Repair(t.Context(), "u1", c.ID, "Onik's laptop", "https://h.example.com", "tok2")
		require.NoError(t, err)
		assert.Equal(t, harness.KindT3CodeV2, updated.Kind)
		assert.Equal(t, []string{TopicComputerPaired, TopicComputerPaired, TopicHarnessSwitched}, topics(repo.outbox))
		assert.Equal(t, HarnessSwitchedEvent{
			ComputerID: c.ID, UserID: "u1", FromKind: harness.KindT3Code, ToKind: harness.KindT3CodeV2, HarnessVersion: "0.0.46-nightly.20261003.2632",
		}, repo.outbox[2].Payload)
	})
}

func TestPair_HarnessItCannotFollow_ShowsItsOwnMessageUnderTheAddress(t *testing.T) {
	t.Parallel()
	exch := pairedExchanger()
	exch.exchangeErr = harness.ProtocolRefusal(wentBack)
	svc, _ := newTunnelService(newFakeRepo(), exch, &fakeTunnels{})
	id := tunnelComputer(t, svc)

	rec := doRequest(NewHandler(svc).Routes(), http.MethodPost, "/api/pairing/computers/"+id+"/pair", "u1", pairComputerRequest{Token: "tok"})
	require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
	var body struct {
		Message string              `json:"message"`
		Errors  map[string][]string `json:"errors"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, wentBack, body.Message, "no fresh t3 pair token can fix this, so none is asked for")
	assert.Equal(t, []string{wentBack}, body.Errors["server_url"])
}

func TestRequireSetup_HarnessMovedOnDuringTheCheck_TargetsWhatTheComputerHoldsNow(t *testing.T) {
	t.Parallel()
	f := newGateFixture(t)
	f.confirmOverall(t)
	f.confirmProvider(t, "codex")
	moved := &fakeExchanger{version: "0.0.46-nightly.20261003.2632"}
	moved.ListProvidersFn = f.exch.ListProvidersFn
	f.exch.ListProvidersFn = func(context.Context, harness.Session) ([]harness.Provider, error) {
		return nil, &harness.MovedError{To: harness.KindT3CodeV2}
	}
	f.svc.harnesses = harness.Registry{
		harness.KindT3Code:   harness.Forward(f.exch, moved, f.svc.SwitchHarness),
		harness.KindT3CodeV2: moved,
	}

	target, err := f.svc.ResolveTarget(t.Context(), "u1", "")
	require.NoError(t, err)
	assert.Equal(t, harness.KindT3CodeV2, target.Computer.Kind, "the turn runs on the new kind's client directly")
	assert.Equal(t, "0.0.46-nightly.20261003.2632", target.Computer.HarnessVersion,
		"the drift check compares the live version against this, so the first turn after the switch warns about nothing")
	assert.Equal(t, "codex-main", target.Provider, "setup confirmations survive the switch")
}
