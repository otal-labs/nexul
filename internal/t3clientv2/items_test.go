package t3clientv2

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/harness"
)

const (
	toolSteps   = "tool-steps.source-65731f986b.ndjson"
	childThread = "subagent-child-thread.source-65731f986b.ndjson"
	claudeItem  = "turn-item:provider:claudeAgent:native-item:"
	question1   = "runtime-request:provider:claudeAgent:native-request:native-rq-1"
	approval1   = "runtime-request:provider:claudeAgent:native-request:native-rq-2"
	// questionItem and approvalItem are those requests' turn items, their ids escaped the way T3 joins them.
	questionItem   = "turn-item:runtime-request:runtime-request%3Aprovider%3AclaudeAgent%3Anative-request%3Anative-rq-1"
	approvalItem   = "turn-item:runtime-request:runtime-request%3Aprovider%3AclaudeAgent%3Anative-request%3Anative-rq-2"
	approvalPrompt = "Claude wants to run: rm -rf build"
)

// recordedItem is the first version of turn item id in status that a fixture's streams delivered.
func recordedItem(t *testing.T, fixture, id, status string) turnItem {
	t.Helper()
	for _, raw := range recorded(t, fixture) {
		var item struct {
			Event struct {
				Type    string          `json:"type"`
				Payload json.RawMessage `json:"payload"`
			} `json:"event"`
			Projection struct {
				TurnItems []json.RawMessage `json:"turnItems"`
			} `json:"projection"`
		}
		require.NoError(t, json.Unmarshal(raw, &item))
		candidates := item.Projection.TurnItems
		if item.Event.Type == "turn-item.updated" {
			candidates = append(candidates, item.Event.Payload)
		}
		for _, c := range candidates {
			var it turnItem
			require.NoError(t, json.Unmarshal(c, &it))
			if it.ID == id && it.Status == status {
				return it
			}
		}
	}
	t.Fatalf("%s has no item %s in status %s", fixture, id, status)
	return turnItem{}
}

func stepOf(kind harness.ActivityKind, callID, tool, summary, detail string, second int) *harness.Update {
	return &harness.Update{Activity: &harness.Activity{Kind: kind, CallID: callID, Tool: tool, Summary: summary, Detail: detail,
		At: time.Date(2026, 10, 3, 17, 0, second, 0, time.UTC)}}
}

