package composite

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/codereview"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/mcptool"
	"github.com/otal-labs/nexul/internal/tickets"
)

func TestTicketTools_Names(t *testing.T) {
	var names []string
	for _, tool := range newFixture(t).ticketTools() {
		names = append(names, tool.Name)
		assert.NotEmpty(t, tool.Title, tool.Name)
	}
	assert.Equal(t, []string{"ticket_list", "ticket_get", "ticket_create", "ticket_update"}, names)
}

func TestTicketList_Errors(t *testing.T) {
	for name, args := range map[string]string{
		"an unknown filter is invalid":  `{"status":"st-todo"}`,
		"a limit of the wrong type":     `{"limit":"ten"}`,
		"blocked_only must be a bool":   `{"blocked_only":"yes"}`,
		"project_id must be a string":   `{"project_id":7}`,
		"arguments must be JSON object": `[]`,
	} {
		t.Run(name, func(t *testing.T) {
			_, err := call(t, t.Context(), newFixture(t).ticketTools(), "ticket_list", args)
			require.ErrorIs(t, err, apperrs.ErrInvalid)
		})
	}
	t.Run("a missing project is not found, not an empty list", func(t *testing.T) {
		_, err := call(t, t.Context(), newFixture(t).ticketTools(), "ticket_list", `{"project_id":"nope"}`)
		require.ErrorIs(t, err, apperrs.ErrNotFound)
		assert.Contains(t, err.Error(), "project_list")
	})
}

func TestTicketList(t *testing.T) {
	f := newFixture(t)
	f.w.addTicket(&tickets.Ticket{ID: "t-3", ProjectID: "p-2", Title: "Web login", Status: "st-web", DocID: "doc-1"})
	f.w.links = append(f.w.links, tickets.TicketLink{TicketID: "t-1", Kind: tickets.LinkBlockedBy, TargetID: "t-3"},
		tickets.TicketLink{TicketID: "t-1", Kind: tickets.LinkBlockedBy, TargetID: "t-2"})
	list := func(args string) mcptool.Page[ticketResult] {
		t.Helper()
		got, err := call(t, t.Context(), f.ticketTools(), "ticket_list", args)
		require.NoError(t, err)
		return got.(mcptool.Page[ticketResult])
	}
	keys := func(p mcptool.Page[ticketResult]) []string {
		var out []string
		for _, r := range p.Items {
			out = append(out, r.Key)
		}
		return out
	}

	assert.Equal(t, []string{"REF-1", "REF-2", "WEB-1"}, keys(list(`{}`)))
	assert.Equal(t, []string{"REF-1", "REF-2"}, keys(list(`{"project_id":"p-1"}`)))
	assert.Equal(t, []string{"WEB-1"}, keys(list(`{"doc_id":"doc-1"}`)))
	assert.Equal(t, []string{"REF-1", "WEB-1"}, keys(list(`{"query":"LOGIN"}`)))
	assert.Equal(t, []string{"REF-1"}, keys(list(`{"query":"login","project_id":"p-1"}`)))

	blocked := list(`{"blocked_only":true}`)
	require.Len(t, blocked.Items, 1)
	first := blocked.Items[0]
	assert.Equal(t, []string{"WEB-1"}, first.WaitingOn, "a done blocker no longer holds the ticket")
	assert.Empty(t, first.Body, "a list leaves bodies to ticket_get")
	assert.Equal(t, "Todo", first.Status)
	assert.Equal(t, "task", first.Type)
	assert.Equal(t, "Sprint 1", first.Category)
	assert.Equal(t, []string{"urgent"}, first.Labels)

	page := list(`{"limit":2}`)
	assert.Equal(t, 3, page.Total)
	assert.True(t, page.HasMore)
	assert.Equal(t, []string{"WEB-1"}, keys(list(`{"limit":2,"offset":2}`)))
}

func TestTicketGet_Errors(t *testing.T) {
	tests := []struct {
		name    string
		args    string
		wantErr error
		hint    string
	}{
		{"missing id is invalid", `{}`, apperrs.ErrInvalid, ""},
		{"a missing ticket is not found", `{"id":"nope"}`, apperrs.ErrNotFound, "ticket_list"},
		{"a missing key is not found", `{"id":"REF-99"}`, apperrs.ErrNotFound, "ticket_list"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := call(t, t.Context(), newFixture(t).ticketTools(), "ticket_get", tt.args)
			require.ErrorIs(t, err, tt.wantErr)
			assert.Contains(t, err.Error(), tt.hint)
		})
	}
}

