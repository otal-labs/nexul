package access

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// TestDecide_FolderBetweenProjectAndDoc is the spec's worked example with Pricing and Roadmap in Storefront's folder Sales.
func TestDecide_FolderBetweenProjectAndDoc(t *testing.T) {
	const (
		read  = permissions.DocsRead
		write = permissions.DocsWrite
	)
	dev := HeldRole{ID: "role-dev", Permissions: permissions.SetOf(read, write)}
	front := HeldRole{ID: "role-front", Permissions: permissions.SetOf(read, write)}
	client := HeldRole{ID: "role-client", Permissions: permissions.SetOf(read)}
	held := map[string][]HeldRole{"alice": {{ID: "role-owner", IsOwner: true}}, "bob": {dev, front}, "lena": {client}, "sam": {client}}
	project := []*Overwrite{roleRule("project", "p-store", TargetPerson, "bob", StateDeny, write)}
	sales := []*Overwrite{
		roleRule("folder", "f-sales", TargetRole, "role-client", StateDeny, read),
		roleRule("folder", "f-sales", TargetPerson, "lena", StateAllow, read),
		roleRule("folder", "f-sales", TargetRole, "role-dev", StateAllow, write),
	}
	pricing := []*Overwrite{
		roleRule("doc", "d-price", TargetRole, "role-dev", StateDeny, write),
		roleRule("doc", "d-price", TargetRole, "role-front", StateAllow, write),
		roleRule("doc", "d-price", TargetRole, "role-client", StateDeny, read),
		roleRule("doc", "d-price", TargetPerson, "sam", StateAllow, read),
	}
	// reaching is what a read of one entity's rules hands back for user: the rule naming them, and every role's.
	reaching := func(user string, rules []*Overwrite) *Rules {
		r := &Rules{}
		for _, ow := range rules {
			if ow.Target == TargetPerson && ow.UserID == user {
				r.Person = ow
			}
			if ow.Target == TargetRole {
				r.Roles = append(r.Roles, ow)
			}
		}
		return r
	}
	onFolder := func(target, id, state string) Source { return src(SourceRule, "folder", "f-sales", target, id, state) }
	onDoc := func(target, id, state string) Source { return src(SourceRule, "doc", "d-price", target, id, state) }
	tests := []struct {
		name        string
		user        string
		action      permissions.Action
		folder, doc []*Overwrite
		allowed     bool
		source      Source
		sentence    string
	}{
		{"the Owner", "alice", write, sales, pricing, true,
			src(SourceOwner, "workspace", "ws", TargetRole, "role-owner", StateAllow), "Owner of Acme"},
		{"a doc in Main answers from the project", "bob", write, nil, nil, false,
			src(SourceRule, "project", "p-store", TargetPerson, "bob", StateDeny), "Denied on project Storefront for bob"},
		{"the folder's role rule beats the project's rule naming him", "bob", write, sales, nil, true,
			onFolder(TargetRole, "role-dev", StateAllow), "Allowed on folder Sales for role Developer"},
		{"the doc's role rules beat the folder's, Allow winning between them", "bob", write, sales, pricing, true,
			onDoc(TargetRole, "role-front", StateAllow), "Allowed on doc Pricing for role Frontend"},
		{"on the folder the person beats the role", "lena", read, sales, nil, true,
			onFolder(TargetPerson, "lena", StateAllow), "Allowed on folder Sales for lena"},
		{"the doc's role rule beats the folder's rule naming her", "lena", read, sales, pricing, false,
			onDoc(TargetRole, "role-client", StateDeny), "Denied on doc Pricing for role Client"},
		{"the folder's role rule denies a role that grants it", "sam", read, sales, nil, false,
			onFolder(TargetRole, "role-client", StateDeny), "Denied on folder Sales for role Client"},
		{"the doc's rule naming him beats the folder's role rule", "sam", read, sales, pricing, true,
			onDoc(TargetPerson, "sam", StateAllow), "Allowed on doc Pricing for sam"},
		{"nothing on the chain sets it", "sam", write, sales, pricing, false,
			src(SourceRole, "workspace", "ws", TargetRole, "role-client", StateDeny), "Not allowed: role Client doesn't grant it"},
	}
	s := newService(newFakeRepo(), newFakeUsers())
	s.SetNames(&fakeNames{})
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			roles := RolesHeld(held[tt.user]...)
			ws := workspaceLayers{userID: tt.user, workspaceID: "ws", member: true, owner: roles.IsOwnerRole, roles: roles}
			ws = ws.in("p-store", reaching(tt.user, project))
			ws.listed = true
			allowed, got := decide(ws, tt.action, func() []*Rules {
				return []*Rules{reaching(tt.user, tt.folder), reaching(tt.user, tt.doc)}
			})
			assert.Equal(t, tt.allowed, allowed)
			assert.Equal(t, tt.source, got)
			assert.Equal(t, tt.sentence, s.namer(context.Background()).named(got).Sentence)
		})
	}
}

