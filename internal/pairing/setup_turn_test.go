package pairing

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/harness"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/redact"
)

type fakeInstance struct {
	url string
	err error
}

func (f fakeInstance) GetInstanceURL(context.Context) (string, error) { return f.url, f.err }

// setupBus records every ephemeral frame the setup turns publish.
type setupBus struct {
	mu     sync.Mutex
	frames []SetupTurnActivityEvent
}

func (b *setupBus) Publish(_ context.Context, topic string, payload any) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if topic == TopicSetupTurnActivity {
		b.frames = append(b.frames, payload.(SetupTurnActivityEvent))
	}
	return nil
}

// setupFixture pairs a computer listing Codex twice, then Claude; its fake harness plays the agent confirming through the use-cases.
type setupFixture struct {
	svc      *Service
	repo     *fakeRepo
	exch     *fakeExchanger
	tokens   *fakeTokens
	bus      *setupBus
	computer *Computer

	mu          sync.Mutex
	titles      []string
	prompts     []string
	fail        map[string]harness.Update
	skipConfirm map[string]bool
	interrupted int
}

var setupDrivers = map[string]string{"codex-main": "codex", "codex-2": "codex", "claude": "claudeAgent"}

func newSetupFixture(t *testing.T) *setupFixture {
	t.Helper()
	repo := newFakeRepo()
	exch := &fakeExchanger{result: harness.PairResult{BearerToken: "secret", ExpiresIn: 30 * 24 * time.Hour}}
	exch.ListProvidersFn = func(context.Context, harness.Session) ([]harness.Provider, error) {
		return []harness.Provider{
			{ID: "codex-main", Driver: "codex", Name: "Codex"},
			{ID: "codex-2", Driver: "codex", Name: "Codex (work)"},
			{ID: "claude", Driver: "claudeAgent", Name: "Claude"},
		}, nil
	}
	exch.ListProjectsFn = func(context.Context, harness.Session) ([]harness.Project, error) {
		return []harness.Project{{ID: "t3-home", Title: "Home"}}, nil
	}
	var clockMu sync.Mutex
	now := testNow
	f := &setupFixture{repo: repo, exch: exch, tokens: newFakeTokens(), bus: &setupBus{}, fail: map[string]harness.Update{}, skipConfirm: map[string]bool{}}
	f.svc = NewService(Config{
		Repo: repo, Harnesses: registry(exch), EncryptionKey: testEncKey, Tokens: f.tokens, Bus: f.bus,
		Instance: fakeInstance{url: "https://nexul.example.com/app"},
		Now: func() time.Time {
			clockMu.Lock()
			defer clockMu.Unlock()
			now = now.Add(time.Second)
			return now
		},
	})
	exch.StartTurnFn = f.startTurn
	exch.InterruptFn = func(context.Context, harness.Target) error {
		f.mu.Lock()
		defer f.mu.Unlock()
		f.interrupted++
		return nil
	}
	c, err := f.svc.Pair(t.Context(), "u1", harness.KindT3Code, "Laptop", "https://laptop.example.com", "tok")
	require.NoError(t, err)
	f.computer = c
	return f
}

// startTurn is the agent: it reports a step carrying the prompt, and a check session confirms through the use-cases.
func (f *setupFixture) startTurn(ctx context.Context, target harness.Target, title string, prompts harness.TurnPrompts) (harness.StartResult, error) {
	f.mu.Lock()
	f.titles = append(f.titles, title)
	f.prompts = append(f.prompts, prompts.Full)
	last, failing := f.fail[title]
	skip := f.skipConfirm[title]
	f.mu.Unlock()
	updates := make(chan harness.Update, 4)
	updates <- harness.Update{Activity: &harness.Activity{Kind: harness.ActivityToolCall, Tool: "bash", Summary: "writing the config", Detail: prompts.Full}}
	if !failing && !skip && strings.HasPrefix(title, "Nexul setup check") {
		driver := setupDrivers[target.Provider]
		if _, err := f.svc.ConfirmProviderSetup(ctx, "u1", f.computer.ID, driver, []string{"tdd", "nexul-memory"}); err != nil {
			return harness.StartResult{}, err
		}
		if strings.Contains(prompts.Full, "call `computer_setup_confirm` with") {
			if _, err := f.svc.ConfirmSetup(ctx, "u1", f.computer.ID); err != nil {
				return harness.StartResult{}, err
			}
		}
	}
	updates <- harness.Update{Snapshot: &harness.Snapshot{MessageID: "m1", Text: "All set."}}
	if !failing {
		last = harness.Update{Terminal: &harness.TurnResult{State: harness.TurnDone}}
	}
	updates <- last
	close(updates)
	return harness.StartResult{SessionID: "s-" + title, Updates: updates}, nil
}

