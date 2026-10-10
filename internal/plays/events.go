package plays

import (
	"time"

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

	TopicAutoPlayCreated = "auto_play.created"
	TopicAutoPlayUpdated = "auto_play.updated"
	TopicAutoPlayDeleted = "auto_play.deleted"

	TopicAutoPlayLimitsUpdated = "auto_play.limits_updated"
	TopicQueued                = "play.queued"
	TopicQueueUpdated          = "play.queue_updated"
	TopicQueueResumed          = "play.queue_resumed"
)

// TopicPlayRun is the live-hub topic for trail state and activity; ephemeral, never persisted or catalogued.
const TopicPlayRun = "play.run"

// Topics returns every topic the plays domain publishes.
func Topics() []eventbus.Topic {
	return []eventbus.Topic{
		{Name: TopicCreated, Payload: CreatedEvent{}},
		{Name: TopicUpdated, Payload: UpdatedEvent{}},
		{Name: TopicDeleted, Payload: DeletedEvent{}},
		{Name: TopicRunStarted, Payload: RunStartedEvent{}},
		{Name: TopicRunWaiting, Payload: RunWaitingEvent{}},
		{Name: TopicRunFinished, Payload: RunFinishedEvent{}},
		{Name: TopicAutoPlayCreated, Payload: AutoPlayEvent{}},
		{Name: TopicAutoPlayUpdated, Payload: AutoPlayEvent{}},
		{Name: TopicAutoPlayDeleted, Payload: AutoPlayDeletedEvent{}},
		{Name: TopicAutoPlayLimitsUpdated, Payload: AutoPlayLimitsEvent{}},
		{Name: TopicQueued, Payload: QueueItem{}, Description: "An auto play's moment matched a ticket or doc: the run waits in its person's queue, or did not run when there was nobody to run it on or they may not run the play."},
		{Name: TopicQueueUpdated, Payload: QueueItem{}, Description: "A queued auto run changed: it waits for another reason, started, was skipped because it no longer matched, did not run, or was cancelled."},
		{Name: TopicQueueResumed, Payload: QueueResumedEvent{}, Description: "Someone resumed auto plays on a ticket or doc the daily cap had paused; its count of automatic runs starts again."},
	}
}

// AutoPlayEvent is the payload for auto_play.created and auto_play.updated.
type AutoPlayEvent struct {
	AutoPlay AutoPlay `json:"auto_play"`
}

// AutoPlayDeletedEvent is the auto_play.deleted payload; a play's delete takes its auto plays without one each.
type AutoPlayDeletedEvent struct {
	ID          string `json:"id"`
	PlayID      string `json:"play_id"`
	WorkspaceID string `json:"workspace_id"`
}

// AutoPlayLimitsEvent is the auto_play.limits_updated payload: a workspace's caps on automatic runs after a change.
type AutoPlayLimitsEvent struct {
	WorkspaceID       string `json:"workspace_id"`
	DailyCapPerTicket int    `json:"daily_cap_per_ticket" jsonschema:"How many automatic runs auto plays may start on one ticket per rolling day."`
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
	ID          string `json:"id"`
	Label       string `json:"label"`
	WorkspaceID string `json:"workspace_id"`
}

// RunRef identifies one run in the run events: the trail, the play, the target, and who pressed it.
type RunRef struct {
	TrailID     string     `json:"trail_id"`
	PlayID      string     `json:"play_id"`
	PlayLabel   string     `json:"play_label"`
	TargetType  TargetType `json:"target_type" enum:"ticket,doc,interview"`
	TargetID    string     `json:"target_id"`
	TargetTitle string     `json:"target_title"`
	StarterID   string     `json:"starter_id"`
	Via         Via        `json:"via" enum:"web,mcp"`
	WorkspaceID string     `json:"workspace_id,omitempty"`
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
	Outcome        TrailState `json:"outcome" enum:"done,failed,interrupted"`
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
	// ProjectID and WorkspaceID place the run, so a client refetches only that project's and workspace's views.
	ProjectID   string `json:"project_id"`
	WorkspaceID string `json:"workspace_id"`
}

func runRef(t *Trail, targetTitle string) RunRef {
	return RunRef{
		TrailID: t.ID, PlayID: t.PlayID, PlayLabel: t.PlayLabel, TargetType: t.TargetType, TargetID: t.TargetID,
		TargetTitle: targetTitle, StarterID: t.StarterID, Via: t.Via, WorkspaceID: t.WorkspaceID,
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
		ProjectID: t.ProjectID, WorkspaceID: t.WorkspaceID,
	}
}
