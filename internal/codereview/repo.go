package codereview

import (
	"context"

	"github.com/otal-labs/nexul/internal/platform/eventbus"
)

// Repo's review records are mirrored state fed by git provider webhook events (ADR 0034).
type Repo interface {
	Create(ctx context.Context, r *CodeReview) error
	Get(ctx context.Context, id string) (*CodeReview, error)
	UpdateStatus(ctx context.Context, id string, status Status, reviewer string, evts ...eventbus.OutboxEvent) error
	ListByTicket(ctx context.Context, ticketID string) ([]*CodeReview, error)
	ListByPR(ctx context.Context, repo string, number int) ([]*CodeReview, error)
}
