package plays

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
)

func TestRuleHolds(t *testing.T) {
	ticket := Facts{
		ProjectID: "p-1", Type: "Bug", Stage: StageProgress, Status: "st-1", Labels: []string{"ui", "urgent"},
		Developer: "u-1", SourceDoc: true,
	}
	tests := []struct {
		name string
		rule Rule
		want bool
	}{
		{"type is, without case", rule(FieldType, OpIs, "bug"), true},
		{"type is another", rule(FieldType, OpIs, "task", "feature"), false},
		{"type is not", rule(FieldType, OpIsNot, "task"), true},
		{"a label among several", rule(FieldLabel, OpIs, "URGENT"), true},
		{"no such label", rule(FieldLabel, OpIsNot, "backend"), true},
		{"labels set", rule(FieldLabel, OpSet), true},
		{"category unset", rule(FieldCategory, OpUnset), true},
		{"category is, when it has none", rule(FieldCategory, OpIs, "web"), false},
		{"tester unset", rule(FieldTester, OpUnset), true},
		{"developer is", rule(FieldDeveloper, OpIs, "u-1"), true},
		{"stage is", rule(FieldStage, OpIs, "progress"), true},
		{"source doc set", rule(FieldSourceDoc, OpSet), true},
		{"linked pr set, without one", rule(FieldLinkedPR, OpSet), false},
		{"not blocked", rule(FieldBlocked, OpUnset), true},
		{"a field nothing reads", rule("colour", OpSet), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.rule.holds(ticket))
		})
	}
}

func TestConditionsHold(t *testing.T) {
	bug := rule(FieldType, OpIs, "bug")
	task := rule(FieldType, OpIs, "task")
	facts := Facts{Type: "bug"}
	tests := []struct {
		name string
		c    Conditions
		want bool
	}{
		{"no groups fires on every match", Conditions{Match: MatchAll}, true},
		{"all of one group that holds", Conditions{Match: MatchAll, Groups: []Group{when(bug)}}, true},
		{"all of a group with a rule that fails", Conditions{Match: MatchAll, Groups: []Group{when(bug, task)}}, false},
		{"any of a group with one rule that holds", Conditions{Match: MatchAll, Groups: []Group{{Match: MatchAny, Rules: []Rule{task, bug}}}}, true},
		{"all groups, one failing", Conditions{Match: MatchAll, Groups: []Group{when(bug), when(task)}}, false},
		{"any group, one holding", Conditions{Match: MatchAny, Groups: []Group{when(task), when(bug)}}, true},
		{"any group, none holding", Conditions{Match: MatchAny, Groups: []Group{when(task)}}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.c.holds(facts))
		})
	}
}

func TestPriorityLevel_FirstRuleThatHoldsElseOtherwise(t *testing.T) {
	p := Priority{Otherwise: LevelLow, Rules: []PriorityRule{
		{Level: LevelHigh, When: when(rule(FieldType, OpIs, "bug"))},
		{Level: LevelNormal, When: when(rule(FieldLabel, OpSet))},
	}}
	assert.Equal(t, LevelHigh, p.level(Facts{Type: "bug", Labels: []string{"x"}}), "the first holding rule wins")
	assert.Equal(t, LevelNormal, p.level(Facts{Type: "task", Labels: []string{"x"}}))
	assert.Equal(t, LevelLow, p.level(Facts{Type: "task"}))
}

func TestAutoPlayPerson(t *testing.T) {
	facts := Facts{Developer: "u-dev", Tester: "u-test"}
	tests := []struct {
		name  string
		runOn RunOn
		m     moment
		want  string
	}{
		{"developer", RunOnDeveloper, moment{causerID: "u-mover"}, "u-dev"},
		{"tester", RunOnTester, moment{causerID: "u-mover"}, "u-test"},
		{"whoever caused it", RunOnCauser, moment{causerID: "u-mover"}, "u-mover"},
		{"an automation's moment runs on the developer", RunOnCauser, moment{automation: true}, "u-dev"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &AutoPlay{RunOn: tt.runOn}
			assert.Equal(t, tt.want, a.person(facts, tt.m))
		})
	}
}

