package plays

import (
	"context"
	"encoding/json"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/otal-labs/nexul/internal/harness"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// FoundMemory is a memory a run read with memory_get: by its title now, or once deleted by the one it had then.
type FoundMemory struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Deleted bool   `json:"deleted"`
}

// memoryGetTool is Nexul's tool a run reads a memory with; harnesses prefix it (mcp__nexul__, nexul_, "nexul · ").
const memoryGetTool = "memory_get"

// failedStep is how a harness marks a failed step's summary (harness.Activity).
const failedStep = " · failed"

// MemoryReadOf is the memory a finished memory_get step read; false for any other step, a failed call, or one naming no id.
func MemoryReadOf(e ActivityEntry) (string, bool) {
	if e.Kind != harness.ActivityToolResult || strings.HasSuffix(e.Summary, failedStep) || !isMemoryGet(e.Tool) {
		return "", false
	}
	id := inputID(e.Detail)
	return id, id != ""
}

// isMemoryGet matches the tool by its last name, so another tool that merely ends in the same letters does not.
func isMemoryGet(tool string) bool {
	rest, ok := strings.CutSuffix(tool, memoryGetTool)
	if !ok || rest == "" {
		return ok
	}
	r, _ := utf8.DecodeLastRuneInString(rest)
	return !unicode.IsLetter(r) && !unicode.IsDigit(r)
}

// inputID reads the id argument from a step's detail, {"input": ...} first, even when the cap cut the result after it.
func inputID(detail string) string {
	rest, ok := strings.CutPrefix(detail, `{"input":`)
	if !ok {
		return ""
	}
	var raw json.RawMessage
	if err := json.NewDecoder(strings.NewReader(rest)).Decode(&raw); err != nil {
		return ""
	}
	// Some providers send the arguments as a JSON string holding the object.
	var text string
	if json.Unmarshal(raw, &text) == nil {
		raw = json.RawMessage(text)
	}
	var in struct {
		ID string `json:"id"`
	}
	if json.Unmarshal(raw, &in) != nil {
		return ""
	}
	return strings.TrimSpace(in.ID)
}

// recordMemoryRead notes a memory the step read, once per memory per run; a failed write only costs the run's view the entry.
func (o *trailObserver) recordMemoryRead(e ActivityEntry) {
	id, ok := MemoryReadOf(e)
	if !ok || o.trail.ProjectID == "" || o.read[id] {
		return
	}
	if err := o.r.trails.RecordMemoryRead(o.ctx, o.trail.ID, o.trail.ProjectID, id, e.At); err != nil {
		o.r.log.Warn("plays: record memory read failed", "trail", o.trail.ID, "memory", id, "error", err)
		return
	}
	if o.read == nil {
		o.read = map[string]bool{}
	}
	o.read[id] = true
}

// withFoundMemories fills the memories the run read that it was not told to read, for a caller who may read the
// project's memories; anyone else gets none.
func (r *Runner) withFoundMemories(ctx context.Context, t *Trail) error {
	if t.ProjectID == "" || !r.perm.HasPermission(ctx, actorID(ctx), t.WorkspaceID, permissions.MemoriesRead, resourceTypeProject, t.ProjectID) {
		return nil
	}
	read, err := r.trails.ListMemoryReads(ctx, t.ID)
	if err != nil {
		return err
	}
	t.FoundMemories = slices.DeleteFunc(read, func(m FoundMemory) bool { return slices.Contains(t.SelectedMemoryIDs, m.ID) })
	return nil
}
