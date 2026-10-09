package main

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/eventcatalog"
	"github.com/otal-labs/nexul/internal/platform/storage"
)

// TestAutomationScope_EveryPublishedTopicHasARule keeps a new topic from silently reaching no automation.
func TestAutomationScope_EveryPublishedTopicHasARule(t *testing.T) {
	for _, topic := range eventcatalog.AllTopics() {
		_, ok := scopeRules[topic]
		assert.True(t, ok, "topic %s has no workspace rule for automations", topic)
	}
}

func TestAutomationScope_Workspaces(t *testing.T) {
	db := mentionsTestDB(t)
	_, err := db.Exec(`
INSERT INTO workspaces (id, name, slug, created_at, updated_at) VALUES ('ws-b', 'B', 'b', 1, 1);
INSERT INTO projects (id, name, position, workspace_id, created_at, updated_at) VALUES
    ('p-a', 'A', 1, 'workspace-default', 1, 1), ('p-b', 'B', 1, 'ws-b', 1, 1);
INSERT INTO tickets (id, project_id, title, status, created_at, updated_at) VALUES
    ('t-a', 'p-a', 'A', 's', 1, 1), ('t-b', 'p-b', 'B', 's', 1, 1);
INSERT INTO project_repos (project_id, owner, name, added_at) VALUES ('p-b', 'acme', 'web', 1);
INSERT INTO stacks (id, project_id, name, slug, machine, strategy, created_at, updated_at) VALUES
    ('st-b', 'p-b', 'web', 'web', 'm', 'compose', 1, 1), ('st-instance', NULL, 'nexul', 'nexul', 'm', 'compose', 1, 1);
INSERT INTO deploys (id, stack_id, service, target, image, status, strategy, created_at, updated_at) VALUES
    ('d-b', 'st-b', 'web', 'm', 'i', 'done', 'compose', 1, 1), ('d-instance', 'st-instance', 'nexul', 'm', 'i', 'done', 'compose', 1, 1);
`)
	require.NoError(t, err)
	scope := automationScope{lookup: storage.New(db, []byte("0123456789abcdef0123456789abcdef")).EventWorkspaces}

	tests := []struct {
		name, topic string
		payload     any
		want        []string
		every       bool
	}{
		{"a ticket event is its project's workspace", "ticket.finished", map[string]any{"ticket": map[string]any{"id": "t-b", "project_id": "p-b"}}, []string{"ws-b"}, false},
		{"a doc folder event is its project's workspace", "doc.folder.deleted", map[string]any{"folder": map[string]any{"id": "f-b", "project_id": "p-b"}}, []string{"ws-b"}, false},
		{"a deleted ticket names its project", "ticket.deleted", map[string]any{"id": "t-gone", "project_id": "p-a"}, []string{"workspace-default"}, false},
		{"a PR reaches every workspace among its linked tickets", "git.pr_opened", map[string]any{"owner": "acme", "repo": "web", "pr": map[string]any{"linked_ticket_ids": []string{"t-a", "t-b"}}}, []string{"workspace-default", "ws-b"}, false},
		{"a PR linking nothing is its repository's", "git.pr_merged", map[string]any{"owner": "acme", "repo": "web", "pr": map[string]any{}}, []string{"ws-b"}, false},
		{"a project stack's deploy is the project's", "deploy.status_changed", map[string]any{"id": "d-b"}, []string{"ws-b"}, false},
		{"an instance stack's deploy reaches every workspace", "deploy.status_changed", map[string]any{"id": "d-instance"}, nil, true},
		{"a notice names the workspace whose inbox holds it", "notification.created", map[string]any{"user_ids": []string{"u"}, "workspace_id": "ws-b"}, []string{"ws-b"}, false},
		{"a notice outside any workspace reaches every workspace", "notification.created", map[string]any{"user_ids": []string{"u"}, "workspace_id": ""}, nil, true},
		{"a runner event reaches every workspace", "runner.connected", map[string]any{"runner_id": "r"}, nil, true},
		{"a workspace's canvas change is that workspace's", "topology.updated", map[string]any{"environment": "ws-b", "workspace_id": "ws-b"}, []string{"ws-b"}, false},
		{"a service registry change reaches nobody", "topology.updated", map[string]any{"environment": "default"}, nil, false},
		{"a ticket that is gone reaches nobody", "ticket.link_created", map[string]any{"link": map[string]any{"ticket_id": "t-gone"}}, nil, false},
		{"an unknown topic reaches nobody", "made.up", map[string]any{}, nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			raw, err := json.Marshal(tt.payload)
			require.NoError(t, err)
			ids, every, err := scope.Workspaces(t.Context(), tt.topic, raw)
			require.NoError(t, err)
			assert.Equal(t, tt.every, every)
			assert.ElementsMatch(t, tt.want, ids)
		})
	}
}
