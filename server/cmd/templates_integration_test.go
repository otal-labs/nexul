package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/auth"
	"github.com/otal-labs/nexul/internal/memories"
	"github.com/otal-labs/nexul/internal/platform/permissions"
	"github.com/otal-labs/nexul/internal/plays"
	"github.com/otal-labs/nexul/internal/roles"
	"github.com/otal-labs/nexul/internal/templates"
	"github.com/otal-labs/nexul/internal/tenancy"
	"github.com/otal-labs/nexul/internal/workspace"
)

const uTemplater = "u-templater"

var (
	atDefault = templates.Location{Scope: templates.ScopeWorkspace, WorkspaceID: wsDefault}
	atGeneral = templates.Location{Scope: templates.ScopeProject, ProjectID: pGeneral}
)

// templateFixture is the permission fixture plus uTemplater, who holds templates:write and nothing else, and a second
// workspace with a project, both made after the fixture's first instance edits so they show what creation copies.
type templateFixture struct {
	permFixture
	tpl *templates.Service
}

func newTemplateFixture(t *testing.T) templateFixture {
	t.Helper()
	f := newPermFixture(t)
	now := time.Now()
	ctx := context.Background()
	_, _, err := f.store.Users.UpsertUser(ctx, &auth.Identity{UserID: uTemplater, Provider: auth.ProviderGitHub, ProviderUserID: uTemplater, Login: uTemplater})
	require.NoError(t, err)
	require.NoError(t, f.store.Roles.Create(ctx, &roles.Role{ID: "role-templater", WorkspaceID: wsDefault, Name: "Templater", Permissions: grant("templates:write"), CreatedAt: now, UpdatedAt: now}))
	require.NoError(t, f.store.WorkspaceMembers.AddMember(ctx, &tenancy.Member{UserID: uTemplater, WorkspaceID: wsDefault, RoleID: "role-templater", CreatedAt: now}))
	listRolesIn(t, f.db, pGeneral)
	return templateFixture{permFixture: f, tpl: f.svc.templatesSvc}
}

func (f templateFixture) newWorkspace(t *testing.T, name, prefix string) (*tenancy.Workspace, *workspace.Project) {
	t.Helper()
	ws, err := f.svc.tenancySvc.Create(as(uOwner), uOwner, name)
	require.NoError(t, err)
	p, err := f.svc.workspaceSvc.Create(as(uOwner), ws.ID, name+" app", prefix, "")
	require.NoError(t, err)
	return ws, p
}

func (f templateFixture) play(t *testing.T, workspaceID, key string) *plays.Play {
	t.Helper()
	p, err := f.svc.playsSvc.BuiltinPlay(as(uOwner), workspaceID, key)
	require.NoError(t, err)
	return p
}

func (f templateFixture) bodyTemplate(t *testing.T, projectID, name string) string {
	t.Helper()
	tt, err := f.svc.workspaceSvc.TicketTypeNamed(as(uOwner), projectID, name)
	require.NoError(t, err)
	return tt.BodyTemplate
}

func (f templateFixture) typeID(t *testing.T, projectID, name string) string {
	t.Helper()
	tt, err := f.svc.workspaceSvc.TicketTypeNamed(as(uOwner), projectID, name)
	require.NoError(t, err)
	return tt.ID
}

func (f templateFixture) interview(t *testing.T, workspaceID string) *memories.InterviewTemplate {
	t.Helper()
	got, err := f.svc.memoriesSvc.InterviewTemplate(as(uOwner), workspaceID)
	require.NoError(t, err)
	return got
}

func (f templateFixture) chip(t *testing.T, workspaceID string) *tenancy.Workspace {
	t.Helper()
	got, err := f.svc.tenancySvc.Get(as(uOwner), workspaceID)
	require.NoError(t, err)
	return got
}

func (f templateFixture) setInstance(t *testing.T, kind, key, body string) {
	t.Helper()
	_, err := f.tpl.Update(as(uOwner), kind, key, templates.Instance, body)
	require.NoError(t, err)
}

