package tickets

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/platform/colors"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/mcptool"
)

func toolByName(t *testing.T, tools []mcptool.Tool, name string) mcptool.Tool {
	t.Helper()
	for _, tool := range tools {
		if tool.Name == name {
			return tool
		}
	}
	t.Fatalf("tool %s not found", name)
	return mcptool.Tool{}
}

func TestMCPTools_Shape(t *testing.T) {
	tools := MCPTools(newTestService(newFakeRepo()))
	require.Len(t, tools, 26)
	var names []string
	for _, tool := range tools {
		names = append(names, tool.Name)
		assert.NotEmpty(t, tool.Description)
		assert.NotNil(t, tool.InputSchema)
		assert.NotNil(t, tool.Call)
	}
	assert.ElementsMatch(t, []string{
		"ticket_create", "ticket_get", "ticket_update", "ticket_update_status", "ticket_set_type",
		"ticket_set_developer", "ticket_set_tester",
		"ticket_add_label", "ticket_remove_label", "ticket_list_labels", "ticket_list_all_labels",
		"ticket_set_label_color", "ticket_label_colors",
		"ticket_search", "ticket_link_pr", "ticket_link_branch", "ticket_get_links",
		"ticket_get_ticket_links", "ticket_set_found_in", "ticket_remove_found_in",
		"ticket_add_blocker", "ticket_remove_blocker", "ticket_list_blocked",
		"ticket_get_test_target", "ticket_test_pass", "ticket_test_fail",
	}, names)
}

