package identity

import (
	"context"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// InstanceAdmin reports the instance-admin fact for a user, read from the store rather than trusted from the context.
type InstanceAdmin interface {
	CanCreateWorkspace(ctx context.Context, userID string) (bool, error)
}

// RequireInstanceAdmin returns ErrUnauthorized when ctx carries no actor and ErrForbidden when the actor is not an instance admin.
func RequireInstanceAdmin(ctx context.Context, gate InstanceAdmin) error {
	a, ok := ActorFromCtx(ctx)
	if !ok || a.ID == "" {
		return apperrs.ErrUnauthorized
	}
	if gate == nil {
		return apperrs.ErrForbidden
	}
	can, err := gate.CanCreateWorkspace(ctx, a.ID)
	if err != nil {
		return err
	}
	if !can {
		return apperrs.ErrForbidden
	}
	return nil
}
