package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/runner"
)

// seedEdgeAndCore writes two machines, each with a deploy runner, and lets carol see edge alone. Carol's role holds
// every bit, so what keeps her off core is the machine's rules and nothing else.
func seedEdgeAndCore(t *testing.T, db *sql.DB) {
	t.Helper()
	_, err := db.Exec(`INSERT INTO machines (id, name, stack_root, reported_hostname, first_seen, last_seen) VALUES
    ('m-edge', 'edge-box', '/data/nexul', 'edge-box', 1, 1), ('m-core', 'core-box', '/data/nexul', 'core-box', 1, 1);
INSERT INTO runners (id, name, last_seen, connected, created_at, version, machine_id) VALUES
    ('r-edge', 'edge-runner', 1, 1, 1, 'v1', 'm-edge'), ('r-core', 'core-runner', 1, 1, 1, 'v1', 'm-core');
INSERT INTO permission_overwrites (resource_type, resource_id, target, user_id, allow, deny, created_at, updated_at)
VALUES ('machine', 'm-edge', 'person', 'u-carol', '["machines:see"]', '[]', 1, 1)`)
	require.NoError(t, err)
}

// TestMachines_NotOnTheMachineIsNotFound: a machine nobody's rules let a person see is not found to them, never
// forbidden, whatever they hold: out of the lists, 404 on every route that names it or its runners, a tool error over MCP,
// and no live frame for its runners. The instance Owner sees both and the person on edge sees edge alone.
func TestMachines_NotOnTheMachineIsNotFound(t *testing.T) {
	c, db := newPrivacyCast(t)
	seedEdgeAndCore(t, db)
	people := []struct {
		name, user, token string
		sees, hidden      []string
	}{
		{"the instance Owner", "u-bob", c.owner, []string{"edge", "core"}, nil},
		{"a person on edge", "u-carol", c.everyBits, []string{"edge"}, []string{"core"}},
		{"a member on no machine", "u-alice", c.alice, nil, []string{"edge", "core"}},
	}
	for _, p := range people {
		t.Run(p.name, func(t *testing.T) {
			for _, name := range p.sees {
				listed := callAPI(t, c.h, http.MethodGet, "/api/machines", p.token, "").Body.String()
				assert.Contains(t, listed, name+"-box", "GET /api/machines")
				assert.Contains(t, callAPI(t, c.h, http.MethodGet, "/api/runners", p.token, "").Body.String(), name+"-runner", "GET /api/runners")
				assert.Contains(t, callTool(t, c.h, p.token, "machine_list", `{}`), name+"-box", "machine_list")
				a := liveAudience{access: c.svc.accessSvc}
				for topic, frame := range runnerFrames(t, "r-"+name, "m-"+name) {
					assert.True(t, a.allows(as(p.user), topic, frame), "%s of %s", topic, name)
				}
			}
			for _, name := range p.hidden {
				id, runnerID := "m-"+name, "r-"+name
				assert.NotContains(t, callAPI(t, c.h, http.MethodGet, "/api/machines", p.token, "").Body.String(), name+"-box", "GET /api/machines")
				assert.NotContains(t, callAPI(t, c.h, http.MethodGet, "/api/runners", p.token, "").Body.String(), name+"-runner", "GET /api/runners")
				assert.NotContains(t, callTool(t, c.h, p.token, "machine_list", `{}`), name+"-box", "machine_list")
				for _, call := range []struct{ method, path, body string }{
					{http.MethodPatch, "/api/machines/" + id, `{"stack_root":"/srv/nexul"}`},
					{http.MethodPost, "/api/machines/" + id + "/discover", ""},
					{http.MethodDelete, "/api/runners/" + runnerID, ""},
				} {
					rec := callAPI(t, c.h, call.method, call.path, p.token, call.body)
					assert.Equal(t, http.StatusNotFound, rec.Code, "%s %s: %s", call.method, call.path, rec.Body.String())
				}
				body := callToolResult(t, c.h, p.token, "machine_discover", `{"id":"`+id+`"}`)
				assert.Contains(t, body, `"isError":true`, "machine_discover")
				assert.Contains(t, body, "not found", "machine_discover")

				a := liveAudience{access: c.svc.accessSvc}
				for topic, frame := range runnerFrames(t, runnerID, id) {
					assert.False(t, a.allows(as(p.user), topic, frame), "%s of %s", topic, name)
				}
			}
		})
	}

	rec := callAPI(t, c.h, http.MethodPatch, "/api/machines/m-edge", c.everyBits, `{"stack_root":"/srv/nexul"}`)
	assert.Equal(t, http.StatusOK, rec.Code, "a person on the machine, holding Manage, still changes it: %s", rec.Body.String())

	a := liveAudience{access: c.svc.accessSvc}
	noMachine, err := json.Marshal(runner.RunnerConnectedEvent{RunnerID: "r-edge"})
	require.NoError(t, err)
	assert.False(t, a.allows(as("u-bob"), runner.TopicRunnerConnected, json.RawMessage(noMachine)), "a frame naming no machine reaches nobody")
}

// runnerFrames is the presence frames a runner on machineID publishes, by topic.
func runnerFrames(t *testing.T, runnerID, machineID string) map[string]json.RawMessage {
	t.Helper()
	connected, err := json.Marshal(runner.RunnerConnectedEvent{RunnerID: runnerID, MachineID: machineID})
	require.NoError(t, err)
	disconnected, err := json.Marshal(runner.RunnerDisconnectedEvent{RunnerID: runnerID, MachineID: machineID})
	require.NoError(t, err)
	return map[string]json.RawMessage{runner.TopicRunnerConnected: connected, runner.TopicRunnerDisconnected: disconnected}
}
