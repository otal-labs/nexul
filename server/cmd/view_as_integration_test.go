package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/auth"
	"github.com/otal-labs/nexul/internal/chat"
	"github.com/otal-labs/nexul/internal/platform/storage/testutil/migrated"
	"github.com/otal-labs/nexul/internal/roles"
	"github.com/otal-labs/nexul/internal/tenancy"
	"github.com/otal-labs/nexul/internal/workspace"
)

const (
	uAuditor  = "u-auditor"
	uLite     = "u-lite"
	uOwnerToo = "u-owner-too"
	wsOther   = "workspace-other"
)

// viewAsFixture is the permission fixture behind the real router: sessions for everyone, an Auditor who holds only
// members:view_as and tickets:read, a Lite member who holds only tickets:read, a second Owner, a second workspace
// that Reader also belongs to, and a second project nobody but the Owner is listed in.
type viewAsFixture struct {
	h        http.Handler
	f        permFixture
	sessions map[string]string
}

func newViewAsFixture(t *testing.T) viewAsFixture {
	t.Helper()
	ctx := t.Context()
	db := migrated.DB(t)
	h, store, svc := routerOver(t, db)
	f := seedPermFixture(t, db, svc, store)
	now := time.Now()

	for _, id := range []string{uAuditor, uLite, uOwnerToo} {
		_, _, err := store.Users.UpsertUser(ctx, &auth.Identity{UserID: id, Provider: auth.ProviderGitHub, ProviderUserID: id, Login: id, Name: strings.TrimPrefix(id, "u-")})
		require.NoError(t, err)
	}
	for _, r := range []*roles.Role{
		{ID: "role-auditor", WorkspaceID: wsDefault, Name: "Auditor", Permissions: grant("members:view_as", "tickets:read")},
		{ID: "role-lite", WorkspaceID: wsDefault, Name: "Lite", Permissions: grant("tickets:read")},
	} {
		r.CreatedAt, r.UpdatedAt = now, now
		require.NoError(t, store.Roles.Create(ctx, r))
	}
	for user, role := range map[string]string{uAuditor: "role-auditor", uLite: "role-lite", uOwnerToo: "role-owner"} {
		require.NoError(t, store.WorkspaceMembers.AddMember(ctx, &tenancy.Member{UserID: user, WorkspaceID: wsDefault, RoleID: role, CreatedAt: now}))
	}
	listRolesIn(t, db, pGeneral)

	require.NoError(t, store.Workspaces.Create(ctx, &tenancy.Workspace{ID: wsOther, Name: "Other", Slug: "other", CreatedAt: now, UpdatedAt: now}))
	require.NoError(t, store.Roles.Create(ctx, &roles.Role{ID: "role-other", WorkspaceID: wsOther, Name: "Member", Permissions: grant("logs:read"), CreatedAt: now, UpdatedAt: now}))
	require.NoError(t, store.WorkspaceMembers.AddMember(ctx, &tenancy.Member{UserID: uReader, WorkspaceID: wsOther, RoleID: "role-other", CreatedAt: now}))

	require.NoError(t, store.Projects.Create(ctx, &workspace.Project{ID: "project-two", Name: "Two", Prefix: "TWO", WorkspaceID: wsDefault, CreatedAt: now, UpdatedAt: now}))

	sessions := map[string]string{}
	for _, id := range []string{uOwner, uAuditor, uManager, uReader, uLite} {
		token, err := svc.authSvc.CreateSession(ctx, id)
		require.NoError(t, err)
		sessions[id] = token
	}
	return viewAsFixture{h: h, f: f, sessions: sessions}
}

// do sends one request with the viewer's session token and any extra headers.
func (v viewAsFixture) do(t *testing.T, viewer, method, path, body string, headers ...string) *httptest.ResponseRecorder {
	t.Helper()
	return v.doWith(t, v.sessions[viewer], method, path, body, headers...)
}

