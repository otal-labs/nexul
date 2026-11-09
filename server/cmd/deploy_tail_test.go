package main

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/access"
	"github.com/otal-labs/nexul/internal/deploy"
	"github.com/otal-labs/nexul/internal/platform/identity"
)

// TestRows_DeployToolsReadOnlyWhatTheyReturn: deploy_list with an id on a 50,000-line log hands back about its 200-line
// tail and a count, and stack_get on a stack with 200 deploys reads its newest ten, where both read everything.
func TestRows_DeployToolsReadOnlyWhatTheyReturn(t *testing.T) {
	t.Parallel()
	f, st, db := newCountedFixtureDB(t)
	_, err := db.Exec(`INSERT INTO stacks (id, project_id, name, slug, machine, strategy, created_at, updated_at)
		VALUES ('stack-tail', ?, 'web', 'web-tail', 'm1', 'run', 1, 1)`, pGeneral)
	require.NoError(t, err)
	for _, stmt := range []string{
		`INSERT INTO deploys (id, stack_id, service, target, image, status, strategy, created_at, updated_at)
			SELECT printf('dep-%03d', i), 'stack-tail', 'web', 'm1', 'web:' || i, 'healthy', 'run', 1700000000 + i, 1700000000 + i FROM n`,
		`INSERT INTO deploy_log_lines (deploy_id, ts, phase, line)
			SELECT 'dep-200', 1700000000000 + i, 'build', 'step ' || i FROM n2`,
	} {
		_, err := db.Exec(`WITH RECURSIVE n(i) AS (SELECT 1 UNION ALL SELECT i + 1 FROM n WHERE i < 200),
			n2(i) AS (SELECT 1 UNION ALL SELECT i + 1 FROM n2 WHERE i < 50000) ` + stmt)
		require.NoError(t, err)
	}
	ctx := access.WithMemo(identity.WithActor(context.Background(), identity.Actor{ID: uOwner}), f.svc.accessSvc.NewMemo())
	tools := deploy.MCPTools(f.svc.deploySvc)

	st.Reset()
	out, err := tool(t, tools, "deploy_list").Call(ctx, json.RawMessage(`{"id":"dep-200"}`))
	require.NoError(t, err)
	raw, err := json.Marshal(out)
	require.NoError(t, err)
	var detail struct {
		Log      []struct{ Text string }
		LogTotal int `json:"log_total"`
	}
	require.NoError(t, json.Unmarshal(raw, &detail))
	require.Len(t, detail.Log, 200)
	assert.Equal(t, "step 49801", detail.Log[0].Text, "the tail starts at the oldest of the last 200 lines")
	assert.Equal(t, "step 50000", detail.Log[199].Text)
	assert.Equal(t, 50000, detail.LogTotal)
	assert.Less(t, st.Rows(), int64(250), "rows handed back for a deploy's log tail")

	st.Reset()
	out, err = tool(t, tools, "stack_get").Call(ctx, json.RawMessage(`{"id":"stack-tail"}`))
	require.NoError(t, err)
	raw, err = json.Marshal(out)
	require.NoError(t, err)
	var stack struct {
		RecentDeploys []struct{ ID string } `json:"recent_deploys"`
	}
	require.NoError(t, json.Unmarshal(raw, &stack))
	require.Len(t, stack.RecentDeploys, 10)
	assert.Equal(t, "dep-200", stack.RecentDeploys[0].ID, "newest first")
	assert.Less(t, st.Rows(), int64(100), "rows handed back for a stack's detail")
}
