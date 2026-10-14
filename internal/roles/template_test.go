package roles

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

func TestParseTemplate_RefusesWhatNoRoleCouldHold(t *testing.T) {
	for name, body := range map[string]string{
		"not JSON":                      `Support`,
		"a misspelt field":              `{"name":"Support","permission":["tickets:read"]}`,
		"no name":                       `{"name":"  ","permissions":[],"welcome_steps":[]}`,
		"an action off the catalog":     `{"name":"Support","permissions":["tickets:fly"]}`,
		"an action only a rule can set": `{"name":"Support","permissions":["projects:open"]}`,
		"a step the wizard lacks":       `{"name":"Support","welcome_steps":["profile"]}`,
	} {
		assert.ErrorIs(t, CheckTemplate(body), apperrs.ErrInvalid, name)
	}

	got, err := ParseTemplate(`{"name":" Support ","permissions":["tickets:write","tickets:read","tickets:read"],"welcome_steps":["projects","github"]}`)
	require.NoError(t, err)
	assert.Equal(t, TemplateBody{Name: "Support", Permissions: permissions.SetOf("tickets:read", "tickets:write"), WelcomeSteps: []string{"github", "projects"}}, got)
}

type fakeTemplateRoles struct {
	bodies []string
	err    error
}

func (f fakeTemplateRoles) Bodies(context.Context, string) ([]string, error) { return f.bodies, f.err }

func TestSeedTemplateRoles(t *testing.T) {
	support := TemplateBody{Name: "Support", Permissions: permissions.SetOf("tickets:read"), WelcomeSteps: []string{"projects"}}.JSON()

	t.Run("a retried seed adds nothing twice, and a name the workspace has is left as it is", func(t *testing.T) {
		repo := newFakeRepo()
		require.NoError(t, repo.Create(context.Background(), &Role{ID: "own-client", WorkspaceID: "ws-1", Name: "client", CreatedAt: fixedNow}))
		s := newTestService(repo, newFakeMemberGate())
		s.SetTemplateRoles(fakeTemplateRoles{bodies: []string{TemplateBody{Name: "Client"}.JSON(), support}})

		require.NoError(t, s.SeedTemplateRoles(context.Background(), "ws-1"))
		require.NoError(t, s.SeedTemplateRoles(context.Background(), "ws-1"))
		rs, err := repo.List(context.Background(), "ws-1")
		require.NoError(t, err)
		var names []string
		for _, r := range rs {
			names = append(names, r.Name)
		}
		assert.ElementsMatch(t, []string{"client", "Support"}, names)
	})

	boom := errors.New("disk")
	for name, tc := range map[string]struct {
		templates fakeTemplateRoles
		repo      func(*fakeRepo)
	}{
		"the templates can't be read": {fakeTemplateRoles{err: boom}, func(*fakeRepo) {}},
		"the roles can't be listed":   {fakeTemplateRoles{bodies: []string{support}}, func(r *fakeRepo) { r.listErr = boom }},
		"a role can't be created":     {fakeTemplateRoles{bodies: []string{support}}, func(r *fakeRepo) { r.createErr = boom }},
	} {
		t.Run(name, func(t *testing.T) {
			repo := newFakeRepo()
			tc.repo(repo)
			s := newTestService(repo, newFakeMemberGate())
			s.SetTemplateRoles(tc.templates)
			assert.ErrorIs(t, s.SeedTemplateRoles(context.Background(), "ws-1"), boom)
		})
	}
	s := newTestService(newFakeRepo(), newFakeMemberGate())
	s.SetTemplateRoles(fakeTemplateRoles{bodies: []string{`{"name":""}`}})
	assert.ErrorIs(t, s.SeedTemplateRoles(context.Background(), "ws-1"), apperrs.ErrInvalid, "a stored body that no longer parses")
}
