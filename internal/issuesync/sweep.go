package issuesync

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// SweepEvery is how often every enabled sync lists the issues changed since its last sweep.
const SweepEvery = 15 * time.Minute

// RunSweepLoop backfills each sync once it is turned on, woken by the commit that stores it, and sweeps every
// enabled sync at start and every SweepEvery.
func (s *Service) RunSweepLoop(ctx context.Context, wake func() <-chan struct{}, logger *slog.Logger) {
	// GitHub never redelivers a missed webhook, so polling it is the only way to catch one; that is the timer's one job.
	ticker := time.NewTicker(SweepEvery)
	defer ticker.Stop()
	all := true
	for {
		committed := wake()
		if err := s.Sweep(ctx, all); err != nil {
			logger.Warn("github issue sweep", "error", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-committed:
			all = false
		case <-ticker.C:
			all = true
		}
	}
}

// Sweep backfills every enabled sync not yet backfilled and, with all, lists every other one's issues changed since
// its last sweep; one repository's failure leaves the others to run.
func (s *Service) Sweep(ctx context.Context, all bool) error {
	syncs, err := s.d.Repo.ListEnabledSyncs(ctx)
	if err != nil {
		return err
	}
	var errs []error
	for _, sy := range syncs {
		if sy.LastSweptAt != nil && !all {
			continue
		}
		if err := s.sweep(ctx, sy); err != nil {
			errs = append(errs, fmt.Errorf("%s/%s: %w", sy.Owner, sy.Repo, err))
		}
	}
	return errors.Join(errs...)
}

// sweep lists one sync's issues page by page: the open ones it matches on the first run, then every issue changed
// since the last. It records the sweep only once every page is read, so a pause or a failure retries from the same point.
func (s *Service) sweep(ctx context.Context, sy Sync) error {
	started := s.d.Now().UTC()
	q := IssueQuery{State: "all"}
	backfill := sy.LastSweptAt == nil
	if !backfill {
		q.Since = *sy.LastSweptAt
	}
	if backfill {
		q.State = "open"
	}
	if backfill && sy.Mode == ModeLabel {
		q.Labels = []string{sy.Label}
	}
	for q.Page = 1; q.Page != 0; {
		resting, err := s.rest(ctx, sy.Owner, sy.Repo)
		if err != nil || resting {
			return err
		}
		issues, next, err := s.d.Issues.ListIssues(ctx, sy.Owner, sy.Repo, q)
		if err != nil {
			s.restOnLimit(sy.Owner, err)
			return err
		}
		for _, i := range issues {
			if err := s.sweepIssue(ctx, sy, i, q.Since); err != nil {
				return err
			}
		}
		q.Page = next
	}
	return s.d.Repo.SetSwept(ctx, sy.Owner, sy.Repo, started)
}

// sweepIssue applies one listed issue and, on a sweep after the backfill, posts the comments its webhooks missed.
func (s *Service) sweepIssue(ctx context.Context, sy Sync, i Issue, since time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.apply(ctx, sy, i, ""); err != nil {
		return err
	}
	if since.IsZero() {
		return nil
	}
	link, err := s.d.Repo.GetLink(ctx, sy.Owner, sy.Repo, i.Number)
	if errors.Is(err, apperrs.ErrNotFound) {
		return nil
	}
	if err != nil || link.Paused {
		return err
	}
	comments, err := s.d.Issues.ListComments(ctx, sy.Owner, sy.Repo, i.Number, since)
	if err != nil {
		s.restOnLimit(sy.Owner, err)
		return err
	}
	for _, c := range comments {
		mine, err := s.fromBot(ctx, c.Author)
		if err != nil {
			return err
		}
		if mine {
			continue
		}
		// shortcut: the sweep adds missed comments only; an edit missed with its webhook waits for the next edit.
		if err := s.comment(ctx, link, "created", c); err != nil {
			return err
		}
	}
	return nil
}

// rest reports whether the sweep waits for owner's rate limit to reset, starting the wait once fewer than rateFloor
// requests are left.
func (s *Service) rest(ctx context.Context, owner, repo string) (bool, error) {
	if !s.pace.restingUntil(owner, s.d.Now()).IsZero() {
		return true, nil
	}
	rate, err := s.d.Issues.RateLimit(ctx, owner, repo)
	if err != nil {
		s.restOnLimit(owner, err)
		return false, err
	}
	if rate.Remaining >= rateFloor {
		return false, nil
	}
	s.pace.rest(owner, rate.Reset)
	return true, nil
}

// restOnLimit starts owner's wait when GitHub refused a call for its rate limit, and reports whether it did.
func (s *Service) restOnLimit(owner string, err error) bool {
	var limited *apperrs.RetryAfterError
	if !errors.As(err, &limited) {
		return false
	}
	s.pace.rest(owner, s.d.Now().Add(limited.Wait))
	return true
}