func TestMomentOf(t *testing.T) {
	f := newDecisionsFixture()
	tests := []struct {
		name    string
		topic   string
		payload string
		want    moment
		ok      bool
	}{
		{"unblocked by an agent", "ticket.unblocked", `{"ticket_id":"t-1","project_id":"p-1","actor":{"kind":"user:mcp","user_id":"u-1"}}`,
			moment{name: MomentTicketUnblocked, targetType: TargetTicket, targetID: "t-1", projectID: "p-1", causerID: "u-1", via: ViaMCP}, true},
		{"created by an automation", "ticket.created", `{"ticket":{"id":"t-1","project_id":"p-1","reporter":{"kind":"automation"}}}`,
			moment{name: MomentTicketCreated, targetType: TargetTicket, targetID: "t-1", projectID: "p-1", automation: true, via: ViaWeb}, true},
		{"created by a person, named by login", "ticket.created", `{"ticket":{"id":"t-1","project_id":"p-1","reporter":{"kind":"user","login":"login-u-2"}}}`,
			moment{name: MomentTicketCreated, targetType: TargetTicket, targetID: "t-1", projectID: "p-1", causerID: "u-2", via: ViaWeb}, true},
		{"entered done", "ticket.status_changed", `{"ticket":{"id":"t-1","project_id":"p-1"},"from":"st-progress","to":"st-done","actor":{"kind":"user","user_id":"u-1"}}`,
			moment{name: MomentTicketEnteredStage, stage: StageDone, targetType: TargetTicket, targetID: "t-1", projectID: "p-1", causerID: "u-1", via: ViaWeb}, true},
		{"between two done columns", "ticket.status_changed", `{"ticket":{"id":"t-1"},"from":"st-done","to":"st-shipped"}`, moment{}, false},
		{"into a deleted column", "ticket.status_changed", `{"ticket":{"id":"t-1"},"from":"st-done","to":"st-gone"}`, moment{}, false},
		{"developer cleared", "ticket.developer_changed", `{"ticket":{"id":"t-1"},"from":"login-u-1","to":""}`, moment{}, false},
		{"tester set", "ticket.tester_changed", `{"ticket":{"id":"t-1","project_id":"p-1"},"to":"login-u-3","actor":{"kind":"user","user_id":"u-1"}}`,
			moment{name: MomentTicketTesterSet, targetType: TargetTicket, targetID: "t-1", projectID: "p-1", causerID: "u-1", via: ViaWeb}, true},
		{"test failed by its tester", "ticket.test_failed", `{"ticket":{"id":"t-1","project_id":"p-1"},"tester":"login-u-3"}`,
			moment{name: MomentTicketTestFailed, targetType: TargetTicket, targetID: "t-1", projectID: "p-1", causerID: "u-3", via: ViaWeb}, true},
		{"doc settled as created", "doc.settled", `{"doc":{"id":"d-1","project_id":"p-1"},"first":true,"actor_id":"u-1"}`,
			moment{name: MomentDocCreated, targetType: TargetDoc, targetID: "d-1", projectID: "p-1", causerID: "u-1", via: ViaWeb}, true},
		{"doc settled as changed", "doc.settled", `{"doc":{"id":"d-1","project_id":"p-1"},"first":false,"actor_id":"u-1"}`,
			moment{name: MomentDocChanged, targetType: TargetDoc, targetID: "d-1", projectID: "p-1", causerID: "u-1", via: ViaWeb}, true},
		{"a topic carrying no moment", "ticket.updated", `{"ticket":{"id":"t-1"}}`, moment{}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var p momentPayload
			require.NoError(t, json.Unmarshal([]byte(tt.payload), &p))
			got, ok, err := f.runner.momentOf(context.Background(), tt.topic, p)
			require.NoError(t, err)
			require.Equal(t, tt.ok, ok)
			if tt.ok {
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestMomentOf_LookupFailuresRetry(t *testing.T) {
	f := newDecisionsFixture()
	var p momentPayload
	require.NoError(t, json.Unmarshal([]byte(`{"ticket":{"id":"t-1"},"tester":"broken"}`), &p))
	_, _, err := f.runner.momentOf(context.Background(), "ticket.test_failed", p)
	require.Error(t, err)
}

func TestHandleAutoPlayMoment_ErrorPaths(t *testing.T) {
	f := newDecisionsFixture()
	f.runner.queue, f.runner.facts = nil, nil
	ev := eventbus.Event{ID: "ev-1", Topic: "ticket.unblocked", Payload: json.RawMessage(`{"ticket_id":"t-1","project_id":"proj-1"}`)}
	require.NoError(t, f.runner.HandleAutoPlayMoment(context.Background(), ev), "a runner without a queue matches nothing")

	f.runner.queue, f.runner.facts = nopQueue{}, &fakeFacts{byID: map[string]Facts{}}
	bad := eventbus.Event{ID: "ev-2", Topic: "ticket.unblocked", Payload: json.RawMessage(`{`)}
	require.ErrorIs(t, f.runner.HandleAutoPlayMoment(context.Background(), bad), apperrs.ErrFatal)

	f.plays.autoPlays = []*AutoPlay{{ID: "ap-1", WorkspaceID: workspaceID, Moment: MomentTicketUnblocked, Enabled: true}}
	require.NoError(t, f.runner.HandleAutoPlayMoment(context.Background(), ev), "a ticket deleted since matches nothing")

	f.plays.autoPlayErr = errors.New("disk gone")
	require.ErrorIs(t, f.runner.HandleAutoPlayMoment(context.Background(), ev), apperrs.ErrRetryable)
}

func TestMomentGone(t *testing.T) {
	review := StageReview
	tests := []struct {
		name string
		a    AutoPlay
		f    Facts
		want string
	}{
		{"still unblocked", AutoPlay{Moment: MomentTicketUnblocked}, Facts{}, ""},
		{"blocked again", AutoPlay{Moment: MomentTicketUnblocked}, Facts{Blocked: true}, "no longer unblocked"},
		{"still in the stage", AutoPlay{Moment: MomentTicketEnteredStage, MomentStage: &review}, Facts{Stage: StageReview}, ""},
		{"moved on", AutoPlay{Moment: MomentTicketEnteredStage, MomentStage: &review}, Facts{Stage: StageDone}, "no longer in review"},
		{"developer cleared", AutoPlay{Moment: MomentTicketDeveloperSet}, Facts{}, "no longer has a developer"},
		{"tester cleared", AutoPlay{Moment: MomentTicketTesterSet}, Facts{}, "no longer has a tester"},
		{"a test failure stays a failure", AutoPlay{Moment: MomentTicketTestFailed}, Facts{}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, momentGone(&tt.a, tt.f))
		})
	}
}

// nopQueue is a QueueRepo with nothing in it, for paths that never reach the queue.
type nopQueue struct{ QueueRepo }