// foldersFixture is projectsFixture with folder f-open, holding d-open, in p-open and f-hidden in p-hidden.
func foldersFixture(t *testing.T) (*Service, *fakeRepo) {
	t.Helper()
	s, repo := projectsFixture(t)
	s.SetDocWorkspaces(&fakeDocWorkspace{
		byDoc: map[string]string{"d-open": "ws", "d-hidden": "ws"}, projectOf: map[string]string{"d-open": "p-open", "d-hidden": "p-hidden"},
		folderOf: map[string]string{"d-open": "f-open"},
		folders:  map[string]DocScope{"f-open": {WorkspaceID: "ws", ProjectID: "p-open"}, "f-hidden": {WorkspaceID: "ws", ProjectID: "p-hidden"}},
	})
	s.SetRoleWorkspaces(fakeRoleWorkspaces{"role-client": "ws", "role-team": "ws"})
	repo.docFolders = map[string]string{"d-open": "f-open"}
	return s, repo
}

func TestSetRules_OnAFolder(t *testing.T) {
	s, repo := foldersFixture(t)
	read := []permissions.Action{permissions.DocsRead}
	ctx := t.Context()

	require.ErrorIs(t, s.SetRules(ctx, "client", "folder", []string{"f-hidden"}, Targets{UserIDs: []string{"team"}}, read, StateDeny), apperrs.ErrNotFound,
		"a folder of a project they are not in")
	require.ErrorIs(t, s.SetRules(ctx, "team", "folder", []string{"f-open"}, Targets{UserIDs: []string{"client"}}, read, StateDeny), apperrs.ErrForbidden,
		"a folder's rules take permissions:write on it")
	assert.False(t, s.ManagesRules(ctx, "team", "folder", "f-open"))
	assert.True(t, s.ManagesRules(ctx, "owner", "folder", "f-open"))

	require.NoError(t, s.SetRules(ctx, "owner", "folder", []string{"f-open"}, Targets{UserIDs: []string{"team"}}, read, StateDeny))
	assert.Equal(t, GrantEvent{ResourceType: "folder", ResourceID: "f-open", Target: TargetPerson, UserID: "team", ActorID: "owner"},
		repo.events[len(repo.events)-1].Payload)
	can, err := s.Can(ctx, "team", "d-open", permissions.DocsRead)
	require.NoError(t, err)
	assert.False(t, can, "a folder's Deny reaches the docs in it")
	held, source, err := s.Decide(ctx, "team", "doc", "d-open", permissions.DocsRead)
	require.NoError(t, err)
	assert.False(t, held)
	assert.Equal(t, src(SourceRule, "folder", "f-open", TargetPerson, "team", StateDeny), source)
	setOverwrite(repo, "doc", "d-open", "team", permissions.SetOf(permissions.DocsRead))
	can, err = s.Can(ctx, "team", "d-open", permissions.DocsRead)
	require.NoError(t, err)
	assert.True(t, can, "the doc's own rule beats its folder's")

	rules, err := s.ListRules(ctx, "owner", "folder", "f-open")
	require.NoError(t, err)
	assert.Len(t, rules, 1)
	_, err = s.ListRules(ctx, "client", "folder", "f-hidden")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
	_, err = s.ListRules(ctx, "owner", "folder", "f-gone")
	require.ErrorIs(t, err, apperrs.ErrNotFound)

	setOverwrite(repo, "folder", "f-open", "client", permissions.SetOf(permissions.PermissionsWrite))
	assert.True(t, s.ManagesRules(ctx, "client", "folder", "f-open"), "permissions:write on the folder manages it")
	require.NoError(t, s.SetRules(ctx, "client", "folder", []string{"f-open"}, Targets{RoleIDs: []string{"role-team"}}, read, StateAllow),
		"allowing what they hold there")
	require.ErrorIs(t, s.SetRules(ctx, "client", "folder", []string{"f-open"}, Targets{RoleIDs: []string{"role-team"}},
		[]permissions.Action{permissions.DocsDelete}, StateAllow), apperrs.ErrForbidden, "nobody allows what they don't hold there")
	for _, a := range []permissions.Action{permissions.ProjectsOpen, permissions.PlaysRun, permissions.TicketsWrite} {
		require.ErrorIs(t, s.SetRules(ctx, "owner", "folder", []string{"f-open"}, Targets{UserIDs: []string{"team"}}, []permissions.Action{a}, StateAllow),
			apperrs.ErrInvalid, "%s on a folder", a)
	}
}

