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
			"title":  "Fix login",
			"body":   "Fixes #42",
			"state":  "open",
			"merged": merged,
			"head":   map[string]any{"ref": "ticket/42-fix-login", "sha": "abc123"},
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
	h := NewWebhookHandler("shhh", bus)
	rec := serveWebhook(t, h, signedWebhookRequest(t, "shhh", "pull_request", prEventPayload("opened", false)))

	assert.Equal(t, http.StatusOK, rec.Code)
	ev := bus.lastPublished()
	assert.Equal(t, TopicPROpened, ev.Topic)
	var got PREvent
	require.NoError(t, json.Unmarshal(ev.Payload, &got))
	assert.Equal(t, "acme", got.Owner)
	assert.Equal(t, "app", got.Repo)
	assert.Equal(t, 7, got.PR.Number)
	assert.Equal(t, "Fixes #42", got.PR.Body)
	assert.Equal(t, []string{"42"}, got.PR.LinkedTicketIDs)
	assert.Equal(t, "abc123", got.PR.HeadSHA)
	assert.Equal(t, "main", got.PR.BaseBranch)
	assert.Equal(t, "onik97", got.PR.Author)
}

func TestWebhookHandler_Merged(t *testing.T) {
	bus := newFakeBus()
	h := NewWebhookHandler("shhh", bus)
	rec := serveWebhook(t, h, signedWebhookRequest(t, "shhh", "pull_request", prEventPayload("closed", true)))

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, TopicPRMerged, bus.lastPublished().Topic)
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

