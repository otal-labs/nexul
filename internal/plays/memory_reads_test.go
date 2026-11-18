package plays

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/harness"
	"github.com/otal-labs/nexul/internal/harness/harnesstest"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

func TestMemoryReadOf_MatchesAFinishedMemoryGetUnderEveryHarnessName(t *testing.T) {
	done := func(tool, detail string) ActivityEntry {
		return ActivityEntry{Kind: harness.ActivityToolResult, Tool: tool, Summary: `{"id":"m-1"}`, Detail: detail}
	}
	read := `{"input":{"id":"m-1"},"result":{"content":"body"}}`
	tests := []struct {
		name string
		step ActivityEntry
		want string
	}{
		{"the mcp__nexul__ prefix", done("mcp__nexul__memory_get", read), "m-1"},
		{"a server · tool name", done("nexul · memory_get", read), "m-1"},
		{"a nexul_ prefix", done("nexul_memory_get", read), "m-1"},
		{"the bare name", done("memory_get", read), "m-1"},
		{"a version is still a read", done("memory_get", `{"input":{"id":"m-1","version":2}}`), "m-1"},
		{"arguments sent as a string", done("memory_get", `{"input":"{\"id\":\"m-1\"}"}`), "m-1"},
		{"a result the cap cut short", done("memory_get", `{"input":{"id":"m-1"},"result":{"content":"bo`), "m-1"},
		{"another tool", done("mcp__nexul__memory_list", `{"input":{"id":"m-1"}}`), ""},
		{"a tool that only ends in the same letters", done("xmemory_get", read), ""},
		{"a call still open", ActivityEntry{Kind: harness.ActivityToolCall, Tool: "memory_get", Detail: read}, ""},
		{"a failed call", ActivityEntry{Kind: harness.ActivityToolResult, Tool: "memory_get", Summary: `{"id":"m-1"} · failed`, Detail: read}, ""},
		{"no id", done("memory_get", `{"input":{}}`), ""},
		{"no detail", done("memory_get", ""), ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := MemoryReadOf(tt.step)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.want != "", ok)
		})
	}
}

func TestRun_AgainstHarness_AMemoryReadIsRecordedOncePerMemory(t *testing.T) {
	f := newRunnerFixture()
	client := &harnesstest.Client{StartTurnFn: func(context.Context, harness.Target, string, harness.TurnPrompts) (harness.StartResult, error) {
		at := time.Unix(1_800_000_000, 0).UTC()
		ch := make(chan harness.Update, 6)
		for _, a := range []harness.Activity{
			{Kind: harness.ActivityToolCall, CallID: "c-1", Tool: "mcp__nexul__memory_get", Detail: `{"input":{}}`, At: at},
			{Kind: harness.ActivityToolResult, CallID: "c-1", Tool: "mcp__nexul__memory_get", Detail: `{"input":{"id":"m-build"}}`, At: at},
			{Kind: harness.ActivityToolResult, CallID: "c-2", Tool: "mcp__nexul__memory_get", Detail: `{"input":{"id":"m-build"}}`, At: at},
			{Kind: harness.ActivityToolResult, CallID: "c-3", Tool: "mcp__nexul__memory_list", Detail: `{"input":{"project_id":"p-1"}}`, At: at},
		} {
			ch <- harness.Update{Activity: &a}
		}
		ch <- harness.Update{Terminal: &harness.TurnResult{State: harness.TurnDone}}
		close(ch)
		return harness.StartResult{SessionID: "sess-1", Updates: ch}, nil
	}}
	newHarnessRunner(f, client)

	trail, err := f.runner.Run(ctxAs(starter), ticketRun())
	require.NoError(t, err)
	<-f.trails.terminal

	f.trails.mu.Lock()
	defer f.trails.mu.Unlock()
	assert.Equal(t, [][3]string{{trail.ID, projectID, "m-build"}}, f.trails.reads, "the open call names no id, and the second read of the same memory writes nothing")
}

func TestGetTrail_FoundMemories_AreTheReadsItWasNotToldOf_ForAReaderOfTheProjectsMemories(t *testing.T) {
	f := newRunnerFixture()
	seededTrail(f, "tr-1", TargetTicket, ticketID, fixedNow)
	found := []FoundMemory{{ID: "m-build", Title: "Building a ticket"}, {ID: "m-gone", Title: "Old release notes", Deleted: true}}
	f.trails.found = map[string][]FoundMemory{"tr-1": append([]FoundMemory{{ID: alwaysMem, Title: "Working here"}}, found...)}

	blind, err := f.runner.GetTrail(ctxAs(starter), "tr-1")
	require.NoError(t, err)
	assert.Empty(t, blind.FoundMemories, "without memories:read on the project no title is shown")

	f.perm.grants[starter] = append(f.perm.grants[starter], permissions.MemoriesRead)
	got, err := f.runner.GetTrail(ctxAs(starter), "tr-1")
	require.NoError(t, err)
	assert.Equal(t, found, got.FoundMemories, "the memory it was told to read stays out")
}
