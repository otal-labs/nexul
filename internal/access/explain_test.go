package access

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// fakeNames resolves the explain fixture's names and counts how often it is asked.
type fakeNames struct {
	mu    sync.Mutex
	calls int
}

var explainNames = map[string]string{
	"workspace/ws": "Acme", "project/p-store": "Storefront", "project/p-back": "Backoffice", "doc/d-price": "Pricing",
	"play/play-notes": "Release notes", "role/role-owner": "Owner", "role/role-dev": "Developer", "role/role-plain": "Member",
	"role/role-client": "Client", "person/alice": "alice", "person/bob": "bob", "person/sam": "sam", "person/lena": "lena",
	"person/outsider": "outsider",
}

func (f *fakeNames) Name(_ context.Context, entityType, id string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	name, ok := explainNames[entityType+"/"+id]
	if !ok {
		return "", apperrs.ErrNotFound
	}
	return name, nil
}

func (f *fakeNames) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls
}

// explainFixture is workspace Acme: alice its Owner, bob a Developer with a workspace deny of tickets:read and a deny
// on doc Pricing and play Release notes, sam a Member with a workspace allow of tickets:write and Pricing shared, and
// lena a Restricted Client with Project access to Storefront only.
func explainFixture(t *testing.T) (*Service, *fakeNames) {
	t.Helper()
	repo := newFakeRepo()
	roles := newFakeRoles()
	roles.set("ws", "alice", RoleInfo{RoleID: "role-owner", IsOwnerRole: true})
	roles.set("ws", "bob", RoleInfo{RoleID: "role-dev", Permissions: permissions.SetOf(permissions.DocsRead, permissions.DocsWrite,
		permissions.TicketsRead, permissions.PlaysRead, permissions.PlaysRun)})
	roles.set("ws", "sam", RoleInfo{RoleID: "role-plain"})
	roles.set("ws", "lena", RoleInfo{RoleID: "role-client", Restricted: true, Permissions: permissions.SetOf(permissions.TicketsRead,
		permissions.DocsRead, permissions.RunnersRead)})
	s := newService(repo, newFakeUsers())
	s.SetRoles(roles)
	s.SetScopes(fakeScopes{projects: map[string]string{"p-store": "ws", "p-back": "ws"}})
	s.SetDocWorkspaces(&fakeDocWorkspace{byDoc: map[string]string{"d-price": "ws", "d-plan": "ws"}, projectOf: map[string]string{"d-price": "p-store", "d-plan": "p-store"}})
	s.SetPlayWorkspaces(&fakePlayWorkspace{byPlay: map[string]string{"play-notes": "ws"}})
	names := &fakeNames{}
	s.SetNames(names)
	ctx := context.Background()
	require.NoError(t, repo.Set(ctx, resourceTypeProject, "p-store", "lena", permissions.SetOf(permissions.TicketsRead, permissions.DocsRead), nil))
	require.NoError(t, repo.Set(ctx, resourceTypeWorkspace, "ws", "sam", permissions.SetOf(permissions.TicketsWrite), nil))
	require.NoError(t, repo.Set(ctx, resourceTypeWorkspace, "ws", "bob", nil, permissions.SetOf(permissions.TicketsRead)))
	require.NoError(t, repo.Set(ctx, resourceTypeDoc, "d-price", "sam", permissions.SetOf(permissions.DocsRead), nil))
	require.NoError(t, repo.Set(ctx, resourceTypeDoc, "d-price", "bob", nil, permissions.SetOf(permissions.DocsWrite)))
	require.NoError(t, repo.Set(ctx, resourceTypePlay, "play-notes", "bob", nil, permissions.SetOf(permissions.PlaysRun)))
	return s, names
}

func src(kind, entityType, entityID, targetType, targetID, state string) Source {
	return Source{Kind: kind, EntityType: entityType, EntityID: entityID, TargetType: targetType, TargetID: targetID, State: state}
}

