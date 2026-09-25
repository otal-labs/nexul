package composite

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/platform/colors"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/workspace"
)

func TestProjectTools_Names(t *testing.T) {
	var names []string
	for _, tool := range newFixture(t).projectTools() {
		names = append(names, tool.Name)
		assert.NotEmpty(t, tool.Title, tool.Name)
	}
	assert.Equal(t, []string{"project_get", "project_update"}, names)
}

func TestProjectGet_Errors(t *testing.T) {
	for name, tt := range map[string]struct {
		args    string
		wantErr error
		hint    string
	}{
		"missing id is invalid":          {`{}`, apperrs.ErrInvalid, ""},
		"a missing project is not found": {`{"id":"nope"}`, apperrs.ErrNotFound, "project_list"},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := call(t, t.Context(), newFixture(t).projectTools(), "project_get", tt.args)
			require.ErrorIs(t, err, tt.wantErr)
			assert.Contains(t, err.Error(), tt.hint)
		})
	}
}

func TestProjectGet(t *testing.T) {
	f := newFixture(t)
	f.w.repos["p-1"] = []workspace.RepoRef{{Owner: "otal-labs", Name: "nexul", FullName: "otal-labs/nexul", ConnectorID: "github", Role: workspace.RepoRoleApp}}
	f.w.labelColor["p-1/urgent"] = colors.Orange

	got, err := call(t, t.Context(), f.projectTools(), "project_get", `{"id":"p-1"}`)
	require.NoError(t, err)
	d := got.(projectDetail)
	assert.Equal(t, "REF", d.Prefix)
	assert.Equal(t, workspace.TestsLocationSame, d.TestsLocation)
	assert.Equal(t, []statusResult{
		{ID: "st-todo", Name: "Todo", Stage: workspace.StatusKindBacklog, Icon: workspace.StatusIconTodo},
		{ID: "st-doing", Name: "Doing", Stage: workspace.StatusKindProgress},
		{ID: "st-done", Name: "Shipped", Stage: workspace.StatusKindDone},
	}, d.Statuses)
	assert.Equal(t, []categoryResult{{ID: "c-1", Name: "Sprint 1", Color: colors.Lime}}, d.Categories)
	assert.Equal(t, "## What needs doing\n", d.TicketTypes[0].BodyTemplate)
	assert.Equal(t, []labelResult{{Name: "urgent", Color: colors.Orange}}, d.Labels)
	assert.Equal(t, "otal-labs/nexul", d.Repositories[0].FullName)
	assert.Equal(t, workspace.DeleteImpact{Tickets: 2, Repos: 1}, d.DeleteImpact)
}

func TestProjectUpdate_Errors(t *testing.T) {
	tests := []struct {
		name    string
		ctx     func(context.Context) context.Context
		owner   bool
		args    string
		wantErr error
	}{
		{"an unknown field is invalid", asUser, true, `{"id":"p-1","kind":"x"}`, apperrs.ErrInvalid},
		{"a status field of the wrong type is invalid", asUser, true, `{"id":"p-1","statuses":{"create":[{"name":1}]}}`, apperrs.ErrInvalid},
		{"no caller is unauthorized", anonymous, true, `{"id":"p-1","name":"New"}`, apperrs.ErrUnauthorized},
		{"a caller who is not an owner is forbidden", asUser, false, `{"id":"p-1","name":"New"}`, apperrs.ErrForbidden},
		{"a missing project is not found", asUser, true, `{"id":"nope","name":"New"}`, apperrs.ErrNotFound},
		{"a blank name is invalid", asUser, true, `{"id":"p-1","name":" "}`, apperrs.ErrInvalid},
		{"a prefix on a project that has one is a conflict", asUser, true, `{"id":"p-1","prefix":"NEW"}`, apperrs.ErrConflict},
		{"another project's column is invalid", asUser, true, `{"id":"p-1","statuses":{"update":[{"id":"st-web","name":"x"}]}}`, apperrs.ErrInvalid},
		{"a missing column is not found", asUser, true, `{"id":"p-1","statuses":{"delete":["st-nope"]}}`, apperrs.ErrNotFound},
		{"a column holding tickets is a conflict", asUser, true, `{"id":"p-1","statuses":{"delete":["st-todo"]}}`, apperrs.ErrConflict},
		{"a bogus stage is invalid", asUser, true, `{"id":"p-1","statuses":{"create":[{"name":"QA","stage":"qa"}]}}`, apperrs.ErrInvalid},
		{"another project's category is invalid", asUser, true, `{"id":"p-2","categories":{"delete":["c-1"]}}`, apperrs.ErrInvalid},
		{"a bogus color is invalid", asUser, true, `{"id":"p-1","categories":{"update":[{"id":"c-1","color":"teal"}]}}`, apperrs.ErrInvalid},
		{"a ticket type in use is a conflict", asUser, true, `{"id":"p-1","ticket_types":{"delete":["tt-task"]}}`, apperrs.ErrConflict},
		{"a repository of another project is invalid", asUser, true, `{"id":"p-2","remove_repos":[{"owner":"otal-labs","name":"nexul"}]}`, apperrs.ErrInvalid},
		{"an unknown tests location is invalid", asUser, true, `{"id":"p-1","tests_location":"elsewhere"}`, apperrs.ErrInvalid},
		{"a bogus label color is invalid", asUser, true, `{"id":"p-1","label_colors":[{"label":"urgent","color":"teal"}]}`, apperrs.ErrInvalid},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture(t)
			f.w.owner = tt.owner
			f.w.repos["p-1"] = []workspace.RepoRef{{Owner: "otal-labs", Name: "nexul"}}
			_, err := call(t, tt.ctx(t.Context()), f.projectTools(), "project_update", tt.args)
			require.ErrorIs(t, err, tt.wantErr)
			p, _ := f.w.projects.get("p-1")
			assert.Equal(t, "Backend", p.Name)
			assert.Len(t, f.w.statuses.items, 4)
			assert.Len(t, f.w.repos["p-1"], 1)
		})
	}
}