func (f *setupFixture) sessionTitles() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.titles...)
}

// turns returns runID's saved turns keyed by provider.
func (f *setupFixture) turns(runID string) map[string]SetupTurn {
	f.repo.mu.Lock()
	defer f.repo.mu.Unlock()
	out := map[string]SetupTurn{}
	for _, turn := range f.repo.turns {
		if turn.RunID == runID {
			out[turn.Provider] = turn
		}
	}
	return out
}

func (f *setupFixture) outboxTopic(topic string) []eventbus.OutboxEvent {
	f.repo.mu.Lock()
	defer f.repo.mu.Unlock()
	var out []eventbus.OutboxEvent
	for _, e := range f.repo.outbox {
		if e.Topic == topic {
			out = append(out, e)
		}
	}
	return out
}

func (f *setupFixture) finished(t *testing.T, runID string) SetupFinishedEvent {
	t.Helper()
	for _, e := range f.outboxTopic(TopicSetupFinished) {
		if p := e.Payload.(SetupFinishedEvent); p.RunID == runID {
			return p
		}
	}
	require.FailNow(t, "no finished event for run "+runID)
	return SetupFinishedEvent{}
}

// resolveForPlay is a play's own gated resolution, once the user has picked a fallback project to run it in.
func (f *setupFixture) resolveForPlay(t *testing.T, provider string) error {
	t.Helper()
	_, err := f.svc.SetDefaults(t.Context(), "u1", Defaults{DefaultComputerID: f.computer.ID, FallbackProjectID: "t3-home"})
	require.NoError(t, err)
	_, err = f.svc.ResolveTargetOverride(t.Context(), "u1", "", f.computer.ID, provider, "")
	return err
}

func (f *setupFixture) start(t *testing.T) *SetupRun {
	t.Helper()
	run, err := f.svc.StartSetup(t.Context(), "u1", f.computer.ID)
	require.NoError(t, err)
	f.svc.setupRuns.Wait()
	return run
}

func TestStartSetup_FreshComputer_ConfirmsEveryProviderAndAPlayRuns(t *testing.T) {
	t.Parallel()
	f := newSetupFixture(t)

	run := f.start(t)

	assert.Equal(t, []SetupProvider{{Provider: "codex", Name: "Codex"}, {Provider: "claudeagent", Name: "Claude"}}, run.Providers, "one turn per driver, in the harness's order")
	assert.Equal(t, []string{"Nexul setup: Codex", "Nexul setup check: Codex", "Nexul setup: Claude", "Nexul setup check: Claude"}, f.sessionTitles())
	turns := f.turns(run.RunID)
	require.Len(t, turns, 2)
	for _, turn := range turns {
		assert.Equal(t, SetupTurnConfirmed, turn.State)
		assert.Equal(t, "Confirmed with 2 skills", turn.Status)
		assert.NotNil(t, turn.EndedAt)
	}
	finished := f.finished(t, run.RunID)
	assert.True(t, finished.Confirmed)
	assert.Equal(t, []SetupTurnOutcome{{Provider: "codex", State: SetupTurnConfirmed, Status: "Confirmed with 2 skills"}, {Provider: "claudeagent", State: SetupTurnConfirmed, Status: "Confirmed with 2 skills"}}, finished.Providers)
	assert.Len(t, f.outboxTopic(TopicSetupTurnChanged), 6, "running, checking, and the end state per provider")

	for _, provider := range []string{"codex-main", "claude"} {
		require.NoError(t, f.resolveForPlay(t, provider), "the gate lets %s run once setup confirmed it", provider)
	}
}

