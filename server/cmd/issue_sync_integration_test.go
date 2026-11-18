package main

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/chat"
	"github.com/otal-labs/nexul/internal/connectors"
	"github.com/otal-labs/nexul/internal/gitprovider"
	"github.com/otal-labs/nexul/internal/issuesync"
	"github.com/otal-labs/nexul/internal/platform/eventbus/inprocess"
	"github.com/otal-labs/nexul/internal/platform/eventbus/outbox"
	"github.com/otal-labs/nexul/internal/tickets"
)

// issueSyncWorld is a wired server whose git.issue deliveries run through the outbox to the real sync consumers.
type issueSyncWorld struct {
	f     permFixture
	hooks *gitprovider.WebhookHandler
}

func newIssueSyncWorld(t *testing.T) issueSyncWorld {
	t.Helper()
	f := newPermFixture(t)
	ctx := t.Context()
	// The fixture already attaches acme/app; acme/quiet is a second repository whose sync stays off.
	_, err := f.db.Exec(`INSERT INTO project_repos (project_id, owner, name, full_name, added_at) VALUES ('project-general', 'acme', 'quiet', 'acme/quiet', 1)`)
	require.NoError(t, err)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	bus := inprocess.New(inprocess.Options{Logger: logger, DedupeStore: f.store.ProcessedEvents, DeadLetterStore: f.store.DeadLetters})
	require.NoError(t, bus.SubscribeWithConsumer(ctx, "issuesync.issue", gitprovider.TopicIssue, f.svc.issueSyncSvc.HandleIssue))
	require.NoError(t, bus.SubscribeWithConsumer(ctx, "issuesync.comment", gitprovider.TopicIssueComment, f.svc.issueSyncSvc.HandleIssueComment))
	runCtx, cancel := context.WithCancel(context.Background())
	relay := outbox.NewRelay(f.store.Outbox, bus, outbox.RelayConfig{Wake: f.store.Commits().Next, Logger: logger})
	var running sync.WaitGroup
	running.Go(func() { _ = relay.Run(runCtx) })
	t.Cleanup(func() { cancel(); running.Wait(); _ = bus.Close() })

	on := true
	_, err = f.svc.issueSyncSvc.Set(as(uOwner), "project-general", issuesync.Input{Owner: "acme", Repo: "app", Enabled: &on})
	require.NoError(t, err)
	off := false
	_, err = f.svc.issueSyncSvc.Set(as(uOwner), "project-general", issuesync.Input{Owner: "acme", Repo: "quiet", Enabled: &off})
	require.NoError(t, err)
	return issueSyncWorld{f: f, hooks: gitprovider.NewWebhookHandler("shhh", f.store.Outbox)}
}

func issuePayload(repo, action, sender, title, state string, labels ...string) map[string]any {
	ls := make([]map[string]any, 0, len(labels))
	for _, l := range labels {
		ls = append(ls, map[string]any{"name": l})
	}
	return map[string]any{
		"action": action,
		"issue": map[string]any{
			"id": 5001, "number": 12, "title": title, "body": "Steps to see it", "state": state, "labels": ls,
			"assignees": []map[string]any{{"login": uWriter}}, "user": map[string]any{"login": "bob-gh"},
			"updated_at": time.Now().UTC().Format(time.RFC3339Nano),
		},
		"sender":     map[string]any{"login": sender, "type": "User"},
		"repository": map[string]any{"name": repo, "owner": map[string]any{"login": "acme"}},
	}
}

// deliver posts one signed delivery and waits until the outbox has handed it to the consumers.
func (w issueSyncWorld) deliver(t *testing.T, event, delivery string, payload map[string]any) {
	t.Helper()
	rec := httptest.NewRecorder()
	w.hooks.ServeHTTP(rec, signedDelivery(t, "shhh", event, delivery, payload))
	require.Equal(t, http.StatusOK, rec.Code)
	// Each issue row the outbox holds is handled once the sync's consumer records its receipt, or dead-letters it.
	require.Eventually(t, func() bool {
		var rows, handled int
		err := w.f.db.QueryRow(`SELECT (SELECT COUNT(*) FROM outbox WHERE topic IN ('git.issue', 'git.issue_comment')),
			(SELECT COUNT(*) FROM processed_events WHERE event_id LIKE 'issuesync.%') + (SELECT COUNT(*) FROM dead_letters)`).Scan(&rows, &handled)
		return err == nil && rows == handled
	}, 10*time.Second, 10*time.Millisecond)
}

