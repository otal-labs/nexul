package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/connectors"
	"github.com/otal-labs/nexul/internal/gitprovider"
	"github.com/otal-labs/nexul/internal/gitprovider/github"
	"github.com/otal-labs/nexul/internal/issuesync"
	"github.com/otal-labs/nexul/internal/platform/config"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus/inprocess"
	"github.com/otal-labs/nexul/internal/platform/eventbus/outbox"
	"github.com/otal-labs/nexul/internal/platform/storage"
	"github.com/otal-labs/nexul/internal/platform/storage/testutil/migrated"
	"github.com/otal-labs/nexul/internal/tickets"
	"github.com/otal-labs/nexul/internal/workspace"
)

// stubGitHub is issue acme/tracker#12 and its comments, recording every write; dropComment makes the next comment
// land and then cuts the connection before GitHub's answer, as a timed-out request GitHub accepted does.
type stubGitHub struct {
	mu          sync.Mutex
	issue       map[string]any
	comments    []map[string]any
	writes      []string
	dropComment bool
}

func (g *stubGitHub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	g.mu.Lock()
	defer g.mu.Unlock()
	now := time.Now().UTC().Format(time.RFC3339)
	path := r.Method + " " + r.URL.Path
	if r.Method != http.MethodGet {
		g.writes = append(g.writes, path)
	}
	switch path {
	case "PATCH /repos/acme/tracker/issues/12":
		var patch map[string]any
		_ = json.NewDecoder(r.Body).Decode(&patch) // test server: a bad body fails the assertions instead
		for k, v := range patch {
			g.issue[k] = v
		}
		if labels, ok := patch["labels"].([]any); ok {
			named := make([]map[string]any, 0, len(labels))
			for _, l := range labels {
				named = append(named, map[string]any{"name": l})
			}
			g.issue["labels"] = named
		}
		g.issue["updated_at"] = now
		_ = json.NewEncoder(w).Encode(g.issue) // test server: write errors are irrelevant
	case "POST /repos/acme/tracker/issues/12/comments":
		var in map[string]string
		_ = json.NewDecoder(r.Body).Decode(&in) // test server: a bad body fails the assertions instead
		c := map[string]any{"id": 8000 + len(g.comments), "body": in["body"], "user": map[string]any{"login": "nexul-app[bot]"}, "created_at": now, "updated_at": now}
		g.comments = append(g.comments, c)
		if g.dropComment {
			g.dropComment = false
			conn, _, err := w.(http.Hijacker).Hijack()
			if err == nil {
				_ = conn.Close() // test server: the client sees the cut connection either way
			}
			return
		}
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(c) // test server: write errors are irrelevant
	case "GET /repos/acme/tracker/issues/12/comments":
		_ = json.NewEncoder(w).Encode(g.comments) // test server: write errors are irrelevant
	default:
		http.NotFound(w, r)
	}
}

func (g *stubGitHub) wrote() []string {
	g.mu.Lock()
	defer g.mu.Unlock()
	return append([]string(nil), g.writes...)
}

// stubIssueGit reads issues from the stub as the router would from GitHub.
type stubIssueGit struct{ c *github.Client }

func (s stubIssueGit) ListIssues(ctx context.Context, owner, name string, opts gitprovider.IssueOpts) ([]*gitprovider.Issue, int, error) {
	return s.c.ListIssues(ctx, owner, name, opts)
}

func (s stubIssueGit) ListIssueComments(ctx context.Context, owner, name string, number int, since time.Time) ([]*gitprovider.IssueComment, error) {
	return s.c.ListIssueComments(ctx, owner, name, number, since)
}

func (s stubIssueGit) resolve(context.Context, string, string) (gitprovider.GitProvider, error) {
	return s.c, nil
}