func TestIntegration_TemplatesResolveThroughTheInstance(t *testing.T) {
	t.Parallel()
	f := newTemplateFixture(t)

	t.Run("an unedited workspace follows the instance live, an edited one keeps its own", func(t *testing.T) {
		assert.Equal(t, memories.DefaultInterviewTemplate, f.interview(t, wsDefault).Body, "nothing edited: the code default")
		assert.Equal(t, tenancy.DefaultMentionChipTemplate, f.chip(t, wsDefault).MentionChipTemplate)

		f.setInstance(t, memories.TemplateKind, "", "## Instance stack")
		f.setInstance(t, tenancy.TemplateKind, "", "{ticket.Status}")
		got := f.interview(t, wsDefault)
		assert.Equal(t, "## Instance stack", got.Body)
		assert.Equal(t, "## Instance stack", got.DefaultBody)
		assert.False(t, got.Edited)
		assert.Equal(t, "Instance stack", got.Questions[0].Text)
		instance, err := f.tpl.Get(as(uOwner), memories.TemplateKind, "", templates.Instance)
		require.NoError(t, err)
		assert.Equal(t, got.Questions, instance.Questions, "the instance view parses through the memories domain")
		ws := f.chip(t, wsDefault)
		assert.Equal(t, "{ticket.Status}", ws.MentionChipTemplate)
		assert.False(t, ws.MentionChipTemplateEdited)
		listed, err := f.svc.tenancySvc.ListForUser(as(uOwner), uOwner)
		require.NoError(t, err)
		assert.Equal(t, "{ticket.Status}", listed[0].MentionChipTemplate, "the workspace list the web reads resolves too")

		_, err = f.svc.memoriesSvc.SaveInterviewTemplate(as(uOwner), wsDefault, "## Our own")
		require.NoError(t, err)
		_, err = f.svc.tenancySvc.SetMentionChipTemplate(as(uOwner), uOwner, wsDefault, "{ticket.Ticket}")
		require.NoError(t, err)
		f.setInstance(t, memories.TemplateKind, "", "## Instance again")
		f.setInstance(t, tenancy.TemplateKind, "", "{ticket.Project}")
		assert.Equal(t, "## Our own", f.interview(t, wsDefault).Body)
		assert.True(t, f.interview(t, wsDefault).Edited)
		assert.Equal(t, "{ticket.Ticket}", f.chip(t, wsDefault).MentionChipTemplate)
	})

	t.Run("a workspace reset follows the instance again, and an instance reset falls to the code default", func(t *testing.T) {
		for _, kind := range []string{memories.TemplateKind, tenancy.TemplateKind} {
			got, err := f.tpl.Reset(as(uOwner), kind, "", atDefault)
			require.NoError(t, err, kind)
			assert.False(t, got.Edited, kind)
		}
		assert.Equal(t, "## Instance again", f.interview(t, wsDefault).Body)
		assert.Equal(t, "{ticket.Project}", f.chip(t, wsDefault).MentionChipTemplate)

		for _, kind := range []string{memories.TemplateKind, tenancy.TemplateKind} {
			_, err := f.tpl.Reset(as(uOwner), kind, "", templates.Instance)
			require.NoError(t, err, kind)
		}
		assert.Equal(t, memories.DefaultInterviewTemplate, f.interview(t, wsDefault).Body)
		assert.Equal(t, tenancy.DefaultMentionChipTemplate, f.chip(t, wsDefault).MentionChipTemplate)
	})

	t.Run("new workspaces and projects copy the instance version, existing copies stay", func(t *testing.T) {
		before := f.play(t, wsDefault, "fix-with-ai").Instructions
		beforeBug := f.bodyTemplate(t, pGeneral, "bug")
		f.setInstance(t, plays.TemplateKind, "fix-with-ai", "Fix it the instance way.")
		f.setInstance(t, workspace.TemplateKind, "bug", "## Instance bug")

		assert.Equal(t, before, f.play(t, wsDefault, "fix-with-ai").Instructions)
		assert.Equal(t, beforeBug, f.bodyTemplate(t, pGeneral, "bug"))
		ws, p := f.newWorkspace(t, "Beta", "BET")
		assert.Equal(t, "Fix it the instance way.", f.play(t, ws.ID, "fix-with-ai").Instructions)
		assert.Equal(t, "## Instance bug", f.bodyTemplate(t, p.ID, "bug"))
		assert.NotEqual(t, before, f.play(t, ws.ID, "to-tickets-via-ai").Instructions, "every built-in play keeps its own key")
		clarify := f.play(t, ws.ID, plays.ClarifyKey)
		assert.Equal(t, "Clarify via AI", clarify.Label, "a new workspace gets Clarify via AI too")
		assert.Equal(t, f.play(t, wsDefault, plays.ClarifyKey).Instructions, clarify.Instructions, "from the unedited instance template")
		assert.Equal(t, f.play(t, wsDefault, plays.DecisionsCheckKey).Instructions, f.play(t, ws.ID, plays.DecisionsCheckKey).Instructions,
			"the migrated decisions check and a new workspace's start from the same text")

		listed, err := f.tpl.List(as(uPlain))
		require.NoError(t, err, "any member reads the instance templates")
		assert.Len(t, listed, 21)
	})

	t.Run("a copy resets to the instance version, not the code default", func(t *testing.T) {
		got, err := f.tpl.Reset(as(uOwner), plays.TemplateKind, "fix-with-ai", atDefault)
		require.NoError(t, err)
		assert.False(t, got.Edited)
		assert.Equal(t, "Fix it the instance way.", f.play(t, wsDefault, "fix-with-ai").Instructions)
		_, err = f.tpl.Reset(as(uOwner), workspace.TemplateKind, "Bug", atGeneral)
		require.NoError(t, err)
		assert.Equal(t, "## Instance bug", f.bodyTemplate(t, pGeneral, "bug"))
	})
}

