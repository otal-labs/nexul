package t3client

import (
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/harness"
)

// Frames below are trimmed copies of what T3 emits on thread.activity-appended for one tool call's lifecycle.
const (
	capturedToolStarted = `{"type":"thread.activity-appended","payload":{"threadId":"th-1","activity":{"id":"ab404f45","createdAt":"2026-09-18T10:01:02.195Z","tone":"tool","kind":"tool.started","summary":"Command run started","payload":{"itemType":"command_execution","toolCallId":"toolu_016Pxx","status":"inProgress","title":"Command run","detail":"Bash: {}","data":{"toolName":"Bash","input":{}}},"turnId":"2e274277"}}}`
	capturedToolUpdated = `{"type":"thread.activity-appended","payload":{"threadId":"th-1","activity":{"id":"312f8e50","createdAt":"2026-09-18T10:01:03.000Z","tone":"tool","kind":"tool.updated","summary":"Tool call","payload":{"itemType":"dynamic_tool_call","toolCallId":"toolu_016Pxx","status":"inProgress","title":"Tool call","detail":"Bash: {\"command\":\"go test ./...\"}","data":{"toolName":"Bash","input":{"command":"go test ./...","description":"Run the tests"}}},"turnId":"2e274277"}}}`
	capturedToolDone    = `{"type":"thread.activity-appended","payload":{"threadId":"th-1","activity":{"id":"ee236506","createdAt":"2026-09-18T09:48:34.358Z","tone":"tool","kind":"tool.completed","summary":"Tool call","payload":{"itemType":"dynamic_tool_call","toolCallId":"toolu_01469P","status":"completed","title":"Tool call","detail":"Read: {\"file_path\":\"/workspace/nexul/.golangci.yml\"}","data":{"toolName":"Read","input":{"file_path":"/workspace/nexul/.golangci.yml"},"result":{"tool_use_id":"toolu_01469P","type":"tool_result","content":"1\tversion: \"2\"\n"}}},"turnId":"b53b23b9"}}}`
	capturedQuestion    = `{"type":"thread.activity-appended","payload":{"threadId":"th-1","activity":{"id":"96c2326a","createdAt":"2026-09-18T09:38:09.687Z","tone":"tool","kind":"tool.completed","summary":"Tool call","payload":{"itemType":"dynamic_tool_call","toolCallId":"toolu_01AoUM","status":"failed","title":"Tool call","detail":"AskUserQuestion: {...}","data":{"toolName":"AskUserQuestion","input":{"questions":[{"question":"How do you want to proceed?","header":"Nexul MCP down","options":[{"label":"Proceed without Nexul tools"},{"label":"Stop and wait"}],"multiSelect":false}]}}},"turnId":"0f302641"}}}`
)

// Frames below are trimmed copies of T3's built-in item types: a command run, a file change (in progress and done),
// and an image view. An in-progress file change carries no input, only T3's cut-short detail label.
const (
	capturedCommandUpdated   = `{"type":"thread.activity-appended","payload":{"threadId":"th-1","activity":{"id":"155edc72","createdAt":"2026-09-18T11:37:50.783Z","tone":"tool","kind":"tool.updated","summary":"Command run","payload":{"itemType":"command_execution","toolCallId":"toolu_016556","status":"inProgress","title":"Command run","detail":"Bash: cd /workspace/nexul && rg -n 'HandleFunc' internal/memories/handler.go | head -30","data":{"command":"cd /workspace/nexul && rg -n 'HandleFunc' internal/memories/handler.go | head -30","toolName":"Bash"}},"turnId":"157f87cf"}}}`
	capturedCommandDone      = `{"type":"thread.activity-appended","payload":{"threadId":"th-1","activity":{"id":"4642e7fb","createdAt":"2026-09-18T11:37:50.977Z","tone":"tool","kind":"tool.completed","summary":"Command run","payload":{"itemType":"command_execution","toolCallId":"toolu_016556","status":"completed","title":"Command run","detail":"Bash: cd /workspace/nexul && rg -n 'HandleFunc' internal/memories/handler.go | head -30","data":{"toolName":"Bash","input":{"command":"cd /workspace/nexul && rg -n 'HandleFunc' internal/memories/handler.go | head -30","description":"Read memory routes"},"result":{"tool_use_id":"toolu_016556","type":"tool_result","content":"37:\tmux.HandleFunc(\"POST /api/memories\", h.create)\n"}}},"turnId":"157f87cf"}}}`
	capturedFileChangeUpdate = `{"type":"thread.activity-appended","payload":{"threadId":"th-1","activity":{"id":"ede7caad","createdAt":"2026-09-18T09:48:40.518Z","tone":"tool","kind":"tool.updated","summary":"File change","payload":{"itemType":"file_change","toolCallId":"toolu_019Tp4","status":"inProgress","title":"File change","detail":"Edit: {\"file_path\":\"/workspace/nexul/.golangci.yml\",\"old_string\":\"  exclusions:\\n    paths:\\n      - internal/platform/storage/sqlcgen\\n\\nformatters:\\n  enable:\\n    - gofm...","data":{"toolName":"Edit"}},"turnId":"b53b23b9"}}}`
	capturedFileChangeDone   = `{"type":"thread.activity-appended","payload":{"threadId":"th-1","activity":{"id":"54f4c3ca","createdAt":"2026-09-18T09:48:40.523Z","tone":"tool","kind":"tool.completed","summary":"File change","payload":{"itemType":"file_change","toolCallId":"toolu_019Tp4","status":"completed","title":"File change","detail":"Edit: {\"file_path\":\"/workspace/nexul/.golangci.yml\",\"old_string\":\"  exclusions:...","data":{"toolName":"Edit","input":{"file_path":"/workspace/nexul/.golangci.yml","old_string":"  exclusions:\n    paths:\n","new_string":"  exclusions:\n    paths:\n      - node_modules\n"},"result":{"tool_use_id":"toolu_019Tp4","type":"tool_result","content":"The file /workspace/nexul/.golangci.yml has been updated successfully."}}},"turnId":"b53b23b9"}}}`
	capturedImageView        = `{"type":"thread.activity-appended","payload":{"threadId":"th-1","activity":{"id":"bdd91722","createdAt":"2026-09-17T18:43:58.967Z","tone":"tool","kind":"tool.updated","summary":"Image view","payload":{"itemType":"image_view","toolCallId":"toolu_013kfA","status":"inProgress","title":"Image view","detail":"/workspace/docs/architecture.png","data":{"imagePath":"/workspace/docs/architecture.png","toolName":"Read"}},"turnId":"024a1ded"}}}`
	capturedFileChangeMCP    = `{"type":"thread.activity-appended","payload":{"threadId":"th-1","activity":{"id":"e12bcfd2","createdAt":"2026-09-18T09:49:37.566Z","tone":"tool","kind":"tool.started","summary":"File change started","payload":{"itemType":"file_change","toolCallId":"toolu_01DCSg","status":"inProgress","title":"File change","detail":"mcp__github__create_pull_request: {}","data":{"toolName":"mcp__github__create_pull_request","input":{}}},"turnId":"b53b23b9"}}}`
)