// TestIssueSync_BothWays_AgainstAStubGitHub: an issue filed in Nexul writes nothing back; a burst of edits in Nexul is
// one write, done closes the issue, a comment posts once though its first answer was lost, the App's own webhooks
// replayed change nothing, a paused link writes nothing, and deleting the ticket leaves the issue untouched.
func TestIssueSync_BothWays_AgainstAStubGitHub(t *testing.T) {
	t.Parallel()
	db := migrated.DB(t)
	store := storage.New(db, []byte("0123456789abcdef0123456789abcdef"))
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	bus := inprocess.New(inprocess.Options{Logger: logger, DedupeStore: store.ProcessedEvents, DeadLetterStore: store.DeadLetters})
	t.Cleanup(func() { _ = bus.Close() })
	cfg := &config.Config{DBPath: filepath.Join(t.TempDir(), "nexul.db"), HTTPAddr: "127.0.0.1:0", AuthSecret: "0123456789abcdef0123456789abcdef"}
	svc := wireCoreServices(cfg, store, []byte("0123456789abcdef0123456789abcdef"), bus, logger)
	ctx := t.Context()

	stub := &stubGitHub{}
	srv := httptest.NewServer(stub)
	t.Cleanup(srv.Close)
	base, err := url.Parse(srv.URL)
	require.NoError(t, err)
	client := github.New("installation-token", github.WithBaseURL(base))
	deps := issueSyncDeps(store, svc.ticketsSvc, svc.chatSvc, svc.workspaceSvc, svc.accessSvc, gitProviderRouter{}, store.ConnectorAppConfig)
	writer := issueWriter{forRepo: func(context.Context, string, string) (gitprovider.GitProvider, error) { return client, nil }}
	deps.Issues, deps.IssueWriter, deps.CommentWriter = issueReader{git: stubIssueGit{client}}, writer, writer
	deps.Quiet, deps.Retry = 200*time.Millisecond, 50*time.Millisecond
	svc.issueSyncSvc = issuesync.New(deps)

	wireDomainEventSubscriptions(ctx, bus, svc, logger)
	seedPermFixture(t, db, svc, store)
	runCtx, cancel := context.WithCancel(ctx)
	relay := outbox.NewRelay(store.Outbox, bus, outbox.RelayConfig{Wake: store.Commits().Next, Logger: logger})
	var running sync.WaitGroup
	running.Go(func() { _ = relay.Run(runCtx) })
	running.Go(func() { svc.issueSyncSvc.RunPushLoop(runCtx, store.CommitsTo(storage.IssuePushWrites...).Next, logger) })
	t.Cleanup(func() { cancel(); running.Wait() })

	require.NoError(t, store.Projects.AddRepo(ctx, pGeneral, workspace.RepoRef{Owner: "acme", Name: "tracker", FullName: "acme/tracker", ConnectorID: "github", Role: workspace.RepoRoleApp}))
	on := true
	_, err = svc.issueSyncSvc.Set(as(uOwner), pGeneral, issuesync.Change{Owner: "acme", Repo: "tracker", Enabled: &on})
	require.NoError(t, err)
	require.NoError(t, store.ConnectorAppConfig.SetAppConfig(ctx, connectors.AppConfig{ConnectorID: githubConnectorID, ClientID: "Iv1.test", AppSlug: "nexul-app"}))

	hook := gitprovider.NewWebhookHandler("shhh", store.Outbox)
	deliveries := 0
	deliver := func(event string, payload map[string]any) {
		t.Helper()
		deliveries++
		rec := httptest.NewRecorder()
		hook.ServeHTTP(rec, signedDelivery(t, "shhh", event, "push-"+strconv.Itoa(deliveries), payload))
		require.Equal(t, http.StatusOK, rec.Code)
	}
	at := time.Now().UTC().Add(-time.Hour)
	stub.issue = map[string]any{
		"id": 5001, "number": 12, "title": "Login fails on Safari", "body": "Steps to reproduce", "state": "open",
		"labels": []map[string]any{{"name": "nexul"}, {"name": "bug"}}, "assignees": []map[string]any{}, "user": map[string]any{"login": "alice"},
		"html_url": "https://github.com/acme/tracker/issues/12", "updated_at": at.Format(time.RFC3339),
	}
	issueEvent := func(action, sender string) map[string]any {
		stub.mu.Lock()
		defer stub.mu.Unlock()
		raw, err := json.Marshal(stub.issue)
		require.NoError(t, err)
		var copied map[string]any
		require.NoError(t, json.Unmarshal(raw, &copied))
		return map[string]any{
			"action": action, "issue": copied,
			"sender":     map[string]any{"login": sender, "type": "User"},
			"repository": map[string]any{"name": "tracker", "owner": map[string]any{"login": "acme"}},
		}
	}
	// settled waits until every event the write-back consumes is handled and nothing waits to be written.
	settled := func(what string) {
		t.Helper()
		require.Eventually(t, func() bool {
			var unhandled, waiting int
			require.NoError(t, db.QueryRowContext(ctx, `SELECT COUNT(*) FROM outbox o WHERE o.topic IN (?, ?, ?, ?, ?, ?, ?)
				AND NOT EXISTS (SELECT 1 FROM processed_events p WHERE p.event_id = 'issuesync.push.' || o.topic || ':' || o.id)`,
				tickets.TopicUpdated, tickets.TopicStatusChanged, tickets.TopicDeveloperChanged, tickets.TopicLabelsChanged,
				"chat.message.created", "chat.message.updated", "chat.message.deleted").Scan(&unhandled))
			require.NoError(t, db.QueryRowContext(ctx, `SELECT (SELECT COUNT(*) FROM github_issue_pushes) + (SELECT COUNT(*) FROM github_comment_pushes)`).Scan(&waiting))
			return unhandled == 0 && waiting == 0
		}, 15*time.Second, 10*time.Millisecond, what)
	}

	deliver("issues", issueEvent("opened", "alice"))
	var ticket *tickets.Ticket
	require.Eventually(t, func() bool {
		link, err := store.IssueSync.GetLink(ctx, "acme", "tracker", 12)
		if err != nil {
			return false
		}
		ticket, err = svc.ticketsSvc.Get(ctx, link.TicketID)
		return err == nil
	}, 10*time.Second, 10*time.Millisecond)
	settled("the filed ticket's own events are handled")
	assert.Empty(t, stub.wrote(), "a ticket filed from GitHub writes nothing back")

	for n := range 20 {
		_, err := svc.ticketsSvc.UpdateTicket(as(uOwner), ticket.ID, "Login fails on Safari 18", fmt.Sprintf("Steps to reproduce, take %d", n))
		require.NoError(t, err)
	}
	settled("the burst of edits is written")
	require.Equal(t, []string{"PATCH /repos/acme/tracker/issues/12"}, stub.wrote(), "twenty fast edits are one write")
	stub.mu.Lock()
	assert.Equal(t, "Login fails on Safari 18", stub.issue["title"])
	assert.True(t, strings.HasPrefix(stub.issue["body"].(string), "Steps to reproduce, take 19"))
	assert.Contains(t, stub.issue["body"], "<!-- nexul:ticket:"+ticket.ID+" -->")
	stub.mu.Unlock()

	_, err = svc.ticketsSvc.UpdateStatus(as(uOwner), ticket.ID, tickets.Status("done"))
	require.NoError(t, err)
	settled("the move to done is written")
	require.Len(t, stub.wrote(), 2)
	stub.mu.Lock()
	assert.Equal(t, "closed", stub.issue["state"], "moving the ticket to done closes the issue")
	assert.Equal(t, "completed", stub.issue["state_reason"])
	stub.mu.Unlock()

	thread, err := svc.chatSvc.GetOrCreateTicketThread(ctx, "workspace-default", ticket.ID, uOwner)
	require.NoError(t, err)
	stub.mu.Lock()
	stub.dropComment = true
	stub.mu.Unlock()
	msg, err := svc.chatSvc.PostMessage(as(uOwner), thread.ID, uOwner, "Shipped in 2.4")
	require.NoError(t, err)
	settled("the comment is posted")
	stub.mu.Lock()
	require.Len(t, stub.comments, 1, "the retry finds the comment its lost first attempt posted")
	body := stub.comments[0]["body"].(string)
	comment := stub.comments[0]
	stub.mu.Unlock()
	assert.Contains(t, body, "wrote in Nexul:\n\nShipped in 2.4")
	assert.Contains(t, body, "<!-- nexul:message:"+msg.ID+" -->")

	writes := len(stub.wrote())
	deliver("issues", issueEvent("edited", "nexul-app[bot]"))
	deliver("issues", issueEvent("closed", "nexul-app[bot]"))
	deliver("issues", issueEvent("edited", "alice"))
	deliver("issue_comment", map[string]any{
		"action": "created", "issue": map[string]any{"number": 12}, "comment": comment,
		"sender":     map[string]any{"login": "nexul-app[bot]", "type": "Bot"},
		"repository": map[string]any{"name": "tracker", "owner": map[string]any{"login": "acme"}},
	})
	settled("the replayed webhooks are handled")
	messages, err := svc.chatSvc.ListMessages(ctx, thread.ID, uOwner, 50)
	require.NoError(t, err)
	for _, m := range messages {
		assert.NotEqual(t, "external", string(m.AuthorKind), "Nexul's own comment never comes back as a GitHub comment")
	}
	fresh, err := svc.ticketsSvc.Get(ctx, ticket.ID)
	require.NoError(t, err)
	assert.Equal(t, "Login fails on Safari 18", fresh.Title)
	assert.Len(t, stub.wrote(), writes, "what Nexul wrote, replayed as webhooks, writes nothing back out")

	stub.mu.Lock()
	stub.issue["labels"] = []map[string]any{{"name": "bug"}}
	stub.issue["updated_at"] = time.Now().UTC().Add(time.Minute).Format(time.RFC3339)
	stub.mu.Unlock()
	deliver("issues", issueEvent("unlabeled", "alice"))
	require.Eventually(t, func() bool {
		link, err := store.IssueSync.GetLink(ctx, "acme", "tracker", 12)
		return err == nil && link.Paused
	}, 10*time.Second, 10*time.Millisecond)
	_, err = svc.ticketsSvc.UpdateTicket(as(uOwner), ticket.ID, "Edited while paused", "Body")
	require.NoError(t, err)
	settled("the paused ticket's edit is handled")
	assert.Len(t, stub.wrote(), writes, "a paused ticket writes nothing")

	require.NoError(t, svc.ticketsSvc.Delete(as(uOwner), ticket.ID))
	_, err = store.IssueSync.GetLink(ctx, "acme", "tracker", 12)
	require.ErrorIs(t, err, apperrs.ErrNotFound)
	settled("the delete is handled")
	assert.Len(t, stub.wrote(), writes, "deleting the ticket leaves the issue untouched")
}
