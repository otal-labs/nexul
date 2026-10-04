package t3clientv2

import (
	"cmp"
	"reflect"
	"slices"
	"strings"

	"github.com/otal-labs/nexul/internal/harness"
)

const (
	maxChildren    = 20
	maxChildSteps  = 200
	maxChildDetail = 2 << 10
	maxPrompt      = 2 << 10
	askingNote     = "A handed-off agent is waiting for an answer in T3 Code"
)

// handoffStates maps a subagent's status to its pill's state; any other status is still running.
var handoffStates = map[string]string{
	"completed":   harness.HandoffDone,
	"idle":        harness.HandoffDone,
	"failed":      harness.HandoffFailed,
	"cancelled":   harness.HandoffInterrupted,
	"interrupted": harness.HandoffInterrupted,
}

// pills are a turn's hand-offs as their pills show them, in the order T3 started them; a pill outlives its row.
type pills struct {
	order []*pill
	// asked holds the requests of handed-off agents already noted on the turn.
	asked map[string]bool
}

// pill is one handed-off agent: its subagent row as last seen, and what its own thread showed.
type pill struct {
	row    subagent
	cursor int64
	// model is the child thread's, for a row that names none.
	model string
	steps []harness.Activity
	// text is the child's newest assistant text, from the item at ordinal textAt.
	text   string
	textAt int
	// ended is the state the turn's end leaves a pill in that is still running.
	ended string
	sent  *harness.Handoff
}

// rows takes the latest rows of the followed runs' subagents, adding a pill for each new one up to maxChildren.
func (ps *pills) rows(rows []subagent) {
	slices.SortFunc(rows, func(a, b subagent) int {
		return cmp.Or(strings.Compare(a.StartedAt, b.StartedAt), strings.Compare(a.ID, b.ID))
	})
	for _, r := range rows {
		p := ps.find(r.ID)
		if p == nil && len(ps.order) >= maxChildren {
			continue
		}
		if p == nil {
			p = &pill{}
			ps.order = append(ps.order, p)
		}
		p.row = r
	}
}

func (ps *pills) find(id string) *pill {
	i := slices.IndexFunc(ps.order, func(p *pill) bool { return p.row.ID == id })
	if i < 0 {
		return nil
	}
	return ps.order[i]
}

// apply folds one item of a handed-off agent's own thread into its pill; it returns the turn's note when the agent asks.
func (ps *pills) apply(id string, item streamItem) []harness.Update {
	p := ps.find(id)
	if p == nil {
		return nil
	}
	if item.Kind == "snapshot" {
		p.cursor = item.SnapshotSequence
		p.model = item.Projection.Thread.ModelSelection.Model
		items := slices.Clone(item.Projection.TurnItems)
		slices.SortStableFunc(items, func(a, b turnItem) int { return a.Ordinal - b.Ordinal })
		var out []harness.Update
		for _, it := range items {
			out = append(out, ps.item(p, it)...)
		}
		return out
	}
	if item.Kind != "event" || item.Sequence <= p.cursor {
		return nil
	}
	p.cursor = item.Sequence
	if item.Event.Type == "turn-item.updated" {
		var it turnItem
		if decode(item.Event, &it) {
			return ps.item(p, it)
		}
	}
	var t appThread
	if strings.HasPrefix(item.Event.Type, "thread.") && decode(item.Event, &t) {
		p.model = t.ModelSelection.Model
	}
	return nil
}

// item maps a child's turn item with the turn's mapper; a question or approval is a step, noted on the turn while open.
func (ps *pills) item(p *pill, it turnItem) []harness.Update {
	if it.Type == "assistant_message" {
		if it.Text != "" && it.Ordinal >= p.textAt {
			p.text, p.textAt = it.Text, it.Ordinal
		}
		return nil
	}
	u, open := mapItem(it)
	step := u.Activity
	asks := u.Question != nil || u.Approval != nil
	if u.Question != nil {
		step = request(it, firstQuestion(u.Question))
	}
	if u.Approval != nil {
		step = request(it, u.Approval.Summary)
	}
	if step == nil {
		return nil
	}
	step.Detail = harness.CapBytes(step.Detail, maxChildDetail)
	p.step(*step)
	if !asks || !open || ps.asked[it.RequestID] {
		return nil
	}
	if ps.asked == nil {
		ps.asked = map[string]bool{}
	}
	ps.asked[it.RequestID] = true
	return []harness.Update{note(askingNote)}
}

func firstQuestion(q *harness.Question) string {
	if len(q.Questions) == 0 {
		return ""
	}
	return q.Questions[0].Text
}

func request(it turnItem, summary string) *harness.Activity {
	return &harness.Activity{Kind: harness.ActivityQuestion, CallID: it.ID, Summary: harness.Preview(summary, summaryRunes), At: itemTime(it.UpdatedAt)}
}

// step replaces the step sharing a's call id, else appends it, keeping the newest maxChildSteps.
func (p *pill) step(a harness.Activity) {
	if i := slices.IndexFunc(p.steps, func(s harness.Activity) bool { return s.CallID == a.CallID }); i >= 0 {
		p.steps[i] = a
		return
	}
	p.steps = append(p.steps, a)
	if len(p.steps) > maxChildSteps {
		p.steps = slices.Delete(p.steps, 0, len(p.steps)-maxChildSteps)
	}
}

func (p *pill) handoff() harness.Handoff {
	prompt := harness.CapBytes(p.row.Prompt, maxPrompt)
	h := harness.Handoff{ID: p.row.ID, Driver: p.row.Driver, Model: p.row.Model, Title: p.row.Title, Prompt: prompt,
		State: p.state(), Reply: p.row.Result, Steps: slices.Clone(p.steps)}
	if h.Model == "" {
		h.Model = p.model
	}
	line, _, _ := strings.Cut(strings.TrimSpace(prompt), "\n")
	h.Title = harness.Preview(cmp.Or(h.Title, line), summaryRunes)
	if h.Reply == "" {
		h.Reply = p.text
	}
	return h
}

func (p *pill) state() string {
	if s, ok := handoffStates[p.row.Status]; ok {
		return s
	}
	if p.ended != "" {
		return p.ended
	}
	return harness.HandoffRunning
}

// changed is a Handoff update for each pill that changed since it was last emitted.
func (ps *pills) changed() []harness.Update {
	var out []harness.Update
	for _, p := range ps.order {
		h := p.handoff()
		if p.sent != nil && reflect.DeepEqual(*p.sent, h) {
			continue
		}
		p.sent = &h
		out = append(out, harness.Update{Handoff: &h})
	}
	return out
}

// end leaves every pill still running in state, the turn's last word on it.
func (ps *pills) end(state string) []harness.Update {
	for _, p := range ps.order {
		p.ended = state
	}
	return ps.changed()
}
