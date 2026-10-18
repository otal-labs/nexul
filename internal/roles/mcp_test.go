package roles

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/identity"
)

func callRoleTool(ctx context.Context, t *testing.T, s *Service, name, args string) (any, error) {
	t.Helper()
	for _, tool := range MCPTools(s) {
		if tool.Name == name {
			return tool.Call(ctx, json.RawMessage(args))
		}
	}
	t.Fatalf("tool %s not found", name)
	return nil, nil
}

func asActor(id string) context.Context {
	return identity.WithActor(context.Background(), identity.Actor{ID: id})
}

// newToolFixture: u-owner owns ws-src and ws-dst, u-plain holds no role permissions in ws-src, ws-src has role-editors.
func newToolFixture(t *testing.T) (*Service, *fakeRepo) {
	t.Helper()
	s, repo := newCloneFixture(t, "u-owner", cloneMember{workspaceID: "ws-src", owner: true}, cloneMember{workspaceID: "ws-dst", owner: true})
	require.NoError(t, repo.Create(context.Background(), &Role{ID: "role-plain", WorkspaceID: "ws-src", Name: "Plain"}))
	s.members.(*fakeMemberGate).roleIDs[memberKey("ws-src", "u-plain")] = "role-plain"
	return s, repo
}

func TestMCPTools_Errors(t *testing.T) {
	tests := []struct {
		name  string
		actor string
		tool  string
		args  string
		want  error
	}{
		{"no caller is unauthorized", "", "role_update", `{"workspace_id":"ws-src","name":"X"}`, apperrs.ErrUnauthorized},
		{"no caller cannot delete", "", "role_delete", `{"workspace_id":"ws-src","id":"role-editors"}`, apperrs.ErrUnauthorized},
		{"an unknown argument is invalid", "u-owner", "role_update", `{"workspace_id":"ws-src","user_id":"u-2"}`, apperrs.ErrInvalid},
		{"missing workspace_id is invalid", "u-owner", "role_update", `{"name":"X"}`, apperrs.ErrInvalid},
		{"an unknown permission is invalid", "u-owner", "role_update", `{"workspace_id":"ws-src","name":"X","permissions":["docs:fly"]}`, apperrs.ErrInvalid},
		{"create without a name is invalid", "u-owner", "role_update", `{"workspace_id":"ws-src"}`, apperrs.ErrInvalid},
		{"clone with a description is invalid", "u-owner", "role_update", `{"workspace_id":"ws-dst","clone_from_id":"role-editors","description":"X"}`, apperrs.ErrInvalid},
		{"clone with a name is invalid", "u-owner", "role_update", `{"workspace_id":"ws-dst","clone_from_id":"role-editors","name":"X"}`, apperrs.ErrInvalid},
		{"clone with permissions is invalid", "u-owner", "role_update", `{"workspace_id":"ws-dst","clone_from_id":"role-editors","permissions":[]}`, apperrs.ErrInvalid},
		{"clone with an id is invalid", "u-owner", "role_update", `{"workspace_id":"ws-dst","clone_from_id":"role-editors","id":"role-editors"}`, apperrs.ErrInvalid},
		{"clone into its own workspace is invalid", "u-owner", "role_update", `{"workspace_id":"ws-src","clone_from_id":"role-editors"}`, apperrs.ErrInvalid},
		{"clone of the Owner role is invalid", "u-owner", "role_update", `{"workspace_id":"ws-dst","clone_from_id":"role-owner-src"}`, apperrs.ErrInvalid},
		{"clone of a missing role is not found", "u-owner", "role_update", `{"workspace_id":"ws-dst","clone_from_id":"ghost"}`, apperrs.ErrNotFound},
		{"clone without roles:clone is forbidden", "u-plain", "role_update", `{"workspace_id":"ws-dst","clone_from_id":"role-editors"}`, apperrs.ErrForbidden},
		{"create without roles:write is forbidden", "u-plain", "role_update", `{"workspace_id":"ws-src","name":"X"}`, apperrs.ErrForbidden},
		{"update without roles:write is forbidden", "u-plain", "role_update", `{"workspace_id":"ws-src","id":"role-editors","name":"X"}`, apperrs.ErrForbidden},
		{"update of a missing role is not found", "u-owner", "role_update", `{"workspace_id":"ws-src","id":"ghost","name":"X"}`, apperrs.ErrNotFound},
		{"update of a role in another workspace is not found", "u-owner", "role_update", `{"workspace_id":"ws-dst","id":"role-editors","name":"X"}`, apperrs.ErrNotFound},
		{"renaming the Owner role is invalid", "u-owner", "role_update", `{"workspace_id":"ws-src","id":"role-owner-src","name":"Boss"}`, apperrs.ErrInvalid},
		{"changing the Owner role's permissions is invalid", "u-owner", "role_update", `{"workspace_id":"ws-src","id":"role-owner-src","permissions":[]}`, apperrs.ErrInvalid},
		{"changing the Owner role's steps is invalid", "u-owner", "role_update", `{"workspace_id":"ws-src","id":"role-owner-src","welcome_steps":[]}`, apperrs.ErrInvalid},
		{"an unknown welcome step is invalid", "u-owner", "role_update", `{"workspace_id":"ws-src","id":"role-editors","welcome_steps":["profile"]}`, apperrs.ErrInvalid},
		{"clone with welcome steps is invalid", "u-owner", "role_update", `{"workspace_id":"ws-dst","clone_from_id":"role-editors","welcome_steps":[]}`, apperrs.ErrInvalid},
		{"clone with the switch is invalid", "u-owner", "role_update", `{"workspace_id":"ws-dst","clone_from_id":"role-editors","show_locked_actions":true}`, apperrs.ErrInvalid},
		{"delete without roles:write is forbidden", "u-plain", "role_delete", `{"workspace_id":"ws-src","id":"role-editors"}`, apperrs.ErrForbidden},
		{"delete of a missing role is not found", "u-owner", "role_delete", `{"workspace_id":"ws-src","id":"ghost"}`, apperrs.ErrNotFound},
		{"delete of the Owner role is invalid", "u-owner", "role_delete", `{"workspace_id":"ws-src","id":"role-owner-src"}`, apperrs.ErrInvalid},
		{"delete without an id is invalid", "u-owner", "role_delete", `{"workspace_id":"ws-src"}`, apperrs.ErrInvalid},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, _ := newToolFixture(t)
			_, err := callRoleTool(asActor(tt.actor), t, s, tt.tool, tt.args)
			require.ErrorIs(t, err, tt.want)
		})
	}
}

