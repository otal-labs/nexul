package pairing

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// OrderShare is a computer shared with a person under Run agents that they put in their own run order (ADR 0211).
type OrderShare struct {
	ComputerID string
	Position   int
}

// orderSlot is one place in a person's run order: their own computer, or a shared one they added.
type orderSlot struct {
	id       string
	position int
	own      *Computer
}

// order is a person's run order: their own computers and the shared ones they added, by position, their own first on a tie.
func (s *Service) order(ctx context.Context, userID string) ([]orderSlot, error) {
	own, err := s.repo.ListComputers(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list computers: %w", err)
	}
	shares, err := s.repo.ListOrderShares(ctx, userID)
	if err != nil {
		return nil, err
	}
	slots := make([]orderSlot, 0, len(own)+len(shares))
	for i := range own {
		slots = append(slots, orderSlot{id: own[i].ID, position: own[i].Position, own: &own[i]})
	}
	for _, sh := range shares {
		slots = append(slots, orderSlot{id: sh.ComputerID, position: sh.Position})
	}
	slices.SortStableFunc(slots, func(a, b orderSlot) int { return cmp.Compare(a.position, b.position) })
	return slots, nil
}

// orderedComputers is a person's order as computers; a shared one counts only while its owner still allows Run agents.
func (s *Service) orderedComputers(ctx context.Context, userID string) ([]Computer, error) {
	slots, err := s.order(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]Computer, 0, len(slots))
	for _, slot := range slots {
		if slot.own != nil {
			out = append(out, *slot.own)
			continue
		}
		shared, err := s.computerWith(ctx, userID, slot.id, permissions.ComputersAgents)
		if errors.Is(err, apperrs.ErrNotFound) {
			continue
		}
		if err != nil {
			return nil, err
		}
		out = append(out, *shared)
	}
	return out, nil
}

// AddToOrder puts a computer shared with userID under Run agents at the end of their run order.
func (s *Service) AddToOrder(ctx context.Context, userID, computerID string) error {
	c, err := s.computerWith(ctx, userID, strings.TrimSpace(computerID), permissions.ComputersAgents)
	if err != nil {
		return err
	}
	if c.UserID == userID {
		return fmt.Errorf("%w: %s is yours, so it is in your order already", apperrs.ErrInvalid, c.Name)
	}
	slots, err := s.order(ctx, userID)
	if err != nil {
		return err
	}
	last := -1
	for _, slot := range slots {
		last = max(last, slot.position)
	}
	if err := s.repo.AddOrderShare(ctx, userID, c.ID, last+1, s.now().UTC()); err != nil {
		return err
	}
	s.placementChanged(ctx, userID)
	return nil
}

// RemoveFromOrder takes a shared computer back out of userID's run order; it stays shared with them.
func (s *Service) RemoveFromOrder(ctx context.Context, userID, computerID string) error {
	if strings.TrimSpace(userID) == "" {
		return fmt.Errorf("%w: user is required", apperrs.ErrUnauthorized)
	}
	removed, err := s.repo.RemoveOrderShare(ctx, userID, strings.TrimSpace(computerID))
	if err != nil {
		return err
	}
	if !removed {
		return fmt.Errorf("%w: computer %s is not in your order", apperrs.ErrNotFound, computerID)
	}
	s.placementChanged(ctx, userID)
	return nil
}

// sharedComputer is one computer shared with userID as their list shows it, with none of its owner's setup.
func (s *Service) sharedComputer(ctx context.Context, userID, computerID string) (*Computer, error) {
	cs, err := s.ListComputers(ctx, userID)
	if err != nil {
		return nil, err
	}
	i := slices.IndexFunc(cs, func(c Computer) bool { return c.ID == computerID && c.Shared != nil })
	if i < 0 {
		return nil, fmt.Errorf("%w: computer %s", apperrs.ErrNotFound, computerID)
	}
	return &cs[i], nil
}

// endOrderShare drops a shared computer from a person's order once its owner no longer lets them run agents there.
func (s *Service) endOrderShare(ctx context.Context, userID, computerID string) error {
	removed, err := s.repo.RemoveOrderShare(ctx, userID, computerID)
	if err != nil || !removed {
		return err
	}
	s.placementChanged(ctx, userID)
	return nil
}
