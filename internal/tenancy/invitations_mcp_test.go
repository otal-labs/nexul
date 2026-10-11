package tenancy

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/mcptool"
)

func callInvitationTool(t *testing.T, svc *InvitationService, name, args string) (any, error) {
	t.Helper()
	ctx := identity.WithActor(t.Context(), identity.Actor{ID: "actor"})
	for _, tool := range MCPTools(svc) {
		if tool.Name == name {
			return tool.Call(ctx, json.RawMessage(args))
		}
	}
	t.Fatalf("tool %s not registered", name)
	return nil, nil
}

func TestInvitationTools_Surface(t *testing.T) {
	t.Parallel()
	svc, _ := newInvitationServiceFixture()
	var names []string
	for _, tool := range MCPTools(svc) {
		names = append(names, tool.Name)
		assert.NotEmpty(t, tool.Title)
	}
	assert.Equal(t, []string{"invitation_create", "invitation_list", "invitation_delete"}, names)
}

func TestInvitationTools_ErrorPaths(t *testing.T) {
	t.Parallel()
	forbidden := fmt.Errorf("%w: members:write required in every invited workspace", apperrs.ErrForbidden)
	tests := []struct {
		name    string
		repo    func(*invitationRepoFake)
		tool    string
		args    string
		wantErr error
	}{
		{"create with a bare string grant", nil, "invitation_create", `{"grants": ["ws-1"]}`, apperrs.ErrInvalid},
		{"create with a grant missing its roles", nil, "invitation_create", `{"grants": [{"workspace_id": "ws-1"}]}`, apperrs.ErrInvalid},
		{"create with a grant of no roles", nil, "invitation_create", `{"grants": [{"workspace_id": "ws-1", "roles": []}]}`, apperrs.ErrInvalid},
		{"create with no grants", nil, "invitation_create", `{"grants": []}`, apperrs.ErrInvalid},
		{"create lasting three days", nil, "invitation_create", `{"grants": [{"workspace_id": "ws-1", "roles": ["r"]}], "expires_in_days": 3}`, apperrs.ErrInvalid},
		{"delete without an id", nil, "invitation_delete", `{}`, apperrs.ErrInvalid},
		{"delete an unknown invitation", func(r *invitationRepoFake) { r.revokeErr = apperrs.ErrNotFound }, "invitation_delete", `{"id": "nope"}`, apperrs.ErrNotFound},
		{"create without members:write", func(r *invitationRepoFake) { r.createErr = forbidden }, "invitation_create", `{"grants": [{"workspace_id": "ws-1", "roles": ["r"]}]}`, apperrs.ErrForbidden},
		{"delete without members:write", func(r *invitationRepoFake) { r.revokeErr = forbidden }, "invitation_delete", `{"id": "inv-1"}`, apperrs.ErrForbidden},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			svc, repo := newInvitationServiceFixture()
			if tt.repo != nil {
				tt.repo(repo)
			}
			_, err := callInvitationTool(t, svc, tt.tool, tt.args)
			require.ErrorIs(t, err, tt.wantErr)
			assert.Empty(t, repo.events, "a refused call publishes nothing")
		})
	}
}

func TestInvitationCreate_TypedGrantsAndTheLinkOnce(t *testing.T) {
	t.Parallel()
	svc, repo := newInvitationServiceFixture()

	got, err := callInvitationTool(t, svc, "invitation_create", `{"grants": [{"workspace_id": "ws-1", "roles": ["role-editor", "role-viewer"]}]}`)
	require.NoError(t, err)
	created := got.(createdInvitationResult)
	assert.True(t, strings.HasPrefix(created.URL, "https://nexul.example/invite#"))
	assert.Equal(t, []*TeamRole{{ID: "role-editor"}, {ID: "role-viewer"}}, created.Invitation.Grants[0].Roles)
	require.Len(t, repo.created, 1)
	assert.Equal(t, 7*24*time.Hour, repo.created[0].ExpiresAt.Sub(repo.created[0].CreatedAt), "an omitted lifetime is seven days")
	assert.Equal(t, []string{"role-editor", "role-viewer"}, repo.created[0].Grants[0].Held())
}

// TestInvitationCreate_RemovedFieldsFailTheSchema: each field the wipe removed is gone from the tool's input, so
// sending one is a schema error before anything runs.
func TestInvitationCreate_RemovedFieldsFailTheSchema(t *testing.T) {
	t.Parallel()
	for _, field := range []string{`"role_id": "r"`, `"allow": []`, `"deny": []`, `"every_project": "role"`, `"project_access": []`} {
		svc, repo := newInvitationServiceFixture()
		_, err := callInvitationTool(t, svc, "invitation_create", `{"grants": [{"workspace_id": "ws-1", "roles": ["r"], `+field+`}]}`)
		require.ErrorIs(t, err, apperrs.ErrInvalid, field)
		assert.Contains(t, err.Error(), "input schema", field)
		assert.Empty(t, repo.created, field)
	}
}

func TestInvitationListAndDelete(t *testing.T) {
	t.Parallel()
	svc, repo := newInvitationServiceFixture()
	roles := []*TeamRole{{ID: "role-editor", Name: "Editor"}, {ID: "role-viewer", Name: "Viewer"}}
	repo.listed = []*Invitation{{ID: "inv-1", Grants: []*InvitationGrant{{WorkspaceID: "ws-1", RoleID: "role-editor", Roles: roles}}}, {ID: "inv-2"}}

	got, err := callInvitationTool(t, svc, "invitation_list", `{"limit": 1}`)
	require.NoError(t, err)
	page := got.(mcptool.Page[invitationResult])
	assert.Equal(t, "inv-1", page.Items[0].ID)
	assert.Equal(t, []invitationGrantResult{{WorkspaceID: "ws-1", Roles: roles}}, page.Items[0].Grants)
	assert.Equal(t, 1, page.NextOffset)

	got, err = callInvitationTool(t, svc, "invitation_delete", `{"id": "inv-1"}`)
	require.NoError(t, err)
	assert.Equal(t, mcptool.Gone("inv-1"), got)
	require.Len(t, repo.events, 1)
	assert.Equal(t, TopicInvitationRevoked, repo.events[0].Topic)
}