func TestIntegration_TemplatesClone(t *testing.T) {
	t.Parallel()
	f := newTemplateFixture(t)
	wsB, pB := f.newWorkspace(t, "Beta", "BET")
	atB := templates.Location{Scope: templates.ScopeWorkspace, WorkspaceID: wsB.ID}
	atProjectB := templates.Location{Scope: templates.ScopeProject, ProjectID: pB.ID}

	t.Run("instance interview into a workspace, which then holds its own", func(t *testing.T) {
		f.setInstance(t, memories.TemplateKind, "", "## Promoted later")
		got, err := f.tpl.Clone(as(uOwner), memories.TemplateKind, "", templates.Instance, atB)
		require.NoError(t, err)
		assert.True(t, got.Edited)
		f.setInstance(t, memories.TemplateKind, "", "## Moved on")
		assert.Equal(t, "## Promoted later", f.interview(t, wsB.ID).Body)
	})

	t.Run("a workspace's interview promoted to the instance", func(t *testing.T) {
		_, err := f.svc.memoriesSvc.SaveInterviewTemplate(as(uOwner), wsDefault, "## Default's own")
		require.NoError(t, err)
		got, err := f.tpl.Clone(as(uOwner), memories.TemplateKind, "", atDefault, templates.Instance)
		require.NoError(t, err)
		assert.Equal(t, "## Default's own", got.Body)
		assert.True(t, got.Edited)
		assert.Equal(t, uOwner, got.UpdatedBy)
	})

	t.Run("a built-in play's instructions from one workspace to the same play in another", func(t *testing.T) {
		_, err := f.svc.playsSvc.SetBuiltinInstructions(as(uOwner), wsDefault, "test-with-ai", "Test like the default workspace.")
		require.NoError(t, err)
		_, err = f.tpl.Clone(as(uOwner), plays.TemplateKind, "test-with-ai", atDefault, atB)
		require.NoError(t, err)
		assert.Equal(t, "Test like the default workspace.", f.play(t, wsB.ID, "test-with-ai").Instructions)
	})

	t.Run("a mention chip from a workspace to the instance", func(t *testing.T) {
		_, err := f.svc.tenancySvc.SetMentionChipTemplate(as(uOwner), uOwner, wsDefault, "{ticket.Ticket}")
		require.NoError(t, err)
		_, err = f.tpl.Clone(as(uOwner), tenancy.TemplateKind, "", atDefault, templates.Instance)
		require.NoError(t, err)
		assert.Equal(t, "{ticket.Ticket}", f.chip(t, wsB.ID).MentionChipTemplate, "an unedited workspace follows the promoted chip")
	})

	t.Run("a body template between projects matches the type name ignoring case, and promotes to the instance", func(t *testing.T) {
		_, err := f.svc.workspaceSvc.SetTicketTypeTemplate(as(uOwner), f.typeID(t, pGeneral, "bug"), "## General's bug")
		require.NoError(t, err)
		_, err = f.tpl.Clone(as(uOwner), workspace.TemplateKind, "Bug", atGeneral, atProjectB)
		require.NoError(t, err)
		assert.Equal(t, "## General's bug", f.bodyTemplate(t, pB.ID, "bug"))
		got, err := f.tpl.Clone(as(uOwner), workspace.TemplateKind, "BUG", atGeneral, templates.Instance)
		require.NoError(t, err)
		assert.Equal(t, "bug", got.Key)
		assert.Equal(t, "## General's bug", got.Body)
	})

	t.Run("no match on the other side is not found, and saying so", func(t *testing.T) {
		_, err := f.svc.workspaceSvc.CreateTicketType(as(uOwner), pGeneral, "Chore", "")
		require.NoError(t, err)
		_, err = f.tpl.Clone(as(uOwner), workspace.TemplateKind, "chore", atGeneral, atProjectB)
		assert.Equal(t, notFound, outcome(err))
		assert.Contains(t, err.Error(), `no ticket type named "chore"`)
		_, err = f.tpl.Clone(as(uOwner), workspace.TemplateKind, "chore", atGeneral, templates.Instance)
		assert.Equal(t, notFound, outcome(err))
		assert.Contains(t, err.Error(), "keys are task, bug, feature")

		require.NoError(t, f.svc.playsSvc.Delete(as(uOwner), wsB.ID, f.play(t, wsB.ID, "interview").ID))
		_, err = f.tpl.Clone(as(uOwner), plays.TemplateKind, "interview", atDefault, atB)
		assert.Equal(t, notFound, outcome(err))
		assert.Contains(t, err.Error(), `no built-in play "interview"`)
	})

	t.Run("each side is checked where it lives", func(t *testing.T) {
		for name, tc := range map[string]struct {
			user     string
			kind     string
			key      string
			from, to templates.Location
			want     string
		}{
			"a reader may not write a workspace's interview":    {uReader, memories.TemplateKind, "", templates.Instance, atDefault, forbidden},
			"a member may not write a workspace's chip":         {uPlain, tenancy.TemplateKind, "", templates.Instance, atDefault, forbidden},
			"a member can't write a play they can't see":        {uPlain, plays.TemplateKind, "fix-with-ai", templates.Instance, atDefault, notFound},
			"a member may not write a project's body template":  {uPlain, workspace.TemplateKind, "bug", templates.Instance, atGeneral, forbidden},
			"a play a member can't see is not found":            {uPlain, plays.TemplateKind, "fix-with-ai", atDefault, templates.Instance, notFound},
			"instance powers without templates:write refused":   {uSteward, memories.TemplateKind, "", atDefault, templates.Instance, forbidden},
			"an outsider cannot read another workspace's chip":  {uOutsider, tenancy.TemplateKind, "", atDefault, templates.Instance, notFound},
			"a project writer may copy a body template there":   {uWriter, workspace.TemplateKind, "task", templates.Instance, atGeneral, ok},
			"templates:write promotes what its holder can read": {uTemplater, workspace.TemplateKind, "task", atGeneral, templates.Instance, ok},
			"templates:write alone cannot reach a workspace":    {uTemplater, memories.TemplateKind, "", templates.Instance, atDefault, forbidden},
		} {
			_, err := f.tpl.Clone(as(tc.user), tc.kind, tc.key, tc.from, tc.to)
			assert.Equal(t, tc.want, outcome(err), name)
		}
	})
}

