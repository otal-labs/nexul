package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/platform/storage"
	"github.com/otal-labs/nexul/internal/platform/storage/testutil/migrated"
	"github.com/otal-labs/nexul/internal/tickets"
)

// TestPRTicketResolver_ResolvesKeysInTheRepositorysWorkspace guards ADR 0089: two workspaces share the REF
// prefix, and a PR links only the ticket in the workspace whose project holds the repository.
func TestPRTicketResolver_ResolvesKeysInTheRepositorysWorkspace(t *testing.T) {
	t.Parallel()
	db := migrated.DB(t)
	_, err := db.Exec(`
INSERT INTO workspaces (id, name, slug, created_at, updated_at) VALUES ('ws-b', 'B', 'b', 1, 1);
INSERT INTO projects (id, name, prefix, position, workspace_id, created_at, updated_at) VALUES
    ('p-a', 'A', 'REF', 1, 'workspace-default', 1, 1), ('p-b', 'B', 'REF', 1, 'ws-b', 1, 1);
INSERT INTO tickets (id, project_id, number, title, status, created_at, updated_at) VALUES
    ('t-a1', 'p-a', 1, 'A', 's', 1, 1), ('t-b1', 'p-b', 1, 'B', 's', 1, 1), ('t-b2', 'p-b', 2, 'B2', 's', 1, 1),
    ('t-b3', 'p-b', 3, 'Moved in', 's', 1, 1), ('t-b3-old', 'p-b', 3, 'Kept its number', 's', 2, 2);
INSERT INTO project_repos (project_id, owner, name, added_at) VALUES ('p-b', 'acme', 'web', 1);
`)
	require.NoError(t, err)
	s := storage.New(db, []byte("0123456789abcdef0123456789abcdef"))
	r := prTicketResolver{repoWorkspace: s.EventWorkspaces, tickets: tickets.NewService(s.Tickets, s.Statuses, workspaceUserStore{users: s.Users})}

	tests := []struct {
		name, repo string
		keys       []string
		want       []string
	}{
		{"keys resolve in the repository's workspace only", "web", []string{"REF-1", "REF-2"}, []string{"t-b1", "t-b2"}},
		{"a key naming no ticket links nothing", "web", []string{"REF-9", "UTF-8", "REF-2"}, []string{"t-b2"}},
		{"a key naming two tickets links neither", "web", []string{"REF-3"}, nil},
		{"a repository no project holds links nothing", "unattached", []string{"REF-1"}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ids, err := r.TicketIDs(t.Context(), "acme", tt.repo, tt.keys)
			require.NoError(t, err)
			assert.Equal(t, tt.want, ids)
		})
	}
}
