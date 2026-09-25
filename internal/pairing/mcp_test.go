package pairing

import (
	"context"
	"encoding/json"
	"errors"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/mcptool"
	shipped "github.com/otal-labs/nexul/internal/platform/skills"
)

func toolNamed(t *testing.T, tools []mcptool.Tool, name string) mcptool.Tool {
	t.Helper()
	for _, tool := range tools {
		if tool.Name == name {
			return tool
		}
	}
	t.Fatalf("tool %s not found", name)
	return mcptool.Tool{}
}

func actorCtx(t *testing.T, userID string) context.Context {
	return identity.WithActor(t.Context(), identity.Actor{ID: userID})
}

func callTool(t *testing.T, ctx context.Context, svc *Service, name, args string) (any, error) {
	t.Helper()
	return toolNamed(t, MCPTools(svc), name).Call(ctx, json.RawMessage(args))
}

func TestMCPTools_Surface(t *testing.T) {
	t.Parallel()
	svc, _ := newSetupService(t)
	var names []string
	for _, tool := range MCPTools(svc) {
		names = append(names, tool.Name)
		assert.NotEmpty(t, tool.Title, tool.Name)
		assert.NotEmpty(t, tool.Description, tool.Name)
		assert.NotNil(t, tool.InputSchema, tool.Name)
	}
	assert.Equal(t, []string{
		"computer_list", "computer_create", "computer_pair", "computer_delete", "computer_tunnel_token_get",
		"computer_setup_run", "computer_setup_update", "computer_mcp_token_create", "computer_mcp_token_delete", "skill_get",
	}, names)
}

func TestMCPTools_InvalidArguments(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		tool string
		args string
	}{
		{"list with a numeric id", "computer_list", `{"id": 7}`},
		{"create without a name", "computer_create", `{}`},
		{"create on a port outside TCP", "computer_create", `{"name": "Desk", "port": 70000}`},
		{"pair without a token", "computer_pair", `{"id": "c1"}`},
		{"pair by URL without a name", "computer_pair", `{"server_url": "https://vps.example.com", "token": "tok"}`},
		{"delete without an id", "computer_delete", `{}`},
		{"tunnel token without a computer", "computer_tunnel_token_get", `{}`},
		{"setup run without a computer", "computer_setup_run", `{}`},
		{"setup run with model but no provider", "computer_setup_run", `{"computer_id": "c1", "model": "gpt-big"}`},
		{"setup run with provider and models", "computer_setup_run", `{"computer_id": "c1", "provider": "codex", "models": {"codex": "gpt-mini"}}`},
		{"setup run with a non-string model", "computer_setup_run", `{"computer_id": "c1", "models": {"codex": 7}}`},
		{"setup update without confirmed", "computer_setup_update", `{"computer_id": "c1"}`},
		{"setup update confirming a provider without skills", "computer_setup_update", `{"computer_id": "c1", "provider": "claude", "confirmed": true}`},
		{"setup update with skills but no provider", "computer_setup_update", `{"computer_id": "c1", "confirmed": true, "skills": ["tdd"]}`},
		{"setup update withdrawing with skills", "computer_setup_update", `{"computer_id": "c1", "provider": "claude", "confirmed": false, "skills": ["tdd"]}`},
		{"setup update with a non-string skill", "computer_setup_update", `{"computer_id": "c1", "provider": "claude", "confirmed": true, "skills": ["tdd", 3]}`},
		{"token create naming another user", "computer_mcp_token_create", `{"computer_id": "c1", "user_id": "u2"}`},
		{"token delete without a computer", "computer_mcp_token_delete", `{}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			svc, repo := newSetupService(t)
			_, err := callTool(t, actorCtx(t, "u1"), svc, tt.tool, tt.args)
			require.ErrorIs(t, err, apperrs.ErrInvalid)
			assert.Empty(t, repo.outbox)
		})
	}
}

func TestMCPTools_MissingEntity(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		tool string
		args string
	}{
		{"list one unknown computer", "computer_list", `{"id": "nope"}`},
		{"delete an unknown computer", "computer_delete", `{"id": "nope"}`},
		{"tunnel token of an unknown computer", "computer_tunnel_token_get", `{"computer_id": "nope"}`},
		{"confirm an unknown computer", "computer_setup_update", `{"computer_id": "nope", "confirmed": true}`},
		{"delete a token the computer does not have", "computer_mcp_token_delete", `{"computer_id": "c1"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			svc, _ := newSetupService(t)
			_, err := callTool(t, actorCtx(t, "u1"), svc, tt.tool, tt.args)
			require.ErrorIs(t, err, apperrs.ErrNotFound)
		})
	}
}

