package automations

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/mcptool"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

func mcpCtx(userID string) context.Context {
	return identity.WithActor(context.Background(), identity.Actor{ID: userID})
}

func callTool(t *testing.T, tools []mcptool.Tool, ctx context.Context, name, args string) (any, error) {
	t.Helper()
	for _, tool := range tools {
		if tool.Name == name {
			return tool.Call(ctx, json.RawMessage(args))
		}
	}
	t.Fatalf("tool %s not found", name)
	return nil, nil
}

// mcpFixture holds one custom automation owned by "owner"; "viewer" may only read.
func mcpFixture(t *testing.T, repo Repo) ([]mcptool.Tool, string, string) {
	t.Helper()
	perm := newFakePerm(map[string][]permissions.Action{
		"owner":  {permissions.AutomationsRead, permissions.AutomationsWrite, permissions.AutomationsDelete},
		"viewer": {permissions.AutomationsRead},
	})
	svc := newTestService(repo, perm)
	a, token, err := svc.Create(mcpCtx("owner"), "owner", "Close stale tickets", []string{"tickets:read"})
	require.NoError(t, err)
	return MCPTools(svc), a.ID, token
}

func TestMCPTools_Surface(t *testing.T) {
	tools := MCPTools(newTestService(newFakeRepo(), allowAll("owner")))
	names := make([]string, 0, len(tools))
	for _, tool := range tools {
		names = append(names, tool.Name)
		assert.NotEmpty(t, tool.Title, tool.Name)
		assert.NotNil(t, tool.InputSchema, tool.Name)
	}
	assert.ElementsMatch(t, []string{
		"automation_list", "automation_create", "automation_update", "automation_delete", "automation_token_create",
	}, names)
	assert.True(t, tools[0].Hints.ReadOnly, "automation_list only reads")
}

func TestMCPTools_ErrorPaths(t *testing.T) {
	tests := []struct {
		name  string
		actor string
		tool  string
		args  string
		want  error
	}{
		{"an unknown argument is invalid", "owner", "automation_list", `{"user_id":"u-2"}`, apperrs.ErrInvalid},
		{"create without scopes is invalid", "owner", "automation_create", `{"name":"x"}`, apperrs.ErrInvalid},
		{"create with only blank scopes is invalid", "owner", "automation_create", `{"name":"x","scopes":[" "]}`, apperrs.ErrInvalid},
		{"an update that changes nothing is invalid", "owner", "automation_update", `{"id":"$ID"}`, apperrs.ErrInvalid},
		{"config_values must be an object", "owner", "automation_update", `{"id":"$ID","config_values":"done"}`, apperrs.ErrInvalid},
		{"get of a missing automation", "owner", "automation_list", `{"id":"missing"}`, apperrs.ErrNotFound},
		{"update of a missing automation", "owner", "automation_update", `{"id":"missing","enabled":true}`, apperrs.ErrNotFound},
		{"delete of a missing automation", "owner", "automation_delete", `{"id":"missing"}`, apperrs.ErrNotFound},
		{"rotating a missing automation's token", "owner", "automation_token_create", `{"id":"missing"}`, apperrs.ErrNotFound},
		{"no actor", "", "automation_list", `{}`, apperrs.ErrUnauthorized},
		{"a stranger cannot list", "stranger", "automation_list", `{}`, apperrs.ErrForbidden},
		{"a viewer cannot create", "viewer", "automation_create", `{"name":"x","scopes":["tickets:read"]}`, apperrs.ErrForbidden},
		{"a viewer cannot update", "viewer", "automation_update", `{"id":"$ID","enabled":true}`, apperrs.ErrForbidden},
		{"a viewer cannot delete", "viewer", "automation_delete", `{"id":"$ID"}`, apperrs.ErrForbidden},
		{"a viewer cannot rotate the token", "viewer", "automation_token_create", `{"id":"$ID"}`, apperrs.ErrForbidden},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tools, id, _ := mcpFixture(t, newFakeRepo())
			ctx := context.Background()
			if tt.actor != "" {
				ctx = mcpCtx(tt.actor)
			}
			_, err := callTool(t, tools, ctx, tt.tool, replaceID(tt.args, id))
			require.ErrorIs(t, err, tt.want)
		})
	}
}

func TestMCPTools_MissingAutomationNamesTheListTool(t *testing.T) {
	tools, _, _ := mcpFixture(t, newFakeRepo())
	_, err := callTool(t, tools, mcpCtx("owner"), "automation_update", `{"id":"missing","enabled":true}`)
	require.ErrorIs(t, err, apperrs.ErrNotFound)
	assert.Contains(t, err.Error(), "automation_list")
}