func TestProjectUpdate_StopsAtTheFirstFailureAndSaysWhatApplied(t *testing.T) {
	f := newFixture(t)
	_, err := call(t, asUser(t.Context()), f.projectTools(), "project_update",
		`{"id":"p-1","name":"Core","statuses":{"create":[{"name":"QA","stage":"testing"},{"name":"Bad","stage":"qa"},{"name":"Never"}]}}`)
	require.ErrorIs(t, err, apperrs.ErrInvalid)
	assert.Contains(t, err.Error(), "statuses.create[1]: ")
	assert.Contains(t, err.Error(), "already applied: name, statuses.create[0]")
	p, _ := f.w.projects.get("p-1")
	assert.Equal(t, "Core", p.Name)
	assert.Len(t, f.w.statuses.items, 5, "QA was created, nothing after the failure was")
}

func TestProjectUpdate_AnOmittedFieldKeepsItsValue(t *testing.T) {
	f := newFixture(t)
	got, err := call(t, asUser(t.Context()), f.projectTools(), "project_update", `{"id":"p-1",
		"name":"Core",
		"statuses":{"update":[{"id":"st-todo","name":"Backlog"}]},
		"categories":{"update":[{"id":"c-1","name":"Sprint 2"}]},
		"ticket_types":{"update":[{"id":"tt-task","name":"chore"}]}}`)
	require.NoError(t, err)
	d := got.(projectUpdateResult).Project
	assert.Equal(t, "Core", d.Name)
	assert.Equal(t, workspace.ProjectIconServer, d.Icon, "the icon survives a rename")
	assert.Equal(t, "REF", d.Prefix)
	assert.Equal(t, workspace.TestsLocationSame, d.TestsLocation)
	assert.Equal(t, statusResult{ID: "st-todo", Name: "Backlog", Stage: workspace.StatusKindBacklog, Icon: workspace.StatusIconTodo}, d.Statuses[0])
	assert.Equal(t, categoryResult{ID: "c-1", Name: "Sprint 2", Color: colors.Lime}, d.Categories[0])
	assert.Equal(t, ticketTypeResult{ID: "tt-task", Name: "chore", Color: colors.Cyan, BodyTemplate: "## What needs doing\n"}, d.TicketTypes[0])
}

