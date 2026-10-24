package main

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/crashintake"
	"github.com/otal-labs/nexul/internal/integrations"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/tickets"
	"github.com/otal-labs/nexul/internal/workspace"
)

// fixedBuilds answers every fix as merged into the build, or none.
type fixedBuilds bool

func (f fixedBuilds) MergedInto(context.Context, string, string, int, string) (bool, error) {
	return bool(f), nil
}

func crashPost(t *testing.T, fingerprint, version string) crashintake.Post {
	t.Helper()
	fields := []map[string]string{
		{"name": "Contract", "value": crashintake.Contract}, {"name": "Fingerprint", "value": fingerprint},
		{"name": "Kind", "value": "native-process"}, {"name": "App version", "value": version},
		{"name": "Build commit", "value": strings.Repeat("e", 40)}, {"name": "Signature", "value": "SIGSEGV"},
		{"name": "Stack", "value": "renderPanel"},
	}
	raw, err := json.Marshal([]map[string]any{{"fields": fields}})
	require.NoError(t, err)
	return crashintake.Post{Content: crashintake.FingerprintLine + fingerprint, Embeds: raw}
}

// asDefaultAutomation is a request as the gateway gives a shipped default automation: no creator, its workspace only.
func asDefaultAutomation() context.Context {
	return withTestMemo(identity.WithActor(context.Background(), identity.Actor{
		Automation: &identity.AutomationRef{ID: "default-crash-intake", Name: "Crash intake", WorkspaceID: "workspace-default"},
	}))
}

// TestCrashIntake_ADefaultAutomationFilesNotesAndFilesARegression walks one crash through the real tickets and chat: a
// bug for the first report, one running note for the repeats, and a high-priority bug found in it once it is done and
// the build holds its fix.
func TestCrashIntake_ADefaultAutomationFilesNotesAndFilesARegression(t *testing.T) {
	t.Parallel()
	f := newPermFixture(t)
	ctx := asDefaultAutomation()
	desk := crashIntakeDesk{tickets: f.svc.ticketsSvc, statuses: f.store.Statuses, types: f.store.TicketTypes}
	notes := crashIntakeNotes{tickets: f.svc.ticketsSvc, chat: f.svc.chatSvc, projects: f.store.Projects, users: f.store.Users}
	fingerprint := strings.Repeat("12", 32)
	cfg := crashintake.Config{ProjectID: "project-general"}
	svc := crashintake.NewService(desk, notes, fixedBuilds(true), nil)

	res, err := svc.Intake(ctx, cfg, crashPost(t, fingerprint, "2.0.0"))
	require.NoError(t, err)
	require.Equal(t, crashintake.OutcomeFiled, res.Outcome)
	filed, err := f.svc.ticketsSvc.Get(as(uOwner), res.TicketID)
	require.NoError(t, err)
	assert.Equal(t, "ticket-type-bug", filed.TypeID)
	assert.Equal(t, "Crash: SIGSEGV in renderPanel (native-process)", filed.Title)
	links, err := f.svc.ticketsSvc.Links(as(uOwner), filed.ID)
	require.NoError(t, err)
	assert.True(t, links.OriginUnknown)

	for _, version := range []string{"2.0.0", "2.1.0"} {
		res, err = svc.Intake(ctx, cfg, crashPost(t, fingerprint, version))
		require.NoError(t, err)
		assert.Equal(t, crashintake.Result{Outcome: crashintake.OutcomeSeenAgain, TicketID: filed.ID}, res)
	}
	thread, err := f.svc.chatSvc.TicketNotes(as(uOwner), filed.ID, uOwner)
	require.NoError(t, err)
	require.Len(t, thread.Notes, 1, "repeats update one note in place")
	files, err := f.svc.chatSvc.NoteFiles(as(uOwner), thread.Notes)
	require.NoError(t, err)
	note := files[thread.Notes[0].AttachmentID]
	assert.Equal(t, "crash-seen-again.md", note.Name)
	assert.Contains(t, note.Markdown, "Reports: 2")
	assert.Contains(t, note.Markdown, "Versions: 2.0.0, 2.1.0")
	assert.Equal(t, uOwner, thread.Notes[0].AuthorID, "posted on the workspace Owner's behalf")

	moveToDone(t, f, filed.ID)
	require.NoError(t, f.svc.ticketsSvc.LinkPR(as(uOwner), filed.ID, tickets.PRRef{Owner: "acme", Repo: "app", Number: 9}))
	_, err = f.store.Tickets.MarkPRState(context.Background(), "acme", "app", 9, tickets.PRStateMerged)
	require.NoError(t, err)

	res, err = svc.Intake(ctx, cfg, crashPost(t, fingerprint, "2.2.0"))
	require.NoError(t, err)
	require.Equal(t, crashintake.OutcomeRegression, res.Outcome)
	regression, err := f.svc.ticketsSvc.Get(as(uOwner), res.TicketID)
	require.NoError(t, err)
	assert.Equal(t, []string{crashintake.DefaultLabel}, regression.Labels)
	assert.Equal(t, 0, regression.Position, "the top of its column")
	links, err = f.svc.ticketsSvc.Links(as(uOwner), regression.ID)
	require.NoError(t, err)
	require.NotNil(t, links.FoundIn)
	assert.Equal(t, filed.ID, links.FoundIn.ID)
}

func moveToDone(t *testing.T, f permFixture, ticketID string) {
	t.Helper()
	statuses, err := f.store.Statuses.ListByProject(context.Background(), "project-general")
	require.NoError(t, err)
	for _, st := range statuses {
		if st.Kind != workspace.StatusKindDone {
			continue
		}
		_, err := f.svc.ticketsSvc.UpdateStatus(as(uOwner), ticketID, tickets.Status(st.ID))
		require.NoError(t, err)
		return
	}
	t.Fatal("the project has no done column")
}

func TestCrashIntake_AnAutomationTokenWithTicketsWriteReachesTheRoute(t *testing.T) {
	t.Parallel()
	assert.True(t, integrations.ScopeAllows("POST", "/api/crash-reports", []string{"tickets:write"}))
	assert.False(t, integrations.ScopeAllows("POST", "/api/crash-reports", []string{"tickets:read"}))
}
