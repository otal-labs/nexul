package main

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/auth"
	"github.com/otal-labs/nexul/internal/memories"
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

		listed, err := f.tpl.List(as(uPlain))
		require.NoError(t, err, "any member reads the instance templates")
		assert.Len(t, listed, 14)
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
			"a member may not write a workspace's play":         {uPlain, plays.TemplateKind, "fix-with-ai", templates.Instance, atDefault, forbidden},
			"a member may not write a project's body template":  {uPlain, workspace.TemplateKind, "bug", templates.Instance, atGeneral, forbidden},
			"a member may not read a workspace's plays":         {uPlain, plays.TemplateKind, "fix-with-ai", atDefault, templates.Instance, forbidden},
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
	f := newTemplateFixture(t)
	for user, want := range map[string]string{uOwner: ok, uTemplater: ok, uSteward: forbidden, uPlain: forbidden} {
		_, err := f.tpl.Update(as(user), memories.TemplateKind, "", templates.Instance, "## By "+user)
		assert.Equal(t, want, outcome(err), user)
		_, err = f.tpl.Reset(as(user), memories.TemplateKind, "", templates.Instance)
		assert.Equal(t, want, outcome(err), user)
	}
}
