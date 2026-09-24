package plays

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/agent"
	"github.com/otal-labs/nexul/internal/harness"
	"github.com/otal-labs/nexul/internal/harness/harnesstest"
	"github.com/otal-labs/nexul/internal/pairing"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

const (
	starter    = "u-1"
	ticketID   = "t-1"
	docID      = "d-1"
	projectID  = "proj-1"
	fixPlayID  = "play-fix"
	docPlayID  = "play-doc"
	intPlayID  = "play-interview"
	alwaysMem  = "m-always"
	pickedMem  = "m-pick"
	otherMem   = "m-other"
	otherProj  = "proj-other"
	foreignWS  = "ws-2"
	foreignPrj = "proj-ws2"
)

type runnerFixture struct {
	runner  *Runner
	plays   *fakeRepo
	trails  *fakeTrailRepo
	perm    *fakePerm
	targets *fakeTargets
	harness *fakeHarnessResolver
	mems    *fakeMemories
	threads *fakeThreads
	turns   *fakeTurns
	mover   *fakeMover
	live    *fakeLive
	clock   time.Time
}

func newRunnerFixture() *runnerFixture {
	f := &runnerFixture{
		plays:   newFakeRepo(),
		trails:  newFakeTrailRepo(),
		perm:    newFakePerm(map[string][]permissions.Action{starter: {permissions.PlaysRead, permissions.PlaysRun, permissions.DocsThread}}),
		harness: &fakeHarnessResolver{},
		threads: &fakeThreads{},
		turns:   newFakeTurns(),
		mover:   &fakeMover{},
		live:    &fakeLive{},
		clock:   fixedNow,
	}
	f.targets = &fakeTargets{
		tickets: map[string]TicketTarget{
			ticketID:    {ProjectID: projectID, Key: "NEX-1", Stage: StageProgress},
			"t-backlog": {ProjectID: projectID, Key: "NEX-2", Stage: StageBacklog},
			"t-foreign": {ProjectID: foreignPrj, Key: "FOR-1", Stage: StageProgress},
		},
		docs: map[string]DocTarget{docID: {ProjectID: projectID, Title: "Roadmap"}},
		statuses: map[string]StatusTarget{
			"st-review": {Name: "In review", Stage: StageReview},
			"st-done":   {Name: "Done", Stage: StageDone},
		},
	}
	f.mems = &fakeMemories{byProject: map[string][]Memory{
		projectID: {
			{ID: pickedMem, Title: "Deploy quirks", Markdown: "picked body", AlwaysIncluded: false},
			{ID: alwaysMem, Title: "Working here", Markdown: "always body", AlwaysIncluded: true},
		},
		otherProj: {{ID: otherMem, Title: "Elsewhere", Markdown: "other"}},
	}}
	stage := StageProgress
	f.plays.byID[fixPlayID] = &Play{ID: fixPlayID, WorkspaceID: workspaceID, Label: "Fix with AI", Type: TypeTicket, Instructions: "Fix the ticket.", Enabled: true, ShowWhenStage: &stage}
	f.plays.byID[docPlayID] = &Play{ID: docPlayID, WorkspaceID: workspaceID, Label: "To tickets via AI", Type: TypeDoc, Instructions: "Split the doc.", Enabled: true}
	f.plays.byID[intPlayID] = &Play{ID: intPlayID, WorkspaceID: workspaceID, Label: "Interview", Type: TypeInterview, Instructions: "Interview them.", Enabled: true}
	f.runner = NewRunner(RunnerConfig{
		Plays: f.plays, Trails: f.trails, Perm: f.perm, Targets: f.targets,
		Projects: &fakeProjects{
			workspaces: map[string]string{projectID: workspaceID, otherProj: workspaceID, foreignPrj: foreignWS},
			projects:   map[string]ProjectTarget{projectID: {Name: "Nexul", TestsLocation: "separate"}, otherProj: {Name: "Other"}},
		},
		Harness: f.harness, Memories: f.mems, Threads: f.threads, Turns: f.turns, Tickets: f.mover, Live: f.live, Users: fakeUsers{},
		Now: func() time.Time { return f.clock },
	})
	return f
}

func ticketRun() RunInput {
	return RunInput{PlayID: fixPlayID, TargetType: TargetTicket, TargetID: ticketID, Via: ViaWeb}
}

func TestRun_NoActor_Unauthorized(t *testing.T) {
	f := newRunnerFixture()
	_, err := f.runner.Run(context.Background(), ticketRun())
	require.ErrorIs(t, err, apperrs.ErrUnauthorized)
}

