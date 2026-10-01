package pairing

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/harness"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	shipped "github.com/otal-labs/nexul/internal/platform/skills"
)

// outdate sets up every provider, then rolls their recorded skills back to an older release's version.
func (f *setupFixture) outdate(t *testing.T) {
	t.Helper()
	f.start(t)
	setups, err := f.repo.ListProviderSetups(t.Context(), f.computer.ID)
	require.NoError(t, err)
	for _, p := range setups {
		p.SkillsVersion = "0ld0ld0ld0ld"
		require.NoError(t, f.repo.SaveProviderSetup(t.Context(), f.computer.ID, p, testNow, eventbus.OutboxEvent{}))
	}
}

func (f *setupFixture) outdated(t *testing.T) map[string]bool {
	t.Helper()
	setup, err := f.svc.GetSetup(t.Context(), "u1", f.computer.ID)
	require.NoError(t, err)
	out := map[string]bool{}
	for _, p := range setup.Providers {
		out[p.Provider] = p.SkillsOutdated
	}
	return out
}

func (f *setupFixture) updateSkills(t *testing.T) *SetupRun {
	t.Helper()
	run, err := f.svc.UpdateSkills(t.Context(), "u1", f.computer.ID)
	require.NoError(t, err)
	f.svc.setupRuns.Wait()
	return run
}

func TestUpdateSkills_OneTurnMarksEveryConfirmedProviderCurrent(t *testing.T) {
	t.Parallel()
	f := newSetupFixture(t)
	f.outdate(t)
	before, err := f.svc.GetSetup(t.Context(), "u1", f.computer.ID)
	require.NoError(t, err)
	sessions := len(f.sessionTitles())

	run := f.updateSkills(t)

	assert.Equal(t, []SetupProvider{{Provider: "codex", Name: "Codex"}}, run.Providers, "the first confirmed provider in setup's order")
	assert.Equal(t, []string{"Nexul skills update: Codex"}, f.sessionTitles()[sessions:], "one session, no MCP reconnect and no per-provider check")
	f.mu.Lock()
	prompt := f.prompts[len(f.prompts)-1]
	f.mu.Unlock()
	assert.Contains(t, prompt, "`skill_get`")
	assert.Contains(t, prompt, shipped.NexulMemory.Paths()[1])
	assert.NotContains(t, prompt, "Authorization", "the update leaves the MCP config alone")
	assert.Equal(t, map[string]bool{"codex": false, "claudeagent": false}, f.outdated(t), "the shared folders are current for every provider")
	turn := f.turns(run.RunID)["codex"]
	assert.Equal(t, SetupTurnSkills, turn.Kind)
	assert.Equal(t, SetupTurnConfirmed, turn.State)
	finished := f.finished(t, run.RunID)
	assert.Equal(t, []SetupTurnOutcome{{Provider: "codex", State: SetupTurnConfirmed, Status: turn.Status}}, finished.Providers)
	after, err := f.svc.GetSetup(t.Context(), "u1", f.computer.ID)
	require.NoError(t, err)
	for i, p := range after.Providers {
		assert.Equal(t, before.Providers[i].ConfirmedAt, p.ConfirmedAt, "%s keeps its confirmation as it was", p.Provider)
	}
}

func TestUpdateSkills_FirstProviderUnconfirmed_RunsOnTheNextConfirmedOne(t *testing.T) {
	t.Parallel()
	f := newSetupFixture(t)
	f.outdate(t)
	_, err := f.svc.UnconfirmProviderSetup(t.Context(), "u1", f.computer.ID, "codex")
	require.NoError(t, err)

	run := f.updateSkills(t)

	assert.Equal(t, []SetupProvider{{Provider: "claudeagent", Name: "Claude"}}, run.Providers)
	setup, err := f.svc.GetSetup(t.Context(), "u1", f.computer.ID)
	require.NoError(t, err)
	for _, p := range setup.Providers {
		assert.False(t, p.SkillsOutdated, p.Provider)
		if p.Provider == "codex" {
			assert.Nil(t, p.ConfirmedAt, "an unconfirmed provider is not confirmed by a skills update")
		}
	}
}

func TestUpdateSkills_TurnFails_LeavesThemOutdated(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		script func(*setupFixture)
		status string
	}{
		{"the harness reports an error", func(f *setupFixture) {
			f.fail["Nexul skills update: Codex"] = harness.Update{Terminal: &harness.TurnResult{State: harness.TurnError, LastError: "permission denied"}}
		}, "permission denied"},
		{"the agent never reports the version", func(f *setupFixture) {
			f.skipConfirm["Nexul skills update: Codex"] = true
		}, "The turn ended without reporting the skills it wrote"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := newSetupFixture(t)
			f.outdate(t)
			tt.script(f)

			run := f.updateSkills(t)

			turn := f.turns(run.RunID)["codex"]
			assert.Equal(t, SetupTurnFailed, turn.State)
			assert.Equal(t, SetupTurnSkills, turn.Kind, "Retry knows to update again, not re-run setup")
			assert.Equal(t, tt.status, turn.Status)
			assert.Equal(t, map[string]bool{"codex": true, "claudeagent": true}, f.outdated(t))
			require.NoError(t, f.resolveForPlay(t, "codex-main"), "a failed update never withdraws a confirmation")
		})
	}
}