// capturedUserInput is the info activity T3 emits beside the AskUserQuestion call; its requestId is what an answer names.
const capturedUserInput = `{"type":"thread.activity-appended","payload":{"threadId":"th-1","activity":{"id":"cd7ac269","createdAt":"2026-09-18T09:29:24.732Z","tone":"info","kind":"user-input.requested","summary":"User input requested","payload":{"requestId":"f33d416d-24b9","questions":[{"id":"How do you want to proceed?","header":"Nexul MCP down","question":"How do you want to proceed?","options":[{"label":"Proceed without Nexul tools","description":"Skip the ticket links."},{"label":"Stop and wait","description":"Do nothing yet."}],"multiSelect":false}]},"turnId":"0f302641"}}}`

func TestEventUpdate_UserInputRequested_MapsToAQuestion(t *testing.T) {
	c := &Client{log: slog.Default()}
	var w turnWatch
	update, terminal := c.eventUpdate(json.RawMessage(capturedUserInput), &w)
	require.Nil(t, terminal, "a question is not terminal: the turn stays open until it is answered")
	require.NotNil(t, update)
	require.NotNil(t, update.Question)
	assert.Equal(t, harness.Question{RequestID: "f33d416d-24b9", Questions: []harness.QuestionItem{{
		ID: "How do you want to proceed?", Text: "How do you want to proceed?", Header: "Nexul MCP down",
		Options: []harness.QuestionOption{{Label: "Proceed without Nexul tools", Description: "Skip the ticket links."}, {Label: "Stop and wait", Description: "Do nothing yet."}},
	}}}, *update.Question)
}

func TestEventUpdate_UserInputRequested_WithoutRequestID_IsSkipped(t *testing.T) {
	c := &Client{log: slog.Default()}
	var w turnWatch
	frame := strings.Replace(capturedUserInput, `"requestId":"f33d416d-24b9",`, "", 1)
	update, terminal := c.eventUpdate(json.RawMessage(frame), &w)
	assert.Nil(t, terminal)
	assert.Nil(t, update, "a question nobody can answer is not surfaced")
}

func activityOf(t *testing.T, frame string) *harness.Activity {
	t.Helper()
	c := &Client{log: slog.Default()}
	var w turnWatch
	update, terminal := c.eventUpdate(json.RawMessage(frame), &w)
	require.Nil(t, terminal)
	require.NotNil(t, update)
	require.NotNil(t, update.Activity)
	return update.Activity
}

func TestEventUpdate_ToolStarted_MapsToAToolCallWithT3sLabel(t *testing.T) {
	a := activityOf(t, capturedToolStarted)
	assert.Equal(t, harness.ActivityToolCall, a.Kind)
	assert.Equal(t, "toolu_016Pxx", a.CallID)
	assert.Equal(t, "Bash", a.Tool)
	assert.Equal(t, "Command run started", a.Summary, "an empty input falls back to T3's phrase")
	assert.Equal(t, `{"input":{}}`, a.Detail)
	assert.Equal(t, time.Date(2026, 9, 18, 10, 1, 2, 195_000_000, time.UTC), a.At)
}