func TestDecide_Source(t *testing.T) {
	s, _ := explainFixture(t)
	tests := []struct {
		name                   string
		user, project          string
		action                 permissions.Action
		resourceType, resource string
		held                   bool
		source                 Source
		sentence               string
	}{
		{"owner", "alice", "p-store", permissions.DocsWrite, resourceTypeDoc, "d-price", true,
			src(SourceOwner, "workspace", "ws", TargetRole, "role-owner", StateAllow), "Owner of Acme"},
		{"role", "bob", "p-store", permissions.DocsRead, "", "", true,
			src(SourceRole, "workspace", "ws", TargetRole, "role-dev", StateAllow), "Allowed by role Developer"},
		{"role missing", "bob", "p-store", permissions.StacksWrite, "", "", false,
			src(SourceRole, "workspace", "ws", TargetRole, "role-dev", StateDeny), "Not allowed: role Developer doesn't grant it"},
		{"workspace allow", "sam", "p-store", permissions.TicketsWrite, "", "", true,
			src(SourceRule, "workspace", "ws", TargetPerson, "sam", StateAllow), "Allowed on workspace Acme for sam"},
		{"workspace deny", "bob", "p-store", permissions.TicketsRead, "", "", false,
			src(SourceRule, "workspace", "ws", TargetPerson, "bob", StateDeny), "Denied on workspace Acme for bob"},
		{"restricted with project access", "lena", "p-store", permissions.TicketsRead, "", "", true,
			src(SourceRestricted, "project", "p-store", TargetPerson, "lena", StateAllow), "Allowed by lena's Project access on project Storefront"},
		{"restricted, an action their access lacks", "lena", "p-store", permissions.TicketsWrite, "", "", false,
			src(SourceRestricted, "project", "p-store", TargetPerson, "lena", StateDeny), "Not allowed: lena's Project access on project Storefront doesn't grant it"},
		{"restricted without project access", "lena", "p-back", permissions.TicketsRead, "", "", false,
			src(SourceNotInProject, "project", "p-back", TargetPerson, "lena", StateDeny), "Not found: lena isn't in project Backoffice"},
		{"restricted, an instance action", "lena", "", permissions.RunnersRead, "", "", false,
			src(SourceRestricted, "workspace", "ws", TargetPerson, "lena", StateDeny), "Not allowed: lena is a Restricted member of Acme"},
		{"doc allow", "sam", "p-store", permissions.DocsRead, resourceTypeDoc, "d-price", true,
			src(SourceRule, "doc", "d-price", TargetPerson, "sam", StateAllow), "Allowed on doc Pricing for sam"},
		{"doc deny", "bob", "p-store", permissions.DocsWrite, resourceTypeDoc, "d-price", false,
			src(SourceRule, "doc", "d-price", TargetPerson, "bob", StateDeny), "Denied on doc Pricing for bob"},
		{"play deny", "bob", "", permissions.PlaysRun, resourceTypePlay, "play-notes", false,
			src(SourceRule, "play", "play-notes", TargetPerson, "bob", StateDeny), "Denied on play Release notes for bob"},
		{"not a member", "outsider", "", permissions.TicketsRead, "", "", false,
			src(SourceMember, "workspace", "ws", TargetPerson, "outsider", StateDeny), "Not found: outsider isn't a member of Acme"},
		{"a member opening the workspace", "sam", "", permissions.Member, "", "", true,
			src(SourceMember, "workspace", "ws", TargetPerson, "sam", StateAllow), "Member of Acme"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			held, got := s.check(ctx, tt.user, "ws", tt.project, tt.action, tt.resourceType, tt.resource)
			assert.Equal(t, tt.held, held)
			assert.Equal(t, tt.source, got)
			assert.Equal(t, tt.sentence, s.namer(ctx).named(got).Sentence)
		})
	}
	t.Run("Decide asks the same check by the entity alone", func(t *testing.T) {
		held, got, err := s.Decide(context.Background(), "bob", resourceTypeDoc, "d-price", permissions.DocsWrite)
		require.NoError(t, err)
		assert.False(t, held)
		assert.Equal(t, src(SourceRule, "doc", "d-price", TargetPerson, "bob", StateDeny), got)
	})
}

