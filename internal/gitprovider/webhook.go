package gitprovider

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	githubapi "github.com/google/go-github/v71/github"

	"github.com/otal-labs/nexul/internal/platform/logging"
)

// providerGitHub identifies the GitHub adapter in published envelopes.
const providerGitHub = "github"

// Publisher is the slice of the event bus the receiver needs (ADR 0019); every accepted delivery is published fan-out.
type Publisher interface {
	Publish(ctx context.Context, topic string, payload any) error
}

// WebhookHandler validates GitHub signatures and publishes a normalized envelope, plus typed git.pr_* topics.
type WebhookHandler struct {
	secret []byte
	bus    Publisher
}

// NewWebhookHandler wires a webhook receiver; an empty secret skips signature checks (dev only) and logs a warning.
func NewWebhookHandler(secret string, bus Publisher) *WebhookHandler {
	return &WebhookHandler{secret: []byte(secret), bus: bus}
}

func (h *WebhookHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromCtx(r.Context())
	if len(h.secret) == 0 {
		logger.Warn("webhook receiver has no secret; requests are not authenticated")
	}
	payload, err := githubapi.ValidatePayload(r, h.secret)
	if err != nil {
		logger.Warn("webhook signature validation failed", "error", err)
		http.Error(w, "invalid webhook signature", http.StatusBadRequest)
		return
	}
	eventType := githubapi.WebHookType(r)
	if !json.Valid(payload) {
		logger.Warn("webhook body is not valid JSON", "event_type", eventType)
		http.Error(w, "invalid webhook payload", http.StatusBadRequest)
		return
	}

	// Normalize into the envelope before dispatch, so unknown types reach the bus too; failure returns non-2xx.
	if err := h.publish(r.Context(), TopicProviderEvent, h.normalizeProviderEvent(eventType, githubapi.DeliveryID(r), payload)); err != nil {
		h.failDispatch(r.Context(), w, err)
		return
	}

	switch eventType {
	case "pull_request":
		h.dispatchPullRequest(r.Context(), w, payload)
	case "pull_request_review":
		h.dispatchReview(r.Context(), w, payload)
	case "issue_comment":
		h.dispatchIssueComment(r.Context(), w, payload)
	case "pull_request_review_comment":
		h.dispatchReviewComment(r.Context(), w, payload)
	case "push":
		h.dispatchPush(r.Context(), w, payload)
	default:
		w.WriteHeader(http.StatusOK)
	}
}

// normalizeProviderEvent builds the provider-agnostic envelope; GitHub's vocabulary stays inside this receiver.
func (h *WebhookHandler) normalizeProviderEvent(eventType, deliveryID string, body []byte) *ProviderEvent {
	var meta struct {
		Action     string         `json:"action"`
		Repository repositoryBody `json:"repository"`
	}
	// Extraction is best-effort: a delivery missing action/repository still publishes, fields empty, rather than failing.
	_ = json.Unmarshal(body, &meta)

	var repo *RepositoryRef
	if meta.Repository.Name != "" {
		repo = &RepositoryRef{
			ID:            meta.Repository.ID,
			Name:          meta.Repository.Name,
			FullName:      meta.Repository.FullName,
			Owner:         meta.Repository.Owner.Login,
			HTMLURL:       meta.Repository.HTMLURL,
			DefaultBranch: meta.Repository.DefaultBranch,
		}
	}
	return &ProviderEvent{
		Provider:   providerGitHub,
		EventType:  eventType,
		DeliveryID: deliveryID,
		Action:     meta.Action,
		Repository: repo,
		Payload:    body,
		ReceivedAt: time.Now().UTC(),
	}
}

// repositoryBody mirrors GitHub's embedded repository object; only RepositoryRef crosses the published contract.
type repositoryBody struct {
	ID            int64               `json:"id"`
	Name          string              `json:"name"`
	FullName      string              `json:"full_name"`
	Owner         repositoryBodyOwner `json:"owner"`
	HTMLURL       string              `json:"html_url"`
	DefaultBranch string              `json:"default_branch"`
}

type repositoryBodyOwner struct {
	Login string `json:"login"`
}

