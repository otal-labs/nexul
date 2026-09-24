// Package plays implements the Play entity: a workspace-scoped, user-fired Agent turn definition (ADR 0055).
package plays

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/otal-labs/nexul/internal/harness"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// Type says what a play is fired from: a ticket, a doc, or a project's interview.
type Type string

const (
	TypeTicket    Type = "ticket"
	TypeDoc       Type = "doc"
	TypeInterview Type = "interview"
)

func (t Type) valid() bool {
	return t == TypeTicket || t == TypeDoc || t == TypeInterview
}

// Stage mirrors the five fixed board stages (ADR 0022); plays never import internal/workspace for it (ADR 0017).
type Stage string

const (
	StageBacklog  Stage = "backlog"
	StageProgress Stage = "progress"
	StageReview   Stage = "review"
	StageTesting  Stage = "testing"
	StageDone     Stage = "done"
)

var stages = []Stage{StageBacklog, StageProgress, StageReview, StageTesting, StageDone}

func (s Stage) valid() bool {
	for _, known := range stages {
		if s == known {
			return true
		}
	}
	return false
}

// Play is one workspace's pre-configured Agent turn, fired by a person from a ticket or a doc (ADR 0055).
type Play struct {
	ID                 string    `json:"id"`
	WorkspaceID        string    `json:"workspace_id"`
	Label              string    `json:"label"`
	Type               Type      `json:"type"`
	Description        string    `json:"description"`
	Instructions       string    `json:"instructions"`
	Enabled            bool      `json:"enabled"`
	ShowWhenStage      *Stage    `json:"show_when_stage"`
	ExcludedProjectIDs []string  `json:"excluded_project_ids"`
	CreatedBy          string    `json:"created_by"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

// TargetType names what a play was fired from and pairs with Type; an interview target's id is its project's id.
type TargetType string

const (
	TargetTicket    TargetType = "ticket"
	TargetDoc       TargetType = "doc"
	TargetInterview TargetType = "interview"
)

func (t TargetType) valid() bool {
	return t == TargetTicket || t == TargetDoc || t == TargetInterview
}

// Via records which adapter started a run, the provenance ADR 0049 asks for.
type Via string

const (
	ViaWeb Via = "web"
	ViaMCP Via = "mcp"
)

// TrailState is where a run stands; starting precedes the harness accepting, waiting is a turn stopped on a
// question to the user, the last three are terminal.
type TrailState string

const (
	TrailStarting    TrailState = "starting"
	TrailRunning     TrailState = "running"
	TrailWaiting     TrailState = "waiting"
	TrailDone        TrailState = "done"
	TrailFailed      TrailState = "failed"
	TrailInterrupted TrailState = "interrupted"
)

// Active reports whether the run still occupies its target; one active trail per target is the rule.
func (s TrailState) Active() bool {
	return s == TrailStarting || s == TrailRunning || s == TrailWaiting
}

// TrailQuestion is the question a trail stopped on; the answer stays beside it once given so the transcript shows both.
type TrailQuestion struct {
	harness.Question
	Answer  *harness.QuestionAnswer `json:"answer,omitempty"`
	AskedAt time.Time               `json:"asked_at"`
}

// ActivityNote is a runner's own line on the trail (a skipped move, a stop), not harness activity; the transcript
// shows it as a muted line between turns.
const ActivityNote harness.ActivityKind = "note"

// MaxTrailActivityLines caps the captured harness activity; the oldest steps drop first.
const MaxTrailActivityLines = 300

// ActivityEntry is one transcript step on a trail; its fields mirror harness.Activity so the seam value converts directly.
type ActivityEntry struct {
	Kind    harness.ActivityKind `json:"kind"`
	CallID  string               `json:"call_id,omitempty"`
	Tool    string               `json:"tool,omitempty"`
	Summary string               `json:"summary"`
	Detail  string               `json:"detail,omitempty"`
	At      time.Time            `json:"at"`
}

// DecodeActivity reads the stored activity column; a trail written as plain lines decodes as steps of kind other.
func DecodeActivity(raw []byte) ([]ActivityEntry, error) {
	var entries []ActivityEntry
	if err := json.Unmarshal(raw, &entries); err == nil {
		return entries, nil
	}
	var lines []string
	if err := json.Unmarshal(raw, &lines); err != nil {
		return nil, err
	}
	entries = make([]ActivityEntry, 0, len(lines))
	for _, line := range lines {
		entries = append(entries, ActivityEntry{Kind: harness.ActivityOther, Summary: line})
	}
	return entries, nil
}

// Trail is the persisted record of one play run (ADR 0055): what was pressed, on what, by whom, and how it went.
type Trail struct {
	ID                 string          `json:"id"`
	WorkspaceID        string          `json:"workspace_id"`
	PlayID             string          `json:"play_id"`
	PlayLabel          string          `json:"play_label"`
	TargetType         TargetType      `json:"target_type"`
	TargetID           string          `json:"target_id"`
	ProjectID          string          `json:"project_id"`
	ConversationID     string          `json:"conversation_id"`
	StarterID          string          `json:"starter_id"`
	Via                Via             `json:"via"`
	SelectedMemoryIDs  []string        `json:"selected_memory_ids"`
	CustomInstructions string          `json:"custom_instructions"`
	MoveToStatusID     string          `json:"move_to_status_id"`
	ComputerID         string          `json:"computer_id"`
	Provider           string          `json:"provider"`
	Model              string          `json:"model"`
	HarnessSessionID   string          `json:"harness_session_id"`
	State              TrailState      `json:"state"`
	StartedAt          time.Time       `json:"started_at"`
	EndedAt            *time.Time      `json:"ended_at"`
	LastError          string          `json:"last_error"`
	ReplyMessageID     string          `json:"reply_message_id"`
	Activity           []ActivityEntry `json:"activity"`
	// Question is the latest question the run stopped on, nil for a run that never asked one.
	Question *TrailQuestion `json:"question"`
}

// AppendActivity replaces the step sharing e's call id, else appends, keeping the newest MaxTrailActivityLines.
func (t *Trail) AppendActivity(e ActivityEntry) {
	if e.CallID != "" {
		for i := len(t.Activity) - 1; i >= 0; i-- {
			if t.Activity[i].CallID == e.CallID {
				t.Activity[i] = e
				return
			}
		}
	}
	t.Activity = append(t.Activity, e)
	if len(t.Activity) > MaxTrailActivityLines {
		t.Activity = t.Activity[len(t.Activity)-MaxTrailActivityLines:]
	}
}

// Validate enforces ticket 02's column rule: a ticket play carries exactly one show-when stage, any other play none.
func (p *Play) Validate() error {
	if strings.TrimSpace(p.Label) == "" {
		return fmt.Errorf("%w: label is required", apperrs.ErrInvalid)
	}
	if !p.Type.valid() {
		return fmt.Errorf("%w: type must be ticket, doc, or interview", apperrs.ErrInvalid)
	}
	if p.Type == TypeTicket {
		if p.ShowWhenStage == nil {
			return fmt.Errorf("%w: a ticket play requires exactly one show-when stage", apperrs.ErrInvalid)
		}
		if !p.ShowWhenStage.valid() {
			return fmt.Errorf("%w: show-when stage must be one of backlog, progress, review, testing, done", apperrs.ErrInvalid)
		}
	}
	if p.Type != TypeTicket && p.ShowWhenStage != nil {
		return fmt.Errorf("%w: a %s play cannot have a show-when stage", apperrs.ErrInvalid, p.Type)
	}
	return nil
}
