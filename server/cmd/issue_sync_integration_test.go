package main

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/chat"
	"github.com/otal-labs/nexul/internal/connectors"
	"github.com/otal-labs/nexul/internal/gitprovider"
	"github.com/otal-labs/nexul/internal/issuesync"
	"github.com/otal-labs/nexul/internal/platform/config"
	"github.com/otal-labs/nexul/internal/platform/eventbus/inprocess"
	"github.com/otal-labs/nexul/internal/platform/eventbus/outbox"
	"github.com/otal-labs/nexul/internal/platform/storage"
	"github.com/otal-labs/nexul/internal/platform/storage/testutil/migrated"
	"github.com/otal-labs/nexul/internal/tickets"
	"github.com/otal-labs/nexul/internal/workspace"
)

// TestIssueSync_FromWebhookToTicket: a labelled GitHub issue reaches its ticket through the webhook, the outbox and the
// consumer, once however often it is delivered; an edit, a close and a comment follow it, the App's own bot changes
// nothing, and a repository whose sync is off files nothing.
func TestIssueSync_FromWebhookToTicket(t *testing.T) {
	t.Parallel()
	db := migrated.DB(t)
	store := storage.New(db, []byte("0123456789abcdef0123456789abcdef"))
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	bus := inprocess.New(inprocess.Options{Logger: logger, DedupeStore: store.ProcessedEvents, DeadLetterStore: store.DeadLetters})
	t.Cleanup(func() { _ = bus.Close() })
	cfg := &config.Config{DBPath: filepath.Join(t.TempDir(), "nexul.db"), HTTPAddr: "127.0.0.1:0", AuthSecret: "0123456789abcdef0123456789abcdef"}
	svc := wireCoreServices(cfg, store, []byte("0123456789abcdef0123456789abcdef"), bus, logger)
	ctx := t.Context()
	wireDomainEventSubscriptions(ctx, bus, svc, logger)
	seedPermFixture(t, db, svc, store)
	runCtx, cancel := context.WithCancel(ctx)
	relay := outbox.NewRelay(store.Outbox, bus, outbox.RelayConfig{Wake: store.Commits().Next, Logger: logger})
	var running sync.WaitGroup
	running.Go(func() { _ = relay.Run(runCtx) })
	t.Cleanup(func() { cancel(); running.Wait() })

	for _, name := range []string{"tracker", "quiet"} {
		require.NoError(t, store.Projects.AddRepo(ctx, pGeneral, workspace.RepoRef{Owner: "acme", Name: name, FullName: "acme/" + name, ConnectorID: "github", Role: workspace.RepoRoleApp}))
	}
	on := true
	_, err := svc.issueSyncSvc.Set(as(uOwner), pGeneral, issuesync.Change{Owner: "acme", Repo: "tracker", Enabled: &on})
	require.NoError(t, err)

	hook := gitprovider.NewWebhookHandler("shhh", store.Outbox)
	deliver := func(event, delivery string, payload map[string]any) {
		t.Helper()
		rec := httptest.NewRecorder()
		hook.ServeHTTP(rec, signedDelivery(t, "shhh", event, delivery, payload))
		require.Equal(t, http.StatusOK, rec.Code)
	}
	issue := func(repo, action, title, state, sender string, at time.Time) map[string]any {
		return map[string]any{
			"action": action,
			"issue": map[string]any{"id": 5001, "number": 12, "title": title, "body": "Steps to reproduce", "state": state,
				"labels": []map[string]any{{"name": "nexul"}, {"name": "bug"}}, "user": map[string]any{"login": "alice"},
				"updated_at": at.Format(time.RFC3339)},
			"sender":     map[string]any{"login": sender, "type": "User"},
			"repository": map[string]any{"name": repo, "owner": map[string]any{"login": "acme"}},
		}
	}
	// One subscriber handles a topic's events in order, so seeing a later event's effect means the earlier ones ran.
	eventually := func(what string, ok func(*tickets.Ticket) bool, id string) *tickets.Ticket {
		t.Helper()
		var got *tickets.Ticket
		require.Eventually(t, func() bool {
			fresh, err := svc.ticketsSvc.Get(ctx, id)
			got = fresh
			return err == nil && ok(fresh)
		}, 10*time.Second, 10*time.Millisecond, what)
		return got
	}
	synced := func() []*tickets.Ticket {
		t.Helper()
		all, err := svc.ticketsSvc.ListByProject(ctx, pGeneral)
		require.NoError(t, err)
		var out []*tickets.Ticket
		for _, tk := range all {
			if tk.Reporter.Kind == tickets.ReporterKindGitHub {
				out = append(out, tk)
			}
		}
		return out
	}
	at := time.Date(2026, 11, 18, 10, 0, 0, 0, time.UTC)

	deliver("issues", "d-1", issue("tracker", "opened", "Login fails on Safari", "open", "alice", at))
	deliver("issues", "d-1", issue("tracker", "opened", "Login fails on Safari", "open", "alice", at))
	deliver("issues", "d-2", issue("tracker", "labeled", "Login fails on Safari", "open", "alice", at))
	deliver("issues", "d-3", issue("quiet", "opened", "Never synced", "open", "alice", at))
	var ticket *tickets.Ticket
	require.Eventually(t, func() bool {
		got := synced()
		if len(got) > 0 {
			ticket = got[0]
		}
		return ticket != nil
	}, 10*time.Second, 10*time.Millisecond)
	assert.Equal(t, "Login fails on Safari", ticket.Title)
	assert.Equal(t, tickets.Status("open"), ticket.Status, "it lands in the first backlog column")
	assert.Equal(t, tickets.Reporter{Kind: tickets.ReporterKindGitHub, Login: "alice"}, ticket.Reporter)

	deliver("issues", "d-4", issue("tracker", "edited", "Edited on GitHub", "open", "alice", at.Add(time.Minute)))
	eventually("the edit reaches the ticket", func(tk *tickets.Ticket) bool { return tk.Title == "Edited on GitHub" }, ticket.ID)
	assert.Len(t, synced(), 1, "one ticket however often the issue arrives, and none for a repository whose sync is off")
	labels, err := svc.ticketsSvc.ListLabels(ctx, ticket.ID)
	require.NoError(t, err)
	assert.Equal(t, []string{"bug"}, labels)

	require.NoError(t, store.ConnectorAppConfig.SetAppConfig(ctx, connectors.AppConfig{ConnectorID: githubConnectorID, ClientID: "Iv1.test", AppSlug: "nexul-app"}))
	deliver("issues", "d-5", issue("tracker", "edited", "Echo of our own write", "open", "nexul-app[bot]", at.Add(2*time.Minute)))
	deliver("issues", "d-6", issue("tracker", "closed", "Edited on GitHub", "closed", "alice", at.Add(3*time.Minute)))
	fresh := eventually("closing the issue moves the ticket to the first done column",
		func(tk *tickets.Ticket) bool { return tk.Status == tickets.Status("done") }, ticket.ID)
	assert.Equal(t, "Edited on GitHub", fresh.Title, "an event the App's own bot sent changes nothing")

	deliver("issue_comment", "d-7", map[string]any{
		"action":     "created",
		"issue":      map[string]any{"number": 12},
		"comment":    map[string]any{"id": 901, "body": "Fixed for me in 2.3", "user": map[string]any{"login": "carol", "avatar_url": "https://avatars.example.com/carol"}},
		"sender":     map[string]any{"login": "carol", "type": "User"},
		"repository": map[string]any{"name": "tracker", "owner": map[string]any{"login": "acme"}},
	})
	thread, err := svc.chatSvc.GetOrCreateTicketThread(ctx, "workspace-default", ticket.ID, uOwner)
	require.NoError(t, err)
	var external *chat.Message
	notes := 0
	require.Eventually(t, func() bool {
		messages, err := svc.chatSvc.ListMessages(ctx, thread.ID, uOwner, 50)
		require.NoError(t, err)
		notes = 0
		for _, m := range messages {
			if m.AuthorKind == chat.AuthorExternal {
				external = m
			}
			if m.AuthorKind == chat.AuthorSystem {
				notes++
			}
		}
		return external != nil
	}, 10*time.Second, 10*time.Millisecond, "the comment reaches the ticket's thread")
	require.NotNil(t, external)
	assert.Equal(t, "Fixed for me in 2.3", external.Body)
	assert.Equal(t, "carol", external.AuthorName)
	assert.Equal(t, "github", external.Via)
	assert.Equal(t, 1, notes, "the close says why on the thread")
}