func TestRun_Refusals_LeaveNoTrail(t *testing.T) {
	tests := []struct {
		name    string
		arrange func(f *runnerFixture)
		in      func() RunInput
		wantErr error
		wantMsg string
	}{
		{"bad target type", nil, func() RunInput { in := ticketRun(); in.TargetType = "column"; return in }, apperrs.ErrInvalid, "target type"},
		{"missing target id", nil, func() RunInput { in := ticketRun(); in.TargetID = " "; return in }, apperrs.ErrInvalid, "target id"},
		{"missing play id", nil, func() RunInput { in := ticketRun(); in.PlayID = ""; return in }, apperrs.ErrInvalid, "play id"},
		{"unknown play", nil, func() RunInput { in := ticketRun(); in.PlayID = "nope"; return in }, apperrs.ErrNotFound, ""},
		{"ticket play on a doc target", nil, func() RunInput { in := ticketRun(); in.TargetType, in.TargetID = TargetDoc, docID; return in }, apperrs.ErrInvalid, "ticket plays run on ticket targets"},
		{"unknown ticket", nil, func() RunInput { in := ticketRun(); in.TargetID = "t-missing"; return in }, apperrs.ErrNotFound, ""},
		{"play from another workspace", nil, func() RunInput { in := ticketRun(); in.TargetID = "t-foreign"; return in }, apperrs.ErrNotFound, ""},
		{"disabled", func(f *runnerFixture) { f.plays.byID[fixPlayID].Enabled = false }, ticketRun, apperrs.ErrInvalid, "is disabled"},
		{"excluded from the project", func(f *runnerFixture) { f.plays.byID[fixPlayID].ExcludedProjectIDs = []string{projectID} }, ticketRun, apperrs.ErrInvalid, "excluded from this project"},
		{"denied plays:run", func(f *runnerFixture) { f.perm.grants[starter] = []permissions.Action{permissions.PlaysRead} }, ticketRun, apperrs.ErrForbidden, "plays:run required"},
		{"denied on this play", func(f *runnerFixture) { f.perm.deny(starter, fixPlayID) }, ticketRun, apperrs.ErrForbidden, "plays:run required"},
		{"wrong stage", nil, func() RunInput { in := ticketRun(); in.TargetID = "t-backlog"; return in }, apperrs.ErrInvalid, "runs on tickets in the progress stage; this ticket is in backlog"},
		{"memory from another project", nil, func() RunInput { in := ticketRun(); in.MemoryIDs = []string{otherMem}; return in }, apperrs.ErrInvalid, "memory m-other is not in this project"},
		{"doc thread refused", func(f *runnerFixture) { f.threads.docErr = apperrs.ErrForbidden }, func() RunInput {
			return RunInput{PlayID: docPlayID, TargetType: TargetDoc, TargetID: docID}
		}, apperrs.ErrForbidden, ""},
		{"interview play on a ticket target", nil, func() RunInput { in := ticketRun(); in.PlayID = intPlayID; return in }, apperrs.ErrInvalid, "interview plays run on interview targets"},
		{"unknown interview project", nil, func() RunInput {
			return RunInput{PlayID: intPlayID, TargetType: TargetInterview, TargetID: "proj-missing"}
		}, apperrs.ErrNotFound, "get project proj-missing"},
		{"interview thread refused", func(f *runnerFixture) { f.threads.interviewErr = apperrs.ErrForbidden }, func() RunInput {
			return RunInput{PlayID: intPlayID, TargetType: TargetInterview, TargetID: projectID}
		}, apperrs.ErrForbidden, "open interview thread"},
		{"memories lookup failed", func(f *runnerFixture) { f.mems.err = errors.New("db down") }, ticketRun, nil, "db down"},
		{"trail list failed", func(f *runnerFixture) { f.trails.listErr = errors.New("db down") }, ticketRun, nil, "db down"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newRunnerFixture()
			if tt.arrange != nil {
				tt.arrange(f)
			}
			_, err := f.runner.Run(ctxAs(starter), tt.in())
			require.Error(t, err)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			}
			assert.Contains(t, err.Error(), tt.wantMsg)
			assert.Empty(t, f.trails.all(), "a refused run leaves no trail")
			assert.Empty(t, f.threads.snapshot(), "a refused run posts nothing")
		})
	}
}

func TestRun_SetupNotConfirmed_FailsOnPressWithTheChatRefusal(t *testing.T) {
	f := newRunnerFixture()
	f.harness.err = &pairing.NotConfiguredError{Reason: pairing.ReasonSetupRequired, Provider: "Codex", Computer: "Onik's laptop"}
	want := "@Agent can't use Codex on Onik's laptop until its setup is done — run setup for Onik's laptop in Settings → T3 pairing."

	_, err := f.runner.Run(ctxAs(starter), ticketRun())

	require.ErrorIs(t, err, apperrs.ErrInvalid, "the HTTP and MCP adapters surface it as a bad request carrying the message")
	assert.Equal(t, want, err.Error())
	trails := f.trails.all()
	require.Len(t, trails, 1)
	assert.Equal(t, TrailFailed, trails[0].State)
	assert.Equal(t, want, trails[0].LastError)
	assert.Empty(t, f.threads.snapshot(), "nothing reaches the harness or the thread")
}

func TestRun_HarnessRefusal_KeepsTheFixOnTheFailedTrail(t *testing.T) {
	setup := &pairing.NotConfiguredError{Reason: pairing.ReasonSetupRequired, Provider: "Codex", Computer: "mint", ComputerID: "c-mint", ProviderID: "codex"}
	tests := []struct {
		name         string
		err          error
		wantReason   string
		wantComputer string
		wantProvider string
	}{
		{"setup refusal", &HarnessRefusal{Reason: "setup_required", ComputerID: "c-mint", Provider: "codex", Err: setup}, "setup_required", "c-mint", "codex"},
		{"wrapped refusal", fmt.Errorf("resolve: %w", &HarnessRefusal{Reason: "unpaired", Err: errors.New("no computer")}), "unpaired", "", ""},
		{"plain error", errors.New("db down"), "", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newRunnerFixture()
			f.harness.err = tt.err

			_, err := f.runner.Run(ctxAs(starter), ticketRun())

			require.Error(t, err)
			trails := f.trails.all()
			require.Len(t, trails, 1)
			assert.Equal(t, TrailFailed, trails[0].State)
			assert.Equal(t, tt.err.Error(), trails[0].LastError)
			assert.Equal(t, tt.wantReason, trails[0].FailureReason)
			assert.Equal(t, tt.wantComputer, trails[0].ComputerID)
			assert.Equal(t, tt.wantProvider, trails[0].Provider)
		})
	}
}