func TestStartSetup_Instructions_EveryConfirmSessionConfirmsTheComputer(t *testing.T) {
	t.Parallel()
	f := newSetupFixture(t)
	f.start(t)

	f.mu.Lock()
	prompts := append([]string(nil), f.prompts...)
	f.mu.Unlock()
	require.Len(t, prompts, 4)
	assert.Regexp(t, `Bearer dep_\d{43}`, prompts[0])
	assert.Contains(t, prompts[0], "https://nexul.example.com/mcp")
	assert.Contains(t, prompts[0], "[mcp_servers.nexul]")
	assert.Contains(t, prompts[2], "claude mcp add --scope user --transport http nexul")
	assert.Contains(t, prompts[0], "name: nexul-memory")
	assert.Contains(t, prompts[1], "call `computer_setup_confirm` with")
	assert.Contains(t, prompts[3], "call `computer_setup_confirm` with")
	assert.NotContains(t, prompts[0], "computer_setup_confirm", "the prepare session cannot reach Nexul's MCP server yet")
	assert.Contains(t, prompts[3], `provider "claudeAgent"`)
}

func TestStartSetup_Transcript_IsSavedAndStreamedWithoutTheToken(t *testing.T) {
	t.Parallel()
	f := newSetupFixture(t)
	run := f.start(t)

	for _, turn := range f.turns(run.RunID) {
		require.NotEmpty(t, turn.Transcript)
		for _, a := range turn.Transcript {
			assert.NotContains(t, a.Detail, "dep_0", "the prompt the step echoed carried the token")
		}
		assert.Contains(t, turn.Transcript[0].Detail, redact.Placeholder)
		assert.Equal(t, "All set.", turn.Transcript[len(turn.Transcript)-1].Detail, "the reply closes each session's transcript")
	}
	f.bus.mu.Lock()
	defer f.bus.mu.Unlock()
	require.NotEmpty(t, f.bus.frames)
	assert.Equal(t, "writing the config", f.bus.frames[0].Status)
	assert.Equal(t, run.RunID, f.bus.frames[0].RunID)
}

func TestStartSetup_OneProviderFails_OthersConfirmAndItRetriesAlone(t *testing.T) {
	t.Parallel()
	f := newSetupFixture(t)
	f.fail["Nexul setup: Codex"] = harness.Update{Terminal: &harness.TurnResult{State: harness.TurnError, LastError: "npx: command not found"}}

	run := f.start(t)

	turns := f.turns(run.RunID)
	assert.Equal(t, SetupTurnFailed, turns["codex"].State)
	assert.Equal(t, "npx: command not found", turns["codex"].Status)
	assert.Equal(t, SetupTurnConfirmed, turns["claudeagent"].State)
	assert.True(t, f.finished(t, run.RunID).Confirmed, "the confirmed provider confirmed the computer")
	err := f.resolveForPlay(t, "codex-main")
	var nc *NotConfiguredError
	require.ErrorAs(t, err, &nc)
	assert.Equal(t, ReasonSetupRequired, nc.Reason)

	delete(f.fail, "Nexul setup: Codex")
	before := len(f.sessionTitles())
	retry, err := f.svc.RetrySetupProvider(t.Context(), "u1", f.computer.ID, "Codex")
	require.NoError(t, err)
	f.svc.setupRuns.Wait()

	assert.Equal(t, []SetupProvider{{Provider: "codex", Name: "Codex"}}, retry.Providers)
	assert.Equal(t, []string{"Nexul setup: Codex", "Nexul setup check: Codex"}, f.sessionTitles()[before:], "only the failed provider runs again")
	assert.Equal(t, SetupTurnConfirmed, f.turns(retry.RunID)["codex"].State)
	assert.Empty(t, f.tokens.revoked, "the retry reuses the token the other providers already hold")
	require.NoError(t, f.resolveForPlay(t, "codex-main"))
}

func TestStartSetup_Rerun_ReverifiesWithoutReplacingTheToken(t *testing.T) {
	t.Parallel()
	f := newSetupFixture(t)
	f.start(t)
	first, err := f.svc.GetMCPToken(t.Context(), "u1", f.computer.ID)
	require.NoError(t, err)

	run := f.start(t)

	second, err := f.svc.GetMCPToken(t.Context(), "u1", f.computer.ID)
	require.NoError(t, err)
	assert.Equal(t, first.ID, second.ID, "the providers' configs keep a working token")
	assert.Empty(t, f.tokens.revoked)
	assert.True(t, f.finished(t, run.RunID).Confirmed)
	for _, provider := range []string{"codex-main", "claude"} {
		require.NoError(t, f.resolveForPlay(t, provider))
	}
}

