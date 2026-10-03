package t3clientv2

import (
	"encoding/json"
	"slices"
	"strings"
	"time"
	"unicode"

	"github.com/otal-labs/nexul/internal/harness"
)

// turnItem is the slice of a T3 turn item Nexul reads; T3 strips command output, diffs and tool results from the wire.
type turnItem struct {
	ID        string   `json:"id"`
	RunID     string   `json:"runId"`
	NodeID    string   `json:"nodeId"`
	Ordinal   int      `json:"ordinal"`
	Status    string   `json:"status"`
	Type      string   `json:"type"`
	Title     string   `json:"title"`
	UpdatedAt string   `json:"updatedAt"`
	MessageID string   `json:"messageId"`
	Text      string   `json:"text"`
	Streaming bool     `json:"streaming"`
	Failure   *failure `json:"failure"`
	// Input is a command's text as a JSON string, or a dynamic tool's arguments as any JSON.
	Input                  json.RawMessage `json:"input"`
	Output                 json.RawMessage `json:"output"`
	OutputIndicatesFailure bool            `json:"outputIndicatesFailure"`
	ExitCode               *int            `json:"exitCode"`
	FileName               string          `json:"fileName"`
	Pattern                string          `json:"pattern"`
	Patterns               []string        `json:"patterns"`
	ToolName               string          `json:"toolName"`
	ViewedImagePath        string          `json:"viewedImagePath"`
	RequestID              string          `json:"requestId"`
	RequestKind            string          `json:"requestKind"`
	Prompt                 string          `json:"prompt"`
	Questions              []userQuestion  `json:"questions"`
}

type userQuestion struct {
	ID          string                   `json:"id"`
	Header      string                   `json:"header"`
	Question    string                   `json:"question"`
	Options     []harness.QuestionOption `json:"options"`
	MultiSelect bool                     `json:"multiSelect"`
}

const (
	summaryRunes = 160
	failedSuffix = " · failed"
	// questionTool is Claude's tool for asking; its row is a question step, while user_input_request carries the question.
	questionTool = "AskUserQuestion"
)

// builtinTools names the item types T3 sends without a tool name, as the trail already labels them.
var builtinTools = map[string]string{
	"command_execution": "Shell",
	"file_change":       "Edit",
	"file_search":       "Grep",
	"web_search":        "WebSearch",
	"subagent":          "Agent",
}

// openItems are the statuses of an item still in flight, or a request still waiting for its answer.
var openItems = []string{"pending", "running", "waiting"}

// mapItem is a turn item as a step, an open question or an open approval; it filters no run, so child threads map too.
func mapItem(it turnItem) (harness.Update, bool) {
	open := slices.Contains(openItems, it.Status)
	switch it.Type {
	case "user_input_request":
		return harness.Update{Question: question(it)}, open
	case "approval_request":
		summary := it.Prompt
		if summary == "" {
			summary = it.Title
		}
		return harness.Update{Approval: &harness.Approval{Kind: it.RequestKind, Summary: harness.Preview(summary, summaryRunes)}}, open
	case "dynamic_tool":
		return harness.Update{Activity: step(it, toolName(it.ToolName), open)}, true
	}
	tool, ok := builtinTools[it.Type]
	if !ok {
		return harness.Update{}, false
	}
	return harness.Update{Activity: step(it, tool, open)}, true
}

func step(it turnItem, tool string, open bool) *harness.Activity {
	kind := harness.ActivityToolResult
	if open {
		kind = harness.ActivityToolCall
	}
	if tool == questionTool {
		kind = harness.ActivityQuestion
	}
	pattern := it.Pattern
	if pattern == "" && len(it.Patterns) > 0 {
		pattern = it.Patterns[0]
	}
	summary := harness.Preview(firstNonEmpty(it.FileName, pattern, it.ViewedImagePath, inputSummary(it.Input), it.Title, it.Prompt), summaryRunes)
	if failed(it) {
		summary += failedSuffix
	}
	detail := ""
	if s := string(it.Input); s != "" && s != "null" {
		detail = harness.CapDetail(`{"input":` + s + `}`)
	}
	return &harness.Activity{Kind: kind, CallID: it.ID, Tool: tool, Summary: summary, Detail: detail, At: itemTime(it.UpdatedAt)}
}

// toolName reads the server.tool MCP name Codex and ACP providers send as "server · tool"; an ACP title has spaces and stays.
func toolName(name string) string {
	server, tool, ok := strings.Cut(name, ".")
	if !ok || strings.ContainsFunc(name, unicode.IsSpace) {
		return name
	}
	return server + " · " + tool
}

// inputSummary is a command's text, or any other input as its JSON; "" for none.
func inputSummary(raw json.RawMessage) string {
	var text string
	if json.Unmarshal(raw, &text) == nil {
		return text
	}
	if s := string(raw); s != "null" && s != "{}" {
		return s
	}
	return ""
}

// failed is a step T3 marked failed, whose output looked failed, whose command exited nonzero, or whose result is an error.
func failed(it turnItem) bool {
	if it.Status == "failed" || it.OutputIndicatesFailure || (it.ExitCode != nil && *it.ExitCode != 0) {
		return true
	}
	var out struct {
		IsError bool `json:"isError"`
	}
	return json.Unmarshal(it.Output, &out) == nil && out.IsError
}

func question(it turnItem) *harness.Question {
	q := &harness.Question{RequestID: it.RequestID, Questions: make([]harness.QuestionItem, 0, len(it.Questions))}
	for _, item := range it.Questions {
		q.Questions = append(q.Questions, harness.QuestionItem{ID: item.ID, Text: item.Question, Header: item.Header, Options: item.Options, MultiSelect: item.MultiSelect})
	}
	return q
}

// itemTime reads a turn item's updatedAt; T3 always sends one, so a zero time only marks a malformed item.
func itemTime(iso string) time.Time {
	t, err := time.Parse(time.RFC3339Nano, iso)
	if err != nil {
		return time.Time{}
	}
	return t.UTC()
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