func TestTicketGet(t *testing.T) {
	f := newFixture(t)
	f.w.addTicket(&tickets.Ticket{ID: "t-3", ProjectID: "p-1", Title: "Timeout crash", Status: "st-todo"})
	f.w.links = append(f.w.links,
		tickets.TicketLink{TicketID: "t-3", Kind: tickets.LinkFoundIn, TargetID: "t-1"},
		tickets.TicketLink{TicketID: "t-1", Kind: tickets.LinkBlockedBy, TargetID: "t-2"})
	f.w.prs["t-1"] = []tickets.PRLink{{PRRef: tickets.PRRef{Owner: "otal-labs", Repo: "nexul", Number: 42, Title: "Fix", SHA: "abc123"}, State: tickets.PRStateOpen}}
	f.w.branches["t-1"] = []tickets.BranchLink{{Owner: "otal-labs", Repo: "nexul", Branch: "fix/login"}}
	f.w.reviews["t-1"] = []*codereview.CodeReview{{ID: "r-1", Repo: "otal-labs/nexul", PRNumber: 42, Status: codereview.StatusApproved, Reviewer: "lena"}}
	f.w.target = tickets.TestTarget{URL: "https://fix-login.example.com", Kind: "preview", Branch: "fix/login"}

	got, err := call(t, t.Context(), f.ticketTools(), "ticket_get", `{"id":"ref-1"}`)
	require.NoError(t, err)
	d := got.(ticketDetail)
	assert.Equal(t, "t-1", d.ID)
	assert.Equal(t, "REF-1", d.Key)
	assert.Equal(t, "On slow networks.", d.Body)
	assert.Equal(t, []pullRequestResult{{Owner: "otal-labs", Repo: "nexul", Number: 42, Title: "Fix", State: tickets.PRStateOpen}}, d.PullRequests)
	assert.Equal(t, []tickets.BranchLink{{Owner: "otal-labs", Repo: "nexul", Branch: "fix/login"}}, d.Branches)
	assert.Equal(t, []linkedResult{{ID: "t-3", Key: "REF-3", Title: "Timeout crash", StatusID: "st-todo"}}, d.BugsFound)
	assert.Equal(t, []linkedResult{{ID: "t-2", Key: "REF-2", Title: "Sign-in page", StatusID: "st-done", Done: true}}, d.BlockedBy)
	assert.False(t, d.Blocked, "its only blocker is done")
	assert.Equal(t, []reviewResult{{ID: "r-1", Repo: "otal-labs/nexul", PRNumber: 42, Status: codereview.StatusApproved, Reviewer: "lena"}}, d.Reviews)
	assert.Equal(t, f.w.target, d.TestTarget)

	got, err = call(t, t.Context(), f.ticketTools(), "ticket_get", `{"id":"t-3"}`)
	require.NoError(t, err)
	require.NotNil(t, got.(ticketDetail).FoundIn)
	assert.Equal(t, "REF-1", got.(ticketDetail).FoundIn.Key)
	assert.Empty(t, got.(ticketDetail).PullRequests)
}

func TestTicketCreate_Errors(t *testing.T) {
	tests := []struct {
		name    string
		args    string
		wantErr error
		hint    string
	}{
		{"missing title is invalid", `{"project_id":"p-1"}`, apperrs.ErrInvalid, ""},
		{"a reporter field is not an argument", `{"project_id":"p-1","title":"Crash","reporter":"lena"}`, apperrs.ErrInvalid, ""},
		{"an origin that is not a ticket is invalid", `{"project_id":"p-1","title":"Crash","type_id":"tt-bug","origin_id":"REF-99"}`, apperrs.ErrInvalid, "ticket_list"},
		{"a blank title is invalid", `{"project_id":"p-1","title":"  "}`, apperrs.ErrInvalid, "project_get lists"},
		{"a bug without an origin is invalid", `{"project_id":"p-1","title":"Crash","type_id":"tt-bug"}`, apperrs.ErrInvalid, "project_get lists"},
		{"an unknown ticket type is invalid", `{"project_id":"p-1","title":"Crash","type_id":"tt-nope"}`, apperrs.ErrInvalid, "project_get lists"},
		{"an origin and unknown together are invalid", `{"project_id":"p-1","title":"Crash","origin_id":"REF-1","origin_unknown":true}`, apperrs.ErrInvalid, "project_get lists"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture(t)
			_, err := call(t, asUser(t.Context()), f.ticketTools(), "ticket_create", tt.args)
			require.ErrorIs(t, err, tt.wantErr)
			assert.Contains(t, err.Error(), tt.hint)
			assert.Len(t, f.w.tickets, 2)
		})
	}
}

