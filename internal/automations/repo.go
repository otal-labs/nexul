package automations

import (
	"context"
	"time"

	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// Repo is the consumer-side persistence contract for automations; implemented in internal/platform/storage.
type Repo interface {
	Create(ctx context.Context, a *Automation) error
	Get(ctx context.Context, id string) (*Automation, error)
	// GetByTokenHash looks up by token hash regardless of revocation, so callers tell "revoked" from "unknown" apart.
	GetByTokenHash(ctx context.Context, hash string) (*Automation, error)
	List(ctx context.Context) ([]Automation, error)
	Update(ctx context.Context, a *Automation) error
	Delete(ctx context.Context, id string) error
	// ListHealth reads each automation's last run, its runs since since, and its pending version, keyed by id.
	ListHealth(ctx context.Context, ids []string, since time.Time) (map[string]Health, error)
}

// ConnectionRegistry lets token revocation force-drop a live connection without importing the WS transport.
type ConnectionRegistry interface {
	Disconnect(automationID, reason string)
}

// DeliveryRegistry is the use-cases' handle on dial-in delivery: dropping a connection and skipping missed events.
type DeliveryRegistry interface {
	ConnectionRegistry
	SkipBacklog(ctx context.Context, automationID string) error
}

// PermissionGate is access's check in one workspace (ADR 0087): not found outside it, forbidden without the action.
type PermissionGate interface {
	Require(ctx context.Context, workspaceID string, action permissions.Action) error
}
