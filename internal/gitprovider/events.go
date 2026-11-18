package gitprovider

import (
	"context"
	"encoding/json"

	apperrors "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
)

const (
	// TopicPROpened is published when a pull request is opened.
	TopicPROpened = "git.pr_opened"
	// TopicPRReviewSubmitted is published when a review is submitted (approved/changes_requested/commented/dismissed).
	TopicPRReviewSubmitted = "git.pr_review_submitted"
	// TopicPRMerged is published when a pull request is merged.
	TopicPRMerged = "git.pr_merged"
	// TopicPRClosed is published when a pull request is closed without merging.
	TopicPRClosed = "git.pr_closed"
	// TopicPush covers a push to an existing branch, both exact and wildcard rules; not a delete (see TopicBranchDeleted).
	TopicPush = "git.push"
	// TopicBranchDeleted is published when a branch is deleted, the one unambiguous teardown signal.
	TopicBranchDeleted = "git.branch_deleted"
	// TopicPRCommentCreated is published for both a general conversation comment and an inline review comment on the diff.
	TopicPRCommentCreated = "git.pr_comment"
	// TopicIssue is published for an issue opened, edited, closed, reopened, labeled, assigned, deleted or transferred; never a pull request.
	TopicIssue = "git.issue"
	// TopicIssueComment is published for a comment created, edited or deleted on an issue; a pull request's stays git.pr_comment.
	TopicIssueComment = "git.issue_comment"
)

// Topics returns every topic gitprovider publishes; TopicProviderEvent is declared in envelope.go but lives here.
func Topics() []eventbus.Topic {
	return []eventbus.Topic{
		{Name: TopicProviderEvent, Payload: ProviderEvent{}},
		{Name: TopicPROpened, Payload: PREvent{}},
		{Name: TopicPRReviewSubmitted, Payload: ReviewSubmittedEvent{}},
		{Name: TopicPRMerged, Payload: PREvent{}},
		{Name: TopicPRClosed, Payload: PREvent{}},
		{Name: TopicPush, Payload: PushEvent{}},
		{Name: TopicBranchDeleted, Payload: BranchDeletedEvent{}},
		{Name: TopicPRCommentCreated, Payload: PRCommentEvent{}},
		{Name: TopicIssue, Payload: IssueEvent{}},
		{Name: TopicIssueComment, Payload: IssueCommentEvent{}},
	}
}

// PREvent is the payload for git.pr_opened/pr_merged/pr_closed; fields are part of the contract, additive-only.
type PREvent struct {
	Owner string `json:"owner"`
	Repo  string `json:"repo"`
	PR    PR     `json:"pr"`
}

// ReviewSubmittedEvent is the git.pr_review_submitted payload: one review outcome (State: approved/etc).
type ReviewSubmittedEvent struct {
	Owner string    `json:"owner"`
	Repo  string    `json:"repo"`
	PR    ReviewRef `json:"pr"`
}

// ReviewRef identifies the pull request and the review outcome.
type ReviewRef struct {
	Number   int    `json:"number"`
	State    string `json:"state"`
	Reviewer string `json:"reviewer"`
}

// PushEvent is the git.push payload; SHA is the branch's new HEAD, the ref a branch deployment builds from.
type PushEvent struct {
	Owner  string `json:"owner"`
	Repo   string `json:"repo"`
	Branch string `json:"branch"`
	SHA    string `json:"sha"`
	Pusher string `json:"pusher,omitempty"`
}

// BranchDeletedEvent is the payload for git.branch_deleted.
type BranchDeletedEvent struct {
	Owner  string `json:"owner"`
	Repo   string `json:"repo"`
	Branch string `json:"branch"`
}

// PRCommentEvent is the payload for git.pr_comment.
type PRCommentEvent struct {
	Owner   string       `json:"owner"`
	Repo    string       `json:"repo"`
	PR      PRCommentRef `json:"pr"`
	Comment CommentRef   `json:"comment"`
}

// PRCommentRef identifies the commented-on pull request.
type PRCommentRef struct {
	Number int `json:"number"`
}

// CommentRef is the comment body and author, shared by both general and inline review comments.
type CommentRef struct {
	Body   string `json:"body"`
	Author string `json:"author"`
}

// IssueEvent is the git.issue payload.
type IssueEvent struct {
	Owner  string `json:"owner"`
	Repo   string `json:"repo"`
	Action string `json:"action" enum:"opened,edited,closed,reopened,labeled,unlabeled,assigned,unassigned,deleted,transferred"`
	Issue  Issue  `json:"issue"`
	Sender Sender `json:"sender"`
}

// IssueCommentEvent is the git.issue_comment payload.
type IssueCommentEvent struct {
	Owner       string       `json:"owner"`
	Repo        string       `json:"repo"`
	IssueNumber int          `json:"issue_number"`
	Action      string       `json:"action" enum:"created,edited,deleted"`
	Comment     IssueComment `json:"comment"`
	Sender      Sender       `json:"sender"`
}

// Sender is who caused a delivery; Type is the host's account type, such as User or Bot.
type Sender struct {
	Login string `json:"login"`
	Type  string `json:"type" jsonschema:"The account type, such as User, Bot or Organization."`
}

// TicketLinker lets gitprovider declare LinkPR without importing tickets, which implements it.
type TicketLinker interface {
	LinkPR(ctx context.Context, ticketID string, ref PRRef) error
}

// HandlePROpened links the PR to every ticket it references.
func HandlePROpened(ctx context.Context, linker TicketLinker, ev eventbus.Event) error {
	e, err := unmarshalPREvent(ev)
	if err != nil {
		return err
	}
	for _, id := range e.PR.LinkedTicketIDs {
		if err := linker.LinkPR(ctx, id, refFromEvent(e)); err != nil {
			return apperrors.Retryable(err)
		}
	}
	return nil
}

func unmarshalPREvent(ev eventbus.Event) (PREvent, error) {
	var e PREvent
	if err := json.Unmarshal(ev.Payload, &e); err != nil {
		return PREvent{}, apperrors.Fatal(err)
	}
	return e, nil
}

func refFromEvent(e PREvent) PRRef {
	return PRRef{Owner: e.Owner, Repo: e.Repo, Number: e.PR.Number, Title: e.PR.Title, SHA: e.PR.HeadSHA}
}