func TestRun_HarnessOffline_FailsOnPressWithTheChatRefusal(t *testing.T) {
	f := newRunnerFixture()
	f.harness.err = &pairing.NotConfiguredError{Reason: pairing.ReasonOffline, Computer: "Onik's laptop"}

	_, err := f.runner.Run(ctxAs(starter), ticketRun())

	require.ErrorIs(t, err, apperrs.ErrInvalid)
	assert.Equal(t, "@Agent can't reach Onik's laptop — is T3 Code running there?", err.Error())
	trails := f.trails.all()
	require.Len(t, trails, 1)
	assert.Equal(t, TrailFailed, trails[0].State)
	assert.Equal(t, err.Error(), trails[0].LastError)
}

func TestRun_HarnessNotReady_ReturnsReasonAndLeavesFailedTrail(t *testing.T) {
	for _, reason := range []pairing.NotConfiguredReason{pairing.ReasonUnpaired, pairing.ReasonExpiredToken, pairing.ReasonNoDefault, pairing.ReasonNoDefaultComputer} {
		t.Run(string(reason), func(t *testing.T) {
			f := newRunnerFixture()
			f.harness.err = &pairing.NotConfiguredError{Reason: reason}
			in := ticketRun()
			in.MemoryIDs = []string{pickedMem}
			in.CustomInstructions = "careful"
			in.MoveToStatusID = "st-review"

			_, err := f.runner.Run(ctxAs(starter), in)

			var nc *pairing.NotConfiguredError
			require.ErrorAs(t, err, &nc)
			assert.Equal(t, reason, nc.Reason)
			trails := f.trails.all()
			require.Len(t, trails, 1)
			tr := trails[0]
			assert.Equal(t, TrailFailed, tr.State)
			assert.Equal(t, "pairing not configured: "+string(reason), tr.LastError)
			assert.Equal(t, []string{pickedMem}, tr.SelectedMemoryIDs)
			assert.Equal(t, "careful", tr.CustomInstructions)
			assert.Equal(t, "st-review", tr.MoveToStatusID)
			assert.Equal(t, "Fix with AI", tr.PlayLabel)
			assert.Equal(t, starter, tr.StarterID)
			require.NotNil(t, tr.EndedAt)
			assert.Empty(t, f.threads.snapshot(), "nothing is posted when the harness is not ready")
		})
	}
}

func TestRun_SecondRunOnActiveTarget_RefusedWithConflict(t *testing.T) {
	f := newRunnerFixture()
	first, err := f.runner.Run(ctxAs(starter), ticketRun())
	require.NoError(t, err)
	<-f.turns.done

	_, err = f.runner.Run(ctxAs(starter), RunInput{PlayID: fixPlayID, TargetType: TargetTicket, TargetID: ticketID})
	require.ErrorIs(t, err, apperrs.ErrConflict)
	assert.Equal(t, "a run is in progress", strings.TrimPrefix(err.Error(), apperrs.ErrConflict.Error()+": "))
	assert.Len(t, f.trails.all(), 1)
	assert.Equal(t, first.ID, f.trails.all()[0].ID)
}

func TestRun_FinishedTrail_DoesNotBlockTheNextRun(t *testing.T) {
	f := newRunnerFixture()
	old := &Trail{ID: "old", WorkspaceID: workspaceID, TargetType: TargetTicket, TargetID: ticketID, State: TrailDone, StartedAt: fixedNow.Add(-time.Hour)}
	require.NoError(t, f.trails.CreateTrail(context.Background(), old))
	<-f.trails.terminal

	_, err := f.runner.Run(ctxAs(starter), ticketRun())
	require.NoError(t, err)
}

func TestRun_OverCeiling_RefusedWithTotals(t *testing.T) {
	f := newRunnerFixture()
	f.mems.byProject[projectID] = append(f.mems.byProject[projectID], Memory{ID: "m-huge", Title: "Huge", Markdown: strings.Repeat("x", agent.MaxMemoryChars+1)})
	in := ticketRun()
	in.MemoryIDs = []string{"m-huge"}

	_, err := f.runner.Run(ctxAs(starter), in)

	var oc *agent.OverCeilingError
	require.ErrorAs(t, err, &oc)
	assert.ErrorIs(t, err, apperrs.ErrInvalid)
	assert.Equal(t, 2, oc.Totals.Memories, "the always-included memory counts too")
	assert.Equal(t, "Huge", oc.Totals.LargestTitle)
	assert.Equal(t, agent.MaxMemoryChars+1, oc.Totals.LargestChars)
	assert.Empty(t, f.trails.all())
}

func TestRun_PostStartedMessageFails_TrailFailed(t *testing.T) {
	f := newRunnerFixture()
	f.threads.postErr = errors.New("chat down")

	_, err := f.runner.Run(ctxAs(starter), ticketRun())

	require.Error(t, err)
	trails := f.trails.all()
	require.Len(t, trails, 1)
	assert.Equal(t, TrailFailed, trails[0].State)
	assert.Contains(t, trails[0].LastError, "chat down")
}

