package t3clientv2

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/harness"
	"github.com/otal-labs/nexul/internal/t3rpc/t3rpctest"
)

// shellRow is a thread's sidebar row as T3's shell sends it, with only what the row needs to say; the rest is noise.
func shellRow(id string, latest, activity any, items int) map[string]any {
	return map[string]any{"id": id, "projectId": "pr-1", "title": "Fix login", "latestRunId": latest, "activeRunId": nil,
		"activityRunStatus": activity, "status": "idle", "itemCount": items, "deletedAt": nil}
}

func TestWatchSessions_ReportsEachThreadsNewsOnceUntilClosed(t *testing.T) {
	t.Parallel()
	f, h := newFake(t, 2)
	f.ShellThreads = []any{shellRow("th-1", "run-1", nil, 4), shellRow("th-2", nil, nil, 0)}
	updates := make(chan harness.SessionUpdate, 16)

	conn, err := h.WatchSessions(t.Context(), laptop(f), func(u harness.SessionUpdate) { updates <- u })
	require.NoError(t, err)
	subID := t3rpctest.WaitFor(t, f.ShellSubscribed, "subscribeShell")
	f.Write(t3rpctest.Chunk(subID,
		map[string]any{"kind": "thread.updated", "sequence": 2, "location": "active", "thread": shellRow("th-1", "run-1", nil, 9)},
		map[string]any{"kind": "thread.updated", "sequence": 3, "location": "active", "thread": shellRow("th-1", "run-2", "running", 10)},
		map[string]any{"kind": "thread.updated", "sequence": 4, "location": "active", "thread": shellRow("th-1", "run-2", "waiting", 12)},
		map[string]any{"kind": "thread.removed", "sequence": 5, "location": "active", "threadId": "th-2"},
	))

	want := []harness.SessionUpdate{
		{SessionID: "th-1", Latest: "run-1"},
		{SessionID: "th-2"},
		{SessionID: "th-1", Latest: "run-2", Working: true},
		{SessionID: "th-1", Latest: "run-2"},
		{SessionID: "th-2", Gone: true},
	}
	for i, w := range want {
		select {
		case got := <-updates:
			assert.Equal(t, w, got, "update %d", i)
		case <-time.After(5 * time.Second):
			t.Fatalf("update %d never came", i)
		}
	}
	assert.Empty(t, updates, "the second snapshot and a row whose item count alone moved say nothing new")

	require.NoError(t, conn.Close())
	select {
	case <-conn.Done():
	case <-time.After(5 * time.Second):
		t.Fatal("Done never fired after Close")
	}
}

func TestShellThreadUpdate_NewestRunThatNeverRan_IsNoLatest(t *testing.T) {
	t.Parallel()
	started := "2026-10-06T15:47:27.000Z"
	tests := []struct {
		name       string
		status     string
		startedAt  *string
		wantLatest string
	}{
		{"cancelled before it started, its message steered into the run before", "cancelled", nil, ""},
		{"queued behind a run still going", "queued", nil, ""},
		{"cancelled after it started", "cancelled", &started, "run-2"},
		{"completed", "completed", &started, "run-2"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			run := "run-2"
			row := shellThread{ID: "th-1", LatestRunID: &run, Status: tt.status, LatestRunStartedAt: tt.startedAt}
			assert.Equal(t, harness.SessionUpdate{SessionID: "th-1", Latest: tt.wantLatest}, row.update())
		})
	}
}

func TestWatchSessions_ServerOnTheOldOrchestrator_IsRefused(t *testing.T) {
	t.Parallel()
	f, h := newFake(t, 1)
	_, err := h.WatchSessions(t.Context(), laptop(f), func(harness.SessionUpdate) {})
	require.ErrorIs(t, err, harness.ErrProtocol)
}

func TestShellUpdates_ReadsEveryKindItCarriesAndNothingElse(t *testing.T) {
	t.Parallel()
	latest, preparing := "run-3", "preparing"
	tests := []struct {
		name string
		item shellItem
		want []harness.SessionUpdate
	}{
		{"a deleted row is gone", shellItem{Kind: "thread.updated", Thread: shellThread{ID: "th-1", DeletedAt: &latest}},
			[]harness.SessionUpdate{{SessionID: "th-1", Gone: true}}},
		{"a preparing run is working", shellItem{Kind: "thread.updated", Thread: shellThread{ID: "th-1", LatestRunID: &latest, ActivityRunStatus: &preparing}},
			[]harness.SessionUpdate{{SessionID: "th-1", Latest: "run-3", Working: true}}},
		{"a project change is no thread's news", shellItem{Kind: "project.updated"}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, shellUpdates(tt.item))
		})
	}
}