func TestExplain_OnAFolder(t *testing.T) {
	s, repo := foldersFixture(t)
	ctx := t.Context()
	require.NoError(t, repo.SetRole(ctx, "folder", "f-open", "role-team", nil, permissions.SetOf(permissions.DocsRead)))
	setOverwrite(repo, "folder", "f-open", "client", permissions.SetOf(permissions.DocsWrite))

	e, err := s.Explain(ctx, "owner", "folder", "f-open", "client")
	require.NoError(t, err)
	byAction := map[permissions.Action]Answer{}
	for _, a := range e.Effective {
		byAction[a.Action] = a
	}
	assert.Len(t, e.Effective, len(docActions))
	assert.True(t, byAction[permissions.DocsWrite].Allowed)
	assert.Equal(t, "Allowed on folder (unknown) for someone", byAction[permissions.DocsWrite].Source.Sentence)
	assert.False(t, byAction[permissions.DocsWrite].Fallback.Allowed, "at Fallback the role answers")

	_, err = s.Explain(ctx, "client", "folder", "f-hidden", "")
	require.ErrorIs(t, err, apperrs.ErrNotFound, "a folder of a project they are not in")
	_, err = s.Explain(ctx, "owner", "folder", "f-gone", "")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
	assert.True(t, s.HasPermission(ctx, "client", "ws", permissions.DocsWrite, "folder", "f-open"))
}

// TestExplainRole_ADocsAnswerPassesThroughItsFolder: a role's answer on a doc, and on its folder, carries the folder's rule.
func TestExplainRole_ADocsAnswerPassesThroughItsFolder(t *testing.T) {
	s, _ := explainFixture(t)
	s.SetRoleWorkspaces(fakeRoleWorkspaces{"role-dev": "ws"})
	s.SetDocWorkspaces(&fakeDocWorkspace{
		byDoc: map[string]string{"d-price": "ws"}, projectOf: map[string]string{"d-price": "p-store"}, folderOf: map[string]string{"d-price": "f-sales"},
		folders: map[string]DocScope{"f-sales": {WorkspaceID: "ws", ProjectID: "p-store"}},
	})
	ctx := t.Context()
	require.NoError(t, s.repo.SetRole(ctx, "folder", "f-sales", "role-dev", nil, permissions.SetOf(permissions.DocsRead)))
	for _, entity := range [][2]string{{"doc", "d-price"}, {"folder", "f-sales"}} {
		e, err := s.ExplainRole(ctx, "alice", entity[0], entity[1], "role-dev")
		require.NoError(t, err)
		for _, a := range e.Effective {
			if a.Action == permissions.DocsRead {
				assert.False(t, a.Allowed, entity[0])
				assert.Equal(t, "Denied on folder Sales for role Developer", a.Source.Sentence, entity[0])
			}
		}
	}
}