func TestRun_CreateTrailFails_ReturnsError(t *testing.T) {
	f := newRunnerFixture()
	f.trails.createErr = errors.New("disk full")
	_, err := f.runner.Run(ctxAs(starter), ticketRun())
	require.ErrorContains(t, err, "disk full")
	assert.Empty(t, f.threads.snapshot())
}

func TestRun_StartsTurnWithBlocksInOrder(t *testing.T) {
	f := newRunnerFixture()
	in := ticketRun()
	in.MemoryIDs = []string{pickedMem}
	in.CustomInstructions = "Touch only the docs."
	in.MoveToStatusID = "st-review"

	trail, err := f.runner.Run(ctxAs(starter), in)
	require.NoError(t, err)
	<-f.turns.done

	assert.Equal(t, TrailStarting, trail.State)
	assert.Equal(t, "conv-ticket-"+ticketID, trail.ConversationID)
	assert.Equal(t, []string{alwaysMem, pickedMem}, trail.SelectedMemoryIDs, "always-included first, then the selection")
	assert.Equal(t, ViaWeb, trail.Via)
	assert.Equal(t, projectID, trail.ProjectID)

	posts := f.threads.snapshot()
	require.Len(t, posts, 1)
	assert.Equal(t, fakePost{"conv-ticket-" + ticketID, starter, "Started Fix with AI\n\nTouch only the docs."}, posts[0])
	assert.NotContains(t, posts[0].body, "@Agent")

	req := f.turns.last()
	assert.Equal(t, trail.ConversationID, req.ConversationID)
	assert.Equal(t, starter, req.ViaUserID)
	assert.Equal(t, posts[0].body, req.RequestBody)
	require.Len(t, req.ExtraRequestBlocks, 3)
	assert.Equal(t, "Play: Fix with AI\nFix the ticket.", req.ExtraRequestBlocks[0])
	assert.Equal(t, "Memories the user selected for this run, follow them:\n### Deploy quirks\npicked body", req.ExtraRequestBlocks[1])
	assert.Equal(t, "Instructions from login-u-1 for this run; where these conflict with the play's instructions, these win:\nTouch only the docs.", req.ExtraRequestBlocks[2])
	require.NotNil(t, req.Observer)
}

func TestRun_WorkspaceScopedMemory_PickableEvenThoughItIsNotTheProjectsOwn(t *testing.T) {
	// A workspace-scoped memory shows up under every project ListForProject is asked about (ADR 0059); the
	// fake mirrors that by listing it under both projectID and otherProj, standing in for the union the real
	// memories.Service.ListForProject returns.
	f := newRunnerFixture()
	wsMem := "m-workspace"
	f.mems.byProject[projectID] = append(f.mems.byProject[projectID], Memory{ID: wsMem, Title: "Team tone", Markdown: "be terse"})
	f.mems.byProject[otherProj] = append(f.mems.byProject[otherProj], Memory{ID: wsMem, Title: "Team tone", Markdown: "be terse"})
	in := ticketRun()
	in.MemoryIDs = []string{wsMem}

	trail, err := f.runner.Run(ctxAs(starter), in)
	require.NoError(t, err)
	assert.Contains(t, trail.SelectedMemoryIDs, wsMem)
}

func TestRun_HarnessChoice_PassedToTheResolverAndRecordedOnTheTrail(t *testing.T) {
	f := newRunnerFixture()
	f.harness.resolved = HarnessChoice{ComputerID: "c-resolved", Provider: "claude", Model: "sonnet-5"}
	in := ticketRun()
	in.ComputerID, in.Provider, in.Model = "c-picked", "opencode", "gpt"

	trail, err := f.runner.Run(ctxAs(starter), in)
	require.NoError(t, err)
	<-f.turns.done

	assert.Equal(t, HarnessChoice{ComputerID: "c-picked", Provider: "opencode", Model: "gpt"}, f.harness.lastChoice,
		"the caller's pick reaches the resolver")
	assert.Equal(t, "c-resolved", trail.ComputerID, "the trail records what the resolver actually picked")
	assert.Equal(t, "claude", trail.Provider)
	assert.Equal(t, "sonnet-5", trail.Model)

	req := f.turns.last()
	require.NotNil(t, req.Target)
	assert.Equal(t, &agent.TargetOverride{ComputerID: "c-resolved", Provider: "claude", Model: "sonnet-5"}, req.Target,
		"the turn resolves against exactly what the runner already checked, not re-resolving on its own")
}

func TestRun_NoHarnessChoice_RecordsWhateverTheResolverPicked(t *testing.T) {
	f := newRunnerFixture()
	f.harness.resolved = HarnessChoice{ComputerID: "c-default", Provider: "claude", Model: "sonnet-5"}

	trail, err := f.runner.Run(ctxAs(starter), ticketRun())
	require.NoError(t, err)
	<-f.turns.done

	assert.Equal(t, HarnessChoice{}, f.harness.lastChoice, "no override means an empty choice reaches the resolver")
	assert.Equal(t, "c-default", trail.ComputerID)
	assert.Equal(t, "claude", trail.Provider)
	assert.Equal(t, "sonnet-5", trail.Model)
}

