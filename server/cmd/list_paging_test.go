package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/access"
	"github.com/otal-labs/nexul/internal/chat"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/mcptool"
)

// tool finds name among tools, failing the test when it is missing.
func tool(t *testing.T, tools []mcptool.Tool, name string) mcptool.Tool {
	t.Helper()
	for _, tl := range tools {
		if tl.Name == name {
			return tl
		}
	}
	t.Fatalf("no tool %s", name)
	return mcptool.Tool{}
}

// callPage calls a list tool as user on a fresh request memo and decodes its page, ids and all.
func callPage(t *testing.T, a *access.Service, tl mcptool.Tool, user, args string) (ids []string, page mcptool.Page[json.RawMessage]) {
	t.Helper()
	ctx := access.WithMemo(identity.WithActor(context.Background(), identity.Actor{ID: user}), a.NewMemo())
	out, err := tl.Call(ctx, json.RawMessage(args))
	require.NoError(t, err, "%s as %s", tl.Name, user)
	raw, err := json.Marshal(out)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(raw, &page))
	for _, item := range page.Items {
		var withID struct {
			ID string `json:"id"`
		}
		require.NoError(t, json.Unmarshal(item, &withID))
		ids = append(ids, withID.ID)
	}
	return ids, page
}

// TestRows_MessageListReadsOnlyItsPage: a page of 50 from a 20,000-message conversation hands back about 50 rows, where
// paging in memory read the whole history, and the page query walks the live-message index without sorting.
func TestRows_MessageListReadsOnlyItsPage(t *testing.T) {
	f, st, db := newCountedFixtureDB(t)
	_, err := db.Exec(`WITH RECURSIVE n(i) AS (SELECT 1 UNION ALL SELECT i + 1 FROM n WHERE i < 20000)
		INSERT INTO messages (id, conversation_id, author_id, body, created_at, updated_at, deleted_at)
		SELECT printf('m-%05d', i), ?, ?, 'status update ' || i, 1700000000 + i / 3, 1700000000 + i / 3, CASE WHEN i % 10 = 0 THEN 1700000000 END FROM n`,
		f.channel.ID, uOwner)
	require.NoError(t, err)
	messageList := tool(t, chat.MCPTools(f.svc.chatSvc), "message_list")

	st.Reset()
	ids, page := callPage(t, f.svc.accessSvc, messageList, uOwner, fmt.Sprintf(`{"conversation_id":%q,"limit":50,"offset":100}`, f.channel.ID))

	assert.Len(t, ids, 50)
	assert.Equal(t, 18000, page.Total, "every tenth message is deleted")
	assert.Equal(t, 150, page.NextOffset)
	assert.Less(t, st.Rows(), int64(100), "rows handed back for one page")

	plan := explain(t, db, `SELECT * FROM messages WHERE conversation_id = ? AND deleted_at IS NULL ORDER BY created_at DESC, id DESC LIMIT 50 OFFSET 100`, f.channel.ID)
	assert.Contains(t, plan, "idx_messages_live")
	assert.NotContains(t, plan, "TEMP B-TREE")
}

// explain is SQLite's query plan for query, its detail lines joined.
func explain(t *testing.T, db interface {
	Query(string, ...any) (*sql.Rows, error)
}, query string, args ...any) string {
	t.Helper()
	rows, err := db.Query("EXPLAIN QUERY PLAN "+query, args...)
	require.NoError(t, err)
	defer func() { require.NoError(t, rows.Close()) }()
	var lines []string
	for rows.Next() {
		var id, parent, notused int
		var detail string
		require.NoError(t, rows.Scan(&id, &parent, &notused, &detail))
		lines = append(lines, detail)
	}
	require.NoError(t, rows.Err())
	return strings.Join(lines, "\n")
}
