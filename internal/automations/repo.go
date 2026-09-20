package automations

import (
	"context"

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
}

// ConnectionRegistry lets token revocation force-drop a live connection without importing the WS transport.
type ConnectionRegistry interface {
	Disconnect(automationID, reason string)
}

// PermissionGate is the consumer-side slice of access's HasPermission check (ADR 0017), gating automation.* actions.
type PermissionGate interface {
	HasPermission(ctx context.Context, userID string, action permissions.Action) bool
}