func TestRun_HarnessChoice_ComputerNotOwnedByCaller_FailsTrail(t *testing.T) {
	f := newRunnerFixture()
	f.harness.err = fmt.Errorf("%w: computer c-theirs is not yours", apperrs.ErrForbidden)
	in := ticketRun()
	in.ComputerID = "c-theirs"

	_, err := f.runner.Run(ctxAs(starter), in)

	require.ErrorIs(t, err, apperrs.ErrForbidden)
	trails := f.trails.all()
	require.Len(t, trails, 1)
	assert.Equal(t, TrailFailed, trails[0].State)
	assert.Empty(t, trails[0].ComputerID, "a refused choice leaves nothing to record")
}

func TestRun_NoMemoriesNoCustom_OmitsEmptyBlocks(t *testing.T) {
	f := newRunnerFixture()
	f.mems.byProject[projectID] = nil
	_, err := f.runner.Run(ctxAs(starter), ticketRun())
	require.NoError(t, err)
	<-f.turns.done

	req := f.turns.last()
	assert.Equal(t, []string{"Play: Fix with AI\nFix the ticket."}, req.ExtraRequestBlocks)
	assert.Equal(t, "Started Fix with AI", req.RequestBody)
}

func TestRun_DocPlay_UsesDocThread(t *testing.T) {
	f := newRunnerFixture()
	trail, err := f.runner.Run(ctxAs(starter), RunInput{PlayID: docPlayID, TargetType: TargetDoc, TargetID: docID, Via: ViaMCP})
	require.NoError(t, err)
	<-f.turns.done
	assert.Equal(t, "conv-doc-"+docID, trail.ConversationID)
	assert.Equal(t, ViaMCP, trail.Via)
	assert.Equal(t, TargetDoc, trail.TargetType)
}

func TestRun_InterviewPlay_PostsInTheInterviewThreadWithTheProjectsAnswers(t *testing.T) {
	tests := []struct {
		name, project, testsLocation, want string
	}{
		{"separate tests repository", projectID, "separate", "in a separate tests repository"},
		{"same repository", projectID, "same", "in the deployed repository"},
		{"not answered", otherProj, "", "not answered yet; ask it"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newRunnerFixture()
			projects := f.runner.projects.(*fakeProjects)
			p := projects.projects[tt.project]
			p.TestsLocation = tt.testsLocation
			projects.projects[tt.project] = p
			trail, err := f.runner.Run(ctxAs(starter), RunInput{PlayID: intPlayID, TargetType: TargetInterview, TargetID: tt.project, Via: ViaWeb})
			require.NoError(t, err)
			<-f.turns.done
			assert.Equal(t, "conv-interview-"+tt.project, trail.ConversationID)
			assert.Equal(t, TargetInterview, trail.TargetType)
			assert.Equal(t, tt.project, trail.ProjectID)
			blocks := strings.Join(f.turns.last().ExtraRequestBlocks, "\n")
			assert.Contains(t, blocks, "Play: Interview\nInterview them.")
			assert.Contains(t, blocks, fmt.Sprintf("(project id %s)", tt.project))
			assert.Contains(t, blocks, "Where its tests live, as answered in the project wizard: "+tt.want+".")
		})
	}
}

// newHarnessRunner swaps the fake turn runner for the real pipeline over harnesstest.Client.
func newHarnessRunner(f *runnerFixture, client *harnesstest.Client) *agentConvs {
	convs := &agentConvs{}
	svc := agent.NewService(agent.Config{Conversations: convs, Targets: agentTargets{}, Harnesses: harnesstest.Registry(client), Live: agentLive{}})
	f.runner.turns = svc
	return convs
}

func TestRun_AgainstHarness_TrailFollowsTheTurn(t *testing.T) {
	f := newRunnerFixture()
	var full, incremental string
	client := &harnesstest.Client{StartTurnFn: func(_ context.Context, _ harness.Target, _ string, prompts harness.TurnPrompts) (harness.StartResult, error) {
		full, incremental = prompts.Full, prompts.Incremental
		ch := make(chan harness.Update, 4)
		read := step("Read", "Read main.go")
		read.CallID = "call-1"
		ch <- harness.Update{Activity: &read}
		bash := step("Bash", "Bash go test")
		ch <- harness.Update{Activity: &bash}
		ch <- harness.Update{Snapshot: &harness.Snapshot{MessageID: "m-1", Text: "Opened PR #7", Streaming: false}}
		ch <- harness.Update{Terminal: &harness.TurnResult{State: harness.TurnDone}}
		close(ch)
		return harness.StartResult{SessionID: "sess-9", Updates: ch}, nil
	}}
	convs := newHarnessRunner(f, client)
	in := ticketRun()
	in.MemoryIDs = []string{pickedMem}
	in.CustomInstructions = "Touch only the docs."

	trail, err := f.runner.Run(ctxAs(starter), in)
	require.NoError(t, err)
	assert.Equal(t, TrailStarting, trail.State)

	final := <-f.trails.terminal
	assert.Equal(t, TrailDone, final.State)
	assert.Equal(t, "sess-9", final.HarnessSessionID)
	assert.Equal(t, []string{"Read main.go", "Bash go test", "Opened PR #7"}, summaries(final.Activity), "the closed reply is the last step")
	assert.Equal(t, ActivityEntry{Kind: harness.ActivityToolCall, CallID: "call-1", Tool: "Read", Summary: "Read main.go", At: time.Unix(1_800_000_000, 0).UTC()}, final.Activity[0], "the step is stored whole")
	assert.Equal(t, harness.ActivityText, final.Activity[2].Kind)
	assert.Equal(t, "reply-1", final.ReplyMessageID)
	assert.Empty(t, final.LastError)
	require.NotNil(t, final.EndedAt)
	assert.Equal(t, []string{"Opened PR #7"}, convs.replies)
	assert.Equal(t, []TrailState{TrailStarting, TrailRunning, TrailRunning, TrailRunning, TrailRunning, TrailDone}, f.trails.recordedStates())

	for name, prompt := range map[string]string{"full": full, "incremental": incremental} {
		play := strings.Index(prompt, "Play: Fix with AI\nFix the ticket.")
		mems := strings.Index(prompt, "Memories the user selected for this run, follow them:\n### Deploy quirks\npicked body")
		custom := strings.Index(prompt, "Instructions from login-u-1 for this run; where these conflict with the play's instructions, these win:\nTouch only the docs.")
		request := strings.Index(prompt, "Started Fix with AI")
		assert.True(t, request >= 0 && request < play && play < mems && mems < custom, "%s prompt orders request, play, memories, custom: %d %d %d %d", name, request, play, mems, custom)
	}
}

