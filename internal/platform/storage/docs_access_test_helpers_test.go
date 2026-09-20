package storage_test

import (
	"context"

	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// allowAll is a permissive docs.AccessChecker for storage integration tests
// that exercise repo + use-case wiring rather than permission behavior.
type allowAll struct{}

func (allowAll) Can(context.Context, string, string, permissions.Action) (bool, error) {
	return true, nil
}

func (allowAll) GrantCreator(context.Context, string, string) error {
	return nil
}

func (allowAll) DeleteByDoc(context.Context, string) error {
	return nil
}

// actorCtx returns a context carrying a fixed actor for use-case calls.
func actorCtx() context.Context {
	return identity.WithActor(context.Background(), identity.Actor{ID: "tester", CanCreateWorkspace: true})
}
