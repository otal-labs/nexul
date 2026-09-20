package tenancy

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/platform/identity"
)

func TestInvitationMCPTools_ExposeCreateListAndRevoke(t *testing.T) {
	t.Parallel()
	svc, _ := newInvitationServiceFixture()
	tools := MCPTools(svc)
	assert.Len(t, tools, 3)
	assert.Equal(t, "create_invitation", tools[0].Name)

	ctx := identity.WithActor(t.Context(), identity.Actor{ID: "actor"})
	result, err := tools[0].Call(ctx, map[string]any{"grants": []any{map[string]any{"workspace_id": "ws-1", "role_id": "role-editor"}}, "expires_in_days": float64(7)})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestInvitationMCPTools_Create_OmittedLifetimeDefaultsToSevenDays(t *testing.T) {
	t.Parallel()
	svc, repo := newInvitationServiceFixture()
	tools := MCPTools(svc)
	ctx := identity.WithActor(t.Context(), identity.Actor{ID: "actor"})
	_, err := tools[0].Call(ctx, map[string]any{"grants": []any{map[string]any{"workspace_id": "ws-1", "role_id": "role-editor"}}})
	require.NoError(t, err)
	require.Len(t, repo.created, 1)
	assert.Equal(t, 7*24*time.Hour, repo.created[0].ExpiresAt.Sub(repo.created[0].CreatedAt))
}

func TestInvitationMCPTools_RejectMalformedGrantsAndSurfaceServiceErrors(t *testing.T) {
	t.Parallel()
	svc, repo := newInvitationServiceFixture()
	tools := MCPTools(svc)
	ctx := identity.WithActor(t.Context(), identity.Actor{ID: "actor"})
	_, err := tools[0].Call(ctx, map[string]any{"grants": []any{"wrong"}})
	assert.Error(t, err)

	repo.createErr = errors.New("storage failed")
	_, err = tools[0].Call(ctx, map[string]any{"grants": []any{map[string]any{"workspace_id": "ws-1", "role_id": "role-editor"}}, "expires_in_days": float64(7)})
	assert.Error(t, err)

	_, err = tools[2].Call(ctx, map[string]any{})
	assert.Error(t, err)
}
