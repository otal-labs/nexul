package pairing

import (
	"context"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/harness"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// gateFixture pairs "Onik's laptop" for u1 with a fallback project; the harness lists Codex first, so Codex is its default.
type gateFixture struct {
	repo     *fakeRepo
	exch     *fakeExchanger
	svc      *Service
	computer *Computer
}

func newGateFixture(t *testing.T) *gateFixture {
	t.Helper()
	repo := newFakeRepo()
	exch := &fakeExchanger{result: harness.PairResult{BearerToken: "secret", ExpiresIn: 30 * 24 * time.Hour}, version: "0.0.34"}
	exch.ListProvidersFn = func(context.Context, harness.Session) ([]harness.Provider, error) {
		return []harness.Provider{
			{ID: "codex-main", Driver: "codex", Name: "Codex"},
			{ID: "claude", Driver: "claudeAgent", Name: "Claude"},
		}, nil
	}
	svc := newTestService(repo, exch)
	c, err := svc.Pair(t.Context(), "u1", harness.KindT3Code, "Onik's laptop", "https://h.example.com", "tok")
	require.NoError(t, err)
	_, err = svc.SetDefaults(t.Context(), "u1", Defaults{DefaultComputerID: c.ID, FallbackProjectID: "default-proj"})
	require.NoError(t, err)
	_, err = svc.SetProjectLink(t.Context(), "u1", "proj-1", ProjectLink{ComputerID: c.ID, HarnessProjectID: "linked-proj"})
	require.NoError(t, err)
	return &gateFixture{repo: repo, exch: exch, svc: svc, computer: c}
}

func (f *gateFixture) confirmOverall(t *testing.T) {
	t.Helper()
	_, err := f.svc.ConfirmSetup(t.Context(), "u1", f.computer.ID)
	require.NoError(t, err)
}

func (f *gateFixture) confirmProvider(t *testing.T, driver string) {
	t.Helper()
	_, err := f.svc.ConfirmProviderSetup(t.Context(), "u1", f.computer.ID, driver, []string{"grilling"})
	require.NoError(t, err)
}

func requireSetupRefusal(t *testing.T, err error, provider string) {
	t.Helper()
	var nc *NotConfiguredError
	require.ErrorAs(t, err, &nc)
	assert.Equal(t, ReasonSetupRequired, nc.Reason)
	assert.Equal(t, provider, nc.Provider)
	assert.Equal(t, "Onik's laptop", nc.Computer)
	assert.NotEmpty(t, nc.ComputerID)
	assert.Equal(t, RefusalDetails{Reason: ReasonSetupRequired, ComputerID: nc.ComputerID, Computer: "Onik's laptop", ProviderID: "codex-main", Provider: provider}, nc.ErrorDetails())
	assert.ErrorIs(t, err, apperrs.ErrInvalid)
}

func TestResolveTarget_SetupGate_NeedsComputerAndProviderConfirmed(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		overall   bool
		providers []string
		refused   bool
	}{
		{"nothing confirmed", false, nil, true},
		{"provider confirmed but not the computer", false, []string{"codex"}, true},
		{"computer confirmed but not the provider", true, []string{"claudeAgent"}, true},
		{"both confirmed", true, []string{"codex"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := newGateFixture(t)
			if tt.overall {
				f.confirmOverall(t)
			}
			for _, driver := range tt.providers {
				f.confirmProvider(t, driver)
			}

			target, err := f.svc.ResolveTarget(t.Context(), "u1", "")
			if tt.refused {
				requireSetupRefusal(t, err, "Codex")
				assert.Nil(t, target)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, "codex-main", target.Provider, "an empty provider is pinned to the harness default it was checked as")
		})
	}
}

func TestResolveTargetOverride_StoredInstance_MapsToItsDriverKind(t *testing.T) {
	t.Parallel()
	f := newGateFixture(t)
	f.confirmOverall(t)
	f.confirmProvider(t, "claudeAgent")

	target, err := f.svc.ResolveTargetOverride(t.Context(), "u1", "", f.computer.ID, "claude", "")
	require.NoError(t, err)
	assert.Equal(t, "claude", target.Provider)

	_, err = f.svc.ResolveTargetOverride(t.Context(), "u1", "", f.computer.ID, "codex-main", "")
	requireSetupRefusal(t, err, "Codex")
}

func TestRequireSetup_HarnessAndStoreFailures_NeverPass(t *testing.T) {
	t.Parallel()

	t.Run("harness unreachable reads as offline, not unconfigured", func(t *testing.T) {
		t.Parallel()
		f := newGateFixture(t)
		f.exch.ListProvidersFn = func(context.Context, harness.Session) ([]harness.Provider, error) { return nil, errBoom }
		_, err := f.svc.ResolveTarget(t.Context(), "u1", "")
		var nc *NotConfiguredError
		require.ErrorAs(t, err, &nc)
		assert.Equal(t, ReasonOffline, nc.Reason)
		assert.Equal(t, "@Agent can't reach Onik's laptop — is T3 Code running there?", err.Error())
		assert.ErrorIs(t, err, apperrs.ErrInvalid, "the play press and MCP play run surface it as a bad request with this message")
		assert.ErrorIs(t, err, errBoom, "the harness failure stays in the chain for logs")
	})

	t.Run("harness lists no provider for an empty choice", func(t *testing.T) {
		t.Parallel()
		f := newGateFixture(t)
		f.exch.ListProvidersFn = func(context.Context, harness.Session) ([]harness.Provider, error) { return nil, nil }
		_, err := f.svc.ResolveTarget(t.Context(), "u1", "")
		require.ErrorIs(t, err, apperrs.ErrInvalid)
	})

	t.Run("stored provider no longer on the computer", func(t *testing.T) {
		t.Parallel()
		f := newGateFixture(t)
		_, err := f.svc.ResolveTargetOverride(t.Context(), "u1", "", f.computer.ID, "gone", "")
		require.ErrorIs(t, err, apperrs.ErrInvalid)
		assert.Contains(t, err.Error(), "gone")
	})

	t.Run("setup store fails", func(t *testing.T) {
		t.Parallel()
		f := newGateFixture(t)
		f.repo.listSetupErr = errBoom
		_, err := f.svc.ResolveTarget(t.Context(), "u1", "")
		require.ErrorIs(t, err, errBoom)
	})
}

