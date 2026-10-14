package runner

import (
	"context"
	"fmt"
	"slices"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/ids"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// ViewerGate names who sees a machine, so a deleted machine's live frame reaches them and nobody else.
type ViewerGate interface {
	MachineViewers(ctx context.Context, machineID string) ([]string, error)
}

// DeleteMachine is refused while anything uses the machine; its offline runners go with it (ADR 0153).
func (s *Service) DeleteMachine(ctx context.Context, id string) error {
	if s.machines == nil {
		return fmt.Errorf("%w: machines are not configured", apperrs.ErrConflict)
	}
	if err := s.requireMachine(ctx, id, permissions.MachinesDelete); err != nil {
		return err
	}
	m, err := s.machines.Get(ctx, id)
	if err != nil {
		return fmt.Errorf("get machine %s: %w", id, err)
	}
	runners, err := s.repo.ListOn(ctx, []string{id})
	if err != nil {
		return fmt.Errorf("list runners on machine %s: %w", m.Name, err)
	}
	users, err := s.machineUsers(ctx, m, runners)
	if err != nil {
		return err
	}
	if len(users) > 0 {
		return &apperrs.InUseError{Thing: fmt.Sprintf("machine %q", m.Name), Users: users, Fix: "move or remove them first"}
	}
	viewers, err := s.machineViewers(ctx, id)
	if err != nil {
		return err
	}
	evt := eventbus.OutboxEvent{ID: ids.New(), Topic: TopicMachineDeleted, Payload: MachineDeletedEvent{MachineID: id, Name: m.Name, UserIDs: viewers}}
	if err := s.machines.Delete(ctx, m, s.now().UTC(), evt); err != nil {
		return fmt.Errorf("delete machine %s: %w", id, err)
	}
	for _, r := range runners {
		s.live.Uninstall(ctx, r.ID)
	}
	return nil
}

// machineUsers names what stops a machine going: its stacks, its gateways and its connected runners.
func (s *Service) machineUsers(ctx context.Context, m *Machine, runners []*Runner) ([]string, error) {
	dep, err := s.machines.Dependents(ctx, m.Name)
	if err != nil {
		return nil, fmt.Errorf("list what targets machine %s: %w", m.Name, err)
	}
	users := make([]string, 0, len(dep.Stacks)+len(dep.Gateways))
	for _, name := range dep.Stacks {
		users = append(users, "stack "+name)
	}
	for _, network := range dep.Gateways {
		users = append(users, "gateway on network "+network)
	}
	live := s.live.Runners()
	for _, r := range runners {
		if slices.ContainsFunc(live, func(c RunnerStatus) bool { return c.RunnerID == r.ID }) {
			users = append(users, "connected runner "+r.Name)
		}
	}
	return users, nil
}

func (s *Service) machineViewers(ctx context.Context, id string) ([]string, error) {
	if s.viewers == nil {
		return []string{}, nil
	}
	viewers, err := s.viewers.MachineViewers(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("list who sees machine %s: %w", id, err)
	}
	return viewers, nil
}
