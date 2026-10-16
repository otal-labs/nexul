package plays

import (
	"context"
	"errors"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// ComputerGate reads what a person holds on a computer through its owner's rules (ADR 0148).
type ComputerGate interface {
	ComputerPermissions(ctx context.Context, userID, computerID string) (ownerID string, held permissions.Set, err error)
}

// mayRunAgents is whether userID owns computerID or holds computers:run_agents on its share; a computer they cannot
// see, or no gate to ask, is no.
func mayRunAgents(ctx context.Context, gate ComputerGate, userID, computerID string) (bool, error) {
	if gate == nil || userID == "" || computerID == "" {
		return false, nil
	}
	_, held, err := gate.ComputerPermissions(ctx, userID, computerID)
	if errors.Is(err, apperrs.ErrNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return held.Has(permissions.ComputersAgents), nil
}
