package pairing

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/otal-labs/nexul/internal/harness"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/logging"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// guestRunLimit is the longest a run's token lives; the run revokes it at its end, and the expiry covers a server that
// stopped before it could.
const guestRunLimit = 12 * time.Hour

// guestInstanceLabel names the provider instance a run on someone else's computer uses, as T3 Code lists it there.
const guestInstanceLabel = "Nexul shared run"

// sharedRunsListed is how many runs by other people a computer's owner sees on it.
const sharedRunsListed = 10

// SharedRun is one play run someone else ran on a computer, as its owner sees it: never what it did (ADR 0210).
type SharedRun struct {
	TrailID     string     `json:"trail_id"`
	Play        string     `json:"play"`
	StarterID   string     `json:"starter_id"`
	StarterName string     `json:"starter_name"`
	State       string     `json:"state"`
	StartedAt   time.Time  `json:"started_at"`
	EndedAt     *time.Time `json:"ended_at"`
}

// ListSharedRuns is the owner's log of the play runs other people ran on their computer, newest first.
func (s *Service) ListSharedRuns(ctx context.Context, userID, computerID string) ([]SharedRun, error) {
	c, err := s.ownComputer(ctx, userID, computerID)
	if err != nil {
		return nil, err
	}
	runs, err := s.repo.ListSharedRuns(ctx, c.ID, c.UserID, sharedRunsListed)
	if err != nil {
		return nil, fmt.Errorf("list the runs others ran on %s: %w", c.Name, err)
	}
	return runs, nil
}

// RunTokens mints and revokes the short-lived token a run on someone else's computer reaches Nexul with (ADR 0210).
type RunTokens interface {
	// MintRunToken mints a token acting as userID until expires and returns it once, with its id.
	MintRunToken(ctx context.Context, userID, name string, expires time.Time) (token, id string, err error)
	RevokeRunToken(ctx context.Context, userID, id string) error
}

// OpenGuestRun readies a turn userID starts on a computer someone else owns: it runs on a provider instance of its own
// there, whose Nexul MCP token is one minted for userID for this run, never the owner's. The returned close removes
// both, and a turn on one's own computer is left as it is (ADR 0210).
func (s *Service) OpenGuestRun(ctx context.Context, userID string, computer Computer, client harness.Client, target *harness.Target) (func(context.Context), error) {
	noop := func(context.Context) {}
	if computer.UserID == userID {
		return noop, nil
	}
	if _, err := s.computerWith(ctx, userID, computer.ID, permissions.ComputersAgents); err != nil {
		return nil, err
	}
	guests, ok := client.(harness.GuestInstances)
	if !ok {
		return nil, fmt.Errorf("%w: T3 Code on %s runs someone else's agent only on its newer orchestrator; it moves to it once updated", apperrs.ErrConflict, computer.Name)
	}
	if s.runTokens == nil {
		return nil, apperrs.Fatal(fmt.Errorf("%w: run tokens are not wired", apperrs.ErrFatal))
	}
	mcpURL, err := s.setupMCPURL(ctx)
	if err != nil {
		return nil, err
	}
	token, tokenID, err := s.runTokens.MintRunToken(ctx, userID, "Run on "+computer.Name, s.now().Add(guestRunLimit))
	if err != nil {
		return nil, fmt.Errorf("mint the run's token: %w", err)
	}
	instance, err := guests.OpenGuest(ctx, target.Session, target.Provider, mcpURL, token, guestInstanceLabel)
	if err != nil {
		return nil, errors.Join(err, s.runTokens.RevokeRunToken(context.WithoutCancel(ctx), userID, tokenID))
	}
	target.Provider = instance
	session := target.Session
	return func(ctx context.Context) {
		ctx = context.WithoutCancel(ctx)
		if err := guests.CloseGuest(ctx, session, instance); err != nil {
			logging.FromCtx(ctx).Warn("remove a shared run's provider instance", "computer_id", computer.ID, "instance", instance, "error", err)
		}
		if err := s.runTokens.RevokeRunToken(ctx, userID, tokenID); err != nil {
			logging.FromCtx(ctx).Error("revoke a shared run's token", "computer_id", computer.ID, "error", err)
		}
	}, nil
}