func TestAutomationUpdate_OmittedFieldsKeepTheirValues(t *testing.T) {
	tools, id, _ := mcpFixture(t, newFakeRepo())
	ctx := mcpCtx("owner")
	update := func(args string) automationResult {
		t.Helper()
		out, err := callTool(t, tools, ctx, "automation_update", replaceID(args, id))
		require.NoError(t, err)
		return out.(automationResult)
	}

	got := update(`{"id":"$ID","config_values":{"status":"done"},"enabled":true}`)
	assert.JSONEq(t, `{"status":"done"}`, string(got.ConfigValues))
	assert.True(t, got.Enabled)

	got = update(`{"id":"$ID","enabled":false}`)
	assert.False(t, got.Enabled)
	assert.JSONEq(t, `{"status":"done"}`, string(got.ConfigValues), "omitting config_values keeps them")

	got = update(`{"id":"$ID","config_values":{"status":"closed"}}`)
	assert.False(t, got.Enabled, "omitting enabled keeps it")
	assert.JSONEq(t, `{"status":"closed"}`, string(got.ConfigValues))

	got = update(`{"id":"$ID","revoke_token":true}`)
	assert.True(t, got.TokenRevoked)
	assert.JSONEq(t, `{"status":"closed"}`, string(got.ConfigValues))

	got = update(`{"id":"$ID","config_values":{}}`)
	assert.JSONEq(t, `{}`, string(got.ConfigValues), "an empty object clears the values")
}

// failingUpdates lets the first okUpdates writes through, then fails every later one.
type failingUpdates struct {
	*fakeRepo
	okUpdates int
}

var errDiskFull = errors.New("disk full")

func (f *failingUpdates) Update(ctx context.Context, a *Automation) error {
	if f.okUpdates == 0 {
		return errDiskFull
	}
	f.okUpdates--
	return f.fakeRepo.Update(ctx, a)
}

func TestAutomationUpdate_StopsAtTheFirstFailureAndSaysWhatApplied(t *testing.T) {
	repo := &failingUpdates{fakeRepo: newFakeRepo(), okUpdates: 1}
	tools, id, _ := mcpFixture(t, repo)

	_, err := callTool(t, tools, mcpCtx("owner"), "automation_update",
		replaceID(`{"id":"$ID","config_values":{"status":"done"},"enabled":true,"revoke_token":true}`, id))
	require.ErrorIs(t, err, errDiskFull)
	assert.Contains(t, err.Error(), "already applied: config_values")

	a, getErr := repo.Get(context.Background(), id)
	require.NoError(t, getErr)
	assert.JSONEq(t, `{"status":"done"}`, string(a.ConfigValues))
	assert.False(t, a.Enabled)
	assert.Nil(t, a.TokenRevokedAt, "nothing after the failure ran")
}

func TestMCPTools_Lifecycle(t *testing.T) {
	svc := newTestService(newFakeRepo(), allowAll("owner"))
	tools := MCPTools(svc)
	ctx := mcpCtx("owner")

	out, err := callTool(t, tools, ctx, "automation_create", `{"name":"My automation","scopes":["tickets:read"]}`)
	require.NoError(t, err)
	created := out.(mintedTokenResult)
	id := created.Automation.ID
	assert.Contains(t, created.Token, tokenPrefix, "create returns the token once")
	assert.False(t, created.Automation.Enabled, "a new automation starts disabled")

	t.Run("list pages every automation without token material", func(t *testing.T) {
		out, err := callTool(t, tools, ctx, "automation_list", `{}`)
		require.NoError(t, err)
		page := out.(mcptool.Page[automationResult])
		require.Len(t, page.Items, 1)
		assert.Equal(t, id, page.Items[0].ID)
		raw, err := json.Marshal(page)
		require.NoError(t, err)
		assert.NotContains(t, string(raw), created.Token)
		assert.NotContains(t, string(raw), "token_hash")
	})
	t.Run("list with id returns that automation", func(t *testing.T) {
		out, err := callTool(t, tools, ctx, "automation_list", replaceID(`{"id":"$ID"}`, id))
		require.NoError(t, err)
		assert.Equal(t, "My automation", out.(automationResult).Name)
	})
	t.Run("token_create rotates the token and restores a revoked one", func(t *testing.T) {
		_, err := callTool(t, tools, ctx, "automation_update", replaceID(`{"id":"$ID","revoke_token":true}`, id))
		require.NoError(t, err)
		out, err := callTool(t, tools, ctx, "automation_token_create", replaceID(`{"id":"$ID"}`, id))
		require.NoError(t, err)
		minted := out.(mintedTokenResult)
		assert.NotEqual(t, created.Token, minted.Token)
		assert.False(t, minted.Automation.TokenRevoked)
	})
	t.Run("delete reports what it deleted", func(t *testing.T) {
		out, err := callTool(t, tools, ctx, "automation_delete", replaceID(`{"id":"$ID"}`, id))
		require.NoError(t, err)
		assert.Equal(t, mcptool.Gone(id), out)
		_, err = callTool(t, tools, ctx, "automation_list", replaceID(`{"id":"$ID"}`, id))
		require.ErrorIs(t, err, apperrs.ErrNotFound)
	})
}

func replaceID(args, id string) string {
	return strings.ReplaceAll(args, "$ID", id)
}