func TestStartSetup_TokenRevokedSinceLastRun_MintsAFreshOne(t *testing.T) {
	t.Parallel()
	f := newSetupFixture(t)
	f.start(t)
	require.NoError(t, f.svc.RevokeMCPToken(t.Context(), "u1", f.computer.ID))

	f.start(t)

	token, err := f.svc.GetMCPToken(t.Context(), "u1", f.computer.ID)
	require.NoError(t, err)
	require.NotNil(t, token)
	assert.Equal(t, "pat-2", token.ID)
}

func TestStartSetup_StaleConfirmation_DoesNotCountForThisTurn(t *testing.T) {
	t.Parallel()
	f := newSetupFixture(t)
	f.start(t)
	f.skipConfirm["Nexul setup check: Codex"] = true

	run := f.start(t)

	codex := f.turns(run.RunID)["codex"]
	assert.Equal(t, SetupTurnFailed, codex.State, "a confirmation from an earlier run is not this turn's evidence")
	assert.Equal(t, "The turn ended without confirming Codex", codex.Status)
}

func TestStartSetup_SessionFailures_FailTheProvider(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		script      func(*setupFixture)
		status      string
		interrupted int
	}{
		{"a question stops the unattended turn", func(f *setupFixture) {
			f.fail["Nexul setup: Claude"] = harness.Update{Question: &harness.Question{RequestID: "q1"}}
		}, "The agent stopped to ask a question; setup runs unattended", 1},
		{"an interrupted turn without an error", func(f *setupFixture) {
			f.fail["Nexul setup check: Claude"] = harness.Update{Terminal: &harness.TurnResult{State: harness.TurnInterrupted}}
		}, "The setup turn ended interrupted", 0},
		{"the stream closes without a result", func(f *setupFixture) {
			f.fail["Nexul setup: Claude"] = harness.Update{Approval: &harness.Approval{Kind: "exec"}}
		}, "The harness closed the setup turn without a result", 0},
		{"the harness refuses to start", func(f *setupFixture) {
			start := f.exch.StartTurnFn
			f.exch.StartTurnFn = func(ctx context.Context, target harness.Target, title string, prompts harness.TurnPrompts) (harness.StartResult, error) {
				if target.Provider == "claude" {
					return harness.StartResult{}, errBoom
				}
				return start(ctx, target, title, prompts)
			}
		}, "Could not start the setup turn: boom", 0},
		{"no project to run in", func(f *setupFixture) {
			f.exch.ListProjectsFn = func(context.Context, harness.Session) ([]harness.Project, error) { return nil, nil }
		}, "open any project in T3 Code on Laptop first", 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := newSetupFixture(t)
			tt.script(f)
			run := f.start(t)
			claude := f.turns(run.RunID)["claudeagent"]
			assert.Equal(t, SetupTurnFailed, claude.State)
			assert.Contains(t, claude.Status, tt.status)
			codexConfirmed := f.turns(run.RunID)["codex"].State == SetupTurnConfirmed
			assert.Equal(t, codexConfirmed, f.finished(t, run.RunID).Confirmed, "any confirmed provider confirms the computer")
			if codexConfirmed {
				require.NoError(t, f.resolveForPlay(t, "codex-main"), "a failed provider never locks out a confirmed one")
			}
			f.mu.Lock()
			defer f.mu.Unlock()
			assert.Equal(t, tt.interrupted, f.interrupted)
		})
	}
}

