package gitprovider

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	githubapi "github.com/google/go-github/v71/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/platform/logging"
)

func prEventPayload(action string, merged bool) []byte {
	payload, _ := json.Marshal(map[string]any{
		"action": action,
		"number": 7,
		"pull_request": map[string]any{
			"number": 7,
			"title":  "REF-42 Fix login",
			"body":   "Also fixes ref-43, after REF-44.",
			"state":  "open",
			"merged": merged,
			"head":   map[string]any{"ref": "feature/REF-42-fix-login", "sha": "abc123"},
			"base":   map[string]any{"ref": "main"},
			"user":   map[string]any{"login": "onik97"},
		},
		"repository": map[string]any{
			"owner": map[string]any{"login": "acme"},
			"name":  "app",
		},
	})
	return payload
}

func signedWebhookRequest(t *testing.T, secret, eventType string, payload []byte) *http.Request {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/hooks/github", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(githubapi.EventTypeHeader, eventType)
	req.Header.Set(githubapi.DeliveryIDHeader, "delivery-123")
	if secret != "" {
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write(payload)
		req.Header.Set(githubapi.SHA256SignatureHeader, "sha256="+hex.EncodeToString(mac.Sum(nil)))
	}
	return req
}

// envelopeFrom decodes the first published event (the envelope) into a ProviderEvent.
func envelopeFrom(t *testing.T, bus *fakeBus) ProviderEvent {
	t.Helper()
	if len(bus.published) == 0 {
		t.Fatal("expected at least the normalized envelope to be published")
	}
	var env ProviderEvent
	require.NoError(t, json.Unmarshal(bus.published[0].Payload, &env))
	return env
}

