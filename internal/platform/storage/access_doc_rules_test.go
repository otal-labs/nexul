package storage

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/platform/permissions"
	"github.com/otal-labs/nexul/internal/roles"
	"github.com/otal-labs/nexul/internal/tenancy"
)

// TestAccessRepo_DocRuleTurns_DecidesEveryRuleShape: u-r, holding role-r and role-s, has a rule of every shape on the
// paged docs, and only the docs whose answer turns come back, within the ruled projects alone.
func TestAccessRepo_DocRuleTurns_DecidesEveryRuleShape(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	seedPagedDocs(t, s)
	ctx := t.Context()
	now := time.Now()
	require.NoError(t, s.Workspaces.Create(ctx, newTestWorkspace("ws-2", "Beta")))
	for id, workspaceID := range map[string]string{"role-r": "workspace-default", "role-s": "workspace-default", "role-x": "workspace-default", "role-far": "ws-2"} {
		require.NoError(t, s.Roles.Create(ctx, &roles.Role{ID: id, WorkspaceID: workspaceID, Name: id, CreatedAt: now, UpdatedAt: now}))
	}
	mustCreateUser(t, s, "u-r", "ur")
	require.NoError(t, s.WorkspaceMembers.AddMember(ctx, &tenancy.Member{UserID: "u-r", WorkspaceID: "workspace-default", RoleID: "role-r", CreatedAt: now}))
	require.NoError(t, s.WorkspaceMembers.AddMember(ctx, &tenancy.Member{UserID: "u-r", WorkspaceID: "ws-2", RoleID: "role-far", CreatedAt: now}))
	_, err := s.db.Exec(`INSERT INTO member_roles (user_id, workspace_id, role_id, created_at) VALUES ('u-r', 'workspace-default', 'role-s', 1)`)
	require.NoError(t, err)
	read := permissions.SetOf(permissions.DocsRead)
	for _, r := range []struct {
		doc, role, person string
		allow, deny       permissions.Set
	}{
		{doc: "doc-04", person: "u-r", allow: read},
		{doc: "doc-01", person: "u-r", deny: read},
		{doc: "doc-02", role: "role-r", deny: read},
		{doc: "doc-08", role: "role-s", allow: read},
		{doc: "doc-03", role: "role-r", allow: read},
		{doc: "doc-03", person: "u-r", deny: read},
		{doc: "doc-06", role: "role-r", deny: read},
		{doc: "doc-06", person: "u-r", allow: read},
		{doc: "doc-07", role: "role-r", deny: read},
		{doc: "doc-07", role: "role-s", allow: read},
		{doc: "doc-10", person: "u-r", allow: read, deny: read},
		{doc: "doc-09", role: "role-x", deny: read},
		{doc: "doc-11", role: "role-far", deny: read},
		{doc: "doc-05", person: "u-r", allow: permissions.SetOf(permissions.DocsWrite)},
		{doc: "doc-13", person: "u-other", deny: read},
	} {
		if r.person == "u-other" {
			mustCreateUser(t, s, "u-other", "other")
		}
		if r.person != "" {
			require.NoError(t, s.Access.Set(ctx, "doc", r.doc, r.person, r.allow, r.deny))
			continue
		}
		require.NoError(t, s.Access.SetRole(ctx, "doc", r.doc, r.role, r.allow, r.deny))
	}

	tests := []struct {
		name                    string
		ruled, readable         []string
		wantAllowed, wantDenied []string
	}{
		{"both projects ruled", []string{"project-general", "p-hidden"}, []string{"project-general"},
			[]string{"doc-04", "doc-08"}, []string{"doc-01", "doc-02", "doc-03", "doc-10"}},
		{"the project listed alone", []string{"project-general"}, []string{"project-general"},
			[]string{}, []string{"doc-01", "doc-02", "doc-03", "doc-10"}},
		{"a project whose answer reads them turns only denials", []string{"p-hidden"}, []string{"project-general", "p-hidden"},
			[]string{}, []string{}},
		{"nothing ruled", []string{}, []string{"project-general"}, []string{}, []string{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			allowed, denied, err := s.Access.DocRuleTurns(ctx, "u-r", permissions.DocsRead, tt.ruled, tt.readable)
			require.NoError(t, err)
			assert.Equal(t, tt.wantAllowed, allowed)
			assert.Equal(t, tt.wantDenied, denied)
		})
	}
}