func TestStartSetup_ErrorPaths(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		script  func(*setupFixture)
		call    func(context.Context, *setupFixture) error
		wantErr error
	}{
		{"another user's computer", nil, func(ctx context.Context, f *setupFixture) error {
			_, err := f.svc.StartSetup(ctx, "u2", f.computer.ID)
			return err
		}, apperrs.ErrNotFound},
		{"no instance url", func(f *setupFixture) { f.svc.instance = fakeInstance{} }, nil, apperrs.ErrInvalid},
		{"instance url unreadable", func(f *setupFixture) { f.svc.instance = fakeInstance{err: errBoom} }, nil, errBoom},
		{"instance url not wired", func(f *setupFixture) { f.svc.instance = nil }, nil, apperrs.ErrFatal},
		{"harness offline", func(f *setupFixture) {
			f.exch.ListProvidersFn = func(context.Context, harness.Session) ([]harness.Provider, error) { return nil, errBoom }
		}, nil, errBoom},
		{"harness lists no provider", func(f *setupFixture) {
			f.exch.ListProvidersFn = func(context.Context, harness.Session) ([]harness.Provider, error) { return nil, nil }
		}, nil, apperrs.ErrInvalid},
		{"retry a provider the harness does not list", nil, func(ctx context.Context, f *setupFixture) error {
			_, err := f.svc.RetrySetupProvider(ctx, "u1", f.computer.ID, "grok")
			return err
		}, apperrs.ErrInvalid},
		{"retry without a provider", nil, func(ctx context.Context, f *setupFixture) error {
			_, err := f.svc.RetrySetupProvider(ctx, "u1", f.computer.ID, " ")
			return err
		}, apperrs.ErrInvalid},
		{"token mint fails", func(f *setupFixture) { f.tokens.mintErr = errBoom }, nil, errBoom},
		{"token store fails", nil, func(ctx context.Context, f *setupFixture) error {
			_, err := f.svc.setupToken(ctx, Computer{ID: "missing", UserID: "u1", Name: "Ghost"})
			return err
		}, apperrs.ErrNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := newSetupFixture(t)
			if tt.script != nil {
				tt.script(f)
			}
			call := tt.call
			if call == nil {
				call = func(ctx context.Context, f *setupFixture) error {
					_, err := f.svc.StartSetup(ctx, "u1", f.computer.ID)
					return err
				}
			}
			require.ErrorIs(t, call(t.Context(), f), tt.wantErr)
			f.svc.setupRuns.Wait()
			assert.Empty(t, f.sessionTitles(), "a refused start runs no turn")
			assert.True(t, f.svc.claimSetup(f.computer.ID), "a refused start leaves the computer free")
		})
	}
}

func TestStartSetup_WhileRunning_IsAConflict(t *testing.T) {
	t.Parallel()
	f := newSetupFixture(t)
	release := make(chan struct{})
	start := f.exch.StartTurnFn
	f.exch.StartTurnFn = func(ctx context.Context, target harness.Target, title string, prompts harness.TurnPrompts) (harness.StartResult, error) {
		<-release
		return start(ctx, target, title, prompts)
	}
	_, err := f.svc.StartSetup(t.Context(), "u1", f.computer.ID)
	require.NoError(t, err)

	_, err = f.svc.RetrySetupProvider(t.Context(), "u1", f.computer.ID, "codex")
	require.ErrorIs(t, err, apperrs.ErrConflict)

	close(release)
	f.svc.setupRuns.Wait()
	_, err = f.svc.StartSetup(t.Context(), "u1", f.computer.ID)
	require.NoError(t, err, "a finished run frees the computer")
	f.svc.setupRuns.Wait()
}

func TestSetupTurn_SaveFailure_IsLoggedAndTheRunCarriesOn(t *testing.T) {
	t.Parallel()
	f := newSetupFixture(t)
	f.start(t)
	f.repo.mu.Lock()
	f.repo.saveErr = errBoom
	f.repo.mu.Unlock()

	_, err := f.svc.StartSetup(t.Context(), "u1", f.computer.ID)
	require.NoError(t, err)
	f.svc.setupRuns.Wait()
	assert.Len(t, f.sessionTitles(), 8, "every session still ran")
}