func TestMCPTools_OnlyTheOwnerReachesAComputer(t *testing.T) {
	t.Parallel()
	calls := map[string]string{
		"computer_list":             `{"id": "c1"}`,
		"computer_pair":             `{"id": "c1", "token": "tok"}`,
		"computer_delete":           `{"id": "c1"}`,
		"computer_tunnel_token_get": `{"computer_id": "c1"}`,
		"computer_setup_run":        `{"computer_id": "c1"}`,
		"computer_setup_update":     `{"computer_id": "c1", "provider": "claude", "confirmed": true, "skills": ["tdd"]}`,
		"computer_mcp_token_create": `{"computer_id": "c1"}`,
		"computer_mcp_token_delete": `{"computer_id": "c1"}`,
	}
	for tool, args := range calls {
		t.Run(tool, func(t *testing.T) {
			t.Parallel()
			svc, repo := newSetupService(t)
			_, err := callTool(t, actorCtx(t, "u2"), svc, tool, args)
			require.ErrorIs(t, err, apperrs.ErrNotFound, "another user's computer is invisible")
			_, err = callTool(t, t.Context(), svc, tool, args)
			require.ErrorIs(t, err, apperrs.ErrUnauthorized)
			assert.Empty(t, repo.outbox)
			assert.Contains(t, repo.computers, "c1")
		})
	}
	t.Run("computer_create", func(t *testing.T) {
		t.Parallel()
		svc, _ := newSetupService(t)
		_, err := callTool(t, t.Context(), svc, "computer_create", `{"name": "Desk"}`)
		require.ErrorIs(t, err, apperrs.ErrUnauthorized)
	})
}

func TestComputerSetupUpdate_ChangesOnlyTheNamedConfirmation(t *testing.T) {
	t.Parallel()
	svc, _, tokens := newTokenService(t)
	ctx := actorCtx(t, "u1")
	update := func(args string) Setup {
		t.Helper()
		out, err := callTool(t, ctx, svc, "computer_setup_update", args)
		require.NoError(t, err)
		return out.(Setup)
	}
	minted, err := callTool(t, ctx, svc, "computer_mcp_token_create", `{"computer_id": "c1"}`)
	require.NoError(t, err)
	update(`{"computer_id": "c1", "provider": "codex", "confirmed": true, "skills": ["tdd", "nexul-memory"]}`)
	update(`{"computer_id": "c1", "provider": "claude", "confirmed": true, "skills": ["tdd"]}`)
	setup := update(`{"computer_id": "c1", "confirmed": true}`)
	require.NotNil(t, setup.ConfirmedAt)

	setup = update(`{"computer_id": "c1", "provider": "codex", "confirmed": false}`)
	assert.NotNil(t, setup.ConfirmedAt, "withdrawing a provider keeps the overall confirmation")
	assert.Equal(t, map[string]bool{"codex": false, "claude": true}, confirmedProviders(setup))

	setup = update(`{"computer_id": "c1", "confirmed": false}`)
	assert.Nil(t, setup.ConfirmedAt)
	assert.Equal(t, map[string]bool{"codex": false, "claude": true}, confirmedProviders(setup), "withdrawing the computer keeps each provider's own")
	assert.Equal(t, []string{minted.(*MintedMCPToken).ID}, tokens.revoked, "an unconfirmed computer loses its MCP token")

	setup = update(`{"computer_id": "c1", "confirmed": false}`)
	assert.Nil(t, setup.ConfirmedAt, "withdrawing again changes nothing more")
}

func confirmedProviders(s Setup) map[string]bool {
	out := map[string]bool{}
	for _, p := range s.Providers {
		out[p.Provider] = p.ConfirmedAt != nil
	}
	return out
}

func TestComputerCreatePairAndDelete_ThroughATunnel(t *testing.T) {
	t.Parallel()
	exch := pairedExchanger()
	tunnels := &fakeTunnels{}
	repo := newFakeRepo()
	svc, _ := newTunnelService(repo, exch, tunnels)
	ctx := actorCtx(t, "u1")

	out, err := callTool(t, ctx, svc, "computer_create", `{"name": "Laptop"}`)
	require.NoError(t, err)
	created := out.(computerResult)
	assert.Equal(t, "https://laptop-ab12cd34.example.com", created.ServerURL)
	assert.False(t, created.Paired)
	assert.Equal(t, "tun-1", created.Tunnel.TunnelID)

	token, err := callTool(t, ctx, svc, "computer_tunnel_token_get", `{"computer_id": "`+created.ID+`"}`)
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"computer_id": created.ID, "token": "connector-token-tun-1"}, token)

	out, err = callTool(t, ctx, svc, "computer_pair", `{"id": "`+created.ID+`", "token": "tok"}`)
	require.NoError(t, err)
	paired := out.(computerResult)
	assert.Equal(t, created.ID, paired.ID)
	assert.True(t, paired.Paired)
	assert.NotNil(t, paired.SessionExpiresAt)
	assert.Equal(t, "https://laptop-ab12cd34.example.com", exch.pairedURL, "a tunnel computer pairs over its hostname")

	out, err = callTool(t, ctx, svc, "computer_delete", `{"id": "`+created.ID+`"}`)
	require.NoError(t, err)
	assert.Equal(t, mcptool.Gone(created.ID), out)
	assert.NotContains(t, repo.computers, created.ID)
	assert.Len(t, tunnels.deleted, 1)
}