func (v viewAsFixture) doWith(t *testing.T, token, method, path, body string, headers ...string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	for i := 0; i+1 < len(headers); i += 2 {
		req.Header.Set(headers[i], headers[i+1])
	}
	rec := httptest.NewRecorder()
	v.h.ServeHTTP(rec, req)
	return rec
}

// asMember are the headers a web view of target in the default workspace carries.
func asMember(target string) []string {
	return []string{headerViewAsUser, target, headerViewAsWorkspace, wsDefault}
}

func startViewAs(workspaceID, target string) string {
	return "/api/workspaces/" + workspaceID + "/members/" + target + "/view-as"
}

func decodeBody[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	var out T
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &out), rec.Body.String())
	return out
}

func idsOf[T any](items []T, id func(T) string) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		out = append(out, id(item))
	}
	return out
}

func projectIDs(t *testing.T, rec *httptest.ResponseRecorder) []string {
	t.Helper()
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	return idsOf(decodeBody[[]workspace.Project](t, rec), func(p workspace.Project) string { return p.ID })
}

func listedConversationIDs(t *testing.T, rec *httptest.ResponseRecorder) []string {
	t.Helper()
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	return idsOf(decodeBody[[]chat.Conversation](t, rec), func(c chat.Conversation) string { return c.ID })
}

// TestViewAs_Start_RefusesWhatTheCheckRefuses is the check behind both the start route and every header request: each
// row fails if its rule is removed from tenancy.CheckViewAs.
func TestViewAs_Start_RefusesWhatTheCheckRefuses(t *testing.T) {
	t.Parallel()
	v := newViewAsFixture(t)
	tests := []struct {
		name     string
		viewer   string
		target   string
		want     int
		contains string
	}{
		{"a caller without members:view_as", uManager, uReader, http.StatusForbidden, "members:view_as"},
		{"a target holding what the caller can't give", uAuditor, uReader, http.StatusForbidden, "docs:read"},
		{"an Owner target", uOwner, uOwnerToo, http.StatusBadRequest, "nothing to check"},
		{"oneself", uOwner, uOwner, http.StatusBadRequest, "view as yourself"},
		{"someone outside the workspace", uOwner, uOutsider, http.StatusNotFound, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := v.do(t, tt.viewer, http.MethodPost, startViewAs(wsDefault, tt.target), "")
			require.Equal(t, tt.want, rec.Code, rec.Body.String())
			assert.Contains(t, rec.Body.String(), tt.contains)
		})
	}
}