func TestRun_AgainstHarness_StartTurnError_TrailFailedWithReason(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		f := newRunnerFixture()
		client := &harnesstest.Client{StartTurnFn: func(context.Context, harness.Target, string, harness.TurnPrompts) (harness.StartResult, error) {
			return harness.StartResult{}, errors.New("harness refused the session")
		}}
		newHarnessRunner(f, client)

		_, err := f.runner.Run(ctxAs(starter), ticketRun())
		require.NoError(t, err, "the refusal arrives after the trail is returned")

		final := <-f.trails.terminal
		synctest.Wait()
		assert.Equal(t, TrailFailed, final.State)
		assert.Equal(t, "harness refused the session", final.LastError)
		assert.Empty(t, final.HarnessSessionID)
		assert.Equal(t, []TrailState{TrailStarting, TrailFailed}, f.trails.recordedStates())
		f.runner.mu.Lock()
		assert.Empty(t, f.runner.runs, "a finished run releases its live entry")
		f.runner.mu.Unlock()
	})
}

func TestRun_AgainstHarness_Interrupted(t *testing.T) {
	f := newRunnerFixture()
	client := &harnesstest.Client{StartTurnFn: func(context.Context, harness.Target, string, harness.TurnPrompts) (harness.StartResult, error) {
		ch := make(chan harness.Update, 1)
		ch <- harness.Update{Terminal: &harness.TurnResult{State: harness.TurnInterrupted}}
		close(ch)
		return harness.StartResult{SessionID: "sess-1", Updates: ch}, nil
	}}
	newHarnessRunner(f, client)

	_, err := f.runner.Run(ctxAs(starter), ticketRun())
	require.NoError(t, err)
	final := <-f.trails.terminal
	assert.Equal(t, TrailInterrupted, final.State)
	assert.Empty(t, final.ReplyMessageID)
}

func doneUpdates() chan harness.Update {
	ch := make(chan harness.Update, 1)
	ch <- harness.Update{Terminal: &harness.TurnResult{State: harness.TurnDone}}
	close(ch)
	return ch
}

func TestRun_MemoryEmbeddedImage_ProducesOneAttachment(t *testing.T) {
	f := newRunnerFixture()
	f.mems.byProject[projectID] = append(f.mems.byProject[projectID],
		Memory{ID: "m-img", Title: "Screenshot memory", Markdown: "see ![shot](/api/attachments/att-1)"})
	f.runner.attachments = &fakeAttachmentReader{items: map[string]agent.StoredAttachment{
		"att-1": {Name: "shot.png", MIME: "image/png", Bytes: []byte{1, 2, 3}},
	}}
	var got harness.TurnPrompts
	client := &harnesstest.Client{StartTurnFn: func(_ context.Context, _ harness.Target, _ string, prompts harness.TurnPrompts) (harness.StartResult, error) {
		got = prompts
		return harness.StartResult{SessionID: "sess-1", Updates: doneUpdates()}, nil
	}}
	newHarnessRunner(f, client)
	in := ticketRun()
	in.MemoryIDs = []string{"m-img"}

	_, err := f.runner.Run(ctxAs(starter), in)
	require.NoError(t, err)
	<-f.trails.terminal

	require.Len(t, got.Attachments, 1)
	assert.Equal(t, "shot.png", got.Attachments[0].Name)
	assert.Equal(t, "image/png", got.Attachments[0].MIME)
	assert.Contains(t, got.Full, "### Screenshot memory\nsee [image: shot.png, attached to this turn]")
}