func TestTicketCreate(t *testing.T) {
	t.Run("a bug found in a ticket named by its key", func(t *testing.T) {
		f := newFixture(t)
		got, err := call(t, asUser(t.Context()), f.ticketTools(), "ticket_create",
			`{"project_id":"p-1","title":"Crash","body":"Boom","type_id":"tt-bug","origin_id":"REF-1","tester":"lena"}`)
		require.NoError(t, err)
		r := got.(ticketResult)
		assert.Equal(t, "REF-3", r.Key)
		assert.Equal(t, "bug", r.Type)
		assert.Equal(t, "lena", r.Tester)
		assert.Equal(t, tickets.Reporter{Kind: tickets.ActorKindUserMCP, Login: "u-1"}, r.Reporter)
		assert.Equal(t, []tickets.TicketLink{{TicketID: r.ID, Kind: tickets.LinkFoundIn, TargetID: "t-1", CreatedAt: f.w.links[0].CreatedAt}}, f.w.links)
	})
	t.Run("a typed ticket without a body starts from the type's template", func(t *testing.T) {
		f := newFixture(t)
		got, err := call(t, asUser(t.Context()), f.ticketTools(), "ticket_create", `{"project_id":"p-1","title":"Chore","type_id":"tt-task"}`)
		require.NoError(t, err)
		assert.Equal(t, "## What needs doing\n", got.(ticketResult).Body)
	})
}

func TestTicketUpdate_Errors(t *testing.T) {
	tests := []struct {
		name    string
		args    string
		wantErr error
	}{
		{"missing id is invalid", `{"title":"x"}`, apperrs.ErrInvalid},
		{"an unknown field is invalid", `{"id":"t-1","assignee":"lena"}`, apperrs.ErrInvalid},
		{"two found-in fields are invalid", `{"id":"t-1","found_in_id":"REF-2","clear_found_in":true}`, apperrs.ErrInvalid},
		{"a pull request without a number is invalid", `{"id":"t-1","link_pr":{"owner":"o","repo":"r"}}`, apperrs.ErrInvalid},
		{"a missing ticket is not found", `{"id":"REF-99","title":"x"}`, apperrs.ErrNotFound},
		{"a blank title is invalid", `{"id":"t-1","title":" "}`, apperrs.ErrInvalid},
		{"an unknown status is invalid", `{"id":"t-1","status_id":"st-nope"}`, apperrs.ErrInvalid},
		{"a missing category is not found", `{"id":"t-1","category_id":"c-nope"}`, apperrs.ErrNotFound},
		{"a missing blocker is not found", `{"id":"t-1","add_blocker_ids":["REF-99"]}`, apperrs.ErrNotFound},
		{"a blocker cycle is a conflict", `{"id":"t-2","add_blocker_ids":["REF-1"]}`, apperrs.ErrConflict},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture(t)
			f.w.links = append(f.w.links, tickets.TicketLink{TicketID: "t-1", Kind: tickets.LinkBlockedBy, TargetID: "t-2"})
			before := *f.w.tickets["t-1"]
			_, err := call(t, asUser(t.Context()), f.ticketTools(), "ticket_update", tt.args)
			require.ErrorIs(t, err, tt.wantErr)
			assert.Equal(t, before.Title, f.w.tickets["t-1"].Title)
			assert.Equal(t, before.CategoryID, f.w.tickets["t-1"].CategoryID)
		})
	}
}

