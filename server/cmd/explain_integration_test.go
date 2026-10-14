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
	"github.com/otal-labs/nexul/internal/docs"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// TestExplain_AnswersWhatTheCheckAnswers holds explain to enforcement: over the list parity fixture's viewers, every
// catalog action on every doc, folder, play, project, stack and channel answers as the check does, and so does each line
// of an explanation.
func TestExplain_AnswersWhatTheCheckAnswers(t *testing.T) {
	f := listParityFixture(t)
	postsChannel(t, f, f.channel.ID)
	require.NoError(t, f.svc.accessSvc.SetRules(as(uOwner), uOwner, "channel", []string{f.channel.ID}, access.Targets{UserIDs: []string{uReader}},
		[]permissions.Action{permissions.ChannelsPost}, access.StateAllow))
	require.NoError(t, f.svc.accessSvc.SetRules(as(uOwner), uOwner, "channel", []string{f.channel.ID}, access.Targets{RoleIDs: []string{"role-plain"}},
		[]permissions.Action{permissions.ChannelsRead}, access.StateDeny))
	a := f.svc.accessSvc
	ctx := context.Background()
	ps, err := f.store.Plays.List(ctx, wsDefault)
	require.NoError(t, err)
	require.NotEmpty(t, ps)
	checks := map[string]func(user, id string, action permissions.Action) bool{
		"doc": func(user, id string, action permissions.Action) bool {
			ok, err := a.Can(ctx, user, id, action)
			require.NoError(t, err)
			return ok
		},
		"play": func(user, id string, action permissions.Action) bool {
			return a.HasPermission(ctx, user, wsDefault, action, "play", id)
		},
		"project": func(user, id string, action permissions.Action) bool { return a.CanInProject(ctx, user, id, action) },
		"folder": func(user, id string, action permissions.Action) bool {
			return a.HasPermission(ctx, user, wsDefault, action, "folder", id)
		},
		"stack": func(user, id string, action permissions.Action) bool {
			return a.HasPermission(ctx, user, wsDefault, action, "stack", id)
		},
		"channel": func(user, id string, action permissions.Action) bool {
			if action == permissions.ChannelsRead {
				_, err := f.svc.chatSvc.GetConversation(identity.WithActor(ctx, identity.Actor{ID: user}), id)
				return err == nil
			}
			if action == permissions.ChannelsPost {
				return a.ChannelPost(ctx, user, wsDefault, id) == nil
			}
			return a.HasPermission(ctx, user, wsDefault, action, "channel", id)
		},
	}
	entities := map[string][]string{"doc": {f.doc, "doc-private", "doc-shared", "doc-client", "doc-folder"}, "project": {pGeneral, pClient},
		"folder": {parityFolder}, "stack": {f.stack, "stack-client"}, "channel": {f.channel.ID}}
	for _, p := range ps {
		entities["play"] = append(entities["play"], p.ID)
	}

	for _, user := range []string{uOwner, uReader, uWriter, uPlain, uOverwrite, uOutsider, uClient} {
		for kind, ids := range entities {
			for _, id := range ids {
				for _, action := range permissions.AllActions() {
					held, _, err := a.Decide(ctx, user, kind, id, action)
					require.NoError(t, err)
					assert.Equal(t, checks[kind](user, id, action), held, "%s %s on %s %s", user, action, kind, id)
				}
				e, err := a.Explain(ctx, uOwner, kind, id, user)
				require.NoError(t, err)
				for _, answer := range e.Effective {
					assert.Equal(t, checks[kind](user, id, answer.Action), answer.Allowed, "explained %s %s on %s %s", user, answer.Action, kind, id)
					assert.NotEmpty(t, answer.Source.Sentence)
				}
			}
		}
	}
}

// accessHTTP calls the permissions gateway as user and returns the status and body.
func accessHTTP(t *testing.T, f permFixture, user, path string) (int, string) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req = req.WithContext(identity.WithActor(req.Context(), identity.Actor{ID: user}))
	rec := httptest.NewRecorder()
	access.NewHandler(f.svc.accessSvc).Routes().ServeHTTP(rec, req)
	return rec.Code, rec.Body.String()
}