func TestSetupHandlers_StartRetryAndMemorySkill(t *testing.T) {
	t.Parallel()
	f := newSetupFixture(t)
	routes := NewHandler(f.svc).Routes()

	rec := doRequest(routes, http.MethodPost, "/api/pairing/computers/"+f.computer.ID+"/setup/runs", "u1", nil)
	require.Equal(t, http.StatusAccepted, rec.Code, rec.Body.String())
	assert.Contains(t, rec.Body.String(), `"run_id"`)
	f.svc.setupRuns.Wait()

	rec = doRequest(routes, http.MethodPost, "/api/pairing/computers/"+f.computer.ID+"/setup/providers/claudeAgent/retry", "u1", nil)
	require.Equal(t, http.StatusAccepted, rec.Code, rec.Body.String())
	assert.Contains(t, rec.Body.String(), `"provider":"claudeagent"`)
	f.svc.setupRuns.Wait()

	rec = doRequest(routes, http.MethodPost, "/api/pairing/computers/"+f.computer.ID+"/setup/runs", "u2", nil)
	assert.Equal(t, http.StatusNotFound, rec.Code, "only the owner starts a computer's setup")
	rec = doRequest(routes, http.MethodPost, "/api/pairing/computers/"+f.computer.ID+"/setup/providers/grok/retry", "u1", nil)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	rec = doRequest(routes, http.MethodGet, "/api/pairing/memory-skill", "u1", nil)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "name: nexul-memory")
	assert.Contains(t, rec.Body.String(), "memory_create")
}

func TestSetupMCPTools_StartAndRetry(t *testing.T) {
	t.Parallel()
	f := newSetupFixture(t)
	ctx := identity.WithActor(t.Context(), identity.Actor{ID: "u1"})
	tools := MCPTools(f.svc)

	out, err := toolNamed(t, tools, "computer_setup_start").Call(ctx, map[string]any{"computer_id": f.computer.ID})
	require.NoError(t, err)
	assert.Len(t, out.(*SetupRun).Providers, 2)
	f.svc.setupRuns.Wait()

	out, err = toolNamed(t, tools, "computer_setup_retry_provider").Call(ctx, map[string]any{"computer_id": f.computer.ID, "provider": "codex"})
	require.NoError(t, err)
	assert.Equal(t, "codex", out.(*SetupRun).Providers[0].Provider)
	f.svc.setupRuns.Wait()

	_, err = toolNamed(t, tools, "computer_setup_start").Call(ctx, map[string]any{})
	require.ErrorIs(t, err, apperrs.ErrInvalid)
	_, err = toolNamed(t, tools, "computer_setup_retry_provider").Call(ctx, map[string]any{"computer_id": f.computer.ID})
	require.ErrorIs(t, err, apperrs.ErrInvalid)
}

func TestResolveSetupTurnTarget_NoLinkedProject_UsesTheHarnessFirstProject(t *testing.T) {
	t.Parallel()
	f := newSetupFixture(t)

	target, err := f.svc.ResolveSetupTurnTarget(t.Context(), "u1", f.computer.ID, "claude")
	require.NoError(t, err)
	assert.Equal(t, "t3-home", target.HarnessProjectID)
	assert.Equal(t, "claude", target.Provider)
	assert.Equal(t, "secret", target.Computer.BearerToken)

	f.exch.ListProjectsFn = func(context.Context, harness.Session) ([]harness.Project, error) { return nil, errBoom }
	_, err = f.svc.ResolveSetupTurnTarget(t.Context(), "u1", f.computer.ID, "claude")
	require.ErrorIs(t, err, errBoom)
}

func TestSetupInstructions_PerDriver(t *testing.T) {
	t.Parallel()
	tests := []struct {
		driver string
		want   string
	}{
		{"claudeAgent", "claude mcp add --scope user --transport http nexul https://n.example.com/mcp --header \"Authorization: Bearer dep_x\""},
		{"codex", "[mcp_servers.nexul]"},
		{"opencode", "~/.config/opencode/opencode.json"},
		{"cursor", "~/.cursor/mcp.json"},
		{"grok", "Grok's own user-level MCP configuration"},
	}
	for _, tt := range tests {
		t.Run(tt.driver, func(t *testing.T) {
			t.Parallel()
			p := setupPrompt{ComputerID: "c1", ComputerName: "Laptop", Driver: tt.driver, ProviderName: "Grok", MCPURL: "https://n.example.com/mcp", Token: "dep_x"}
			prepare := prepareInstructions(p)
			assert.Contains(t, prepare, tt.want)
			assert.Contains(t, prepare, "~/.claude/skills/ and ~/.agents/skills/")
			assert.Contains(t, prepare, "npx -y skills@latest add mattpocock/skills")
			assert.Contains(t, prepare, "Never overwrite a skill directory that already exists")
			assert.True(t, strings.HasSuffix(prepare, memorySkill+"````\n"), "the nexul-memory file closes the instructions whole")
			assert.NotContains(t, confirmInstructions(p), "dep_x", "the confirm session never needs the token")
		})
	}
}

