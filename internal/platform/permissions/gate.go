package permissions

import (
	"context"
	"errors"
	"fmt"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/identity"
)

// Ungated answers a check when a use-case has no gate wired: the server's own calls (no actor) pass and a
// person's call is refused, so a wiring slip fails closed.
func Ungated(ctx context.Context) error {
	if _, ok := identity.ActorFromCtx(ctx); !ok {
		return nil
	}
	return fmt.Errorf("%w: no permission gate wired", apperrs.ErrForbidden)
}

// Refused reports a gate's answer that drops an item from a list rather than failing it.
func Refused(err error) bool {
	return errors.Is(err, apperrs.ErrForbidden) || errors.Is(err, apperrs.ErrNotFound)
}

// Filter keeps the items whose scope passes check, asking once per scope; a refusal drops the item, any other
// error fails the whole list.
func Filter[T any](items []T, scope func(T) string, check func(scope string) error) ([]T, error) {
	allowed := map[string]bool{}
	out := make([]T, 0, len(items))
	for _, item := range items {
		key := scope(item)
		ok, seen := allowed[key]
		if !seen {
			err := check(key)
			if err != nil && !Refused(err) {
				return nil, err
			}
			ok = err == nil
			allowed[key] = ok
		}
		if ok {
			out = append(out, item)
		}
	}
	return out, nil
}

// RequireHeld refuses a grant carrying an action its giver does not hold, so nobody hands out more than they have
// (ADR 0088). held is the giver's own grid in the workspace the grant lands in; an Owner's is every action.
func RequireHeld(grant, held Set) error {
	var missing Set
	for _, action := range grant {
		if !held.Has(action) {
			missing = append(missing, action)
		}
	}
	if len(missing) == 0 {
		return nil
	}
	return fmt.Errorf("%w: you can only grant permissions you hold yourself; you lack %s", apperrs.ErrForbidden, missing)
}