// Chat, the play run dialog, and MCP play run all resolve through these two; no argument combination gets past the gate.
func TestResolveTargetOverride_NoRequestArgument_BypassesTheGate(t *testing.T) {
	t.Parallel()
	f := newGateFixture(t)
	f.confirmProvider(t, "claudeAgent")
	f.confirmProvider(t, "codex")

	for _, projectID := range []string{"", "proj-1", "unlinked"} {
		for _, computerID := range []string{"", f.computer.ID} {
			for _, provider := range []string{"", "codex-main", "claude", "CLAUDE", "claudeAgent", "codex", "setup"} {
				for _, model := range []string{"", "gpt"} {
					target, err := f.svc.ResolveTargetOverride(t.Context(), "u1", projectID, computerID, provider, model)
					require.Error(t, err, "project=%q computer=%q provider=%q model=%q", projectID, computerID, provider, model)
					assert.Nil(t, target)
				}
			}
		}
		_, err := f.svc.ResolveTarget(t.Context(), "u1", projectID)
		require.Error(t, err, "project=%q", projectID)
	}
}

func TestResolveSetupTurnTarget_UnconfirmedComputer_ResolvesWithoutTheGate(t *testing.T) {
	t.Parallel()
	f := newGateFixture(t)

	target, err := f.svc.ResolveSetupTurnTarget(t.Context(), "u1", f.computer.ID, "claude")
	require.NoError(t, err)
	assert.Equal(t, "claude", target.Provider)
	assert.Equal(t, "secret", target.Computer.BearerToken)

	_, err = f.svc.ResolveSetupTurnTarget(t.Context(), "u1", "  ", "claude")
	require.ErrorIs(t, err, apperrs.ErrInvalid, "a setup turn never falls back to the caller's defaults")

	_, err = f.svc.ResolveSetupTurnTarget(t.Context(), "u2", f.computer.ID, "claude")
	var nc *NotConfiguredError
	require.ErrorAs(t, err, &nc)
	assert.Equal(t, ReasonUnpaired, nc.Reason, "only the computer's owner runs its setup")
}

// wizardSetupTurnCallers lists the files allowed to call ResolveSetupTurnTarget: the setup wizard's own use-case.
var wizardSetupTurnCallers = map[string]bool{"../../internal/pairing/setup_turn.go": true}

func TestResolveSetupTurnTarget_OnlyTheWizardCallsIt(t *testing.T) {
	t.Parallel()
	root := filepath.Join("..", "..")
	var callers []string
	for _, dir := range []string{"internal", "server", "runner"} {
		err := filepath.WalkDir(filepath.Join(root, dir), func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return err
			}
			file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
			if err != nil {
				return err
			}
			ast.Inspect(file, func(n ast.Node) bool {
				sel, ok := n.(*ast.SelectorExpr)
				if ok && sel.Sel.Name == "ResolveSetupTurnTarget" {
					callers = append(callers, filepath.ToSlash(path))
				}
				return true
			})
			return nil
		})
		require.NoError(t, err)
	}
	for _, c := range callers {
		assert.True(t, wizardSetupTurnCallers[c], "%s calls the ungated setup-turn resolution; only the wizard's setup turn may", c)
	}
}

func TestPreviewTarget_UnconfirmedComputer_ResolvesWithoutTokenOrGate(t *testing.T) {
	t.Parallel()
	f := newGateFixture(t)
	target, err := f.svc.PreviewTarget(t.Context(), "u1", "proj-1")
	require.NoError(t, err)
	assert.Equal(t, "linked-proj", target.HarnessProjectID)
	assert.Empty(t, target.Computer.BearerToken)

	_, err = f.svc.PreviewTarget(t.Context(), "", "proj-1")
	require.ErrorIs(t, err, apperrs.ErrUnauthorized)
}

func TestListProviders_TagsProvidersThatNeedSetup(t *testing.T) {
	t.Parallel()
	f := newGateFixture(t)
	f.confirmProvider(t, "codex")

	needs := func() map[string]bool {
		providers, err := f.svc.ListProviders(t.Context(), "u1", f.computer.ID)
		require.NoError(t, err)
		out := map[string]bool{}
		for _, p := range providers {
			out[p.ID] = p.NeedsSetup
		}
		return out
	}
	assert.Equal(t, map[string]bool{"codex-main": true, "claude": true}, needs(), "nothing runs until the computer is confirmed too")

	f.confirmOverall(t)
	assert.Equal(t, map[string]bool{"codex-main": false, "claude": true}, needs())

	f.repo.listSetupErr = errBoom
	_, err := f.svc.ListProviders(t.Context(), "u1", f.computer.ID)
	require.ErrorIs(t, err, errBoom)
}
