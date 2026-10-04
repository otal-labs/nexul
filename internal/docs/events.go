package docs

import "time"

// Topics published by the docs domain. Consumers: mcp (re-index).
const (
	TopicCreated       = "doc.created"
	TopicUpdated       = "doc.updated"
	TopicDeleted       = "doc.deleted"
	TopicMoved         = "doc.moved"
	TopicFolderCreated = "doc.folder.created"
	TopicFolderUpdated = "doc.folder.updated"
	TopicFolderDeleted = "doc.folder.deleted"
	// TopicWatchersChanged is someone starting or stopping watching a doc; an edit adding its editor rides doc.updated.
	TopicWatchersChanged = "doc.watchers.changed"

	// Clarification topics carry the question and the author, never an answer or the "Anything else?" text.
	TopicClarificationRoundStarted      = "doc.clarification.round_started"
	TopicClarificationRoundPosted       = "doc.clarification.round_posted"
	TopicClarificationRoundEnded        = "doc.clarification.round_ended"
	TopicClarificationRoundAnswered     = "doc.clarification.round_answered"
	TopicClarificationAnswerSaved       = "doc.clarification.answer_saved"
	TopicClarificationAnswerCleared     = "doc.clarification.answer_cleared"
	TopicClarificationAnythingElseSaved = "doc.clarification.anything_else_saved"
	TopicClarificationClosed            = "doc.clarification.closed"
)

// Topics returns every topic the docs domain publishes.
func Topics() []string {
	return []string{TopicCreated, TopicUpdated, TopicDeleted, TopicMoved, TopicFolderCreated, TopicFolderUpdated, TopicFolderDeleted, TopicWatchersChanged,
		TopicClarificationRoundStarted, TopicClarificationRoundPosted, TopicClarificationRoundEnded, TopicClarificationRoundAnswered,
		TopicClarificationAnswerSaved, TopicClarificationAnswerCleared, TopicClarificationAnythingElseSaved, TopicClarificationClosed}
}

// CreatedEvent is the payload for doc.created; field names are part of the event contract (ADR 0044) and additive-only.
type CreatedEvent struct {
	Doc Doc `json:"doc"`
	// ActorID is the user who created the doc, so notifications skip them.
	ActorID string `json:"actor_id,omitempty"`
	// MentionedUserIDs are the people the body @-mentions, each told once in their inbox.
	MentionedUserIDs []string `json:"mentioned_user_ids,omitempty"`
}

// UpdatedEvent is the payload for doc.updated.
type UpdatedEvent struct {
	Doc Doc `json:"doc"`
	// ActorID is the user whose edit, archive, or collaborative commit this is; empty when none is known.
	ActorID string `json:"actor_id,omitempty"`
	// MentionedUserIDs are the people this save @-mentions that the previous version did not.
	MentionedUserIDs []string `json:"mentioned_user_ids,omitempty"`
}

// DeletedEvent is the doc.deleted payload; the doc is already gone by publish time, so consumers get identity only.
type DeletedEvent struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	ProjectID string `json:"project_id"`
}

// MovedEvent is the doc.moved payload: the doc as it now stands, in its new folder.
type MovedEvent struct {
	Doc          Doc    `json:"doc"`
	FromFolderID string `json:"from_folder_id"`
	ActorID      string `json:"actor_id,omitempty"`
}

// FolderEvent is the doc.folder.created payload.
type FolderEvent struct {
	Folder  Folder `json:"folder"`
	ActorID string `json:"actor_id,omitempty"`
}

// FolderUpdatedEvent is the doc.folder.updated payload, sent when a folder is renamed.
type FolderUpdatedEvent struct {
	Folder       Folder `json:"folder"`
	PreviousName string `json:"previous_name"`
	ActorID      string `json:"actor_id,omitempty"`
}

// FolderDeletedEvent is the doc.folder.deleted payload; the folder's docs moved to MovedToFolderID, never deleted.
type FolderDeletedEvent struct {
	Folder          Folder `json:"folder"`
	MovedToFolderID string `json:"moved_to_folder_id"`
	ActorID         string `json:"actor_id,omitempty"`
}

// WatchersChangedEvent is the doc.watchers.changed payload: who started (watching true) or stopped watching which doc.
type WatchersChangedEvent struct {
	Doc      WatchedDoc `json:"doc"`
	UserID   string     `json:"user_id"`
	Watching bool       `json:"watching"`
}

// WatchedDoc is the slice of a doc a watcher or clarification event names; the body never travels with it.
type WatchedDoc struct {
	ID        string `json:"id"`
	ProjectID string `json:"project_id"`
	Title     string `json:"title"`
}

// ClarificationRoundEvent is the payload of a round starting, ending, being fully answered, its "Anything else?" being
// saved, and the clarification closing; StartedBy is the round's starter, ActorID whoever caused the event.
type ClarificationRoundEvent struct {
	Doc       WatchedDoc `json:"doc"`
	Round     int        `json:"round"`
	StartedBy string     `json:"started_by"`
	ActorID   string     `json:"actor_id,omitempty"`
	// Removed is set when a round ended having asked nothing and found no gaps, so it is gone.
	Removed bool `json:"removed,omitempty"`
}

// ClarificationPostedEvent is the doc.clarification.round_posted payload: a run posted its questions or found no gaps.
type ClarificationPostedEvent struct {
	ClarificationRoundEvent
	QuestionCount int  `json:"question_count"`
	NoGaps        bool `json:"no_gaps"`
}

// ClarificationAnswerEvent is the answer_saved and answer_cleared payload: which question, never the answer.
type ClarificationAnswerEvent struct {
	Doc        WatchedDoc `json:"doc"`
	Round      int        `json:"round"`
	QuestionID string     `json:"question_id"`
	Question   string     `json:"question"`
	AuthorID   string     `json:"author_id"`
	At         time.Time  `json:"at"`
}