func TestMCPTools_LabelColors(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo)
	tools := MCPTools(s)
	setCall := toolByName(t, tools, "ticket_set_label_color").Call
	getCall := toolByName(t, tools, "ticket_label_colors").Call

	t.Run("sets a label's color", func(t *testing.T) {
		got, err := setCall(context.Background(), map[string]any{"project_id": "p-1", "label": "bug", "color": "cyan"})
		require.NoError(t, err)
		lc, ok := got.(*LabelColor)
		require.True(t, ok)
		assert.Equal(t, "bug", lc.Label)
	})
	t.Run("missing project id is invalid", func(t *testing.T) {
		_, err := setCall(context.Background(), map[string]any{"label": "bug", "color": "cyan"})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("missing label is invalid", func(t *testing.T) {
		_, err := setCall(context.Background(), map[string]any{"project_id": "p-1", "color": "cyan"})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("missing color is invalid", func(t *testing.T) {
		_, err := setCall(context.Background(), map[string]any{"project_id": "p-1", "label": "bug"})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("invalid color is invalid", func(t *testing.T) {
		_, err := setCall(context.Background(), map[string]any{"project_id": "p-1", "label": "bug", "color": "magenta"})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("batch-fetches colors for the requested labels", func(t *testing.T) {
		_, err := setCall(context.Background(), map[string]any{"project_id": "p-1", "label": "bug", "color": "cyan"})
		require.NoError(t, err)
		got, err := getCall(context.Background(), map[string]any{"project_id": "p-1", "labels": []any{"bug", "nope"}})
		require.NoError(t, err)
		colorsByLabel, ok := got.(map[string]colors.Color)
		require.True(t, ok)
		assert.Equal(t, colors.Cyan, colorsByLabel["bug"])
		_, exists := colorsByLabel["nope"]
		assert.False(t, exists)
	})
	t.Run("missing project id is invalid on fetch", func(t *testing.T) {
		_, err := getCall(context.Background(), map[string]any{"labels": []any{"bug"}})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("missing labels is invalid", func(t *testing.T) {
		_, err := getCall(context.Background(), map[string]any{"project_id": "p-1"})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("empty labels list is invalid", func(t *testing.T) {
		_, err := getCall(context.Background(), map[string]any{"project_id": "p-1", "labels": []any{}})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("non-string label entry is invalid", func(t *testing.T) {
		_, err := getCall(context.Background(), map[string]any{"labels": []any{1}})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
}

func TestMCPTools_TicketCreate(t *testing.T) {
	tools := MCPTools(newTestService(newFakeRepo()))
	call := toolByName(t, tools, "ticket_create").Call

	t.Run("happy path links to doc", func(t *testing.T) {
		got, err := call(context.Background(), map[string]any{"project_id": "p-1", "title": "Fix", "body": "b", "doc_id": "doc-1", "developer": "onik97", "tester": "lena"})
		require.NoError(t, err)
		tk, ok := got.(*Ticket)
		require.True(t, ok)
		assert.Equal(t, "Fix", tk.Title)
		assert.Equal(t, "doc-1", tk.DocID)
		assert.Equal(t, "p-1", tk.ProjectID)
		assert.Equal(t, StatusOpen, tk.Status)
	})
	t.Run("missing project id is invalid", func(t *testing.T) {
		_, err := call(context.Background(), map[string]any{"title": "Fix"})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("missing title is invalid", func(t *testing.T) {
		_, err := call(context.Background(), map[string]any{})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
}

func TestMCPTools_TicketGet(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo)
	created, err := s.Create(context.Background(), "p-1", "Fix", "", "", "")
	require.NoError(t, err)
	call := toolByName(t, MCPTools(s), "ticket_get").Call

	t.Run("returns stored ticket", func(t *testing.T) {
		got, err := call(context.Background(), map[string]any{"id": created.ID})
		require.NoError(t, err)
		assert.Equal(t, created.ID, got.(*Ticket).ID)
	})
	t.Run("missing id is invalid", func(t *testing.T) {
		_, err := call(context.Background(), map[string]any{})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("unknown ticket is not found", func(t *testing.T) {
		_, err := call(context.Background(), map[string]any{"id": "nope"})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrNotFound))
	})
}

func TestMCPTools_TicketUpdate(t *testing.T) {
	t.Run("edits title and body", func(t *testing.T) {
		repo := newFakeRepo()
		s := newTestService(repo)
		created, err := s.Create(context.Background(), "p-1", "Fix", "old", "", "")
		require.NoError(t, err)
		call := toolByName(t, MCPTools(s), "ticket_update").Call
		got, err := call(context.Background(), map[string]any{"id": created.ID, "title": "Fixed", "body": "new"})
		require.NoError(t, err)
		assert.Equal(t, "Fixed", got.(*Ticket).Title)
		assert.Equal(t, "new", got.(*Ticket).Body)
	})
	t.Run("body may be omitted to clear it", func(t *testing.T) {
		repo := newFakeRepo()
		s := newTestService(repo)
		created, err := s.Create(context.Background(), "p-1", "Fix", "old", "", "")
		require.NoError(t, err)
		call := toolByName(t, MCPTools(s), "ticket_update").Call
		got, err := call(context.Background(), map[string]any{"id": created.ID, "title": "Fix"})
		require.NoError(t, err)
		assert.Equal(t, "", got.(*Ticket).Body)
	})
	t.Run("missing title is invalid", func(t *testing.T) {
		call := toolByName(t, MCPTools(newTestService(newFakeRepo())), "ticket_update").Call
		_, err := call(context.Background(), map[string]any{"id": "t-1"})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
}

func TestMCPTools_TicketUpdateStatus(t *testing.T) {
	t.Run("transitions status", func(t *testing.T) {
		repo := newFakeRepo()
		s := newTestService(repo)
		created, err := s.Create(context.Background(), "p-1", "Fix", "", "", "")
		require.NoError(t, err)
		call := toolByName(t, MCPTools(s), "ticket_update_status").Call
		got, err := call(context.Background(), map[string]any{"id": created.ID, "status": "in_progress"})
		require.NoError(t, err)
		assert.Equal(t, StatusInProgress, got.(*Ticket).Status)
	})
	t.Run("missing id is invalid", func(t *testing.T) {
		call := toolByName(t, MCPTools(newTestService(newFakeRepo())), "ticket_update_status").Call
		_, err := call(context.Background(), map[string]any{"status": "done"})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("missing status is invalid", func(t *testing.T) {
		call := toolByName(t, MCPTools(newTestService(newFakeRepo())), "ticket_update_status").Call
		_, err := call(context.Background(), map[string]any{"id": "t-1"})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("unconfigured status is rejected", func(t *testing.T) {
		repo := newFakeRepo()
		s := newTestService(repo)
		created, err := s.Create(context.Background(), "p-1", "Fix", "", "", "")
		require.NoError(t, err)
		call := toolByName(t, MCPTools(s), "ticket_update_status").Call
		_, err = call(context.Background(), map[string]any{"id": created.ID, "status": "nope"})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
}

func TestMCPTools_TicketSearch(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo)
	_, err := s.Create(context.Background(), "p-1", "Storage spine", "sqlite migrations", "", "")
	require.NoError(t, err)
	call := toolByName(t, MCPTools(s), "ticket_search").Call

	t.Run("finds matching tickets", func(t *testing.T) {
		got, err := call(context.Background(), map[string]any{"query": "sqlite"})
		require.NoError(t, err)
		results, ok := got.([]SearchResult)
		require.True(t, ok)
		require.Len(t, results, 1)
	})
	t.Run("missing query is invalid", func(t *testing.T) {
		_, err := call(context.Background(), map[string]any{})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
}

func TestMCPTools_Links(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo)
	created, err := s.Create(context.Background(), "p-1", "ticket", "", "", "")
	require.NoError(t, err)
	tools := MCPTools(s)

	t.Run("ticket_link_pr links and returns the links", func(t *testing.T) {
		call := toolByName(t, tools, "ticket_link_pr").Call
		got, err := call(context.Background(), map[string]any{"id": created.ID, "owner": "acme", "repo": "app", "number": 7, "title": "F"})
		require.NoError(t, err)
		links, ok := got.(map[string]any)
		require.True(t, ok)
		prs := links["prs"].([]PRLink)
		require.Len(t, prs, 1)
		assert.Equal(t, 7, prs[0].Number)
	})
	t.Run("ticket_link_pr rejects a bad number", func(t *testing.T) {
		call := toolByName(t, tools, "ticket_link_pr").Call
		_, err := call(context.Background(), map[string]any{"id": created.ID, "owner": "acme", "repo": "app", "number": 0})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("ticket_link_branch links a branch", func(t *testing.T) {
		call := toolByName(t, tools, "ticket_link_branch").Call
		got, err := call(context.Background(), map[string]any{"id": created.ID, "owner": "acme", "repo": "app", "branch": "ticket/1"})
		require.NoError(t, err)
		links, ok := got.(map[string]any)
		require.True(t, ok)
		branches := links["branches"].([]BranchLink)
		require.Len(t, branches, 1)
		assert.Equal(t, "ticket/1", branches[0].Branch)
	})
	t.Run("ticket_get_links lists the links", func(t *testing.T) {
		call := toolByName(t, tools, "ticket_get_links").Call
		got, err := call(context.Background(), map[string]any{"id": created.ID})
		require.NoError(t, err)
		links, ok := got.(map[string]any)
		require.True(t, ok)
		assert.Len(t, links["prs"].([]PRLink), 1)
		assert.Len(t, links["branches"].([]BranchLink), 1)
	})
	t.Run("ticket_get_links requires an id", func(t *testing.T) {
		call := toolByName(t, tools, "ticket_get_links").Call
		_, err := call(context.Background(), map[string]any{})
		require.Error(t, err)
	})
}

func TestMCPTools_TicketSetType(t *testing.T) {
	s := newTestService(newFakeRepo())
	created, err := s.Create(context.Background(), "p-1", "Fix", "", "", "")
	require.NoError(t, err)
	call := toolByName(t, MCPTools(s), "ticket_set_type").Call

	t.Run("sets a type", func(t *testing.T) {
		got, err := call(context.Background(), map[string]any{"id": created.ID, "type_id": "bug"})
		require.NoError(t, err)
		assert.Equal(t, "bug", got.(*Ticket).TypeID)
	})
	t.Run("missing type id is invalid", func(t *testing.T) {
		_, err := call(context.Background(), map[string]any{"id": created.ID})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
}

func TestMCPTools_TicketAddLabel(t *testing.T) {
	s := newTestService(newFakeRepo())
	created, err := s.Create(context.Background(), "p-1", "Fix", "", "", "")
	require.NoError(t, err)
	call := toolByName(t, MCPTools(s), "ticket_add_label").Call

	t.Run("adds a label", func(t *testing.T) {
		got, err := call(context.Background(), map[string]any{"id": created.ID, "label": "bug"})
		require.NoError(t, err)
		assert.Contains(t, got.(*Ticket).Labels, "bug")
	})
	t.Run("missing label is invalid", func(t *testing.T) {
		_, err := call(context.Background(), map[string]any{"id": created.ID})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
}

func TestMCPTools_TicketRemoveLabel(t *testing.T) {
	s := newTestService(newFakeRepo())
	created, err := s.Create(context.Background(), "p-1", "Fix", "", "", "")
	require.NoError(t, err)
	_, err = s.AddLabel(context.Background(), created.ID, "bug")
	require.NoError(t, err)
	call := toolByName(t, MCPTools(s), "ticket_remove_label").Call

	got, err := call(context.Background(), map[string]any{"id": created.ID, "label": "bug"})
	require.NoError(t, err)
	assert.NotContains(t, got.(*Ticket).Labels, "bug")
}

func TestMCPTools_TicketListLabels(t *testing.T) {
	s := newTestService(newFakeRepo())
	created, err := s.Create(context.Background(), "p-1", "Fix", "", "", "")
	require.NoError(t, err)
	_, err = s.AddLabel(context.Background(), created.ID, "bug")
	require.NoError(t, err)

	call := toolByName(t, MCPTools(s), "ticket_list_labels").Call
	got, err := call(context.Background(), map[string]any{"id": created.ID})
	require.NoError(t, err)
	assert.Contains(t, got.([]string), "bug")
}

func TestMCPTools_TicketListAllLabels(t *testing.T) {
	s := newTestService(newFakeRepo())
	created, err := s.Create(context.Background(), "p-1", "Fix", "", "", "")
	require.NoError(t, err)
	_, err = s.AddLabel(context.Background(), created.ID, "bug")
	require.NoError(t, err)

	call := toolByName(t, MCPTools(s), "ticket_list_all_labels").Call
	got, err := call(context.Background(), map[string]any{})
	require.NoError(t, err)
	assert.Contains(t, got.([]string), "bug")
}
