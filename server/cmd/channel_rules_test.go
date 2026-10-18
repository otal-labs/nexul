package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/access"
	"github.com/otal-labs/nexul/internal/platform/permissions"
	"github.com/otal-labs/nexul/internal/platform/storage/testutil/migrated"
)

// postsChannel is the one rule that makes a channel announcements-only: its Writers post, and every other role is denied.
func postsChannel(t *testing.T, f permFixture, channelID string) {
	t.Helper()
	post := []permissions.Action{permissions.ChannelsPost}
	others := access.Targets{RoleIDs: []string{"role-reader", "role-plain", "role-steward", "role-clerk", "role-manager", "role-cloner", "role-editor", "role-copier"}}
	require.NoError(t, f.svc.accessSvc.SetRules(as(uOwner), uOwner, "channel", []string{channelID}, access.Targets{RoleIDs: []string{"role-writer"}}, post, access.StateAllow))
	require.NoError(t, f.svc.accessSvc.SetRules(as(uOwner), uOwner, "channel", []string{channelID}, others, post, access.StateDeny))
}

// TestChannelRules_OnlyTheRolePosts: on a channel only Writers post in, a Writer posts and a Reader is refused with the
// rule's sentence, through the HTTP gateway, MCP, a bot each of them made, and an automation each of them made; the
// Reader still reads the channel, and the Owner posts whatever the rules say.
func TestChannelRules_OnlyTheRolePosts(t *testing.T) {
	t.Parallel()
	db := migrated.DB(t)
	h, store, svc := routerOver(t, db)
	f := seedPermFixture(t, db, svc, store)
	postsChannel(t, f, f.channel.ID)
	const refusal = "channels:post required (Denied on channel #eng for role Reader)"
	ctx := context.Background()

	tokens := map[string]string{}
	bots := map[string]string{}
	automationTokens := map[string]string{}
	for _, user := range []string{uWriter, uReader, uOwner} {
		pat, _, err := svc.authSvc.MintPAT(ctx, user, "posting test")
		require.NoError(t, err)
		tokens[user] = pat
		b, err := svc.botwebhookSvc.Create(as(uOwner), f.channel.ID, "CI "+user, "")
		require.NoError(t, err)
		_, err = db.Exec(`UPDATE botwebhooks SET created_by = ? WHERE id = ?`, user, b.ID)
		require.NoError(t, err)
		bots[user] = botPath(b)
		a, raw, err := svc.automationsSvc.Create(as(uOwner), uOwner, wsDefault, "Release notes "+user, []string{"channels:post"})
		require.NoError(t, err)
		_, err = svc.automationsSvc.SetEnabled(as(uOwner), uOwner, a.ID, true)
		require.NoError(t, err)
		_, err = db.Exec(`UPDATE automations SET created_by = ? WHERE id = ?`, user, a.ID)
		require.NoError(t, err)
		automationTokens[user] = raw
	}
	messages := "/api/chat/conversations/" + f.channel.ID + "/messages"

	paths := []struct {
		name string
		post func(user string) (status int, body string)
		// refused is what the Reader is told, when it isn't the rule's own sentence.
		refused string
	}{
		{name: "over HTTP", post: func(user string) (int, string) {
			rec := callAPI(t, h, http.MethodPost, messages, tokens[user], `{"body":"shipped"}`)
			return rec.Code, rec.Body.String()
		}},
		{name: "over MCP", post: func(user string) (int, string) {
			out := callToolResult(t, h, tokens[user], "message_post", `{"conversation_id":"`+f.channel.ID+`","body":"shipped"}`)
			if strings.Contains(out, `"isError":true`) {
				return http.StatusForbidden, out
			}
			return http.StatusOK, out
		}},
		{name: "through a bot they made", refused: "Stopped: u-reader can't post in #eng", post: func(user string) (int, string) {
			req := httptest.NewRequest(http.MethodPost, bots[user], strings.NewReader(`{"content":"build green"}`))
			req.RemoteAddr = "203.0.113.9:51000"
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)
			return rec.Code, rec.Body.String()
		}},
		{name: "through an automation they made", post: func(user string) (int, string) {
			rec := callAPI(t, h, http.MethodPost, messages, automationTokens[user], `{"body":"v1.2 is out"}`)
			return rec.Code, rec.Body.String()
		}},
	}
	for _, p := range paths {
		t.Run(p.name, func(t *testing.T) {
			for _, user := range []string{uWriter, uOwner} {
				status, body := p.post(user)
				assert.Less(t, status, 300, "%s: %s", user, body)
			}
			status, body := p.post(uReader)
			assert.Equal(t, http.StatusForbidden, status, body)
			want := refusal
			if p.refused != "" {
				want = p.refused
			}
			assert.Contains(t, body, want)
		})
	}

	rec := callAPI(t, h, http.MethodGet, messages, tokens[uReader], "")
	require.Equal(t, http.StatusOK, rec.Code, "a Reader still reads the channel")
	var listed []json.RawMessage
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &listed))
	assert.Len(t, listed, 8, "the Writer's and the Owner's posts, each path once")

	rec = callAPI(t, h, http.MethodPost, messages, automationTokens[uWriter], `{"body":"x"}`)
	require.Less(t, rec.Code, 300)
	_, err := db.Exec(`UPDATE automations SET scopes = CAST('["chat:write"]' AS BLOB) WHERE created_by = ?`, uWriter)
	require.NoError(t, err)
	rec = callAPI(t, h, http.MethodPost, messages, automationTokens[uWriter], `{"body":"x"}`)
	assert.Equal(t, http.StatusForbidden, rec.Code, "a token posts with channels:post, whatever its creator may")
}