func TestComputerPair_ByURL_AddsAComputerWithoutATunnel(t *testing.T) {
	t.Parallel()
	exch := pairedExchanger()
	svc, _ := newTunnelService(newFakeRepo(), exch, &fakeTunnels{})
	ctx := actorCtx(t, "u1")

	out, err := callTool(t, ctx, svc, "computer_pair", `{"name": "VPS", "server_url": "https://vps.example.com", "token": "tok"}`)
	require.NoError(t, err)
	vps := out.(computerResult)
	assert.Equal(t, "https://vps.example.com", exch.pairedURL)
	assert.Nil(t, vps.Tunnel)

	_, err = callTool(t, ctx, svc, "computer_tunnel_token_get", `{"computer_id": "`+vps.ID+`"}`)
	require.ErrorIs(t, err, apperrs.ErrInvalid, "a computer paired by URL has no tunnel token")
}

func TestComputerPair_WithID_RenamesOrMovesAndKeepsWhatIsOmitted(t *testing.T) {
	t.Parallel()
	exch := pairedExchanger()
	svc, _ := newTunnelService(newFakeRepo(), exch, &fakeTunnels{})
	ctx := actorCtx(t, "u1")
	pair := func(args string) computerResult {
		t.Helper()
		out, err := callTool(t, ctx, svc, "computer_pair", args)
		require.NoError(t, err)
		return out.(computerResult)
	}
	vps := pair(`{"name": "VPS", "server_url": "https://vps.example.com", "token": "tok"}`)

	got := pair(`{"id": "` + vps.ID + `", "token": "tok"}`)
	assert.Equal(t, "VPS", got.Name, "an omitted name survives a re-pair")
	assert.Equal(t, "https://vps.example.com", got.ServerURL, "an omitted server_url survives a re-pair")

	got = pair(`{"id": "` + vps.ID + `", "token": "tok", "server_url": "https://vps2.example.com"}`)
	assert.Equal(t, "VPS", got.Name)
	assert.Equal(t, "https://vps2.example.com", exch.pairedURL)

	got = pair(`{"id": "` + vps.ID + `", "token": "tok", "name": "Box"}`)
	assert.Equal(t, vps.ID, got.ID)
	assert.Equal(t, "Box", got.Name)
	assert.Equal(t, "https://vps2.example.com", got.ServerURL)

	tunnelID := tunnelComputer(t, svc)
	_, err := callTool(t, ctx, svc, "computer_pair", `{"id": "`+tunnelID+`", "token": "tok", "server_url": "https://elsewhere.example.com"}`)
	require.ErrorIs(t, err, apperrs.ErrInvalid, "a tunnel computer only pairs over its own hostname")
}

func TestComputerList_ShowsSetupTokenMetadataAndLiveTunnelStatus(t *testing.T) {
	t.Parallel()
	svc, _ := newTunnelService(newFakeRepo(), &fakeExchanger{}, &fakeTunnels{status: "inactive"})
	id := tunnelComputer(t, svc)
	ctx := actorCtx(t, "u1")
	minted, err := callTool(t, ctx, svc, "computer_mcp_token_create", `{"computer_id": "`+id+`"}`)
	require.NoError(t, err)

	out, err := callTool(t, ctx, svc, "computer_list", `{}`)
	require.NoError(t, err)
	page := out.(mcptool.Page[computerResult])
	require.Len(t, page.Items, 1)
	listed := page.Items[0]
	assert.Equal(t, id, listed.ID)
	assert.Nil(t, listed.TunnelStatus, "the full list never calls Cloudflare")
	require.NotNil(t, listed.Setup)
	assert.Nil(t, listed.Setup.ConfirmedAt)
	require.NotNil(t, listed.MCPToken)
	assert.Equal(t, minted.(*MintedMCPToken).ID, listed.MCPToken.ID)
	raw, err := json.Marshal(page)
	require.NoError(t, err)
	assert.NotContains(t, string(raw), minted.(*MintedMCPToken).Token, "a list never carries the token")

	out, err = callTool(t, ctx, svc, "computer_list", `{"id": "`+id+`"}`)
	require.NoError(t, err)
	one := out.(mcptool.Page[computerResult]).Items[0]
	assert.Equal(t, &TunnelStatus{Tunnel: "inactive"}, one.TunnelStatus)
}