func TestMCPURLFor(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "https://n.example.com/mcp", mcpURLFor(" https://n.example.com/some/path "))
	assert.Equal(t, "http://localhost:8080/mcp", mcpURLFor("http://localhost:8080"))
	assert.Empty(t, mcpURLFor(""))
	assert.Empty(t, mcpURLFor("not a url"))
	assert.Empty(t, mcpURLFor("://bad"))
}

func TestSetupToken_UnreadableStoredCopy_MintsAFreshOne(t *testing.T) {
	t.Parallel()
	f := newSetupFixture(t)
	f.start(t)
	f.repo.mu.Lock()
	c := f.repo.computers[f.computer.ID]
	c.SetupMCPToken = "not-sealed"
	f.repo.computers[f.computer.ID] = c
	f.repo.mu.Unlock()

	f.start(t)

	assert.Equal(t, []string{"pat-1"}, f.tokens.revoked, "an unreadable copy is replaced, not fatal")
}

func TestGetSetup_Turns_ShowEachProviderNewestTurnOverHTTPAndMCP(t *testing.T) {
	t.Parallel()
	f := newSetupFixture(t)
	f.fail["Nexul setup: Codex"] = harness.Update{Terminal: &harness.TurnResult{State: harness.TurnError, LastError: "npx: command not found"}}
	first := f.start(t)
	delete(f.fail, "Nexul setup: Codex")
	retry, err := f.svc.RetrySetupProvider(t.Context(), "u1", f.computer.ID, "codex")
	require.NoError(t, err)
	f.svc.setupRuns.Wait()

	setup, err := f.svc.GetSetup(t.Context(), "u1", f.computer.ID)
	require.NoError(t, err)
	require.Len(t, setup.Turns, 2)
	assert.Equal(t, "claudeagent", setup.Turns[0].Provider)
	assert.Equal(t, first.RunID, setup.Turns[0].RunID, "a retry leaves the other provider's turn in view")
	assert.Equal(t, SetupTurnConfirmed, setup.Turns[0].State)
	assert.Equal(t, "codex", setup.Turns[1].Provider)
	assert.Equal(t, retry.RunID, setup.Turns[1].RunID)
	assert.Equal(t, SetupTurnConfirmed, setup.Turns[1].State)
	assert.Equal(t, "Confirmed with 2 skills", setup.Turns[1].Status)
	assert.False(t, setup.Turns[1].UpdatedAt.IsZero())

	routes := NewHandler(f.svc).Routes()
	rec := doRequest(routes, http.MethodGet, "/api/pairing/computers/"+f.computer.ID+"/setup", "u1", nil)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"run_id":"`+retry.RunID+`"`)
	assert.Contains(t, rec.Body.String(), `"updated_at":"`)
	assert.NotContains(t, rec.Body.String(), "Transcript", "the read never carries a transcript")
	rec = doRequest(routes, http.MethodGet, "/api/pairing/computers/"+f.computer.ID+"/setup", "u2", nil)
	assert.Equal(t, http.StatusNotFound, rec.Code, "only the owner reads a computer's setup turns")

	ctx := identity.WithActor(t.Context(), identity.Actor{ID: "u1"})
	out, err := toolNamed(t, MCPTools(f.svc), "computer_setup_get").Call(ctx, map[string]any{"computer_id": f.computer.ID})
	require.NoError(t, err)
	assert.Len(t, out.(Setup).Turns, 2)
	_, err = toolNamed(t, MCPTools(f.svc), "computer_setup_get").Call(identity.WithActor(t.Context(), identity.Actor{ID: "u2"}), map[string]any{"computer_id": f.computer.ID})
	require.ErrorIs(t, err, apperrs.ErrNotFound)

	f.repo.mu.Lock()
	f.repo.listTurnsErr = errBoom
	f.repo.mu.Unlock()
	_, err = f.svc.GetSetup(t.Context(), "u1", f.computer.ID)
	require.ErrorIs(t, err, errBoom)
}