func TestRun_MemoryOversizedOrNonImage_OmittedWithNoAttachment(t *testing.T) {
	f := newRunnerFixture()
	oversized := make([]byte, agent.MaxAttachmentBytes+1)
	f.mems.byProject[projectID] = append(f.mems.byProject[projectID],
		Memory{ID: "m-big", Title: "Big screenshot", Markdown: "![big](/api/attachments/att-big)"},
		Memory{ID: "m-pdf", Title: "Spec", Markdown: "[spec](/api/attachments/att-pdf)"},
	)
	f.runner.attachments = &fakeAttachmentReader{items: map[string]agent.StoredAttachment{
		"att-big": {Name: "huge.png", MIME: "image/png", Bytes: oversized},
		"att-pdf": {Name: "spec.pdf", MIME: "application/pdf", Bytes: []byte{1, 2, 3}},
	}}
	var got harness.TurnPrompts
	client := &harnesstest.Client{StartTurnFn: func(_ context.Context, _ harness.Target, _ string, prompts harness.TurnPrompts) (harness.StartResult, error) {
		got = prompts
		return harness.StartResult{SessionID: "sess-2", Updates: doneUpdates()}, nil
	}}
	newHarnessRunner(f, client)
	in := ticketRun()
	in.MemoryIDs = []string{"m-big", "m-pdf"}

	_, err := f.runner.Run(ctxAs(starter), in)
	require.NoError(t, err)
	<-f.trails.terminal

	assert.Empty(t, got.Attachments)
	assert.Contains(t, got.Full, "[attachment omitted: huge.png]")
	assert.Contains(t, got.Full, "[attachment omitted: spec.pdf]")
}

func TestTrail_AppendActivity_CapsAtMaxDroppingOldest(t *testing.T) {
	tr := &Trail{}
	for i := 0; i < MaxTrailActivityLines+5; i++ {
		tr.AppendActivity(ActivityEntry{Kind: harness.ActivityOther, Summary: "x" + string(rune('a'+i%26))})
	}
	assert.Len(t, tr.Activity, MaxTrailActivityLines)
	assert.Equal(t, "x"+string(rune('a'+5%26)), tr.Activity[0].Summary, "the oldest five steps are gone")
}

func TestTrail_AppendActivity_SameCallID_ReplacesTheStepInPlace(t *testing.T) {
	tr := &Trail{}
	tr.AppendActivity(ActivityEntry{Kind: harness.ActivityToolCall, CallID: "c-1", Tool: "Bash", Summary: "started"})
	tr.AppendActivity(ActivityEntry{Kind: harness.ActivityToolCall, CallID: "c-2", Tool: "Read", Summary: "main.go"})
	tr.AppendActivity(ActivityEntry{Kind: harness.ActivityToolResult, CallID: "c-1", Tool: "Bash", Summary: "go test", Detail: `{"result":"ok"}`})
	tr.AppendActivity(ActivityEntry{Kind: harness.ActivityText, Summary: "done"})
	tr.AppendActivity(ActivityEntry{Kind: harness.ActivityText, Summary: "and more"})

	assert.Equal(t, []string{"go test", "main.go", "done", "and more"}, summaries(tr.Activity))
	assert.Equal(t, harness.ActivityToolResult, tr.Activity[0].Kind, "the result replaces the call it belongs to")
	assert.Equal(t, `{"result":"ok"}`, tr.Activity[0].Detail)
}

func TestDecodeActivity_ReadsEntriesAndLegacyLines(t *testing.T) {
	at := time.Date(2026, 9, 18, 10, 0, 0, 0, time.UTC)
	entries := []ActivityEntry{{Kind: harness.ActivityToolCall, CallID: "c-1", Tool: "Read", Summary: "main.go", Detail: `{"input":{}}`, At: at}}
	raw, err := json.Marshal(entries)
	require.NoError(t, err)
	got, err := DecodeActivity(raw)
	require.NoError(t, err)
	assert.Equal(t, entries, got)

	legacy, err := DecodeActivity([]byte(`["Read main.go","Bash go test"]`))
	require.NoError(t, err)
	assert.Equal(t, []ActivityEntry{
		{Kind: harness.ActivityOther, Summary: "Read main.go"},
		{Kind: harness.ActivityOther, Summary: "Bash go test"},
	}, legacy)

	empty, err := DecodeActivity([]byte(`[]`))
	require.NoError(t, err)
	assert.Empty(t, empty)

	_, err = DecodeActivity([]byte(`{"not":"a list"}`))
	require.Error(t, err)
}

func seededTrail(f *runnerFixture, id string, targetType TargetType, targetID string, startedAt time.Time) *Trail {
	tr := &Trail{ID: id, WorkspaceID: workspaceID, PlayID: fixPlayID, TargetType: targetType, TargetID: targetID, ProjectID: projectID,
		StarterID: starter, State: TrailDone, StartedAt: startedAt, SelectedMemoryIDs: []string{alwaysMem}, MoveToStatusID: "st-" + id}
	_ = f.trails.CreateTrail(context.Background(), tr)
	<-f.trails.terminal
	return tr
}

func TestGetTrail(t *testing.T) {
	f := newRunnerFixture()
	seededTrail(f, "tr-ticket", TargetTicket, ticketID, fixedNow)
	seededTrail(f, "tr-doc", TargetDoc, docID, fixedNow)

	got, err := f.runner.GetTrail(ctxAs(starter), "tr-ticket")
	require.NoError(t, err)
	assert.Equal(t, "tr-ticket", got.ID)

	_, err = f.runner.GetTrail(ctxAs(starter), "  ")
	assert.ErrorIs(t, err, apperrs.ErrInvalid)
	_, err = f.runner.GetTrail(ctxAs(starter), "missing")
	assert.ErrorIs(t, err, apperrs.ErrNotFound)
	_, err = f.runner.GetTrail(ctxAs("stranger"), "tr-ticket")
	assert.ErrorIs(t, err, apperrs.ErrForbidden)

	f.perm.denyResource(starter, resourceTypeDoc, docID)
	_, err = f.runner.GetTrail(ctxAs(starter), "tr-doc")
	assert.ErrorIs(t, err, apperrs.ErrForbidden, "a doc trail needs docs:thread on the doc")
	_, err = f.runner.GetTrail(ctxAs(starter), "tr-ticket")
	assert.NoError(t, err, "the doc gate does not touch ticket trails")
}