// TestExplain_AnotherPersonTakesTheManagePermission: explaining someone else without the entity's manage permission
// is refused with what refused it, over HTTP and MCP alike, and a doc the caller can't read is not found.
func TestExplain_AnotherPersonTakesTheManagePermission(t *testing.T) {
	f := listParityFixture(t)
	listTool := tool(t, access.MCPTools(f.svc.accessSvc), "permission_overwrite_list")
	explainOverMCP := func(user, args string) (any, error) {
		return listTool.Call(identity.WithActor(context.Background(), identity.Actor{ID: user}), json.RawMessage(args))
	}

	status, body := accessHTTP(t, f, uReader, "/api/permissions/explain?resource_type=doc&resource_id=doc-1&user_id="+uPlain)
	assert.Equal(t, http.StatusForbidden, status)
	assert.Contains(t, body, "permissions:write required (Not allowed: role Reader doesn't grant it)")
	_, err := explainOverMCP(uReader, `{"resource_type":"doc","resource_id":"doc-1","user_id":"`+uPlain+`"}`)
	require.ErrorIs(t, err, apperrs.ErrForbidden)
	assert.Contains(t, err.Error(), "Not allowed: role Reader doesn't grant it")

	status, _ = accessHTTP(t, f, uReader, "/api/permissions/explain?resource_type=doc&resource_id=doc-private")
	assert.Equal(t, http.StatusNotFound, status, "a doc denied to them")
	_, err = explainOverMCP(uReader, `{"resource_type":"doc","resource_id":"doc-private","user_id":"`+uReader+`"}`)
	require.ErrorIs(t, err, apperrs.ErrNotFound)

	status, body = accessHTTP(t, f, uReader, "/api/permissions/explain?resource_type=doc&resource_id=doc-1")
	assert.Equal(t, http.StatusOK, status, "their own access")
	assert.Contains(t, body, "Allowed by role Reader")
	status, body = accessHTTP(t, f, uOwner, "/api/permissions/explain?resource_type=doc&resource_id=doc-private&user_id="+uReader)
	assert.Equal(t, http.StatusOK, status)
	assert.Contains(t, body, "Denied on doc Spec doc-private for u-reader")
}

// TestDocEdit_ForbiddenNamesTheRuleThatDeniedIt: an edit refused by a doc's rule says so over HTTP and MCP, and an
// edit by someone outside the workspace is not found with nothing more.
func TestDocEdit_ForbiddenNamesTheRuleThatDeniedIt(t *testing.T) {
	f := newPermFixture(t)
	require.NoError(t, f.store.Access.Set(context.Background(), "doc", f.doc, uEditor, nil, grant("docs:write")))
	routes := docs.NewHandler(f.svc.docsSvc).Routes()
	editOverHTTP := func(user string) (int, string) {
		req := httptest.NewRequest(http.MethodPut, "/api/docs/"+f.doc, strings.NewReader(`{"title":"Spec","body":"{\"type\":\"doc\",\"content\":[]}"}`))
		req = req.WithContext(identity.WithActor(req.Context(), identity.Actor{ID: user}))
		rec := httptest.NewRecorder()
		routes.ServeHTTP(rec, req)
		return rec.Code, rec.Body.String()
	}
	update := tool(t, docs.MCPTools(f.svc.docsSvc), "doc_update")
	editOverMCP := func(user string) error {
		_, err := update.Call(identity.WithActor(context.Background(), identity.Actor{ID: user}), json.RawMessage(`{"id":"`+f.doc+`","title":"Spec 2"}`))
		return err
	}
	const denied = "docs:write required (Denied on doc Spec for u-editor)"

	status, body := editOverHTTP(uEditor)
	assert.Equal(t, http.StatusForbidden, status)
	assert.Contains(t, body, denied)
	assert.Contains(t, body, `"details":{"action":"docs:write","sentence":"Denied on doc Spec for u-editor"}`, "the sentence alone, for a client to show")
	err := editOverMCP(uEditor)
	require.ErrorIs(t, err, apperrs.ErrForbidden)
	assert.Contains(t, err.Error(), denied)

	status, body = editOverHTTP(uOutsider)
	assert.Equal(t, http.StatusNotFound, status)
	assert.NotContains(t, body, "required")
	err = editOverMCP(uOutsider)
	require.ErrorIs(t, err, apperrs.ErrNotFound)
	assert.NotContains(t, err.Error(), "required")
}
