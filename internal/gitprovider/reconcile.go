package gitprovider

import (
	"context"
	"log/slog"
	"time"
)

// reconcileInterval is how stale a linked PR's state may get when the provider's webhook never arrives.
const reconcileInterval = 2 * time.Minute

// OpenPRLister lists the pull requests Nexul still records as open, declared here so gitprovider never imports tickets.
type OpenPRLister interface {
	ListOpenPRs(ctx context.Context) ([]PRRef, error)
}

// PRReader is the slice of GitProvider the reconciler needs.
type PRReader interface {
	GetPR(ctx context.Context, owner, name string, number int) (*PR, error)
}

// RunPRReconciliation asks the git host about every PR still recorded as open and publishes the merge or close a
// missed webhook would have, until ctx is done.
// ponytail: one GetPR per open PR per tick; batch through the search API if open links reach the hundreds.
func RunPRReconciliation(ctx context.Context, open OpenPRLister, prs PRReader, bus Publisher, logger *slog.Logger) {
	ticker := time.NewTicker(reconcileInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			reconcilePRsOnce(ctx, open, prs, bus, logger)
		}
	}
}

func reconcilePRsOnce(ctx context.Context, open OpenPRLister, prs PRReader, bus Publisher, logger *slog.Logger) {
	refs, err := open.ListOpenPRs(ctx)
	if err != nil {
		logger.Warn("pr reconciliation: list open prs failed", "error", err)
		return
	}
	for _, ref := range refs {
		pr, err := prs.GetPR(ctx, ref.Owner, ref.Repo, ref.Number)
		if err != nil {
			logger.Warn("pr reconciliation: get pr failed", "owner", ref.Owner, "repo", ref.Repo, "number", ref.Number, "error", err)
			continue
		}
		topic := closedTopic(pr)
		if topic == "" {
			continue
		}
		if err := bus.Publish(ctx, topic, PREvent{Owner: ref.Owner, Repo: ref.Repo, PR: *pr}); err != nil {
			logger.Warn("pr reconciliation: publish failed", "topic", topic, "owner", ref.Owner, "repo", ref.Repo, "number", ref.Number, "error", err)
		}
	}
}

// closedTopic is the topic a webhook would have published for pr, or empty while it is still open.
func closedTopic(pr *PR) string {
	if pr.Merged {
		return TopicPRMerged
	}
	if pr.State == PRStateClosed {
		return TopicPRClosed
	}
	return ""
}