func TestSentence_NamesWhatItCan(t *testing.T) {
	s, _ := explainFixture(t)
	n := s.namer(context.Background())
	assert.Equal(t, "Allowed on doc Pricing for role Developer", n.named(src(SourceRule, "doc", "d-price", TargetRole, "role-dev", StateAllow)).Sentence)
	assert.Equal(t, "Denied on doc d-gone for bob", n.named(src(SourceRule, "doc", "d-gone", TargetPerson, "bob", StateDeny)).Sentence, "a name it can't read is its id")
	assert.Empty(t, n.named(Source{Kind: "unknown"}).Sentence)
	unnamed := newService(newFakeRepo(), newFakeUsers()).namer(context.Background())
	assert.Equal(t, "Allowed by role role-dev", unnamed.named(src(SourceRole, "workspace", "ws", TargetRole, "role-dev", StateAllow)).Sentence, "unwired names are ids")
}

// TestRequire_ForbiddenNamesItsSourceOnlyWhenRead: a refusal reads no name until its message is, so a list dropping
// refused rows stays as cheap as before.
func TestRequire_ForbiddenNamesItsSourceOnlyWhenRead(t *testing.T) {
	s, names := explainFixture(t)
	bob := identity.WithActor(context.Background(), identity.Actor{ID: "bob"})

	err := s.RequireProject(bob, "p-store", permissions.StacksWrite)
	require.ErrorIs(t, err, apperrs.ErrForbidden)
	assert.Zero(t, names.count(), "no name read before the message")
	assert.Equal(t, "forbidden: stacks:write required (Not allowed: role Developer doesn't grant it)", err.Error())
	assert.Positive(t, names.count())

	err = s.RefuseDoc(bob, "bob", "d-price", permissions.DocsWrite)
	require.ErrorIs(t, err, apperrs.ErrForbidden)
	assert.Equal(t, "forbidden: docs:write required (Denied on doc Pricing for bob)", err.Error())

	outsider := identity.WithActor(context.Background(), identity.Actor{ID: "outsider"})
	err = s.Require(outsider, "ws", permissions.TicketsRead)
	require.ErrorIs(t, err, apperrs.ErrNotFound)
	assert.NotContains(t, err.Error(), "member", "not found explains nothing")
}

func TestExplain(t *testing.T) {
	s, _ := explainFixture(t)
	ctx := context.Background()
	answer := func(t *testing.T, e *Explanation, action permissions.Action) Answer {
		t.Helper()
		for _, a := range e.Effective {
			if a.Action == action {
				return a
			}
		}
		t.Fatalf("no answer for %s", action)
		return Answer{}
	}

	t.Run("your own access on a doc you can open, with what its rule would leave", func(t *testing.T) {
		e, err := s.Explain(ctx, "sam", resourceTypeDoc, "d-price", "")
		require.NoError(t, err)
		assert.Equal(t, "sam", e.UserID)
		assert.Len(t, e.Effective, len(docActions))
		read := answer(t, e, permissions.DocsRead)
		assert.True(t, read.Allowed)
		assert.Equal(t, "Allowed on doc Pricing for sam", read.Source.Sentence)
		assert.Equal(t, "Pricing", read.Source.EntityName)
		assert.False(t, read.Fallback.Allowed)
		assert.Equal(t, "Not allowed: role Member doesn't grant it", read.Fallback.Source.Sentence)
	})
	t.Run("someone else's, with the manage permission", func(t *testing.T) {
		e, err := s.Explain(ctx, "alice", resourceTypeDoc, "d-price", "bob")
		require.NoError(t, err)
		write := answer(t, e, permissions.DocsWrite)
		assert.False(t, write.Allowed)
		assert.Equal(t, "Denied on doc Pricing for bob", write.Source.Sentence)
		assert.True(t, write.Fallback.Allowed)
		assert.Equal(t, "Allowed by role Developer", write.Fallback.Source.Sentence)
	})
	t.Run("someone else's without the manage permission is refused with its source", func(t *testing.T) {
		_, err := s.Explain(ctx, "bob", resourceTypeDoc, "d-plan", "sam")
		require.ErrorIs(t, err, apperrs.ErrForbidden)
		assert.Equal(t, "forbidden: permissions:write required (Not allowed: role Developer doesn't grant it)", err.Error())
	})
	t.Run("a doc you can't read is not found", func(t *testing.T) {
		_, err := s.Explain(ctx, "sam", resourceTypeDoc, "d-plan", "")
		require.ErrorIs(t, err, apperrs.ErrNotFound)
		_, err = s.Explain(ctx, "outsider", resourceTypeDoc, "d-price", "alice")
		require.ErrorIs(t, err, apperrs.ErrNotFound)
	})
	t.Run("a play", func(t *testing.T) {
		e, err := s.Explain(ctx, "bob", resourceTypePlay, "play-notes", "")
		require.NoError(t, err)
		run := answer(t, e, permissions.PlaysRun)
		assert.Equal(t, "Denied on play Release notes for bob", run.Source.Sentence)
		assert.True(t, run.Fallback.Allowed)
	})
	t.Run("a project: Project access is the rule, and without it the project is not found", func(t *testing.T) {
		e, err := s.Explain(ctx, "alice", resourceTypeProject, "p-store", "lena")
		require.NoError(t, err)
		assert.Len(t, e.Effective, len(projectActions))
		read := answer(t, e, permissions.TicketsRead)
		assert.True(t, read.Allowed)
		assert.Equal(t, "Not found: lena isn't in project Storefront", read.Fallback.Source.Sentence)

		_, err = s.Explain(ctx, "lena", resourceTypeProject, "p-back", "")
		require.ErrorIs(t, err, apperrs.ErrNotFound, "a project hidden from you")
		_, err = s.Explain(ctx, "bob", resourceTypeProject, "p-store", "lena")
		require.ErrorIs(t, err, apperrs.ErrForbidden, "another person's takes members:write")
	})
	t.Run("what it can't explain", func(t *testing.T) {
		_, err := s.Explain(ctx, "", resourceTypeDoc, "d-price", "")
		require.ErrorIs(t, err, apperrs.ErrUnauthorized)
		_, err = s.Explain(ctx, "alice", "ticket", "t-1", "")
		require.ErrorIs(t, err, apperrs.ErrInvalid)
		_, err = s.Explain(ctx, "alice", resourceTypeProject, "p-gone", "")
		require.ErrorIs(t, err, apperrs.ErrNotFound)
		_, _, err = s.Decide(ctx, "alice", "ticket", "t-1", permissions.TicketsRead)
		require.ErrorIs(t, err, apperrs.ErrInvalid)
	})
}

