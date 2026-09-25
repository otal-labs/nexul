// Package composite holds the MCP tools that compose more than one domain; it imports only the domains it composes.
package composite

import (
	"context"
	"fmt"
	"slices"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/mcptool"
)

// step is one use-case call of an update; hint says where the valid values for its field come from.
type step struct {
	field string
	hint  string
	run   func(ctx context.Context) error
}

// runSteps applies steps in order and stops at the first failure, since each use-case commits on its own.
func runSteps(ctx context.Context, steps []step) ([]string, error) {
	applied := []string{}
	for _, s := range steps {
		err := s.run(ctx)
		if err == nil {
			applied = append(applied, s.field)
			continue
		}
		if s.hint != "" {
			err = fmt.Errorf("%w (%s)", err, s.hint)
		}
		if len(applied) == 0 {
			return applied, fmt.Errorf("%s: %w; nothing was changed", s.field, err)
		}
		return applied, &mcptool.PartialError{Applied: applied, Err: fmt.Errorf("%s: %w; nothing after %s was tried", s.field, err, s.field)}
	}
	return applied, nil
}

// ownerActor is the caller the owner gate checks; an empty id would read as a trusted adapter and skip the gate.
func ownerActor(ctx context.Context) (string, error) {
	a, ok := identity.ActorFromCtx(ctx)
	if !ok || a.ID == "" {
		return "", fmt.Errorf("%w: changing a project needs a signed-in owner", apperrs.ErrUnauthorized)
	}
	return a.ID, nil
}

// moveTo returns ids with id moved to position, clamped to the list, for the use-cases that take a whole new order.
func moveTo(ids []string, id string, position int) []string {
	rest := slices.DeleteFunc(slices.Clone(ids), func(x string) bool { return x == id })
	return slices.Insert(rest, min(max(position, 0), len(rest)), id)
}