func TestTicketUpdate_StopsAtTheFirstFailureAndSaysWhatApplied(t *testing.T) {
	f := newFixture(t)
	_, err := call(t, asUser(t.Context()), f.ticketTools(), "ticket_update",
		`{"id":"t-1","title":"Renamed","status_id":"st-nope","type_id":"tt-bug"}`)
	require.ErrorIs(t, err, apperrs.ErrInvalid)
	assert.Contains(t, err.Error(), "status_id: ")
	assert.Contains(t, err.Error(), "already applied: title")
	assert.Contains(t, err.Error(), "project_get lists")
	assert.Equal(t, "Renamed", f.w.tickets["t-1"].Title)
	assert.Equal(t, "tt-task", f.w.tickets["t-1"].TypeID, "nothing after the failure runs")
}

func TestTicketUpdate_AnOmittedFieldKeepsItsValue(t *testing.T) {
	tests := []struct {
		name  string
		args  string
		check func(t *testing.T, got ticketResult)
	}{
		{"a title keeps the body", `{"id":"t-1","title":"Renamed"}`, func(t *testing.T, got ticketResult) {
			assert.Equal(t, "Renamed", got.Title)
			assert.Equal(t, "On slow networks.", got.Body)
		}},
		{"a body keeps the title", `{"id":"t-1","body":""}`, func(t *testing.T, got ticketResult) {
			assert.Equal(t, "Login times out", got.Title)
			assert.Empty(t, got.Body)
		}},
		{"a status keeps everything else", `{"id":"t-1","status_id":"st-doing"}`, func(t *testing.T, got ticketResult) {
			assert.Equal(t, "Doing", got.Status)
			assert.Equal(t, "On slow networks.", got.Body)
			assert.Equal(t, "c-1", got.CategoryID)
			assert.Equal(t, "onik97", got.Developer)
			assert.Equal(t, []string{"urgent"}, got.Labels)
		}},
		{"an empty category uncategorizes and keeps the type", `{"id":"t-1","category_id":""}`, func(t *testing.T, got ticketResult) {
			assert.Empty(t, got.CategoryID)
			assert.Equal(t, "tt-task", got.TypeID)
		}},
		{"a tester keeps the developer", `{"id":"t-1","tester":"lena"}`, func(t *testing.T, got ticketResult) {
			assert.Equal(t, "lena", got.Tester)
			assert.Equal(t, "onik97", got.Developer)
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := call(t, asUser(t.Context()), newFixture(t).ticketTools(), "ticket_update", tt.args)
			require.NoError(t, err)
			tt.check(t, got.(ticketUpdateResult).Ticket)
		})
	}
}

func TestTicketUpdate_EveryField(t *testing.T) {
	f := newFixture(t)
	f.w.addTicket(&tickets.Ticket{ID: "t-3", ProjectID: "p-1", Title: "Blocker", Status: "st-todo"})
	_, err := call(t, asUser(t.Context()), f.ticketTools(), "ticket_update", `{"id":"t-1","found_in_unknown":true,"add_blocker_ids":["t-2"]}`)
	require.NoError(t, err)

	got, err := call(t, asUser(t.Context()), f.ticketTools(), "ticket_update", `{
		"id":"REF-1","title":"New","body":"Body","status_id":"st-doing","position":0,"type_id":"tt-bug","category_id":"",
		"developer":"","tester":"lena","add_labels":["backend"],"remove_labels":["urgent"],"found_in_id":"REF-2",
		"add_blocker_ids":["REF-3"],"remove_blocker_ids":["REF-2"],
		"link_pr":{"owner":"otal-labs","repo":"nexul","number":7},"link_branch":{"owner":"otal-labs","repo":"nexul","branch":"fix/x"}}`)
	require.NoError(t, err)
	res := got.(ticketUpdateResult)
	assert.Equal(t, []string{"title, body", "status_id", "position", "type_id", "category_id", "developer", "tester",
		"add_labels[0]", "remove_labels[0]", "found_in", "add_blocker_ids[0]", "remove_blocker_ids[0]", "link_pr", "link_branch"}, res.Applied)
	tk := res.Ticket
	assert.Equal(t, "New", tk.Title)
	assert.Equal(t, "Doing", tk.Status)
	assert.Equal(t, "bug", tk.Type)
	assert.Empty(t, tk.CategoryID)
	assert.Empty(t, tk.Developer)
	assert.Equal(t, "lena", tk.Tester)
	assert.Equal(t, []string{"backend"}, tk.Labels)

	detail, err := call(t, t.Context(), f.ticketTools(), "ticket_get", `{"id":"t-1"}`)
	require.NoError(t, err)
	d := detail.(ticketDetail)
	require.NotNil(t, d.FoundIn)
	assert.Equal(t, "REF-2", d.FoundIn.Key)
	assert.False(t, d.OriginUnknown)
	require.Len(t, d.BlockedBy, 1)
	assert.Equal(t, "REF-3", d.BlockedBy[0].Key)
	assert.Equal(t, 7, d.PullRequests[0].Number)
	assert.Equal(t, "fix/x", d.Branches[0].Branch)

	got, err = call(t, asUser(t.Context()), f.ticketTools(), "ticket_update", `{"id":"t-1","clear_found_in":true,"project_id":"p-2"}`)
	require.NoError(t, err)
	assert.Equal(t, []string{"project_id", "clear_found_in"}, got.(ticketUpdateResult).Applied)
	assert.Equal(t, "p-2", got.(ticketUpdateResult).Ticket.ProjectID)
	assert.Equal(t, "WEB-1", got.(ticketUpdateResult).Ticket.Key, "the key follows the ticket's new project")
}

