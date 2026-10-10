package docs

import (
	"context"
	"log/slog"
	"time"

	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/ids"
)

// SettleWindow is how long a person's edits to a doc must stop before doc.settled fires.
const SettleWindow = 10 * time.Minute

// settleRetry is how soon a failed settle is tried again.
const settleRetry = 5 * time.Second

// SettleRepo holds the open settle windows, one per doc a person created or edited.
type SettleRepo interface {
	// NextSettleDue is when the earliest open window closes; false when none is open.
	NextSettleDue(ctx context.Context) (time.Time, bool, error)
	// SettleDue closes every window due by now, writing build's events for those whose doc is not archived, in one transaction.
	SettleDue(ctx context.Context, now time.Time, build func([]SettledEvent) []eventbus.OutboxEvent) error
}

// RunSettleLoop publishes doc.settled as windows close, sleeping until the earliest one or a commit, until ctx ends.
func RunSettleLoop(ctx context.Context, repo SettleRepo, wake func() <-chan struct{}, logger *slog.Logger) {
	for {
		committed := wake()
		wait, err := settle(ctx, repo)
		if err != nil {
			logger.Error("settle docs", "error", err)
			wait = settleRetry
		}
		if !sleep(ctx, wait, committed) {
			return
		}
	}
}

// settle publishes the windows already closed and returns how long until the next one closes, or -1 when none is open.
func settle(ctx context.Context, repo SettleRepo) (time.Duration, error) {
	due, ok, err := repo.NextSettleDue(ctx)
	if err != nil {
		return 0, err
	}
	if !ok {
		return -1, nil
	}
	now := time.Now()
	if due.After(now) {
		return due.Sub(now), nil
	}
	return 0, repo.SettleDue(ctx, now, settledEvents)
}

// sleep waits out wait (forever when negative) or a commit, and reports false once ctx ends.
func sleep(ctx context.Context, wait time.Duration, committed <-chan struct{}) bool {
	var timeout <-chan time.Time
	if wait >= 0 {
		timer := time.NewTimer(wait)
		defer timer.Stop()
		timeout = timer.C
	}
	select {
	case <-ctx.Done():
		return false
	case <-timeout:
	case <-committed:
	}
	return true
}

func settledEvents(settled []SettledEvent) []eventbus.OutboxEvent {
	out := make([]eventbus.OutboxEvent, 0, len(settled))
	for _, s := range settled {
		out = append(out, eventbus.OutboxEvent{ID: ids.New(), Topic: TopicSettled, Payload: s})
	}
	return out
}