func TestProjectUpdate_EveryField(t *testing.T) {
	f := newFixture(t)
	must(t, f.w.projects.put(&workspace.Project{ID: "p-3", Name: "Ops", WorkspaceID: "ws-1"}))
	f.w.repos["p-3"] = []workspace.RepoRef{{Owner: "otal-labs", Name: "old"}}
	got, err := call(t, asUser(t.Context()), f.projectTools(), "project_update", `{"id":"p-3",
		"icon":"Cloud","prefix":"ops","position":0,"tests_location":"",
		"add_repos":[{"owner":"otal-labs","name":"e2e","role":"tests"}],"remove_repos":[{"owner":"OTAL-LABS","name":"old"}],
		"statuses":{"create":[{"name":"Todo"},{"name":"Done","stage":"done","icon":"CircleCheckBig"}]},
		"categories":{"create":[{"name":"Now","color":"cyan"}]},
		"ticket_types":{"create":[{"name":"spike","color":"orange","body_template":"## Question\n"}]},
		"label_colors":[{"label":"urgent","color":"fuchsia"}]}`)
	require.NoError(t, err)
	res := got.(projectUpdateResult)
	assert.Equal(t, []string{"icon", "prefix", "position", "tests_location", "add_repos[0]", "remove_repos[0]",
		"statuses.create[0]", "statuses.create[1]", "categories.create[0]", "ticket_types.create[0]", "label_colors[0]"}, res.Applied)
	d := res.Project
	assert.Equal(t, "Ops", d.Name)
	assert.Equal(t, workspace.ProjectIconCloud, d.Icon)
	assert.Equal(t, "OPS", d.Prefix)
	assert.Equal(t, workspace.TestsLocationSeparate, d.TestsLocation, "attaching a tests repository answers the tests location")
	assert.Equal(t, []workspace.RepoRef{{Owner: "otal-labs", Name: "e2e", FullName: "otal-labs/e2e", ConnectorID: "github", Role: workspace.RepoRoleTests}}, d.Repositories)
	require.Len(t, d.Statuses, 2)
	assert.Equal(t, workspace.StatusKindBacklog, d.Statuses[0].Stage, "the stage defaults to backlog")
	assert.Equal(t, workspace.StatusIconDone, d.Statuses[1].Icon)
	assert.Equal(t, colors.Cyan, d.Categories[0].Color)
	assert.Equal(t, "## Question\n", d.TicketTypes[0].BodyTemplate)
	assert.Equal(t, colors.Fuchsia, f.w.labelColor["p-3/urgent"])
	assert.Equal(t, "p-3", f.w.projects.items[0].ID, "position 0 moves the project first")

	todo, done := d.Statuses[0].ID, d.Statuses[1].ID
	got, err = call(t, asUser(t.Context()), f.projectTools(), "project_update", `{"id":"p-3",
		"statuses":{"update":[{"id":"`+done+`","position":0,"stage":"testing","icon":""}],"delete":["`+todo+`"]},
		"categories":{"update":[{"id":"`+d.Categories[0].ID+`","color":""}],"delete":["`+d.Categories[0].ID+`"]},
		"ticket_types":{"update":[{"id":"`+d.TicketTypes[0].ID+`","body_template":"","position":0}],"delete":["`+d.TicketTypes[0].ID+`"]}}`)
	require.NoError(t, err)
	d = got.(projectUpdateResult).Project
	require.Len(t, d.Statuses, 1)
	assert.Equal(t, done, d.Statuses[0].ID)
	assert.Equal(t, workspace.StatusKindTesting, d.Statuses[0].Stage)
	assert.Empty(t, d.Statuses[0].Icon, "an empty icon clears it")
	assert.Empty(t, d.Categories)
	assert.Empty(t, d.TicketTypes)
}

func TestProjectUpdate_ReordersWithinTheProject(t *testing.T) {
	f := newFixture(t)
	must(t, f.w.categories.put(&workspace.Category{ID: "c-2", ProjectID: "p-1", Name: "Sprint 2"}))
	must(t, f.w.types.put(&workspace.TicketType{ID: "tt-feature", ProjectID: "p-1", Name: "feature"}))
	got, err := call(t, asUser(t.Context()), f.projectTools(), "project_update", `{"id":"p-1",
		"statuses":{"update":[{"id":"st-done","position":0}]},
		"categories":{"update":[{"id":"c-2","position":0}]},
		"ticket_types":{"update":[{"id":"tt-feature","position":99}]}}`)
	require.NoError(t, err)
	d := got.(projectUpdateResult).Project
	assert.Equal(t, []string{"st-done", "st-todo", "st-doing"}, idsOf(d.Statuses, func(s statusResult) string { return s.ID }))
	assert.Equal(t, []string{"c-2", "c-1"}, idsOf(d.Categories, func(c categoryResult) string { return c.ID }))
	assert.Equal(t, []string{"tt-task", "tt-bug", "tt-feature"}, idsOf(d.TicketTypes, func(tt ticketTypeResult) string { return tt.ID }), "a position past the end is last")
}

// TestProjectUpdate_ListedValuesAreAccepted keeps the literal value lists in the parameter descriptions true.
func TestProjectUpdate_ListedValuesAreAccepted(t *testing.T) {
	schema := newFixture(t).projectTools()[1].InputSchema
	listed := func(desc string) []string {
		_, rest, _ := strings.Cut(desc, "icon: ")
		rest, _, _ = strings.Cut(rest, ".")
		return strings.Fields(strings.NewReplacer(",", "", " or ", " ").Replace(rest))
	}
	statuses := schema.Properties["statuses"].Properties["create"].Items
	for _, c := range colors.All() {
		assert.Contains(t, schema.Properties["categories"].Properties["create"].Items.Properties["color"].Description, string(c))
		assert.Contains(t, schema.Properties["label_colors"].Items.Properties["color"].Description, string(c))
	}
	for _, k := range workspace.StatusKinds {
		assert.Contains(t, statuses.Properties["stage"].Description, string(k))
	}
	for _, icon := range listed(schema.Properties["icon"].Description) {
		f := newFixture(t)
		_, err := call(t, asUser(t.Context()), f.projectTools(), "project_update", `{"id":"p-1","icon":"`+icon+`"}`)
		assert.NoError(t, err, icon)
	}
	for _, icon := range listed(statuses.Properties["icon"].Description) {
		f := newFixture(t)
		_, err := call(t, asUser(t.Context()), f.projectTools(), "project_update", `{"id":"p-1","statuses":{"create":[{"name":"x","icon":"`+icon+`"}]}}`)
		assert.NoError(t, err, icon)
	}
}
