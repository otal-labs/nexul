package runner

import (
	"context"
	"fmt"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// OpenEnrollment is an unused code as its maker sees it: never the code or its hash, only what names it.
type OpenEnrollment struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	Machine    string    `json:"machine,omitempty"`
	ComputerID string    `json:"computer_id,omitempty" jsonschema:"Set on a code that adds a computer's runner."`
	CreatedAt  time.Time `json:"created_at"`
	ExpiresAt  time.Time `json:"expires_at"`
}

// ListEnrollments returns the unused codes the caller may revoke: their own computers' codes and the runner codes
// they made, plus those of a machine they manage and every runner code for the instance Owner. The server's own
// calls, which are nobody, have none.
func (s *Service) ListEnrollments(ctx context.Context) ([]OpenEnrollment, error) {
	actor, ok := identity.ActorFromCtx(ctx)
	if !ok || actor.ID == "" {
		return nil, nil
	}
	r, err := s.reach(ctx)
	if err != nil {
		return nil, err
	}
	codes, err := s.repo.ListEnrollments(ctx, actor.ID, s.now().UTC())
	if err != nil {
		return nil, err
	}
	out := make([]OpenEnrollment, 0, len(codes))
	for _, c := range codes {
		if s.mayRevoke(ctx, actor.ID, r, c) {
			out = append(out, OpenEnrollment{ID: c.ID, Name: c.Name, Machine: c.Machine, ComputerID: c.ComputerID, CreatedAt: c.CreatedAt, ExpiresAt: c.ExpiresAt})
		}
	}
	return out, nil
}

// RevokeEnrollment voids an unused code so it enrolls nothing; one the caller may not revoke is not found, as is one
// already used or expired.
func (s *Service) RevokeEnrollment(ctx context.Context, id string) error {
	actor, ok := identity.ActorFromCtx(ctx)
	if !ok || actor.ID == "" {
		return fmt.Errorf("%w: an authenticated user is required", apperrs.ErrUnauthorized)
	}
	c, err := s.repo.GetEnrollmentByID(ctx, id, s.now().UTC())
	if err != nil {
		return fmt.Errorf("revoke enrollment code %s: %w", id, err)
	}
	r, err := s.reach(ctx)
	if err != nil {
		return err
	}
	if !s.mayRevoke(ctx, actor.ID, r, c) {
		return fmt.Errorf("revoke enrollment code %s: %w", id, apperrs.ErrNotFound)
	}
	if err := s.repo.RevokeEnrollment(ctx, id); err != nil {
		return fmt.Errorf("revoke enrollment code %s: %w", id, err)
	}
	return nil
}

// mayRevoke is whether userID made the code or could have: Manage on the machine it names, or the instance Owner. The
// server's own code, which has no maker, is nobody's to revoke.
func (s *Service) mayRevoke(ctx context.Context, userID string, r reach, c *EnrollmentCode) bool {
	if c.Personal() {
		return c.OwnerUserID == userID
	}
	if c.EnrolledBy == "" {
		return false
	}
	if c.EnrolledBy == userID || r.all {
		return true
	}
	if c.Machine == "" || s.machines == nil {
		return false
	}
	m, err := s.machines.GetByName(ctx, c.Machine)
	if err != nil {
		return false
	}
	return s.requireMachine(ctx, m.ID, permissions.MachinesWrite) == nil
}