func TestEventUpdate_ToolUpdated_PreviewsTheArguments(t *testing.T) {
	a := activityOf(t, capturedToolUpdated)
	assert.Equal(t, harness.ActivityToolCall, a.Kind)
	assert.Equal(t, "toolu_016Pxx", a.CallID, "the same call id as the started frame, so the trail replaces the step")
	assert.Equal(t, `{"command":"go test ./...","description":"Run the tests"}`, a.Summary)
	assert.Equal(t, `{"input":{"command":"go test ./...","description":"Run the tests"}}`, a.Detail)
}

func TestEventUpdate_ToolCompleted_CarriesInputAndResult(t *testing.T) {
	a := activityOf(t, capturedToolDone)
	assert.Equal(t, harness.ActivityToolResult, a.Kind)
	assert.Equal(t, "Read", a.Tool)
	assert.Equal(t, `{"file_path":"/workspace/nexul/.golangci.yml"}`, a.Summary)
	var detail struct {
		Input  map[string]any `json:"input"`
		Result map[string]any `json:"result"`
	}
	require.NoError(t, json.Unmarshal([]byte(a.Detail), &detail))
	assert.Equal(t, "/workspace/nexul/.golangci.yml", detail.Input["file_path"])
	assert.Equal(t, "1\tversion: \"2\"\n", detail.Result["content"])
}

func TestEventUpdate_AskUserQuestion_IsAQuestionAndAFailedStatusSuffixesTheLabel(t *testing.T) {
	a := activityOf(t, capturedQuestion)
	assert.Equal(t, harness.ActivityQuestion, a.Kind)
	assert.Equal(t, "AskUserQuestion", a.Tool)
	assert.True(t, strings.HasSuffix(a.Summary, " · failed"), a.Summary)
	assert.Contains(t, a.Detail, `"header":"Nexul MCP down"`)
}

func TestEventUpdate_OversizedResult_IsCappedAndRunesSurvive(t *testing.T) {
	big := strings.Repeat("é", harness.MaxActivityDetail)
	frame := strings.Replace(capturedToolDone, `1\tversion: \"2\"\n`, big, 1)
	a := activityOf(t, frame)
	assert.LessOrEqual(t, len(a.Detail), harness.MaxActivityDetail)
	assert.True(t, strings.HasPrefix(a.Detail, `{"input":`))
}

func TestEventUpdate_CommandExecution_LabelIsTheCommand(t *testing.T) {
	updated := activityOf(t, capturedCommandUpdated)
	assert.Equal(t, harness.ActivityToolCall, updated.Kind)
	assert.Equal(t, "Bash", updated.Tool)
	assert.Equal(t, "cd /workspace/nexul && rg -n 'HandleFunc' internal/memories/handler.go | head -30", updated.Summary, "an in-progress command rides in data.command")

	done := activityOf(t, capturedCommandDone)
	assert.Equal(t, harness.ActivityToolResult, done.Kind)
	assert.Equal(t, "toolu_016556", done.CallID, "the same call id, so the trail replaces the row")
	assert.Equal(t, updated.Summary, done.Summary, "a finished command reads its input.command")
	assert.Contains(t, done.Detail, `"content":"37:\tmux.HandleFunc(`)
}

func TestEventUpdate_FileChange_LabelIsThePath(t *testing.T) {
	updated := activityOf(t, capturedFileChangeUpdate)
	assert.Equal(t, "Edit", updated.Tool)
	assert.Equal(t, "/workspace/nexul/.golangci.yml", updated.Summary, "in progress there is no input; the path comes from T3's cut-short label")
	assert.Empty(t, updated.Detail, "no input and no result means nothing to expand")

	done := activityOf(t, capturedFileChangeDone)
	assert.Equal(t, harness.ActivityToolResult, done.Kind)
	assert.Equal(t, "/workspace/nexul/.golangci.yml", done.Summary)
	assert.Contains(t, done.Detail, `"new_string"`)

	mcp := activityOf(t, capturedFileChangeMCP)
	assert.Equal(t, "mcp__github__create_pull_request", mcp.Tool)
	assert.Equal(t, "File change started", mcp.Summary, "a tool T3 files under file_change without a path keeps the generic fallback")
}

func TestEventUpdate_ImageView_LabelIsTheImagePath(t *testing.T) {
	a := activityOf(t, capturedImageView)
	assert.Equal(t, "Read", a.Tool)
	assert.Equal(t, "/workspace/docs/architecture.png", a.Summary)
}

func TestQuotedField(t *testing.T) {
	tests := []struct {
		name, label, want string
	}{
		{"cut short after the key", `Edit: {"file_path":"/a/b.go","old_string":"x...`, "/a/b.go"},
		{"missing key", `Bash: {"command":"ls"}`, ""},
		{"cut inside the value", `Edit: {"file_path":"/a/b`, ""},
		{"empty", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, quotedField(tt.label, "file_path"))
		})
	}
}