func TestHandler_Explain(t *testing.T) {
	s, _ := explainFixture(t)
	h := NewHandler(s).Routes()

	rec := serveAccess(t, h, identity.Actor{ID: "alice"}, http.MethodGet, "/api/permissions/explain?resource_type=doc&resource_id=d-price&user_id=bob", "")
	require.Equal(t, http.StatusOK, rec.Code)
	var e Explanation
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &e))
	assert.Equal(t, "bob", e.UserID)
	assert.NotEmpty(t, e.Effective[0].Source.Sentence)

	for path, want := range map[string]int{
		"/api/permissions/explain?resource_type=doc":                                    http.StatusBadRequest,
		"/api/permissions/explain?resource_type=doc&resource_id=d-plan&user_id=sam":     http.StatusForbidden,
		"/api/permissions/explain?resource_type=doc&resource_id=d-price&user_id=bob":    http.StatusOK,
		"/api/permissions/explain?resource_type=project&resource_id=p-back&user_id=sam": http.StatusForbidden,
	} {
		assert.Equal(t, want, serveAccess(t, h, identity.Actor{ID: "bob"}, http.MethodGet, path, "").Code, path)
	}
	assert.Contains(t, serveAccess(t, h, identity.Actor{ID: "bob"}, http.MethodGet, "/api/permissions/explain?resource_type=doc&resource_id=d-plan&user_id=sam", "").Body.String(),
		"Not allowed: role Developer doesn't grant it")
}

func TestGrantList_UserIDExplains(t *testing.T) {
	s, _ := explainFixture(t)

	got, err := callGrantTool(t, s, "alice", "permission_overwrite_list", `{"resource_type": "project", "resource_id": "p-store", "user_id": "lena"}`)
	require.NoError(t, err)
	assert.Equal(t, "lena", got.(*Explanation).UserID)

	_, err = callGrantTool(t, s, "bob", "permission_overwrite_list", `{"resource_type": "doc", "resource_id": "d-plan", "user_id": "sam"}`)
	require.ErrorIs(t, err, apperrs.ErrForbidden)
	assert.Contains(t, err.Error(), "Not allowed: role Developer doesn't grant it")
}