func TestComputerList_ALiveTunnelFailureStillReturnsTheComputer(t *testing.T) {
	t.Parallel()
	tunnels := &fakeTunnels{status: "inactive"}
	svc, _ := newTunnelService(newFakeRepo(), &fakeExchanger{}, tunnels)
	id := tunnelComputer(t, svc)
	tunnels.readErr = errors.New("cloudflare: 502 bad gateway")

	out, err := callTool(t, actorCtx(t, "u1"), svc, "computer_list", `{"id": "`+id+`"}`)
	require.NoError(t, err)
	one := out.(mcptool.Page[computerResult]).Items[0]
	assert.Nil(t, one.TunnelStatus)
	require.NotNil(t, one.Setup, "the stored setup state still comes back")
	assert.Contains(t, one.TunnelStatusError, "unavailable")
	assert.NotContains(t, one.TunnelStatusError, "502", "a provider's raw error stays in the log")
}

func TestComputerSetupRun_EveryProviderOrOneWithItsModel(t *testing.T) {
	t.Parallel()
	f := newSetupFixture(t)
	ctx := actorCtx(t, "u1")

	out, err := callTool(t, ctx, f.svc, "computer_setup_run", `{"computer_id": "`+f.computer.ID+`", "models": {"codex": "gpt-mini"}}`)
	require.NoError(t, err)
	run := out.(*SetupRun)
	assert.Equal(t, []SetupProvider{{Provider: "codex", Name: "Codex", Model: "gpt-mini"}, {Provider: "claudeagent", Name: "Claude"}}, run.Providers)
	f.svc.setupRuns.Wait()
	assert.True(t, f.finished(t, run.RunID).Confirmed, "the confirm sessions confirmed through computer_setup_update")

	out, err = callTool(t, ctx, f.svc, "computer_setup_run", `{"computer_id": "`+f.computer.ID+`", "provider": "codex", "model": "gpt-big"}`)
	require.NoError(t, err)
	assert.Equal(t, []SetupProvider{{Provider: "codex", Name: "Codex", Model: "gpt-big"}}, out.(*SetupRun).Providers)
	f.svc.setupRuns.Wait()
}

func TestComputerMCPToken_CreateRevealsItOnceAndDeleteRevokesIt(t *testing.T) {
	t.Parallel()
	svc, _, tokens := newTokenService(t)
	ctx := actorCtx(t, "u1")

	out, err := callTool(t, ctx, svc, "computer_mcp_token_create", `{"computer_id": "c1"}`)
	require.NoError(t, err)
	minted := out.(*MintedMCPToken)
	assert.NotEmpty(t, minted.Token)

	out, err = callTool(t, ctx, svc, "computer_mcp_token_delete", `{"computer_id": "c1"}`)
	require.NoError(t, err)
	assert.Equal(t, mcptool.Gone(minted.ID), out)
	assert.Equal(t, []string{minted.ID}, tokens.revoked)

	_, err = callTool(t, ctx, svc, "computer_mcp_token_delete", `{"computer_id": "c1"}`)
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}

func TestConfirmInstructions_NameOnlyToolsThatExist(t *testing.T) {
	t.Parallel()
	svc, _ := newSetupService(t)
	names := map[string]bool{}
	for _, tool := range MCPTools(svc) {
		names[tool.Name] = true
	}
	named := regexp.MustCompile("`(computer_[a-z_]+)`").FindAllStringSubmatch(confirmInstructions(setupPrompt{ComputerID: "c1", Driver: "codex"}), -1)
	require.NotEmpty(t, named)
	for _, m := range named {
		assert.True(t, names[m[1]], "the confirm session is told to call %s", m[1])
	}
}

func TestSkillGet(t *testing.T) {
	t.Parallel()
	call := skillGetTool().Call

	_, err := call(t.Context(), json.RawMessage(`{}`))
	require.ErrorIs(t, err, apperrs.ErrInvalid)
	_, err = call(t.Context(), json.RawMessage(`{"name":"nexul-memroy"}`))
	require.ErrorIs(t, err, apperrs.ErrNotFound)
	assert.Contains(t, err.Error(), "nexul-memory", "a miss names the skills that exist")

	out, err := call(t.Context(), json.RawMessage(`{"name":"nexul-memory"}`))
	require.NoError(t, err)
	got := out.(skillResult)
	assert.Equal(t, shipped.NexulMemory.Version, got.Version)
	assert.Equal(t, shipped.NexulMemory.Content, got.Content)
	assert.Equal(t, shipped.NexulMemory.Paths(), got.Paths)
}