func TestMapItem_RecordedItems(t *testing.T) {
	t.Parallel()
	const (
		bash      = claudeItem + "native-1"
		codexBash = "turn-item:provider:codex:native-item:native-31"
		codexMCP  = "turn-item:provider:codex:native-item:native-33"
		grokItem  = "turn-item:provider:grok:native-item:"
		askInput  = `{"questions":[{"question":"Which database should the cache use?","header":"Database","multiSelect":false,"options":[{"label":"SQLite","description":"One file beside the server"},{"label":"Postgres","description":"A separate server"}]}]}`
	)
	call, result, asked := harness.ActivityToolCall, harness.ActivityToolResult, harness.ActivityQuestion
	tests := []struct {
		name, fixture, id, status string
		// patch changes the recorded item, for a shape no capture shows on its own.
		patch func(*turnItem)
		// want is nil for an item that shows as nothing.
		want *harness.Update
	}{
		{name: "a command in flight is a Shell call", fixture: toolSteps, id: bash, status: "running",
			want: stepOf(call, bash, "Shell", "go test ./internal/t3clientv2/", `{"input":"go test ./internal/t3clientv2/"}`, 12)},
		{name: "the same command failing keeps its call id and is marked failed", fixture: toolSteps, id: bash, status: "failed",
			want: stepOf(result, bash, "Shell", "go test ./internal/t3clientv2/ · failed", `{"input":"go test ./internal/t3clientv2/"}`, 15)},
		{name: "a completed command whose output looked failed is failed", fixture: toolSteps, id: bash, status: "failed",
			patch: func(it *turnItem) { it.Status, it.OutputIndicatesFailure = "completed", true },
			want:  stepOf(result, bash, "Shell", "go test ./internal/t3clientv2/ · failed", `{"input":"go test ./internal/t3clientv2/"}`, 15)},
		{name: "a command that exited nonzero is failed", fixture: toolSteps, id: codexBash, status: "completed",
			want: stepOf(result, codexBash, "Shell", "/bin/bash -lc 'go vet ./...' · failed", `{"input":"/bin/bash -lc 'go vet ./...'"}`, 62)},
		{name: "a nonzero exit code alone is failed", fixture: toolSteps, id: codexBash, status: "completed",
			patch: func(it *turnItem) { it.OutputIndicatesFailure = false },
			want:  stepOf(result, codexBash, "Shell", "/bin/bash -lc 'go vet ./...' · failed", `{"input":"/bin/bash -lc 'go vet ./...'"}`, 62)},
		{name: "a file change is an Edit of the file", fixture: toolSteps, id: claudeItem + "native-2", status: "completed",
			want: stepOf(result, claudeItem+"native-2", "Edit", "internal/t3clientv2/items.go", "", 16)},
		{name: "Claude's Grep keeps its name and shows its arguments", fixture: toolSteps, id: claudeItem + "native-3", status: "completed",
			want: stepOf(result, claudeItem+"native-3", "Grep", `{"pattern":"mapItem","path":"internal/t3clientv2"}`,
				`{"input":{"pattern":"mapItem","path":"internal/t3clientv2"}}`, 17)},
		{name: "a file search is a Grep of its pattern", fixture: toolSteps, id: grokItem + "native-41", status: "completed",
			want: stepOf(result, grokItem+"native-41", "Grep", "TODO", "", 72)},
		{name: "Claude's MCP call keeps its mcp__ name", fixture: toolSteps, id: claudeItem + "native-4", status: "completed",
			want: stepOf(result, claudeItem+"native-4", "mcp__nexul__ticket_get", `{"key":"REF-102"}`, `{"input":{"key":"REF-102"}}`, 18)},
		{name: "Codex's server.tool reads as server · tool", fixture: toolSteps, id: codexMCP, status: "failed",
			want: stepOf(result, codexMCP, "nexul · ticket_get", `{"key":"REF-404"} · failed`, `{"input":{"key":"REF-404"}}`, 64)},
		{name: "an ACP provider's MCP call reads as server · tool too", fixture: toolSteps, id: grokItem + "native-43", status: "completed",
			want: stepOf(result, grokItem+"native-43", "nexul · memory_list", `{"project":"pr-1"}`, `{"input":{"project":"pr-1"}}`, 72)},
		{name: "an ACP tool named by its title keeps the title", fixture: toolSteps, id: grokItem + "native-42", status: "completed",
			want: stepOf(result, grokItem+"native-42", "Check go.mod version", "Check go.mod version", `{"input":{}}`, 72)},
		{name: "a completed call whose result is an error is failed", fixture: toolSteps, id: codexMCP, status: "failed",
			patch: func(it *turnItem) { it.Status = "completed" },
			want:  stepOf(result, codexMCP, "nexul · ticket_get", `{"key":"REF-404"} · failed`, `{"input":{"key":"REF-404"}}`, 64)},
		{name: "an image read names the image", fixture: toolSteps, id: claudeItem + "native-5", status: "completed",
			want: stepOf(result, claudeItem+"native-5", "Read", "/workspace/docs/shot.png", `{"input":{"file_path":"/workspace/docs/shot.png"}}`, 19)},
		{name: "a web search names its query", fixture: toolSteps, id: claudeItem + "native-6", status: "completed",
			want: stepOf(result, claudeItem+"native-6", "WebSearch", "t3code orchestrator v2", "", 20)},
		{name: "a subagent is an Agent step named by its title", fixture: toolSteps, id: claudeItem + "native-8", status: "running",
			want: stepOf(call, claudeItem+"native-8", "Agent", "Find callers", "", 43)},
		{name: "a subagent without a title is named by its prompt", fixture: toolSteps, id: claudeItem + "native-8", status: "running",
			patch: func(it *turnItem) { it.Title = "" },
			want:  stepOf(call, claudeItem+"native-8", "Agent", "Find every caller of mapItem.", "", 43)},
		{name: "a subagent's own item, which has no run, maps", fixture: childThread, id: claudeItem + "native-81", status: "completed",
			want: stepOf(result, claudeItem+"native-81", "Shell", "rg -n mapItem internal", `{"input":"rg -n mapItem internal"}`, 45)},
		{name: "Claude's AskUserQuestion is a question step", fixture: toolSteps, id: claudeItem + "native-7", status: "running",
			want: stepOf(asked, claudeItem+"native-7", questionTool, askInput[:160]+"…", `{"input":`+askInput+`}`, 21)},
		{name: "a waiting question is asked", fixture: toolSteps, id: questionItem, status: "waiting",
			want: &harness.Update{Question: &harness.Question{RequestID: question1, Questions: []harness.QuestionItem{{
				ID: "Which database should the cache use?", Text: "Which database should the cache use?", Header: "Database",
				Options: []harness.QuestionOption{{Label: "SQLite", Description: "One file beside the server"}, {Label: "Postgres", Description: "A separate server"}},
			}}}}},
		{name: "an answered question asks nothing", fixture: toolSteps, id: questionItem, status: "completed"},
		{name: "Codex's question with empty option descriptions decodes", fixture: toolSteps, id: "turn-item:runtime-request:runtime-request%3Aprovider%3Acodex%3Anative-request%3Aasync%253Anative-34", status: "waiting",
			want: &harness.Update{Question: &harness.Question{RequestID: "runtime-request:provider:codex:native-request:async%3Anative-34", Questions: []harness.QuestionItem{{
				ID: "0", Text: "Ship it today?", Header: "Question", Options: []harness.QuestionOption{{Label: "Yes"}, {Label: "No"}},
			}}}}},
		{name: "a waiting approval is raised", fixture: toolSteps, id: approvalItem, status: "waiting",
			want: &harness.Update{Approval: &harness.Approval{Kind: "command", Summary: approvalPrompt}}},
		{name: "a declined approval is raised no more", fixture: toolSteps, id: approvalItem, status: "cancelled"},
		{name: "an assistant message is no step", fixture: toolSteps, id: claudeItem + "native-9", status: "completed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			it := recordedItem(t, tt.fixture, tt.id, tt.status)
			if tt.patch != nil {
				tt.patch(&it)
			}
			got, ok := mapItem(it)
			if tt.want == nil {
				assert.False(t, ok, "shown as %+v", got)
				return
			}
			require.True(t, ok)
			assert.Equal(t, *tt.want, got)
		})
	}
}