// dispatchPullRequest maps opened, merged close, and unmerged close to a git.pr_* event.
func (h *WebhookHandler) dispatchPullRequest(ctx context.Context, w http.ResponseWriter, payload []byte) {
	logger := logging.FromCtx(ctx)
	event, err := githubapi.ParseWebHook("pull_request", payload)
	if err != nil {
		logger.Warn("webhook payload parse failed", "error", err)
		http.Error(w, "invalid webhook payload", http.StatusBadRequest)
		return
	}
	prEvent, ok := event.(*githubapi.PullRequestEvent)
	if !ok {
		logger.Warn("webhook delivered an unexpected pull_request payload")
		http.Error(w, "unexpected webhook payload", http.StatusBadRequest)
		return
	}
	ev, ok := mapPREvent(prEvent)
	if !ok {
		w.WriteHeader(http.StatusOK)
		return
	}
	switch {
	case prEvent.GetAction() == "opened":
		if err := h.publish(ctx, TopicPROpened, ev); err != nil {
			h.failDispatch(ctx, w, err)
			return
		}
	case prEvent.GetAction() == "closed" && prEvent.GetPullRequest().GetMerged():
		if err := h.publish(ctx, TopicPRMerged, ev); err != nil {
			h.failDispatch(ctx, w, err)
			return
		}
	case prEvent.GetAction() == "closed":
		if err := h.publish(ctx, TopicPRClosed, ev); err != nil {
			h.failDispatch(ctx, w, err)
			return
		}
	}
	w.WriteHeader(http.StatusOK)
}

