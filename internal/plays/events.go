package plays

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
)

// Topics published by the plays domain through the outbox.
const (
	TopicCreated     = "play.created"
	TopicUpdated     = "play.updated"
	TopicDeleted     = "play.deleted"
	TopicRunStarted  = "play.run_started"
	TopicRunWaiting  = "play.run_waiting"
	TopicRunFinished = "play.run_finished"
)

// TopicPlayRun is the live-hub topic for trail state and activity; ephemeral, never persisted or catalogued.
const TopicPlayRun = "play.run"

// Topics returns every topic the plays domain publishes.
func Topics() []string {
	return []string{TopicCreated, TopicUpdated, TopicDeleted, TopicRunStarted, TopicRunWaiting, TopicRunFinished}
}

// CreatedEvent is the payload for play.created; field names are part of the event contract (ADR 0044).
type CreatedEvent struct {
	Play Play `json:"play"`
}

// UpdatedEvent is the payload for play.updated.
type UpdatedEvent struct {
	Play Play `json:"play"`
}

// DeletedEvent is the play.deleted payload; the play is already gone by publish time, identity only.
type DeletedEvent struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

// RunRef identifies one run in the run events: the trail, the play, the target, and who pressed it.
type RunRef struct {
	TrailID     string     `json:"trail_id"`
	PlayID      string     `json:"play_id"`
	PlayLabel   string     `json:"play_label"`
	TargetType  TargetType `json:"target_type"`
	TargetID    string     `json:"target_id"`
	TargetTitle string     `json:"target_title"`
	StarterID   string     `json:"starter_id"`
	Via         Via        `json:"via"`
}

// RunStartedEvent is the play.run_started payload, written when the harness accepts the turn.
type RunStartedEvent struct {
	RunRef
	HarnessSessionID string `json:"harness_session_id"`
}

// RunWaitingEvent is the play.run_waiting payload, written when the turn stops on a question to the starter.
type RunWaitingEvent struct {
	RunRef
}

// RunFinishedEvent is the play.run_finished payload; Outcome is done, failed, or interrupted.
type RunFinishedEvent struct {
	RunRef
	Outcome        TrailState `json:"outcome"`
	LastError      string     `json:"last_error,omitempty"`
	ReplyMessageID string     `json:"reply_message_id,omitempty"`
}

// RunFrame is TopicPlayRun's payload: the trail's current state plus its latest step.
type RunFrame struct {
	TrailID    string         `json:"trail_id"`
	PlayID     string         `json:"play_id"`
	TargetType TargetType     `json:"target_type"`
	TargetID   string         `json:"target_id"`
	State      TrailState     `json:"state"`
	Activity   *ActivityEntry `json:"activity"`
	Question   *TrailQuestion `json:"question"`
	EndedAt    *time.Time     `json:"ended_at"`
	LastError  string         `json:"last_error"`
}

func runRef(t *Trail, targetTitle string) RunRef {
	return RunRef{
		TrailID: t.ID, PlayID: t.PlayID, PlayLabel: t.PlayLabel, TargetType: t.TargetType, TargetID: t.TargetID,
		TargetTitle: targetTitle, StarterID: t.StarterID, Via: t.Via,
	}
}

func runFrame(t *Trail) RunFrame {
	var activity *ActivityEntry
	if len(t.Activity) > 0 {
		last := t.Activity[len(t.Activity)-1]
		activity = &last
	}
	return RunFrame{
		TrailID: t.ID, PlayID: t.PlayID, TargetType: t.TargetType, TargetID: t.TargetID,
		State: t.State, Activity: activity, Question: t.Question, EndedAt: t.EndedAt, LastError: t.LastError,
	}
}

// HandleTicketStatusChanged is registered on ticket.status_changed: a ticket entering done fires the decisions check.
func (r *Runner) HandleTicketStatusChanged(ctx context.Context, ev eventbus.Event) error {
	var m ticketMove
	if err := json.Unmarshal(ev.Payload, &m); err != nil {
		return apperrs.Fatal(fmt.Errorf("parse ticket.status_changed: %w", err))
	}
	if m.Ticket.ID == "" || m.To == "" {
		return apperrs.Fatal(fmt.Errorf("ticket.status_changed missing ticket id or destination status"))
	}
	if err := r.onTicketMoved(ctx, m); err != nil {
		return apperrs.Retryable(err)
	}
	return nil
}