// TestViewAs_Start_NamesTheMemberAndIsAuditedAsTheViewer: the route answers who is being viewed, and its audit row is the
// one record of who viewed as whom.
func TestViewAs_Start_NamesTheMemberAndIsAuditedAsTheViewer(t *testing.T) {
	t.Parallel()
	v := newViewAsFixture(t)

	rec := v.do(t, uAuditor, http.MethodPost, startViewAs(wsDefault, uLite), "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.JSONEq(t, `{"workspace_id":"workspace-default","user_id":"u-lite","name":"lite","avatar_url":""}`, rec.Body.String())

	rows := auditRows(t, v.f.store)
	require.Len(t, rows, 1)
	assert.Equal(t, uAuditor, rows[0].ActorID)
	assert.Equal(t, "POST /api/workspaces/workspace-default/members/u-lite/view-as", rows[0].Action)
}

// TestViewAs_Headers_RefusedCheckIs403WithTheRefusedHeader: the web leaves the mode on X-View-As: refused, so a request
// the check refuses must never be a 401 (which signs the viewer out) and one carrying half the pair is a 400.
func TestViewAs_Headers_RefusedCheckIs403WithTheRefusedHeader(t *testing.T) {
	t.Parallel()
	v := newViewAsFixture(t)

	rec := v.do(t, uManager, http.MethodGet, "/api/projects?workspace_id="+wsDefault, "", asMember(uReader)...)
	assert.Equal(t, http.StatusForbidden, rec.Code, rec.Body.String())
	assert.Equal(t, "refused", rec.Header().Get("X-View-As"))

	rec = v.do(t, uOwner, http.MethodGet, "/api/projects?workspace_id="+wsDefault, "", headerViewAsUser, uReader)
	assert.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
	rec = v.do(t, uOwner, http.MethodGet, "/api/projects?workspace_id="+wsDefault, "", headerViewAsWorkspace, wsDefault)
	assert.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
}

// TestViewAs_Headers_OnlyABrowserSessionMayCarryThem: a personal access token, or an integration or automation
// token, that sends the headers is turned away, so the swap can't be reached with a long-lived credential.
func TestViewAs_Headers_OnlyABrowserSessionMayCarryThem(t *testing.T) {
	t.Parallel()
	v := newViewAsFixture(t)
	pat, _, err := v.f.svc.authSvc.MintPAT(t.Context(), uOwner, "view as")
	require.NoError(t, err)

	rec := v.doWith(t, pat, http.MethodGet, "/api/projects?workspace_id="+wsDefault, "", asMember(uReader)...)
	assert.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
	rec = v.doWith(t, pat, http.MethodGet, "/api/projects?workspace_id="+wsDefault, "")
	assert.Equal(t, http.StatusOK, rec.Code, "the same token is fine without them")
}

// TestViewAs_ServesTheTargetsView_AndNothingIsWritten: reads answer as the target, a write is refused before the audit
// middleware can record it, and the reads that arrive as POST still work.
func TestViewAs_ServesTheTargetsView_AndNothingIsWritten(t *testing.T) {
	t.Parallel()
	v := newViewAsFixture(t)
	list := "/api/projects?workspace_id=" + wsDefault

	assert.ElementsMatch(t, []string{"project-general", "project-two"}, projectIDs(t, v.do(t, uOwner, http.MethodGet, list, "")), "the Owner is in both projects")
	assert.Equal(t, []string{"project-general"}, projectIDs(t, v.do(t, uOwner, http.MethodGet, list, "", asMember(uReader)...)), "viewing as Reader lists only the project Reader is in")

	rec := v.do(t, uOwner, http.MethodPost, "/api/chat/channels", `{"workspace_id":"workspace-default","name":"ops"}`, asMember(uReader)...)
	assert.Equal(t, http.StatusForbidden, rec.Code)
	assert.Contains(t, rec.Body.String(), "You're viewing as u-reader; changes are off")
	assert.Empty(t, auditRows(t, v.f.store), "a refused write leaves no audit row")

	rec = v.do(t, uOwner, http.MethodPost, "/api/mentions/resolve", `{"refs":[]}`, asMember(uReader)...)
	assert.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
}

// TestViewAs_PersonalThingsAndOtherWorkspacesAreNotFound: sign-in state, personal computers, the inbox and every other
// workspace stay out of a view, however the target's own account could reach them.
func TestViewAs_PersonalThingsAndOtherWorkspacesAreNotFound(t *testing.T) {
	t.Parallel()
	v := newViewAsFixture(t)
	reader := v.sessions[uReader]

	for _, path := range []string{"/api/auth/tokens", "/api/auth/sessions", "/api/pairing/computers", "/api/notifications", "/api/workspaces/" + wsOther + "/me"} {
		assert.Equal(t, http.StatusOK, v.doWith(t, reader, http.MethodGet, path, "").Code, "%s is open to Reader themselves", path)
		assert.Equal(t, http.StatusNotFound, v.do(t, uOwner, http.MethodGet, path, "", asMember(uReader)...).Code, "%s in a view", path)
	}
	assert.Equal(t, http.StatusNotFound, v.do(t, uOwner, http.MethodGet, "/api/chat/conversations?workspace_id="+wsOther, "", asMember(uReader)...).Code,
		"a workspace named in a query is as unreachable as one named in the path")

	workspaces := decodeBody[[]map[string]any](t, v.do(t, uOwner, http.MethodGet, "/api/workspaces", "", asMember(uReader)...))
	require.Len(t, workspaces, 1, "the switcher lists only the pinned workspace")
	assert.Equal(t, wsDefault, workspaces[0]["id"])
	assert.Len(t, decodeBody[[]map[string]any](t, v.doWith(t, reader, http.MethodGet, "/api/workspaces", "")), 2)
}

// TestViewAs_AuthMe_AnswersAsTheTargetOverThePinnedWorkspace: the instance permissions a client shows are the target's
// roles in the viewed workspace alone, not every workspace they belong to.
func TestViewAs_AuthMe_AnswersAsTheTargetOverThePinnedWorkspace(t *testing.T) {
	t.Parallel()
	v := newViewAsFixture(t)
	type me struct {
		User                struct{ ID string }
		InstancePermissions []string `json:"instance_permissions"`
	}

	own := decodeBody[me](t, v.doWith(t, v.sessions[uReader], http.MethodGet, "/api/auth/me", ""))
	assert.Contains(t, own.InstancePermissions, "logs:read", "Reader's role in the other workspace gives it")

	viewed := decodeBody[me](t, v.do(t, uOwner, http.MethodGet, "/api/auth/me", "", asMember(uReader)...))
	assert.Equal(t, uReader, viewed.User.ID)
	assert.NotContains(t, viewed.InstancePermissions, "logs:read")
	assert.Contains(t, viewed.InstancePermissions, "tickets:read")
}

// TestViewAs_MembersOnlyConversationsShowOnlyWhatTheViewerCouldRead: a DM or private channel the target is in is hidden
// from the list and from a direct read unless the viewer is in it too.
func TestViewAs_MembersOnlyConversationsShowOnlyWhatTheViewerCouldRead(t *testing.T) {
	t.Parallel()
	v := newViewAsFixture(t)
	chatSvc := v.f.svc.chatSvc
	withWriter, err := chatSvc.CreateDM(as(uWriter), wsDefault, uWriter, []string{uLite})
	require.NoError(t, err)
	withAuditor, err := chatSvc.CreateDM(as(uOwner), wsDefault, uOwner, []string{uAuditor, uLite})
	require.NoError(t, err)
	secret, err := chatSvc.CreatePrivateChannel(as(uWriter), wsDefault, uWriter, "secret", chat.KindChannel, []string{uLite})
	require.NoError(t, err)

	list := "/api/chat/conversations?workspace_id=" + wsDefault
	own := listedConversationIDs(t, v.doWith(t, v.sessions[uLite], http.MethodGet, list, ""))
	assert.Subset(t, own, []string{withWriter.ID, withAuditor.ID, secret.ID}, "Lite sees all three themselves")

	viewed := listedConversationIDs(t, v.do(t, uAuditor, http.MethodGet, list, "", asMember(uLite)...))
	assert.Contains(t, viewed, withAuditor.ID, "a DM with the viewer stays")
	assert.NotContains(t, viewed, withWriter.ID)
	assert.NotContains(t, viewed, secret.ID)

	for _, hidden := range []string{withWriter.ID, secret.ID} {
		assert.Equal(t, http.StatusNotFound, v.do(t, uAuditor, http.MethodGet, "/api/chat/conversations/"+hidden+"/messages", "", asMember(uLite)...).Code)
		assert.Equal(t, http.StatusOK, v.doWith(t, v.sessions[uLite], http.MethodGet, "/api/chat/conversations/"+hidden+"/messages", "").Code)
	}
	assert.Equal(t, http.StatusOK, v.do(t, uAuditor, http.MethodGet, "/api/chat/conversations/"+withAuditor.ID+"/messages", "", asMember(uLite)...).Code)
}
