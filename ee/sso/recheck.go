package sso

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"
)

const (
	// recheckBatch is how many grants one read returns.
	recheckBatch = 50
	// recheckAfter is how stale a grant must be to be asked about again: half the hourly tick, so a grant confirmed
	// at the last tick is due at this one.
	recheckAfter = 30 * time.Minute
)

// recheck asks the provider about every person whose grant was last confirmed long enough ago, and ends the sessions
// of those it refuses; it returns how many sessions ended. A provider that cannot answer, or rejects this instance's
// client, changes nothing and stops the pass with an error: its being down must never sign anyone out.
func (s *Service) recheck(ctx context.Context, log *slog.Logger) (int, error) {
	st, err := s.cfg.Repo.Get(ctx)
	if err != nil {
		return 0, err
	}
	if !st.Configured() || !s.unlocked(ctx) {
		return 0, nil
	}
	secret, err := s.clientSecret(ctx, st.ClientSecret)
	if err != nil {
		return 0, err
	}
	idp, err := s.connect(ctx, st, secret, false)
	if err != nil {
		return 0, err
	}
	ended := 0
	for {
		now := s.cfg.Now()
		grants, err := s.cfg.Repo.StaleGrants(ctx, now.Add(-recheckAfter), recheckBatch)
		if err != nil || len(grants) == 0 {
			return ended, err
		}
		for _, g := range grants {
			n, err := s.recheckOne(ctx, idp, g, now, log)
			ended += n
			if err != nil {
				return ended, err
			}
		}
	}
}

// recheckOne refreshes one grant: a refusal ends the person's single sign-on sessions, an answer records the check.
func (s *Service) recheckOne(ctx context.Context, idp IdP, g StoredGrant, now time.Time, log *slog.Logger) (int, error) {
	if g.RefreshToken == "" {
		log.Warn("unreadable single sign-on grant dropped; checked again after their next sign-in", "user_id", g.UserID)
		return 0, s.cfg.Repo.DropGrant(ctx, g.UserID)
	}
	next, err := idp.Refresh(ctx, g.RefreshToken)
	if errors.Is(err, errGrantRefused) {
		n, err := s.cfg.Repo.RefuseGrant(ctx, g.UserID)
		if err != nil {
			return 0, err
		}
		log.Info("single sign-on grant refused by the provider; sessions ended", "user_id", g.UserID, "sessions", n)
		return n, nil
	}
	if err != nil {
		return 0, fmt.Errorf("recheck single sign-on grant: %w", err)
	}
	if next == g.RefreshToken {
		next = ""
	}
	return 0, s.cfg.Repo.ConfirmGrant(ctx, g.UserID, next, now)
}

// RunRecheckLoop runs a recheck pass on interval until ctx is cancelled; the provider is an external system answering by the
// clock, so this is a ticker, not the commit broadcast.
func (s *Service) RunRecheckLoop(ctx context.Context, interval time.Duration, log *slog.Logger) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if _, err := s.recheck(ctx, log); err != nil {
				log.Warn("single sign-on recheck stopped", "error", err)
			}
		}
	}
}