func TestWebhookHandler_OtherPRAction_PublishesOnlyEnvelope(t *testing.T) {
	bus := newFakeBus()
	h := NewWebhookHandler("shhh", bus)
	rec := serveWebhook(t, h, signedWebhookRequest(t, "shhh", "pull_request", prEventPayload("synchronize", false)))

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Len(t, bus.published, 1)
	assert.Equal(t, TopicProviderEvent, bus.published[0].Topic)
	env := envelopeFrom(t, bus)
	assert.Equal(t, "synchronize", env.Action)
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

func TestWebhookHandler_PushTag_PublishesOnlyEnvelope(t *testing.T) {
	bus := newFakeBus()
	h := NewWebhookHandler("shhh", bus)
	rec := serveWebhook(t, h, signedWebhookRequest(t, "shhh", "push", pushEventPayload("refs/tags/v1.0.0", "before123", "after456", false)))

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Len(t, bus.published, 1)
	assert.Equal(t, TopicProviderEvent, bus.published[0].Topic)
}

func TestWebhookHandler_PushMissingRepository_PublishesOnlyEnvelope(t *testing.T) {
	bus := newFakeBus()
	h := NewWebhookHandler("shhh", bus)
	payload, _ := json.Marshal(map[string]any{"ref": "refs/heads/main"})
	rec := serveWebhook(t, h, signedWebhookRequest(t, "shhh", "push", payload))

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Len(t, bus.published, 1)
	assert.Equal(t, TopicProviderEvent, bus.published[0].Topic)
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

func TestWebhookHandler_ReviewChangesRequested(t *testing.T) {
	bus := newFakeBus()
	h := NewWebhookHandler("shhh", bus)
	rec := serveWebhook(t, h, signedWebhookRequest(t, "shhh", "pull_request_review", reviewEventPayload("submitted", "changes_requested", "bob")))

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, TopicPRReviewSubmitted, bus.lastPublished().Topic)
}

func TestWebhookHandler_ReviewNonSubmittedAction_PublishesOnlyEnvelope(t *testing.T) {
	bus := newFakeBus()
	h := NewWebhookHandler("shhh", bus)
	rec := serveWebhook(t, h, signedWebhookRequest(t, "shhh", "pull_request_review", reviewEventPayload("dismissed", "dismissed", "alice")))

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Len(t, bus.published, 1)
	assert.Equal(t, TopicProviderEvent, bus.published[0].Topic)
}

func TestWebhookHandler_ReviewInvalidSignature_Rejected(t *testing.T) {
	bus := newFakeBus()
	h := NewWebhookHandler("shhh", bus)
	rec := serveWebhook(t, h, signedWebhookRequest(t, "wrong-secret", "pull_request_review", reviewEventPayload("submitted", "approved", "alice")))

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Empty(t, bus.published)
}

func TestWebhookHandler_ReviewMissingReviewPayload_PublishesOnlyEnvelope(t *testing.T) {
	bus := newFakeBus()
	h := NewWebhookHandler("shhh", bus)
	payload, _ := json.Marshal(map[string]any{"action": "submitted", "repository": map[string]any{"owner": map[string]any{"login": "acme"}, "name": "app"}})
	rec := serveWebhook(t, h, signedWebhookRequest(t, "shhh", "pull_request_review", payload))

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Len(t, bus.published, 1)
	assert.Equal(t, TopicProviderEvent, bus.published[0].Topic)
}

func TestWebhookHandler_PublishFailure_Returns500(t *testing.T) {
	bus := newFakeBus()
	bus.publishErr = errors.New("bus closed")
	h := NewWebhookHandler("shhh", bus)
	rec := serveWebhook(t, h, signedWebhookRequest(t, "shhh", "pull_request", prEventPayload("opened", false)))

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func issueCommentPayload(isPR bool, body, author string) []byte {
	issue := map[string]any{"number": 7}
	if isPR {
		issue["pull_request"] = map[string]any{"url": "https://api.github.com/repos/acme/app/pulls/7"}
	}
	payload, _ := json.Marshal(map[string]any{
		"action": "created",
		"issue":  issue,
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

func TestWebhookHandler_IssueCommentOnPlainIssue_PublishesOnlyEnvelope(t *testing.T) {
	bus := newFakeBus()
	h := NewWebhookHandler("shhh", bus)
	rec := serveWebhook(t, h, signedWebhookRequest(t, "shhh", "issue_comment", issueCommentPayload(false, "nice catch", "onik97")))

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Len(t, bus.published, 1)
	assert.Equal(t, TopicProviderEvent, bus.published[0].Topic)
}

func TestWebhookHandler_IssueCommentMissingIssuePayload_PublishesOnlyEnvelope(t *testing.T) {
	bus := newFakeBus()
	h := NewWebhookHandler("shhh", bus)
	payload, _ := json.Marshal(map[string]any{"action": "created", "repository": map[string]any{"owner": map[string]any{"login": "acme"}, "name": "app"}})
	rec := serveWebhook(t, h, signedWebhookRequest(t, "shhh", "issue_comment", payload))

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Len(t, bus.published, 1)
	assert.Equal(t, TopicProviderEvent, bus.published[0].Topic)
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

func TestWebhookHandler_ReviewCommentMissingPullRequestPayload_PublishesOnlyEnvelope(t *testing.T) {
	bus := newFakeBus()
	h := NewWebhookHandler("shhh", bus)
	payload, _ := json.Marshal(map[string]any{"action": "created", "repository": map[string]any{"owner": map[string]any{"login": "acme"}, "name": "app"}})
	rec := serveWebhook(t, h, signedWebhookRequest(t, "shhh", "pull_request_review_comment", payload))

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Len(t, bus.published, 1)
	assert.Equal(t, TopicProviderEvent, bus.published[0].Topic)
}

func TestWebhookHandler_MissingPullRequestPayload_PublishesOnlyEnvelope(t *testing.T) {
	bus := newFakeBus()
	h := NewWebhookHandler("shhh", bus)
	payload, _ := json.Marshal(map[string]any{"action": "opened", "repository": map[string]any{"owner": map[string]any{"login": "acme"}, "name": "app"}})
	rec := serveWebhook(t, h, signedWebhookRequest(t, "shhh", "pull_request", payload))

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Len(t, bus.published, 1)
	assert.Equal(t, TopicProviderEvent, bus.published[0].Topic)
}