func TestUpdateSkills_WhileASetupOrUpdateRuns_IsAConflict(t *testing.T) {
	t.Parallel()
	f := newSetupFixture(t)
	f.outdate(t)
	release := make(chan struct{})
	start := f.exch.StartTurnFn
	f.exch.StartTurnFn = func(ctx context.Context, target harness.Target, title string, prompts harness.TurnPrompts) (harness.StartResult, error) {
		<-release
		return start(ctx, target, title, prompts)
	}
	_, err := f.svc.StartSetup(t.Context(), "u1", f.computer.ID, nil, nil, "", nil)
	require.NoError(t, err)

	_, err = f.svc.UpdateSkills(t.Context(), "u1", f.computer.ID)
	require.ErrorIs(t, err, apperrs.ErrConflict, "an update waits for a running setup")

	close(release)
	f.svc.setupRuns.Wait()
	f.outdate(t)
	release = make(chan struct{})
	_, err = f.svc.UpdateSkills(t.Context(), "u1", f.computer.ID)
	require.NoError(t, err)
	_, err = f.svc.UpdateSkills(t.Context(), "u1", f.computer.ID)
	require.ErrorIs(t, err, apperrs.ErrConflict, "a second update waits for the first")
	_, err = f.svc.StartSetup(t.Context(), "u1", f.computer.ID, nil, nil, "", nil)
	require.ErrorIs(t, err, apperrs.ErrConflict, "a setup waits for a running update")
	close(release)
	f.svc.setupRuns.Wait()
}

func TestUpdateSkills_ErrorPaths(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		script  func(*testing.T, *setupFixture)
		userID  string
		wantErr error
	}{
		{"another user's computer", func(t *testing.T, f *setupFixture) { f.outdate(t) }, "u2", apperrs.ErrNotFound},
		{"no provider confirmed yet", nil, "u1", apperrs.ErrInvalid},
		{"skills already current", func(t *testing.T, f *setupFixture) { f.start(t) }, "u1", apperrs.ErrInvalid},
		{"the confirmed providers are gone from the harness", func(t *testing.T, f *setupFixture) {
			f.outdate(t)
			f.exch.ListProvidersFn = func(context.Context, harness.Session) ([]harness.Provider, error) {
				return []harness.Provider{{ID: "cursor", Driver: "cursor", Name: "Cursor"}}, nil
			}
		}, "u1", apperrs.ErrInvalid},
		{"harness offline", func(t *testing.T, f *setupFixture) {
			f.outdate(t)
			f.exch.ListProvidersFn = func(context.Context, harness.Session) ([]harness.Provider, error) { return nil, errBoom }
		}, "u1", errBoom},
		{"setup rows unreadable", func(t *testing.T, f *setupFixture) {
			f.outdate(t)
			f.repo.listSetupErr = errBoom
		}, "u1", errBoom},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := newSetupFixture(t)
			if tt.script != nil {
				tt.script(t, f)
			}
			sessions := len(f.sessionTitles())

			_, err := f.svc.UpdateSkills(t.Context(), tt.userID, f.computer.ID)

			require.ErrorIs(t, err, tt.wantErr)
			f.svc.setupRuns.Wait()
			assert.Len(t, f.sessionTitles(), sessions, "a refused update runs no turn")
			assert.True(t, f.svc.claimSetup(f.computer.ID), "a refused update leaves the computer free")
		})
	}
}

func TestUpdateSkills_OverHTTP(t *testing.T) {
	t.Parallel()
	f := newSetupFixture(t)
	f.outdate(t)
	routes := NewHandler(f.svc).Routes()

	rec := doRequest(routes, http.MethodPost, "/api/pairing/computers/"+f.computer.ID+"/setup/skills", "u2", nil)
	assert.Equal(t, http.StatusNotFound, rec.Code, "only the owner updates a computer's skills")

	rec = doRequest(routes, http.MethodPost, "/api/pairing/computers/"+f.computer.ID+"/setup/skills", "u1", nil)
	require.Equal(t, http.StatusAccepted, rec.Code, rec.Body.String())
	assert.Contains(t, rec.Body.String(), `"provider":"codex"`)
	f.svc.setupRuns.Wait()
	rec = doRequest(routes, http.MethodGet, "/api/pairing/computers/"+f.computer.ID+"/setup", "u1", nil)
	assert.Contains(t, rec.Body.String(), `"kind":"skills"`, "the setup dialog tells an update's turn from a setup turn")
	assert.NotContains(t, rec.Body.String(), `"skills_outdated":true`)
}