func (w issueSyncWorld) issueTickets(t *testing.T) []*tickets.Ticket {
	t.Helper()
	all, err := w.f.svc.ticketsSvc.ListByProject(as(uOwner), "project-general")
	require.NoError(t, err)
	var out []*tickets.Ticket
	for _, tk := range all {
		if tk.Reporter.Kind == tickets.ReporterKindGitHub {
			out = append(out, tk)
		}
	}
	return out
}

// TestIssueSync_WebhookToTicket walks a labelled issue from a signed webhook through the outbox to its ticket: one
// ticket however often it arrives, its title and comments followed, closed to done, and reopening never leaving done.
func TestIssueSync_WebhookToTicket(t *testing.T) {
	t.Parallel()
	w := newIssueSyncWorld(t)
	opened := issuePayload("app", "opened", "bob-gh", "Login fails on Safari", "open", "nexul", "bug")
	w.deliver(t, "issues", "d-1", opened)
	w.deliver(t, "issues", "d-1", opened)
	w.deliver(t, "issues", "d-2", opened)
	filed := w.issueTickets(t)
	require.Len(t, filed, 1, "a redelivery and a resend file nothing more")
	tk := filed[0]
	assert.Equal(t, tickets.Reporter{Kind: tickets.ReporterKindGitHub, Login: "bob-gh"}, tk.Reporter)
	assert.Equal(t, []string{"bug"}, tk.Labels)
	assert.Equal(t, uWriter, tk.Developer, "the assignee signed in to Nexul with that GitHub login")

	w.deliver(t, "issues", "d-3", issuePayload("app", "edited", "bob-gh", "Login fails on Safari 18", "open", "nexul", "bug"))
	got, err := w.f.svc.ticketsSvc.Get(as(uOwner), tk.ID)
	require.NoError(t, err)
	assert.Equal(t, "Login fails on Safari 18", got.Title)

	w.deliver(t, "issue_comment", "d-4", map[string]any{
		"action":     "created",
		"issue":      map[string]any{"number": 12},
		"comment":    map[string]any{"id": 77, "body": "Same on iPad", "user": map[string]any{"login": "bob-gh", "avatar_url": "https://example.com/bob.png"}},
		"sender":     map[string]any{"login": "bob-gh", "type": "User"},
		"repository": map[string]any{"name": "app", "owner": map[string]any{"login": "acme"}},
	})
	thread, err := w.f.svc.chatSvc.GetOrCreateTicketThread(as(uOwner), "workspace-default", tk.ID, uOwner)
	require.NoError(t, err)
	page, err := w.f.svc.chatSvc.ListMessages(as(uOwner), thread.ID, uOwner, 50)
	require.NoError(t, err)
	var comment *chat.Message
	for _, m := range page {
		if m.AuthorKind == chat.AuthorExternal {
			comment = m
		}
	}
	require.NotNil(t, comment)
	assert.Equal(t, "Same on iPad", comment.Body)
	assert.Equal(t, "bob-gh", comment.AuthorName)
	assert.Equal(t, "github", comment.Via)

	w.deliver(t, "issues", "d-5", issuePayload("app", "closed", "bob-gh", "Login fails on Safari 18", "closed", "nexul", "bug"))
	done, err := ticketDone(context.Background(), w.f.store.Statuses, mustTicket(t, w, tk.ID))
	require.NoError(t, err)
	assert.True(t, done)
	w.deliver(t, "issues", "d-6", issuePayload("app", "reopened", "bob-gh", "Login fails on Safari 18", "open", "nexul", "bug"))
	done, err = ticketDone(context.Background(), w.f.store.Statuses, mustTicket(t, w, tk.ID))
	require.NoError(t, err)
	assert.True(t, done, "a done ticket stays done")
}

func mustTicket(t *testing.T, w issueSyncWorld, id string) *tickets.Ticket {
	t.Helper()
	got, err := w.f.svc.ticketsSvc.Get(as(uOwner), id)
	require.NoError(t, err)
	return got
}

// TestIssueSync_ChangesNothingForTheBotOrASyncThatIsOff: the App's own change and a repository whose sync is off file
// no ticket.
func TestIssueSync_ChangesNothingForTheBotOrASyncThatIsOff(t *testing.T) {
	t.Parallel()
	w := newIssueSyncWorld(t)
	require.NoError(t, w.f.store.ConnectorAppConfig.SetAppConfig(context.Background(), connectors.AppConfig{ConnectorID: "github", ClientID: "Iv1.acme", AppSlug: "nexul-app"}))
	w.deliver(t, "issues", "d-1", issuePayload("app", "opened", "nexul-app[bot]", "From the App", "open", "nexul"))
	w.deliver(t, "issues", "d-2", issuePayload("quiet", "opened", "bob-gh", "Sync off", "open", "nexul"))
	assert.Empty(t, w.issueTickets(t))
}