func TestIntegration_TemplatesBitGatesTheInstance(t *testing.T) {
	parallel(t)
	f := newTemplateFixture(t)
	for user, want := range map[string]string{uOwner: ok, uTemplater: ok, uSteward: forbidden, uPlain: forbidden} {
		_, err := f.tpl.Update(as(user), memories.TemplateKind, "", templates.Instance, "## By "+user)
		assert.Equal(t, want, outcome(err), user)
		_, err = f.tpl.Reset(as(user), memories.TemplateKind, "", templates.Instance)
		assert.Equal(t, want, outcome(err), user)
	}
}

// TestIntegration_TemplateEditorOpensEditsAndSavesItFromSettings: a non-Owner whose only bit is memories:template
// reads the workspace Interview template the way Settings does, with no project named, and saves an edit; someone
// holding neither it nor memories:read is still refused.
func TestIntegration_TemplateEditorOpensEditsAndSavesItFromSettings(t *testing.T) {
	t.Parallel()
	f := newTemplateFixture(t)
	now := time.Now()
	ctx := context.Background()
	const uEditor = "u-template-editor"
	_, _, err := f.store.Users.UpsertUser(ctx, &auth.Identity{UserID: uEditor, Provider: auth.ProviderGitHub, ProviderUserID: uEditor, Login: uEditor})
	require.NoError(t, err)
	require.NoError(t, f.store.Roles.Create(ctx, &roles.Role{ID: "role-template-editor", WorkspaceID: wsDefault, Name: "Template editor", Permissions: grant("memories:template"), CreatedAt: now, UpdatedAt: now}))
	require.NoError(t, f.store.WorkspaceMembers.AddMember(ctx, &tenancy.Member{UserID: uEditor, WorkspaceID: wsDefault, RoleID: "role-template-editor", CreatedAt: now}))
	routes := memories.NewHandler(f.svc.memoriesSvc).Routes()
	serve := func(actor, method, body string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		path := "/api/memories/interview-template?workspace_id=" + wsDefault
		routes.ServeHTTP(rec, httptest.NewRequest(method, path, strings.NewReader(body)).WithContext(as(actor)))
		return rec
	}

	opened := serve(uEditor, http.MethodGet, "")
	require.Equal(t, http.StatusOK, opened.Code, opened.Body.String())
	var template memories.InterviewTemplate
	require.NoError(t, json.Unmarshal(opened.Body.Bytes(), &template))
	assert.False(t, template.Edited)

	edited := template.Body + "\n## What do we never ship on a Friday?\n"
	saved := serve(uEditor, http.MethodPut, `{"workspace_id":"`+wsDefault+`","body":`+strconv.Quote(edited)+`}`)
	require.Equal(t, http.StatusOK, saved.Code, saved.Body.String())

	reopened := serve(uEditor, http.MethodGet, "")
	require.Equal(t, http.StatusOK, reopened.Code)
	require.NoError(t, json.Unmarshal(reopened.Body.Bytes(), &template))
	assert.True(t, template.Edited)
	assert.Equal(t, edited, template.Body)

	assert.Equal(t, http.StatusForbidden, serve(uTemplater, http.MethodGet, "").Code, "templates:write alone opens no workspace's template")
}