// dispatchReview handles a pull_request_review delivery: a submitted review becomes git.pr_review_submitted.
func (h *WebhookHandler) dispatchReview(ctx context.Context, w http.ResponseWriter, payload []byte) {
	logger := logging.FromCtx(ctx)
	event, err := githubapi.ParseWebHook("pull_request_review", payload)
	if err != nil {
		logger.Warn("webhook payload parse failed", "error", err)
		http.Error(w, "invalid webhook payload", http.StatusBadRequest)
		return
	}
	reviewEvent, ok := event.(*githubapi.PullRequestReviewEvent)
	if !ok {
		logger.Warn("webhook delivered an unexpected pull_request_review payload")
		http.Error(w, "unexpected webhook payload", http.StatusBadRequest)
		return
	}
	ev, ok := mapReviewEvent(reviewEvent)
	if !ok {
		w.WriteHeader(http.StatusOK)
		return
	}
	if err := h.publish(ctx, TopicPRReviewSubmitted, ev); err != nil {
		h.failDispatch(ctx, w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// dispatchIssueComment maps only a PR comment, not a plain issue, to git.pr_comment (Issue.IsPullRequest).
func (h *WebhookHandler) dispatchIssueComment(ctx context.Context, w http.ResponseWriter, payload []byte) {
	logger := logging.FromCtx(ctx)
	event, err := githubapi.ParseWebHook("issue_comment", payload)
	if err != nil {
		logger.Warn("webhook payload parse failed", "error", err)
		http.Error(w, "invalid webhook payload", http.StatusBadRequest)
		return
	}
	commentEvent, ok := event.(*githubapi.IssueCommentEvent)
	if !ok {
		logger.Warn("webhook delivered an unexpected issue_comment payload")
		http.Error(w, "unexpected webhook payload", http.StatusBadRequest)
		return
	}
	ev, ok := mapIssueCommentEvent(commentEvent)
	if !ok {
		w.WriteHeader(http.StatusOK)
		return
	}
	if err := h.publish(ctx, TopicPRCommentCreated, ev); err != nil {
		h.failDispatch(ctx, w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// dispatchReviewComment maps an inline diff comment to git.pr_comment too, same topic as a conversation comment.
func (h *WebhookHandler) dispatchReviewComment(ctx context.Context, w http.ResponseWriter, payload []byte) {
	logger := logging.FromCtx(ctx)
	event, err := githubapi.ParseWebHook("pull_request_review_comment", payload)
	if err != nil {
		logger.Warn("webhook payload parse failed", "error", err)
		http.Error(w, "invalid webhook payload", http.StatusBadRequest)
		return
	}
	commentEvent, ok := event.(*githubapi.PullRequestReviewCommentEvent)
	if !ok {
		logger.Warn("webhook delivered an unexpected pull_request_review_comment payload")
		http.Error(w, "unexpected webhook payload", http.StatusBadRequest)
		return
	}
	ev, ok := mapReviewCommentEvent(commentEvent)
	if !ok {
		w.WriteHeader(http.StatusOK)
		return
	}
	if err := h.publish(ctx, TopicPRCommentCreated, ev); err != nil {
		h.failDispatch(ctx, w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// branchRefPrefix is the git ref prefix for branches; a tag push carries no branch and is left as envelope-only.
const branchRefPrefix = "refs/heads/"

// dispatchPush maps a push to an existing branch to git.push, a deleted branch to git.branch_deleted.
func (h *WebhookHandler) dispatchPush(ctx context.Context, w http.ResponseWriter, payload []byte) {
	logger := logging.FromCtx(ctx)
	event, err := githubapi.ParseWebHook("push", payload)
	if err != nil {
		logger.Warn("webhook payload parse failed", "error", err)
		http.Error(w, "invalid webhook payload", http.StatusBadRequest)
		return
	}
	pushEvent, ok := event.(*githubapi.PushEvent)
	if !ok {
		logger.Warn("webhook delivered an unexpected push payload")
		http.Error(w, "unexpected webhook payload", http.StatusBadRequest)
		return
	}
	branch, ok := strings.CutPrefix(pushEvent.GetRef(), branchRefPrefix)
	if !ok || pushEvent.Repo == nil {
		w.WriteHeader(http.StatusOK)
		return
	}
	owner := pushEvent.Repo.GetOwner().GetLogin()
	repo := pushEvent.Repo.GetName()
	if pushEvent.GetDeleted() {
		if err := h.publish(ctx, TopicBranchDeleted, BranchDeletedEvent{Owner: owner, Repo: repo, Branch: branch}); err != nil {
			h.failDispatch(ctx, w, err)
			return
		}
		w.WriteHeader(http.StatusOK)
		return
	}
	ev := PushEvent{Owner: owner, Repo: repo, Branch: branch, SHA: pushEvent.GetAfter(), Pusher: pushEvent.GetPusher().GetName()}
	if err := h.publish(ctx, TopicPush, ev); err != nil {
		h.failDispatch(ctx, w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// publish wraps a bus publish so transient failures bubble up.
func (h *WebhookHandler) publish(ctx context.Context, topic string, payload any) error {
	return h.bus.Publish(ctx, topic, payload)
}

// failDispatch reports a failed dispatch; 500 makes GitHub redeliver so a transient bus failure doesn't drop it.
func (h *WebhookHandler) failDispatch(ctx context.Context, w http.ResponseWriter, err error) {
	logging.FromCtx(ctx).Error("webhook event dispatch failed", "error", err)
	http.Error(w, "event dispatch failed", http.StatusInternalServerError)
}

func mapPREvent(e *githubapi.PullRequestEvent) (PREvent, bool) {
	if e.Repo == nil || e.PullRequest == nil {
		return PREvent{}, false
	}
	p := e.PullRequest
	return PREvent{
		Owner: e.Repo.GetOwner().GetLogin(),
		Repo:  e.Repo.GetName(),
		PR: PR{
			Number:          p.GetNumber(),
			Title:           p.GetTitle(),
			Body:            p.GetBody(),
			State:           PRState(p.GetState()),
			HeadSHA:         p.GetHead().GetSHA(),
			BaseBranch:      p.GetBase().GetRef(),
			Author:          p.GetUser().GetLogin(),
			LinkedTicketIDs: LinkedTicketIDs(p.GetBody(), p.GetHead().GetRef()),
		},
	}, true
}

func mapReviewEvent(e *githubapi.PullRequestReviewEvent) (ReviewSubmittedEvent, bool) {
	if e.Repo == nil || e.PullRequest == nil || e.Review == nil {
		return ReviewSubmittedEvent{}, false
	}
	if e.GetAction() != "submitted" {
		return ReviewSubmittedEvent{}, false
	}
	return ReviewSubmittedEvent{
		Owner: e.Repo.GetOwner().GetLogin(),
		Repo:  e.Repo.GetName(),
		PR: ReviewRef{
			Number:   e.GetPullRequest().GetNumber(),
			State:    e.Review.GetState(),
			Reviewer: e.Review.GetUser().GetLogin(),
		},
	}, true
}

func mapIssueCommentEvent(e *githubapi.IssueCommentEvent) (PRCommentEvent, bool) {
	if e.Repo == nil || e.Issue == nil || e.Comment == nil || !e.Issue.IsPullRequest() {
		return PRCommentEvent{}, false
	}
	return PRCommentEvent{
		Owner:   e.Repo.GetOwner().GetLogin(),
		Repo:    e.Repo.GetName(),
		PR:      PRCommentRef{Number: e.Issue.GetNumber()},
		Comment: CommentRef{Body: e.Comment.GetBody(), Author: e.Comment.GetUser().GetLogin()},
	}, true
}

func mapReviewCommentEvent(e *githubapi.PullRequestReviewCommentEvent) (PRCommentEvent, bool) {
	if e.Repo == nil || e.PullRequest == nil || e.Comment == nil {
		return PRCommentEvent{}, false
	}
	return PRCommentEvent{
		Owner:   e.Repo.GetOwner().GetLogin(),
		Repo:    e.Repo.GetName(),
		PR:      PRCommentRef{Number: e.PullRequest.GetNumber()},
		Comment: CommentRef{Body: e.Comment.GetBody(), Author: e.Comment.GetUser().GetLogin()},
	}, true
}
