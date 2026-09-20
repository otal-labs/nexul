package codereview

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/ids"
)

// Service records are created and mutated only by git provider events; HTTP/MCP adapters are read-only (ADR 0034).
type Service struct {
	repo Repo
	now  func() time.Time
}

// NewService wires the codereview use-cases over the given repo.
func NewService(repo Repo) *Service {
	return &Service{repo: repo, now: time.Now}
}

// Create's unique (repo, number) constraint makes redelivered git.pr_opened events idempotent.
func (s *Service) Create(ctx context.Context, repo string, number int) (*CodeReview, error) {
	repo = strings.TrimSpace(repo)
	if repo == "" {
		return nil, fmt.Errorf("%w: repo is required", apperrs.ErrInvalid)
	}
	if number < 1 {
		return nil, fmt.Errorf("%w: pr number must be a positive integer", apperrs.ErrInvalid)
	}
	r := &CodeReview{
		ID:        ids.New(),
		PRNumber:  number,
		Repo:      repo,
		Status:    StatusPending,
		CreatedAt: s.now().UTC(),
	}
	if err := s.repo.Create(ctx, r); err != nil {
		return nil, fmt.Errorf("create review %s#%d: %w", repo, number, err)
	}
	return r, nil
}

// ApplyReviewSubmitted is the git.pr_review_submitted consumer; the latest review wins (ADR 0034).
func (s *Service) ApplyReviewSubmitted(ctx context.Context, repo string, number int, state, reviewer string) (*CodeReview, error) {
	repo = strings.TrimSpace(repo)
	if repo == "" {
		return nil, fmt.Errorf("%w: repo is required", apperrs.ErrInvalid)
	}
	if number < 1 {
		return nil, fmt.Errorf("%w: pr number must be a positive integer", apperrs.ErrInvalid)
	}
	review, err := s.byPR(ctx, repo, number)
	if err != nil {
		if !isNotFound(err) {
			return nil, err
		}
		created, err := s.Create(ctx, repo, number)
		if err != nil {
			return nil, err
		}
		review = created
	}
	switch state {
	case "approved":
		return s.setStatus(ctx, review, StatusApproved, strings.TrimSpace(reviewer))
	case "changes_requested":
		return s.setStatus(ctx, review, StatusChangesRequested, strings.TrimSpace(reviewer))
	default:
		// A commented or dismissed review deliberately leaves the aggregate alone, mirroring GitHub: an
		// approval has to survive later non-blocking activity or a PR stops reading as ready to merge.
		return review, nil
	}
}

// ApplyPRMerged is a no-op for unknown PRs, since nothing was mirrored.
func (s *Service) ApplyPRMerged(ctx context.Context, repo string, number int) (*CodeReview, error) {
	review, err := s.byPR(ctx, strings.TrimSpace(repo), number)
	if err != nil {
		if isNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return s.setStatus(ctx, review, StatusMerged, "")
}

// ApplyPRClosed moves an unmerged-close to closed so it never displays as awaiting review (ADR 0034).
func (s *Service) ApplyPRClosed(ctx context.Context, repo string, number int) (*CodeReview, error) {
	review, err := s.byPR(ctx, strings.TrimSpace(repo), number)
	if err != nil {
		if isNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return s.setStatus(ctx, review, StatusClosed, "")
}

// byPR returns the single review record for a pull request.
func (s *Service) byPR(ctx context.Context, repo string, number int) (*CodeReview, error) {
	rs, err := s.repo.ListByPR(ctx, repo, number)
	if err != nil {
		return nil, fmt.Errorf("get review %s#%d: %w", repo, number, err)
	}
	if len(rs) == 0 {
		return nil, fmt.Errorf("%w: no review for %s#%d", apperrs.ErrNotFound, repo, number)
	}
	return rs[0], nil
}

func isNotFound(err error) bool {
	return err != nil && errors.Is(err, apperrs.ErrNotFound)
}

// setStatus enqueues review.status_changed via the outbox in the same transaction.
func (s *Service) setStatus(ctx context.Context, r *CodeReview, status Status, reviewer string) (*CodeReview, error) {
	if r.Status == status {
		return r, nil
	}
	ev := StatusChangedEvent{ID: r.ID, Repo: r.Repo, PRNumber: r.PRNumber, Status: string(status), Reviewer: reviewer}
	evt := eventbus.OutboxEvent{ID: ids.New(), Topic: TopicStatusChanged, Payload: ev}
	if err := s.repo.UpdateStatus(ctx, r.ID, status, reviewer, evt); err != nil {
		return nil, fmt.Errorf("update review %s status: %w", r.ID, err)
	}
	r.Status = status
	r.Reviewer = reviewer
	return r, nil
}

// Get returns a single review record by id.
func (s *Service) Get(ctx context.Context, id string) (*CodeReview, error) {
	if strings.TrimSpace(id) == "" {
		return nil, fmt.Errorf("%w: id is required", apperrs.ErrInvalid)
	}
	r, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get review %s: %w", id, err)
	}
	return r, nil
}

// ListByTicket returns the review records for every PR linked to a ticket, oldest first.
func (s *Service) ListByTicket(ctx context.Context, ticketID string) ([]*CodeReview, error) {
	if strings.TrimSpace(ticketID) == "" {
		return nil, fmt.Errorf("%w: ticket id is required", apperrs.ErrInvalid)
	}
	rs, err := s.repo.ListByTicket(ctx, ticketID)
	if err != nil {
		return nil, fmt.Errorf("list reviews for ticket %s: %w", ticketID, err)
	}
	return rs, nil
}

// ListByPR returns the review records for a pull request (one per PR), oldest first; repo is the owner/name full name.
func (s *Service) ListByPR(ctx context.Context, repo string, number int) ([]*CodeReview, error) {
	repo = strings.TrimSpace(repo)
	if repo == "" {
		return nil, fmt.Errorf("%w: repo is required", apperrs.ErrInvalid)
	}
	if number < 1 {
		return nil, fmt.Errorf("%w: pr number must be a positive integer", apperrs.ErrInvalid)
	}
	rs, err := s.repo.ListByPR(ctx, repo, number)
	if err != nil {
		return nil, fmt.Errorf("list reviews for %s#%d: %w", repo, number, err)
	}
	return rs, nil
}
