package runner

import (
	"context"
	"time"
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
	// Touch refreshes last_seen and records the reported hostname hint; called at connect and on every heartbeat.
	Touch(ctx context.Context, id, reportedHostname string, at time.Time) error
}