// workspaceRoles reads a workspace's roles as names in list order, and each one by name.
func workspaceRoles(t *testing.T, svc *coreServices, workspaceID string) ([]string, map[string]*roles.Role) {
	t.Helper()
	rs, err := svc.rolesSvc.List(context.Background(), workspaceID)
	require.NoError(t, err)
	var names []string
	byName := map[string]*roles.Role{}
	for _, r := range rs {
		names = append(names, r.Name)
		byName[r.Name] = r
	}
	return names, byName
}

func ticksOf(byName map[string]*roles.Role) map[string][]string {
	out := map[string][]string{}
	for name, r := range byName {
		out[name] = r.WelcomeSteps
	}
	return out
}

// TestIntegration_TemplateRoles: a new workspace copies every template role beside Owner, as the instance holds it at
// creation, and nothing that happens to the templates afterwards reaches it or any workspace that existed before.
func TestIntegration_TemplateRoles(t *testing.T) {
	t.Parallel()
	f := newTemplateFixture(t)
	existingNames, existing := workspaceRoles(t, f.svc, wsDefault)

	f.setInstance(t, roles.TemplateKind, "client", `{"name":"Customer","description":"Reads the tickets of their projects.","permissions":["tickets:read"],"welcome_steps":["projects","phone"]}`)
	added, err := f.tpl.Update(as(uTemplater), roles.TemplateKind, "", templates.Instance, `{"name":"Support","permissions":["tickets:read","chat:write"],"welcome_steps":["github"],"show_locked_actions":true}`)
	require.NoError(t, err)
	assert.True(t, added.Added)
	assert.Equal(t, "Support", added.Name, "named from its body, not its generated key")

	ws, _ := f.newWorkspace(t, "Beta", "BET")
	names, copies := workspaceRoles(t, f.svc, ws.ID)
	require.Equal(t, []string{"Owner", "Customer", "Junior Developer", "Senior Developer", "DevOps", "Administrator", "Support"}, names)
	assert.Equal(t, map[string][]string{
		"Owner": {"github", "computer"}, "Customer": {"projects", "phone"}, "Junior Developer": {"computer", "projects"},
		"Senior Developer": {"github", "computer"}, "DevOps": {"deploy_server"}, "Administrator": {"github"}, "Support": {"github"},
	}, ticksOf(copies))
	assert.Equal(t, "Reads the tickets of their projects.", copies["Customer"].Description)
	assert.Empty(t, copies["Support"].Description, "a body written before descriptions still seeds")
	for _, shipped := range roles.ShippedTemplates()[1:] {
		assert.NotEmpty(t, shipped.Body.Description, shipped.Key)
		assert.Equal(t, shipped.Body.Description, copies[shipped.Body.Name].Description, shipped.Key)
	}
	assert.Equal(t, grant("tickets:read", "chat:write"), copies["Support"].Permissions)
	assert.Equal(t, map[string]bool{"Customer": false, "Junior Developer": true, "Support": true}, map[string]bool{
		"Customer": copies["Customer"].ShowLockedActions, "Junior Developer": copies["Junior Developer"].ShowLockedActions, "Support": copies["Support"].ShowLockedActions,
	}, "a copy starts with the switch its template holds")
	assert.True(t, copies["Administrator"].Permissions.Has(permissions.RolesWrite))
	assert.False(t, copies["Administrator"].Permissions.Has(permissions.WorkspacesCreate), "making a workspace stays with Owner")
	assert.False(t, copies["Administrator"].Permissions.Has("workspaces:delete"), "removing a workspace stays with Owner")
	for _, a := range copies["Administrator"].Permissions {
		if permissions.AreaOf(a) == permissions.AreaInstance {
			domain, _, _ := strings.Cut(string(a), ":")
			assert.Contains(t, []string{"runners", "machines", "dns", "topology"}, domain, "Administrator holds none of the instance's own administration")
		}
	}

	t.Run("later template edits and deletes leave every workspace as it was", func(t *testing.T) {
		f.setInstance(t, roles.TemplateKind, "client", `{"name":"Client again","permissions":[],"welcome_steps":[]}`)
		f.setInstance(t, roles.TemplateKind, added.Key, `{"name":"Support 2","permissions":[],"welcome_steps":[]}`)
		_, err := f.tpl.Reset(as(uOwner), roles.TemplateKind, "devops", templates.Instance)
		require.NoError(t, err)
		require.NoError(t, f.tpl.Delete(as(uOwner), roles.TemplateKind, added.Key))

		after, afterCopies := workspaceRoles(t, f.svc, ws.ID)
		assert.Equal(t, names, after)
		assert.Equal(t, ticksOf(copies), ticksOf(afterCopies))
		assert.Equal(t, copies["Support"].Permissions, afterCopies["Support"].Permissions, "the deleted template's copy stays")
		stillNames, still := workspaceRoles(t, f.svc, wsDefault)
		assert.Equal(t, existingNames, stillNames, "a workspace from before gets no template role")
		assert.Equal(t, ticksOf(existing), ticksOf(still))
	})

	t.Run("a shipped template resets but never deletes, and only templates:write changes any", func(t *testing.T) {
		assert.Equal(t, invalid, outcome(f.tpl.Delete(as(uOwner), roles.TemplateKind, "client")))
		_, err := f.tpl.Get(as(uOwner), roles.TemplateKind, added.Key, templates.Instance)
		assert.Equal(t, notFound, outcome(err), "a deleted template is gone")
		mine, err := f.tpl.Update(as(uTemplater), roles.TemplateKind, "", templates.Instance, `{"name":"Mine","permissions":[],"welcome_steps":[]}`)
		require.NoError(t, err)
		_, err = f.tpl.Reset(as(uOwner), roles.TemplateKind, mine.Key, templates.Instance)
		assert.Equal(t, invalid, outcome(err), "an added template has no default to reset to")
		for _, user := range []string{uSteward, uPlain} {
			_, err := f.tpl.Update(as(user), roles.TemplateKind, "", templates.Instance, `{"name":"Theirs","permissions":[],"welcome_steps":[]}`)
			assert.Equal(t, forbidden, outcome(err), user)
			_, err = f.tpl.Update(as(user), roles.TemplateKind, mine.Key, templates.Instance, `{"name":"Theirs","permissions":[],"welcome_steps":[]}`)
			assert.Equal(t, forbidden, outcome(err), user)
			_, err = f.tpl.Reset(as(user), roles.TemplateKind, "client", templates.Instance)
			assert.Equal(t, forbidden, outcome(err), user)
			assert.Equal(t, forbidden, outcome(f.tpl.Delete(as(user), roles.TemplateKind, mine.Key)), user)
		}
	})
}

// TestIntegration_FreshInstallFirstWorkspaceGetsTheTemplateRoles: the owner wizard binding the pre-seeded default
// workspace seeds the template roles there too, beside the Owner role it creates.
func TestIntegration_FreshInstallFirstWorkspaceGetsTheTemplateRoles(t *testing.T) {
	t.Parallel()
	svc, store := wiredOver(t, mentionsTestDB(t))
	seedMentionsUser(t, store, uOwner)

	require.NoError(t, svc.tenancySvc.BindDefaultWorkspaceOwner(context.Background(), uOwner))
	names, byName := workspaceRoles(t, svc, wsDefault)
	require.Equal(t, []string{"Owner", "Client", "Junior Developer", "Senior Developer", "DevOps", "Administrator"}, names)
	assert.Equal(t, []string{"projects"}, byName["Client"].WelcomeSteps)
	shown := map[string]bool{}
	for name, r := range byName {
		shown[name] = r.ShowLockedActions
	}
	assert.Equal(t, map[string]bool{"Owner": false, "Client": false, "Junior Developer": true, "Senior Developer": true, "DevOps": true, "Administrator": true}, shown,
		"a new workspace shows locked actions to developers and above, not to clients")
}