func TestListTrails(t *testing.T) {
	f := newRunnerFixture()
	seededTrail(f, "older", TargetTicket, ticketID, fixedNow.Add(-time.Hour))
	seededTrail(f, "newer", TargetTicket, ticketID, fixedNow)

	list, err := f.runner.ListTrails(ctxAs(starter), TargetTicket, ticketID)
	require.NoError(t, err)
	require.Len(t, list, 2)
	assert.Equal(t, "newer", list[0].ID)

	empty, err := f.runner.ListTrails(ctxAs("stranger"), TargetTicket, "t-never-run")
	require.NoError(t, err)
	assert.Empty(t, empty)

	_, err = f.runner.ListTrails(ctxAs("stranger"), TargetTicket, ticketID)
	assert.ErrorIs(t, err, apperrs.ErrForbidden)
	_, err = f.runner.ListTrails(ctxAs(starter), "column", ticketID)
	assert.ErrorIs(t, err, apperrs.ErrInvalid)
	f.trails.listErr = errors.New("db down")
	_, err = f.runner.ListTrails(ctxAs(starter), TargetTicket, ticketID)
	assert.ErrorContains(t, err, "db down")
}

func TestLatestChoices(t *testing.T) {
	f := newRunnerFixture()
	none, err := f.runner.LatestChoices(ctxAs(starter), starter, fixPlayID, projectID)
	require.NoError(t, err)
	assert.Equal(t, &Choices{MemoryIDs: []string{}}, none)

	seededTrail(f, "older", TargetTicket, ticketID, fixedNow.Add(-time.Hour))
	seededTrail(f, "newer", TargetTicket, "t-2", fixedNow)
	got, err := f.runner.LatestChoices(ctxAs(starter), starter, fixPlayID, projectID)
	require.NoError(t, err)
	assert.Equal(t, &Choices{MemoryIDs: []string{alwaysMem}, MoveToStatusID: "st-newer"}, got)

	_, err = f.runner.LatestChoices(ctxAs(starter), "", fixPlayID, projectID)
	assert.ErrorIs(t, err, apperrs.ErrInvalid)
	_, err = f.runner.LatestChoices(ctxAs(starter), starter, "nope", projectID)
	assert.ErrorIs(t, err, apperrs.ErrNotFound)
	_, err = f.runner.LatestChoices(ctxAs("stranger"), starter, fixPlayID, projectID)
	assert.ErrorIs(t, err, apperrs.ErrForbidden)
	f.trails.latestErr = errors.New("db down")
	_, err = f.runner.LatestChoices(ctxAs(starter), starter, fixPlayID, projectID)
	assert.ErrorContains(t, err, "db down")
}

func TestLatestChoices_HarnessChoiceRoundTrips(t *testing.T) {
	f := newRunnerFixture()
	tr := seededTrail(f, "tr-1", TargetTicket, ticketID, fixedNow)
	tr.ComputerID, tr.Provider, tr.Model = "c-1", "claude", "sonnet-5"
	require.NoError(t, f.trails.UpdateTrail(context.Background(), tr))
	<-f.trails.terminal

	got, err := f.runner.LatestChoices(ctxAs(starter), starter, fixPlayID, projectID)
	require.NoError(t, err)
	assert.Equal(t, "c-1", got.ComputerID)
	assert.Equal(t, "claude", got.Provider)
	assert.Equal(t, "sonnet-5", got.Model)
}

func TestActiveTrails(t *testing.T) {
	f := newRunnerFixture()
	seededTrail(f, "tr-done", TargetTicket, ticketID, fixedNow)
	running := &Trail{ID: "tr-run", WorkspaceID: workspaceID, PlayID: fixPlayID, TargetType: TargetTicket, TargetID: "t-2", ProjectID: projectID,
		StarterID: starter, State: TrailRunning, StartedAt: fixedNow}
	require.NoError(t, f.trails.CreateTrail(context.Background(), running))

	got, err := f.runner.ActiveTrails(ctxAs(starter), TargetTicket, []string{ticketID, "t-2", " ", "t-3"})
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"t-2": "tr-run"}, got, "only starting or running trails count")

	got, err = f.runner.ActiveTrails(ctxAs("stranger"), TargetTicket, []string{"t-2"})
	require.NoError(t, err)
	assert.Empty(t, got, "a workspace the caller cannot read is dropped, not refused")

	got, err = f.runner.ActiveTrails(ctxAs(starter), TargetTicket, nil)
	require.NoError(t, err)
	assert.Empty(t, got)

	_, err = f.runner.ActiveTrails(ctxAs(starter), TargetType("column"), []string{"t-2"})
	assert.ErrorIs(t, err, apperrs.ErrInvalid)

	f.trails.listErr = errors.New("db down")
	_, err = f.runner.ActiveTrails(ctxAs(starter), TargetTicket, []string{"t-2"})
	assert.ErrorContains(t, err, "db down")
}
