package issuesync

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFieldsFromIssue_LeavesOutTheTriggerLabelAndSorts(t *testing.T) {
	f := FieldsFromIssue(Issue{Title: " Crash ", Body: "b", State: "closed", StateReason: "completed", Labels: []string{"ui", "NEXUL", "bug"}, Assignees: []string{"bob"}}, "nexul")
	assert.Equal(t, TicketFields{Title: "Crash", Body: "b", Closed: true, StateReason: "completed", Labels: []string{"bug", "ui"}, Assignees: []string{"bob"}}, f)
}

func TestHash_ChangesWithEachFieldItCovers(t *testing.T) {
	base := TicketFields{Title: "t", Body: "b", Labels: []string{"a", "b"}, Assignees: []string{"bob", "carol"}}
	same := base
	same.Labels = []string{"b", "a"}
	same.Assignees = []string{"bob", "dave"}
	same.StateReason = "reopened"
	assert.Equal(t, Hash(base), Hash(same), "label order, later assignees and the state reason are not part of it")

	for name, change := range map[string]func(*TicketFields){
		"title":    func(f *TicketFields) { f.Title = "u" },
		"body":     func(f *TicketFields) { f.Body = "c" },
		"state":    func(f *TicketFields) { f.Closed = true },
		"labels":   func(f *TicketFields) { f.Labels = []string{"a"} },
		"assignee": func(f *TicketFields) { f.Assignees = nil },
		"boundary": func(f *TicketFields) { f.Title, f.Body = "tb", "" },
	} {
		changed := base
		change(&changed)
		assert.NotEqual(t, Hash(base), Hash(changed), name)
	}
}
