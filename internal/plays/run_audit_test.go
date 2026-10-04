package plays

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const auditPlayID = "play-audit"

func newAuditFixture() (*runnerFixture, *fakeCheckouts) {
	f, checkouts := newDraftingFixture()
	f.plays.byID[auditPlayID] = &Play{ID: auditPlayID, WorkspaceID: workspaceID, Label: "Audit via AI", Type: TypeInterview, Instructions: "Audit it.", Enabled: true, BuiltinKey: AuditKey}
	return f, checkouts
}

func auditRun() RunInput {
	return RunInput{PlayID: auditPlayID, TargetType: TargetInterview, TargetID: projectID, ComputerID: runComputer, Via: ViaWeb}
}

func TestRun_AuditPlay_OfAPredecessor_NamesItsQuestionSourcesWithTheirCheckouts(t *testing.T) {
	f, _ := newAuditFixture()

	trail, blocks := runBlocks(t, f, auditRun())

	assert.Contains(t, blocks, "(project id "+projectID+")", "the same interview block as the follow-up run")
	assert.Contains(t, blocks, "Today is "+f.clock.UTC().Format(time.DateOnly)+". This run's trail id is "+trail.ID+".")
	assert.Contains(t, blocks, "Project sources with stance question:\n"+
		`- "Old app" (project id p-old, source id s-old): its checkout on this computer is /home/dev/old.`)
	assert.NotContains(t, blocks, "p-standards", "a follow source is the yardstick's input, not what is audited")
	assert.NotContains(t, blocks, "its own code")
	assert.Empty(t, f.answers.cleared, "an audit leaves drafts alone")
}

func TestRun_AuditPlay_WithNoQuestionSource_AuditsTheProjectsOwnCheckout(t *testing.T) {
	tests := []struct {
		name    string
		arrange func(f *runnerFixture)
	}{
		{"only follow sources", func(f *runnerFixture) { f.answers.sources = f.answers.sources[:3] }},
		{"sources unreadable", func(f *runnerFixture) { f.answers.sourcesErr = errors.New("db down") }},
		{"no answers seam", func(f *runnerFixture) { f.runner.answers = nil }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, _ := newAuditFixture()
			tt.arrange(f)

			trail, blocks := runBlocks(t, f, auditRun())

			assert.Contains(t, blocks, "This run's trail id is "+trail.ID)
			assert.Contains(t, blocks, "This project has no project source under question: audit its own code, in the checkout you are running in.")
			assert.NotContains(t, blocks, "Project sources")
		})
	}
}

func TestRun_AuditPlay_QuestionSourceWithoutACheckout_ReadsItsMemoriesAndAnswers(t *testing.T) {
	f, c := newAuditFixture()
	c.projects = nil

	_, blocks := runBlocks(t, f, auditRun())

	assert.Contains(t, blocks, `s-old): no checkout on this computer; use only its memories and interview answers.`)
}