func serveWebhook(t *testing.T, h *WebhookHandler, req *http.Request) *httptest.ResponseRecorder {
	t.Helper()
	silent := slog.New(slog.NewTextHandler(io.Discard, nil))
	req = req.WithContext(logging.CtxWithLogger(req.Context(), silent))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestWebhookHandler_Opened(t *testing.T) {
	bus := newFakeBus()
	resolver := &fakeTicketResolver{ids: map[string]string{"REF-42": "t-42", "REF-43": "t-43"}}
	h := NewWebhookHandler("shhh", bus).WithTicketResolver(resolver)
	rec := serveWebhook(t, h, signedWebhookRequest(t, "shhh", "pull_request", prEventPayload("opened", false)))

	assert.Equal(t, http.StatusOK, rec.Code)
	ev := bus.lastPublished()
	assert.Equal(t, TopicPROpened, ev.Topic)
	var got PREvent
	require.NoError(t, json.Unmarshal(ev.Payload, &got))
	assert.Equal(t, "acme", got.Owner)
	assert.Equal(t, "app", got.Repo)
	assert.Equal(t, 7, got.PR.Number)
	assert.Equal(t, "Also fixes ref-43, after REF-44.", got.PR.Body)
	assert.Equal(t, []string{"REF-42", "REF-43"}, resolver.keys, "the title, the body's closing word and the branch name the keys, each once; a mention links nothing")
	assert.Equal(t, []string{"t-42", "t-43"}, got.PR.LinkedTicketIDs)
	assert.Equal(t, "abc123", got.PR.HeadSHA)
	assert.Equal(t, "main", got.PR.BaseBranch)
	assert.Equal(t, "onik97", got.PR.Author)
}

func TestWebhookHandler_Merged(t *testing.T) {
	bus := newFakeBus()
	h := NewWebhookHandler("shhh", bus)
	rec := serveWebhook(t, h, signedWebhookRequest(t, "shhh", "pull_request", prEventPayload("closed", true)))

	assert.Equal(t, http.StatusOK, rec.Code)
	ev := bus.lastPublished()
	assert.Equal(t, TopicPRMerged, ev.Topic)
	var got PREvent
	require.NoError(t, json.Unmarshal(ev.Payload, &got))
	assert.True(t, got.PR.Merged)
}

func TestWebhookHandler_ClosedWithoutMerge_PublishesPRClosed(t *testing.T) {
	bus := newFakeBus()
	h := NewWebhookHandler("shhh", bus)
	rec := serveWebhook(t, h, signedWebhookRequest(t, "shhh", "pull_request", prEventPayload("closed", false)))

	assert.Equal(t, http.StatusOK, rec.Code)
	ev := bus.lastPublished()
	assert.Equal(t, TopicPRClosed, ev.Topic)
	var got PREvent
	require.NoError(t, json.Unmarshal(ev.Payload, &got))
	assert.Equal(t, "acme", got.Owner)
	assert.Equal(t, "app", got.Repo)
	assert.Equal(t, 7, got.PR.Number)
}

func TestWebhookHandler_UnknownEventType_PublishesEnvelope(t *testing.T) {
	bus := newFakeBus()
	h := NewWebhookHandler("shhh", bus)
	rec := serveWebhook(t, h, signedWebhookRequest(t, "shhh", "workflow_run", []byte(`{"repository":{"name":"app","full_name":"acme/app","owner":{"login":"acme"},"html_url":"https://github.com/acme/app","default_branch":"main"}}`)))

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Len(t, bus.published, 1)
	env := envelopeFrom(t, bus)
	assert.Equal(t, "github", env.Provider)
	assert.Equal(t, "workflow_run", env.EventType)
	assert.Equal(t, "delivery-123", env.DeliveryID)
	assert.Equal(t, "acme", env.Repository.Owner)
	assert.Equal(t, "app", env.Repository.Name)
}

func pushEventPayload(ref, before, after string, deleted bool) []byte {
	payload, _ := json.Marshal(map[string]any{
		"ref":     ref,
		"before":  before,
		"after":   after,
		"deleted": deleted,
		"pusher":  map[string]any{"name": "onik97"},
		"repository": map[string]any{
			"name":  "app",
			"owner": map[string]any{"login": "acme"},
		},
	})
	return payload
}

func TestWebhookHandler_Push_PublishesGitPush(t *testing.T) {
	bus := newFakeBus()
	h := NewWebhookHandler("shhh", bus)
	rec := serveWebhook(t, h, signedWebhookRequest(t, "shhh", "push", pushEventPayload("refs/heads/feature/discord", "before123", "after456", false)))

	assert.Equal(t, http.StatusOK, rec.Code)
	require.Len(t, bus.published, 2)
	assert.Equal(t, TopicProviderEvent, bus.published[0].Topic)
	ev := bus.published[1]
	assert.Equal(t, TopicPush, ev.Topic)
	var got PushEvent
	require.NoError(t, json.Unmarshal(ev.Payload, &got))
	assert.Equal(t, "acme", got.Owner)
	assert.Equal(t, "app", got.Repo)
	assert.Equal(t, "feature/discord", got.Branch)
	assert.Equal(t, "after456", got.SHA)
	assert.Equal(t, "onik97", got.Pusher)
}

func TestWebhookHandler_PushDeleted_PublishesBranchDeleted(t *testing.T) {
	bus := newFakeBus()
	h := NewWebhookHandler("shhh", bus)
	rec := serveWebhook(t, h, signedWebhookRequest(t, "shhh", "push", pushEventPayload("refs/heads/feature/discord", "before123", "0000000000000000000000000000000000000000", true)))

	assert.Equal(t, http.StatusOK, rec.Code)
	require.Len(t, bus.published, 2)
	ev := bus.published[1]
	assert.Equal(t, TopicBranchDeleted, ev.Topic)
	var got BranchDeletedEvent
	require.NoError(t, json.Unmarshal(ev.Payload, &got))
	assert.Equal(t, "acme", got.Owner)
	assert.Equal(t, "app", got.Repo)
	assert.Equal(t, "feature/discord", got.Branch)
}

func TestWebhookHandler_Ping_PublishesEnvelope(t *testing.T) {
	bus := newFakeBus()
	h := NewWebhookHandler("shhh", bus)
	rec := serveWebhook(t, h, signedWebhookRequest(t, "shhh", "ping", []byte(`{"zen":"ok"}`)))

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Len(t, bus.published, 1)
	assert.Equal(t, TopicProviderEvent, bus.published[0].Topic)
	env := envelopeFrom(t, bus)
	assert.Equal(t, "ping", env.EventType)
	assert.Empty(t, env.Action)
	assert.Equal(t, json.RawMessage(`{"zen":"ok"}`), env.Payload)
}

func TestWebhookHandler_InvalidSignature_Rejected(t *testing.T) {
	bus := newFakeBus()
	h := NewWebhookHandler("shhh", bus)
	rec := serveWebhook(t, h, signedWebhookRequest(t, "wrong-secret", "pull_request", prEventPayload("opened", false)))

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Empty(t, bus.published)
}

func TestWebhookHandler_UnsignedWithConfiguredSecret_Rejected(t *testing.T) {
	bus := newFakeBus()
	h := NewWebhookHandler("shhh", bus)
	req := httptest.NewRequest(http.MethodPost, "/hooks/github", bytes.NewReader(prEventPayload("opened", false)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(githubapi.EventTypeHeader, "pull_request")
	rec := serveWebhook(t, h, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Empty(t, bus.published)
}

func TestWebhookHandler_UnsupportedContentType_Rejected(t *testing.T) {
	bus := newFakeBus()
	h := NewWebhookHandler("shhh", bus)
	req := httptest.NewRequest(http.MethodPost, "/hooks/github", bytes.NewReader(prEventPayload("opened", false)))
	req.Header.Set("Content-Type", "text/plain")
	req.Header.Set(githubapi.EventTypeHeader, "pull_request")
	req.Header.Set(githubapi.SHA256SignatureHeader, "sha256=deadbeef")
	rec := serveWebhook(t, h, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Empty(t, bus.published)
}

func TestWebhookHandler_NoSecret_AcceptsDevDelivery(t *testing.T) {
	bus := newFakeBus()
	h := NewWebhookHandler("", bus)
	req := httptest.NewRequest(http.MethodPost, "/hooks/github", bytes.NewReader(prEventPayload("opened", false)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(githubapi.EventTypeHeader, "pull_request")
	rec := serveWebhook(t, h, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, TopicPROpened, bus.lastPublished().Topic)
}

func reviewEventPayload(action, state, reviewer string) []byte {
	payload, _ := json.Marshal(map[string]any{
		"action": action,
		"review": map[string]any{
			"state": state,
			"user":  map[string]any{"login": reviewer},
		},
		"pull_request": map[string]any{
			"number": 7,
			"title":  "Fix login",
		},
		"repository": map[string]any{
			"owner": map[string]any{"login": "acme"},
			"name":  "app",
		},
	})
	return payload
}

func TestWebhookHandler_ReviewSubmitted(t *testing.T) {
	bus := newFakeBus()
	h := NewWebhookHandler("shhh", bus)
	rec := serveWebhook(t, h, signedWebhookRequest(t, "shhh", "pull_request_review", reviewEventPayload("submitted", "approved", "alice")))

	assert.Equal(t, http.StatusOK, rec.Code)
	ev := bus.lastPublished()
	assert.Equal(t, TopicPRReviewSubmitted, ev.Topic)
	var got ReviewSubmittedEvent
	require.NoError(t, json.Unmarshal(ev.Payload, &got))
	assert.Equal(t, "acme", got.Owner)
	assert.Equal(t, "app", got.Repo)
	assert.Equal(t, 7, got.PR.Number)
	assert.Equal(t, "approved", got.PR.State)
	assert.Equal(t, "alice", got.PR.Reviewer)
}

func issueCommentPayload(isPR bool, body, author string) []byte {
	return issueCommentPayloadWithAction("created", isPR, body, author)
}

func issueCommentPayloadWithAction(action string, isPR bool, body, author string) []byte {
	issue := map[string]any{"number": 7}
	if isPR {
		issue["pull_request"] = map[string]any{"url": "https://api.github.com/repos/acme/app/pulls/7"}
	}
	payload, _ := json.Marshal(map[string]any{
		"action": action,
		"issue":  issue,
		"comment": map[string]any{
			"id":         int64(99),
			"body":       body,
			"user":       map[string]any{"login": author, "avatar_url": "https://avatars.example.com/" + author},
			"created_at": "2026-11-01T10:00:00Z",
			"updated_at": "2026-11-02T10:00:00Z",
		},
		"sender": map[string]any{"login": author, "type": "User"},
		"repository": map[string]any{
			"owner": map[string]any{"login": "acme"},
			"name":  "app",
		},
	})
	return payload
}

func TestWebhookHandler_IssueCommentOnPR_PublishesPRComment(t *testing.T) {
	bus := newFakeBus()
	h := NewWebhookHandler("shhh", bus)
	rec := serveWebhook(t, h, signedWebhookRequest(t, "shhh", "issue_comment", issueCommentPayload(true, "/deploy", "onik97")))

	assert.Equal(t, http.StatusOK, rec.Code)
	ev := bus.lastPublished()
	assert.Equal(t, TopicPRCommentCreated, ev.Topic)
	var got PRCommentEvent
	require.NoError(t, json.Unmarshal(ev.Payload, &got))
	assert.Equal(t, "acme", got.Owner)
	assert.Equal(t, "app", got.Repo)
	assert.Equal(t, 7, got.PR.Number)
	assert.Equal(t, "/deploy", got.Comment.Body)
	assert.Equal(t, "onik97", got.Comment.Author)
}

func reviewCommentPayload(body, author string) []byte {
	payload, _ := json.Marshal(map[string]any{
		"action": "created",
		"pull_request": map[string]any{
			"number": 7,
		},
		"comment": map[string]any{
			"body": body,
			"user": map[string]any{"login": author},
		},
		"repository": map[string]any{
			"owner": map[string]any{"login": "acme"},
			"name":  "app",
		},
	})
	return payload
}

func TestWebhookHandler_ReviewComment_PublishesPRComment(t *testing.T) {
	bus := newFakeBus()
	h := NewWebhookHandler("shhh", bus)
	rec := serveWebhook(t, h, signedWebhookRequest(t, "shhh", "pull_request_review_comment", reviewCommentPayload("nit: rename this", "bob")))

	assert.Equal(t, http.StatusOK, rec.Code)
	ev := bus.lastPublished()
	assert.Equal(t, TopicPRCommentCreated, ev.Topic)
	var got PRCommentEvent
	require.NoError(t, json.Unmarshal(ev.Payload, &got))
	assert.Equal(t, "acme", got.Owner)
	assert.Equal(t, "app", got.Repo)
	assert.Equal(t, 7, got.PR.Number)
	assert.Equal(t, "nit: rename this", got.Comment.Body)
	assert.Equal(t, "bob", got.Comment.Author)
}

func TestWebhookHandler_EventsWithoutADomainEvent_PublishOnlyTheEnvelope(t *testing.T) {
	repoOnly := func(action string) []byte {
		payload, _ := json.Marshal(map[string]any{"action": action, "repository": map[string]any{"owner": map[string]any{"login": "acme"}, "name": "app"}})
		return payload
	}
	noRepo, _ := json.Marshal(map[string]any{"ref": "refs/heads/main"})
	tests := []struct {
		name    string
		event   string
		payload []byte
	}{
		{"a pull request action nothing listens for", "pull_request", prEventPayload("synchronize", false)},
		{"a pull request event without its pull request", "pull_request", repoOnly("opened")},
		{"a pushed tag", "push", pushEventPayload("refs/tags/v1.0.0", "before123", "after456", false)},
		{"a push without its repository", "push", noRepo},
		{"a review that is not submitted", "pull_request_review", reviewEventPayload("dismissed", "dismissed", "alice")},
		{"a review event without its review", "pull_request_review", repoOnly("submitted")},
		{"an issue action nothing listens for", "issues", issuesPayload("pinned", false)},
		{"an issues delivery for a pull request", "issues", issuesPayload("opened", true)},
		{"an issues delivery without its issue", "issues", repoOnly("opened")},
		{"a plain issue's comment action nothing listens for", "issue_comment", issueCommentPayloadWithAction("pinned", false, "hm", "bob")},
		{"an issue comment without its issue", "issue_comment", repoOnly("created")},
		{"a review comment without its pull request", "pull_request_review_comment", repoOnly("created")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bus := newFakeBus()
			rec := serveWebhook(t, NewWebhookHandler("shhh", bus), signedWebhookRequest(t, "shhh", tt.event, tt.payload))

			assert.Equal(t, http.StatusOK, rec.Code)
			require.Len(t, bus.published, 1)
			assert.Equal(t, TopicProviderEvent, bus.published[0].Topic)
		})
	}
}

func TestWebhookHandler_TicketResolverFails_Returns500(t *testing.T) {
	bus := newFakeBus()
	h := NewWebhookHandler("shhh", bus).WithTicketResolver(&fakeTicketResolver{err: errors.New("database is locked")})
	rec := serveWebhook(t, h, signedWebhookRequest(t, "shhh", "pull_request", prEventPayload("opened", false)))

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.NotEqual(t, TopicPROpened, bus.lastPublished().Topic, "no PR event goes out without its links")
}

func issuesPayload(action string, isPR bool) []byte {
	issue := map[string]any{
		"id":           int64(5001),
		"number":       12,
		"title":        "Login fails on Safari",
		"body":         "Steps to reproduce",
		"state":        "closed",
		"state_reason": "completed",
		"labels":       []any{map[string]any{"name": "bug"}},
		"assignees":    []any{map[string]any{"login": "bob"}},
		"user":         map[string]any{"login": "alice"},
		"html_url":     "https://github.com/acme/app/issues/12",
		"updated_at":   "2026-11-03T09:30:00Z",
	}
	if isPR {
		issue["pull_request"] = map[string]any{"url": "https://api.github.com/repos/acme/app/pulls/12"}
	}
	payload, _ := json.Marshal(map[string]any{
		"action":     action,
		"issue":      issue,
		"sender":     map[string]any{"login": "nexul-app[bot]", "type": "Bot"},
		"repository": map[string]any{"owner": map[string]any{"login": "acme"}, "name": "app"},
	})
	return payload
}

func TestWebhookHandler_Issues_PublishesGitIssueForEveryAction(t *testing.T) {
	for _, action := range []string{"opened", "edited", "closed", "reopened", "labeled", "unlabeled", "assigned", "unassigned", "deleted", "transferred"} {
		t.Run(action, func(t *testing.T) {
			bus := newFakeBus()
			rec := serveWebhook(t, NewWebhookHandler("shhh", bus), signedWebhookRequest(t, "shhh", "issues", issuesPayload(action, false)))

			assert.Equal(t, http.StatusOK, rec.Code)
			require.Len(t, bus.published, 2, "the envelope and one git.issue")
			ev := bus.lastPublished()
			assert.Equal(t, TopicIssue, ev.Topic)
			var got IssueEvent
			require.NoError(t, json.Unmarshal(ev.Payload, &got))
			assert.Equal(t, IssueEvent{
				Owner: "acme", Repo: "app", Action: action,
				Issue: Issue{
					ID: 5001, Number: 12, Title: "Login fails on Safari", Body: "Steps to reproduce",
					State: "closed", StateReason: "completed", Labels: []string{"bug"}, Assignees: []string{"bob"},
					Author: "alice", HTMLURL: "https://github.com/acme/app/issues/12",
					UpdatedAt: time.Date(2026, 11, 3, 9, 30, 0, 0, time.UTC),
				},
				Sender: Sender{Login: "nexul-app[bot]", Type: "Bot"},
			}, got)
		})
	}
}

func TestWebhookHandler_IssueWithoutLabelsOrAssignees_SendsEmptyLists(t *testing.T) {
	bus := newFakeBus()
	payload, _ := json.Marshal(map[string]any{
		"action":     "opened",
		"issue":      map[string]any{"number": 3, "title": "Bare", "state": "open"},
		"repository": map[string]any{"owner": map[string]any{"login": "acme"}, "name": "app"},
	})
	rec := serveWebhook(t, NewWebhookHandler("shhh", bus), signedWebhookRequest(t, "shhh", "issues", payload))

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, string(bus.lastPublished().Payload), `"labels":[],"assignees":[]`)
}

func TestWebhookHandler_IssueComment_PlainIssueIsGitIssueComment_PRStaysGitPRComment(t *testing.T) {
	tests := []struct {
		action string
		isPR   bool
		topic  string
	}{
		{"created", false, TopicIssueComment},
		{"edited", false, TopicIssueComment},
		{"deleted", false, TopicIssueComment},
		{"created", true, TopicPRCommentCreated},
		{"edited", true, TopicPRCommentCreated},
		{"deleted", true, TopicPRCommentCreated},
	}
	for _, tt := range tests {
		t.Run(tt.topic+" "+tt.action, func(t *testing.T) {
			bus := newFakeBus()
			payload := issueCommentPayloadWithAction(tt.action, tt.isPR, "Seen on 17.1 too", "bob")
			rec := serveWebhook(t, NewWebhookHandler("shhh", bus), signedWebhookRequest(t, "shhh", "issue_comment", payload))

			assert.Equal(t, http.StatusOK, rec.Code)
			require.Len(t, bus.published, 2)
			ev := bus.lastPublished()
			assert.Equal(t, tt.topic, ev.Topic)
			if tt.isPR {
				var got PRCommentEvent
				require.NoError(t, json.Unmarshal(ev.Payload, &got))
				assert.Equal(t, PRCommentEvent{Owner: "acme", Repo: "app", PR: PRCommentRef{Number: 7}, Comment: CommentRef{Body: "Seen on 17.1 too", Author: "bob"}}, got)
				return
			}
			var got IssueCommentEvent
			require.NoError(t, json.Unmarshal(ev.Payload, &got))
			assert.Equal(t, IssueCommentEvent{
				Owner: "acme", Repo: "app", IssueNumber: 7, Action: tt.action,
				Comment: IssueComment{
					ID: 99, Body: "Seen on 17.1 too", Author: "bob", AuthorAvatarURL: "https://avatars.example.com/bob",
					CreatedAt: time.Date(2026, 11, 1, 10, 0, 0, 0, time.UTC), UpdatedAt: time.Date(2026, 11, 2, 10, 0, 0, 0, time.UTC),
				},
				Sender: Sender{Login: "bob", Type: "User"},
			}, got)
		})
	}
}
