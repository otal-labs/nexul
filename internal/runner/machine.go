package runner

import (
	"context"
	"time"

	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/permissions"
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

// MachineView is a machine with the actions its caller holds on it, so a client shows the controls they may use.
type MachineView struct {
	*Machine
	Actions permissions.Set `json:"actions"`
}

// CreatedMachine is a machine an enrollment brings into being, with the person who gets every action on it in the same
// write (ADR 0151); Owner is empty for a machine the server itself enrolls.
type CreatedMachine struct {
	Machine *Machine
	Owner   string
}

// MachineDependents are the stacks and gateways that still target a machine, each as it reads in a refusal.
type MachineDependents struct {
	Stacks   []string
	Gateways []string
}

// MachineRepo persists machines.
type MachineRepo interface {
	Create(ctx context.Context, m *Machine) error
	Get(ctx context.Context, id string) (*Machine, error)
	GetByName(ctx context.Context, name string) (*Machine, error)
	List(ctx context.Context) ([]*Machine, error)
	// ListIn returns the machines ids names, in one statement however many there are.
	ListIn(ctx context.Context, ids []string) ([]*Machine, error)
	Rename(ctx context.Context, id, name string) error
	SetStackRoot(ctx context.Context, id, stackRoot string) error
	// Touch refreshes last_seen and records the reported hostname hint; called at connect and on every heartbeat.
	Touch(ctx context.Context, id, reportedHostname string, at time.Time) error
	// Dependents names the stacks and gateways that target the machine called name.
	Dependents(ctx context.Context, name string) (MachineDependents, error)
	// Delete removes the machine with its rules, its runners and their credentials (revoked at at) and the codes still
	// waiting to enroll one, and writes evt, in one transaction; an unknown id is apperrs.ErrNotFound.
	Delete(ctx context.Context, m *Machine, at time.Time, evt eventbus.OutboxEvent) error
}
