package tenancy

import (
	"testing"

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