func TestTicketUpdate_NothingSentChangesNothing(t *testing.T) {
	got, err := call(t, t.Context(), newFixture(t).ticketTools(), "ticket_update", `{"id":"t-1"}`)
	require.NoError(t, err)
	assert.Empty(t, got.(ticketUpdateResult).Applied)
	assert.Equal(t, "Login times out", got.(ticketUpdateResult).Ticket.Title)
}

// TestTools_StorageFailuresPropagate checks every read a composite tool makes stops the call instead of shaping a partial result.
func TestTools_StorageFailuresPropagate(t *testing.T) {
	tests := []struct {
		fail, tool, args string
	}{
		{"List", "ticket_list", `{}`},
		{"Search", "ticket_list", `{"query":"login"}`},
		{"UnclearedBlockers", "ticket_list", `{}`},
		{"ListStatuses", "ticket_list", `{}`},
		{"ListTypes", "ticket_list", `{}`},
		{"ListCategories", "ticket_list", `{}`},
		{"ListPRLinks", "ticket_get", `{"id":"t-1"}`},
		{"ListLinkEnds", "ticket_get", `{"id":"t-1"}`},
		{"ListReviews", "ticket_get", `{"id":"t-1"}`},
		{"ListStatuses", "ticket_create", `{"project_id":"p-1","title":"x"}`},
		{"UpdateTicket", "ticket_update", `{"id":"t-1","title":"x"}`},
		{"ListStatuses", "project_get", `{"id":"p-1"}`},
		{"ListCategories", "project_get", `{"id":"p-1"}`},
		{"ListTypes", "project_get", `{"id":"p-1"}`},
		{"ListRepos", "project_get", `{"id":"p-1"}`},
		{"ListAllLabels", "project_get", `{"id":"p-1"}`},
		{"LabelColors", "project_get", `{"id":"p-1"}`},
		{"CountTickets", "project_get", `{"id":"p-1"}`},
		{"ListProjects", "project_update", `{"id":"p-1","position":1}`},
		{"ListRepos", "project_update", `{"id":"p-1","remove_repos":[{"owner":"o","name":"n"}]}`},
		{"ListStatuses", "project_update", `{"id":"p-1","statuses":{"update":[{"id":"st-todo","position":1}]}}`},
		{"ListCategories", "project_update", `{"id":"p-1","categories":{"update":[{"id":"c-1","position":1}]}}`},
		{"ListTypes", "project_update", `{"id":"p-1","ticket_types":{"update":[{"id":"tt-task","position":1}]}}`},
	}
	for _, tt := range tests {
		t.Run(tt.tool+" "+tt.fail, func(t *testing.T) {
			f := newFixture(t)
			f.w.fail = tt.fail
			tools := append(f.ticketTools(), f.projectTools()...)
			_, err := call(t, asUser(t.Context()), tools, tt.tool, tt.args)
			require.ErrorIs(t, err, errBoom)
		})
	}
}