// TestChannelRules_ThePanelAndTheAccessGridAgree: a role hidden on the channel's panel shows hidden on the Access page's
// grid, a grid save shows on the panel, and the grid moves only channels:read, leaving a posting rule as it was.
func TestChannelRules_ThePanelAndTheAccessGridAgree(t *testing.T) {
	t.Parallel()
	f := newPermFixture(t)
	a := f.svc.accessSvc
	hidden := func() []string {
		t.Helper()
		g, err := a.Grid(as(uOwner), uOwner, wsDefault)
		require.NoError(t, err)
		for _, c := range g.Channels {
			if c.ID == f.channel.ID {
				return c.HiddenRoleIDs
			}
		}
		t.Fatalf("channel %s is not on the grid", f.channel.ID)
		return nil
	}
	rules := func() map[string][2]permissions.Set {
		t.Helper()
		got, err := a.ListRules(as(uOwner), uOwner, "channel", f.channel.ID)
		require.NoError(t, err)
		out := map[string][2]permissions.Set{}
		for _, ow := range got {
			out[ow.RoleID] = [2]permissions.Set{ow.Allow, ow.Deny}
		}
		return out
	}
	toReaders := access.Targets{RoleIDs: []string{"role-reader"}}

	require.NoError(t, a.SetRules(as(uOwner), uOwner, "channel", []string{f.channel.ID}, toReaders, []permissions.Action{permissions.ChannelsRead, permissions.ChannelsPost}, access.StateDeny))
	assert.Equal(t, []string{"role-reader"}, hidden(), "the panel's Deny is the grid's unticked box")

	require.NoError(t, a.SaveGrid(as(uOwner), uOwner, wsDefault, access.GridSave{Channels: []access.GridChannelSave{{ChannelID: f.channel.ID, HiddenRoleIDs: []string{"role-plain"}}}}))
	assert.Equal(t, []string{"role-plain"}, hidden())
	assert.Equal(t, map[string][2]permissions.Set{
		"role-reader": {nil, permissions.SetOf(permissions.ChannelsPost)},
		"role-plain":  {nil, permissions.SetOf(permissions.ChannelsRead)},
	}, rules(), "the grid's save on the panel, Readers still kept from posting")
}

// TestChannelRules_ABotStopsWithItsCreator: a bot posts only while its creator may post in its channel. Denied, left
// with no role, or gone from the workspace, its creator stops it, and both the bot's row and the sender's 403 say
// "Stopped: <creator> can't post in #<channel>"; a creator who may post again starts it again.
func TestChannelRules_ABotStopsWithItsCreator(t *testing.T) {
	t.Parallel()
	db := migrated.DB(t)
	h, store, svc := routerOver(t, db)
	f := seedPermFixture(t, db, svc, store)
	ctx := context.Background()
	b, err := svc.botwebhookSvc.Create(as(uOwner), f.channel.ID, "CI", "")
	require.NoError(t, err)
	_, err = db.Exec(`UPDATE botwebhooks SET created_by = ? WHERE id = ?`, uWriter, b.ID)
	require.NoError(t, err)
	const stopped = "Stopped: u-writer can't post in #eng"
	post := func() (int, string) {
		req := httptest.NewRequest(http.MethodPost, botPath(b), strings.NewReader(`{"content":"build green"}`))
		req.RemoteAddr = "203.0.113.9:51000"
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec.Code, rec.Body.String()
	}
	row := func() string {
		bots, err := svc.botwebhookSvc.List(as(uOwner), f.channel.ID, false)
		require.NoError(t, err)
		require.Len(t, bots, 1)
		return bots[0].Stopped
	}
	assertStopped := func(why string) {
		t.Helper()
		status, body := post()
		assert.Equal(t, http.StatusForbidden, status, why)
		assert.Contains(t, body, stopped, why)
		assert.Equal(t, stopped, row(), why)
	}

	status, _ := post()
	require.Equal(t, http.StatusNoContent, status, "its creator posts here")
	assert.Empty(t, row())

	require.NoError(t, svc.authSvc.DisableAccount(as(uOwner), uOwner, uWriter))
	assertStopped("its creator's account is disabled")
	require.NoError(t, svc.authSvc.ReactivateAccount(as(uOwner), uOwner, uWriter))
	status, _ = post()
	require.Equal(t, http.StatusNoContent, status, "reactivating its creator starts it again")

	postsChannel(t, f, f.channel.ID)
	require.NoError(t, svc.accessSvc.SetRules(as(uOwner), uOwner, "channel", []string{f.channel.ID}, access.Targets{UserIDs: []string{uWriter}},
		[]permissions.Action{permissions.ChannelsPost}, access.StateDeny))
	assertStopped("denied by a rule naming them")

	require.NoError(t, svc.accessSvc.SetRules(as(uOwner), uOwner, "channel", []string{f.channel.ID}, access.Targets{UserIDs: []string{uWriter}},
		[]permissions.Action{permissions.ChannelsPost}, access.StateFallback))
	status, _ = post()
	require.Equal(t, http.StatusNoContent, status, "allowed again, it starts again")

	require.NoError(t, store.WorkspaceMembers.ChangeRoles(ctx, wsDefault, uWriter, nil, []string{"role-writer"}))
	assertStopped("holding no role")

	require.NoError(t, store.WorkspaceMembers.RemoveMember(ctx, wsDefault, uWriter))
	assertStopped("gone from the workspace")
}