func TestMCPTools_CloneErrorsSayWhatToOmit(t *testing.T) {
	s, _ := newToolFixture(t)
	_, err := callRoleTool(asActor("u-owner"), t, s, "role_update", `{"workspace_id":"ws-dst","clone_from_id":"role-editors","name":"X"}`)
	require.ErrorIs(t, err, apperrs.ErrInvalid)
	assert.Contains(t, err.Error(), "omit id, name, description, permissions, welcome_steps and show_locked_actions")
}

func TestRoleUpdate_Creates(t *testing.T) {
	s, repo := newToolFixture(t)
	out, err := callRoleTool(asActor("u-owner"), t, s, "role_update", `{"workspace_id":"ws-src","name":"Reviewers","description":"Reviews the docs.","permissions":["docs:write","docs:read"]}`)
	require.NoError(t, err)
	got := out.(RoleResult)
	assert.Equal(t, "Reviewers", got.Name)
	assert.Equal(t, "Reviews the docs.", got.Description)
	assert.Equal(t, "ws-src", got.WorkspaceID)
	assert.False(t, got.IsOwnerRole)
	assert.Equal(t, []string{"docs:read", "docs:write"}, got.Permissions)
	stored, err := repo.Get(context.Background(), got.ID)
	require.NoError(t, err)
	assert.Equal(t, "Reviewers", stored.Name)
}

func TestRoleUpdate_Clones(t *testing.T) {
	s, repo := newToolFixture(t)
	require.NoError(t, repo.Create(context.Background(), &Role{ID: "taken", WorkspaceID: "ws-dst", Name: "Editors"}))
	out, err := callRoleTool(asActor("u-owner"), t, s, "role_update", `{"workspace_id":"ws-dst","clone_from_id":"role-editors"}`)
	require.NoError(t, err)
	got := out.(RoleResult)
	assert.Equal(t, "ws-dst", got.WorkspaceID)
	assert.Equal(t, "Editors (copy)", got.Name)
	assert.Equal(t, []string{"docs:read", "memories:clone", "plays:run"}, got.Permissions)
	assert.Equal(t, []string{"computer"}, got.WelcomeSteps)
}

