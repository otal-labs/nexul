package runner

import (
	"context"
	"errors"
	"fmt"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/ids"
)

// defaultStackRoot is a machine's stack root until the owner changes it (spec §2).
const defaultStackRoot = "/data/nexul"

// Machine is a host runners connect from; runners reporting the same machine form a dispatch pool (issue 05).
type Machine struct {
	ID               string    `json:"id"`
	Name             string    `json:"name"`
	StackRoot        string    `json:"stack_root"`
	ReportedHostname string    `json:"reported_hostname,omitempty"`
	FirstSeen        time.Time `json:"first_seen"`
	LastSeen         time.Time `json:"last_seen"`
}

// MachineRepo persists machines.
type MachineRepo interface {
	Create(ctx context.Context, m *Machine) error
	Get(ctx context.Context, id string) (*Machine, error)
	GetByName(ctx context.Context, name string) (*Machine, error)
	List(ctx context.Context) ([]*Machine, error)
	Rename(ctx context.Context, id, name string) error
	SetStackRoot(ctx context.Context, id, stackRoot string) error
	// Touch refreshes last_seen and records the runner-reported hostname hint; called at connect and on every
	// heartbeat (the reported name is known only at connect, so heartbeats just repeat it).
	Touch(ctx context.Context, id, reportedHostname string, at time.Time) error
}

// ensureMachine resolves the machine a connecting runner belongs to (issue 05 handshake): a runner that
// already has one keeps its machine and id — renaming only ever happens through the UI, never a reconnect —
// and the freshly reported name is recorded as a hint via Touch. A runner without one yet is linked to a
// machine found-or-created by the reported name (falling back to the runner id when nothing was reported). A
// created machine takes the runner's reported stack root, else the default; an existing one keeps its own.
func ensureMachine(ctx context.Context, machines MachineRepo, runners Repo, runnerID, reportedName, stackRoot string, now time.Time) (string, error) {
	r, err := runners.GetByID(ctx, runnerID)
	if err != nil {
		return "", fmt.Errorf("get runner %s: %w", runnerID, err)
	}
	if r.MachineID != "" {
		if err := machines.Touch(ctx, r.MachineID, reportedName, now); err != nil {
			return "", fmt.Errorf("touch machine %s: %w", r.MachineID, err)
		}
		return r.MachineID, nil
	}
	name := reportedName
	if name == "" {
		name = runnerID
	}
	m, err := machines.GetByName(ctx, name)
	if err != nil {
		if !errors.Is(err, apperrs.ErrNotFound) {
			return "", fmt.Errorf("get machine %s: %w", name, err)
		}
		if stackRoot == "" {
			stackRoot = defaultStackRoot
		}
		m = &Machine{
			ID: ids.New(), Name: name, StackRoot: stackRoot,
			ReportedHostname: reportedName, FirstSeen: now, LastSeen: now,
		}
		if err := machines.Create(ctx, m); err != nil {
			return "", fmt.Errorf("create machine %s: %w", name, err)
		}
	}
	if err := runners.SetMachine(ctx, runnerID, m.ID); err != nil {
		return "", fmt.Errorf("link runner %s to machine %s: %w", runnerID, m.ID, err)
	}
	return m.ID, nil
}
