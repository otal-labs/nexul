package plays

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/harness"
)

const (
	draftPlayID = "play-draft"
	runComputer = "c-laptop"
)

func newDraftingFixture() (*runnerFixture, *fakeCheckouts) {
	f := newRunnerFixture()
	f.plays.byID[draftPlayID] = &Play{ID: draftPlayID, WorkspaceID: workspaceID, Label: "Draft interview", Type: TypeInterview, Instructions: "Draft them.", Enabled: true, BuiltinKey: DraftInterviewKey}
	f.answers.sources = []InterviewSource{
		{ID: "s-linked", Kind: SourceKindProject, Ref: "p-standards", Label: "Standards", Stance: StanceFollow},
		{ID: "s-elsewhere", Kind: SourceKindProject, Ref: "p-elsewhere", Label: "Elsewhere", Stance: StanceFollow},
		{ID: "s-unlinked", Kind: SourceKindProject, Ref: "p-unlinked", Label: "Unlinked", Stance: StanceFollow},
		{ID: "s-old", Kind: SourceKindProject, Ref: "p-old", Label: "Old app", Stance: StanceQuestion},
		{ID: "s-hidden", Kind: SourceKindProject, Stance: StanceFollow, Unreadable: true},
		{ID: "s-doc", Kind: "doc", Ref: "d-9", Label: "Spec", Stance: StanceFollow},
	}
	checkouts := &fakeCheckouts{
		links: map[string][2]string{
			"p-standards": {runComputer, "t3-standards"},
			"p-elsewhere": {"c-desktop", "t3-elsewhere"},
			"p-old":       {runComputer, "t3-old"},
		},
		projects: []harness.Project{{ID: "t3-standards", Title: "standards", Path: "/home/dev/standards"}, {ID: "t3-old", Path: "/home/dev/old"}},
	}
	f.runner.checkouts = checkouts
	return f, checkouts
}

func draftRun() RunInput {
	return RunInput{PlayID: draftPlayID, TargetType: TargetInterview, TargetID: projectID, ComputerID: runComputer, Via: ViaWeb}
}

func runBlocks(t *testing.T, f *runnerFixture, in RunInput) (*Trail, string) {
	t.Helper()
	trail, err := f.runner.Run(ctxAs(starter), in)
	require.NoError(t, err)
	<-f.turns.done
	play := f.turns.last().Play
	require.NotNil(t, play)
	return trail, strings.Join(play.Blocks, "\n")
}

func TestRun_DraftingPlay_NamesItsTrailAndFollowProjectCheckouts(t *testing.T) {
	f, checkouts := newDraftingFixture()

	trail, blocks := runBlocks(t, f, draftRun())

	assert.Contains(t, blocks, "(project id "+projectID+")", "the same interview block as the follow-up run")
	assert.Contains(t, blocks, "This run's trail id is "+trail.ID+"; pass it as `trail_id` on every draft.")
	assert.Contains(t, blocks, "Project sources with stance follow:\n"+
		`- "Standards" (project id p-standards, source id s-linked): its checkout on this computer is /home/dev/standards.`+"\n"+
		`- "Elsewhere" (project id p-elsewhere, source id s-elsewhere): no checkout on this computer; use only its memories and interview answers.`+"\n"+
		`- "Unlinked" (project id p-unlinked, source id s-unlinked): no checkout on this computer; use only its memories and interview answers.`)
	assert.NotContains(t, blocks, "p-old", "a source under question is not read for drafts")
	assert.NotContains(t, blocks, "s-hidden", "a source the starter cannot read is left out")
	assert.NotContains(t, blocks, "s-doc", "only project sources need a checkout named")
	assert.Equal(t, 1, checkouts.lists, "the computer's projects are listed once per run")
	assert.Equal(t, []string{projectID}, f.answers.cleared, "the run clears the project's suggested changes as it starts")
}

func TestRun_InterviewPlay_NeitherClearsSuggestionsNorNamesItsTrail(t *testing.T) {
	f, _ := newDraftingFixture()

	_, blocks := runBlocks(t, f, RunInput{PlayID: intPlayID, TargetType: TargetInterview, TargetID: projectID, Via: ViaWeb})

	assert.NotContains(t, blocks, "trail id")
	assert.NotContains(t, blocks, "Project sources")
	assert.Empty(t, f.answers.cleared)
}

func TestRun_DraftingPlay_SeamFailuresNeverStopTheRun(t *testing.T) {
	tests := []struct {
		name    string
		arrange func(f *runnerFixture, c *fakeCheckouts)
		want    string
		absent  string
	}{
		{"sources unreadable", func(f *runnerFixture, _ *fakeCheckouts) { f.answers.sourcesErr = errors.New("db down") }, "trail id", "Project sources"},
		{"clearing fails", func(f *runnerFixture, _ *fakeCheckouts) { f.answers.clearErr = errors.New("db down") }, "/home/dev/standards", ""},
		{"project link unreadable", func(_ *runnerFixture, c *fakeCheckouts) { c.linkErr = errors.New("db down") }, `s-linked): no checkout on this computer`, "/home/dev"},
		{"computer unreachable", func(_ *runnerFixture, c *fakeCheckouts) { c.listErr = errors.New("offline") }, `s-linked): no checkout on this computer`, "/home/dev"},
		{"linked project no longer on the computer", func(_ *runnerFixture, c *fakeCheckouts) { c.projects = nil }, `s-linked): no checkout on this computer`, "/home/dev"},
		{"no checkouts seam", func(f *runnerFixture, _ *fakeCheckouts) { f.runner.checkouts = nil }, `s-linked): no checkout on this computer`, "/home/dev"},
		{"no answers seam", func(f *runnerFixture, _ *fakeCheckouts) { f.runner.answers = nil }, "trail id", "Project sources"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, c := newDraftingFixture()
			tt.arrange(f, c)

			_, blocks := runBlocks(t, f, draftRun())

			assert.Contains(t, blocks, tt.want)
			if tt.absent != "" {
				assert.NotContains(t, blocks, tt.absent)
			}
		})
	}
}

func TestRun_DraftingPlay_NoFollowProjectSources_NamesOnlyTheTrail(t *testing.T) {
	f, _ := newDraftingFixture()
	f.answers.sources = f.answers.sources[3:]

	trail, blocks := runBlocks(t, f, draftRun())

	assert.Contains(t, blocks, "This run's trail id is "+trail.ID)
	assert.NotContains(t, blocks, "Project sources")
}