func TestRoleUpdate_OmittedFieldsKeepTheirValue(t *testing.T) {
	tests := []struct {
		name      string
		args      string
		wantName  string
		wantDesc  string
		wantPerms []string
		wantSteps []string
	}{
		{"name only keeps the description, permissions and steps", `{"workspace_id":"ws-src","id":"role-editors","name":"Writers"}`, "Writers", "Writes the docs.", []string{"docs:read", "memories:clone", "plays:run"}, []string{"computer"}},
		{"permissions only keeps the name", `{"workspace_id":"ws-src","id":"role-editors","permissions":["roles:clone"]}`, "Editors", "Writes the docs.", []string{"roles:clone"}, []string{"computer"}},
		{"an empty permission list clears them", `{"workspace_id":"ws-src","id":"role-editors","permissions":[]}`, "Editors", "Writes the docs.", []string{}, []string{"computer"}},
		{"steps only keeps the rest", `{"workspace_id":"ws-src","id":"role-editors","welcome_steps":["phone","github"]}`, "Editors", "Writes the docs.", []string{"docs:read", "memories:clone", "plays:run"}, []string{"github", "phone"}},
		{"an empty step list clears them", `{"workspace_id":"ws-src","id":"role-editors","welcome_steps":[]}`, "Editors", "Writes the docs.", []string{"docs:read", "memories:clone", "plays:run"}, []string{}},
		{"description only replaces it, trimmed", `{"workspace_id":"ws-src","id":"role-editors","description":" Reviews the docs. "}`, "Editors", "Reviews the docs.", []string{"docs:read", "memories:clone", "plays:run"}, []string{"computer"}},
		{"an empty description clears it", `{"workspace_id":"ws-src","id":"role-editors","description":""}`, "Editors", "", []string{"docs:read", "memories:clone", "plays:run"}, []string{"computer"}},
		{"nothing sent changes nothing", `{"workspace_id":"ws-src","id":"role-editors"}`, "Editors", "Writes the docs.", []string{"docs:read", "memories:clone", "plays:run"}, []string{"computer"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, repo := newToolFixture(t)
			out, err := callRoleTool(asActor("u-owner"), t, s, "role_update", tt.args)
			require.NoError(t, err)
			got := out.(RoleResult)
			assert.Equal(t, tt.wantName, got.Name)
			assert.Equal(t, tt.wantDesc, got.Description)
			assert.Equal(t, tt.wantPerms, got.Permissions)
			assert.Equal(t, tt.wantSteps, got.WelcomeSteps)
			stored, err := repo.Get(context.Background(), "role-editors")
			require.NoError(t, err)
			assert.Equal(t, ToRoleResult(stored), got)
		})
	}
}

func TestRoleUpdate_ShowLockedActionsIsSetAloneAndOnCreate(t *testing.T) {
	s, repo := newToolFixture(t)
	created, err := callRoleTool(asActor("u-owner"), t, s, "role_update", `{"workspace_id":"ws-src","name":"Developers","show_locked_actions":true}`)
	require.NoError(t, err)
	assert.True(t, created.(RoleResult).ShowLockedActions)

	out, err := callRoleTool(asActor("u-owner"), t, s, "role_update", `{"workspace_id":"ws-src","id":"role-editors","show_locked_actions":true}`)
	require.NoError(t, err)
	got := out.(RoleResult)
	assert.True(t, got.ShowLockedActions)
	assert.Equal(t, "Editors", got.Name, "the switch alone changes nothing else")
	assert.Equal(t, []string{"docs:read", "memories:clone", "plays:run"}, got.Permissions)
	stored, err := repo.Get(context.Background(), "role-editors")
	require.NoError(t, err)
	assert.True(t, stored.ShowLockedActions)
}

func TestRoleDelete_ReturnsWhatItDeleted(t *testing.T) {
	s, repo := newToolFixture(t)
	out, err := callRoleTool(asActor("u-owner"), t, s, "role_delete", `{"workspace_id":"ws-src","id":"role-editors"}`)
	require.NoError(t, err)
	assert.Equal(t, roleDeleted{ID: "role-editors", Deleted: true}, out)
	_, err = repo.Get(context.Background(), "role-editors")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}
