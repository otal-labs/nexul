package docs

// Topics published by the docs domain. Consumers: mcp (re-index).
const (
	TopicCreated       = "doc.created"
	TopicUpdated       = "doc.updated"
	TopicDeleted       = "doc.deleted"
	TopicMoved         = "doc.moved"
	TopicFolderCreated = "doc.folder.created"
	TopicFolderUpdated = "doc.folder.updated"
	TopicFolderDeleted = "doc.folder.deleted"
)

// Topics returns every topic the docs domain publishes.
func Topics() []string {
	return []string{TopicCreated, TopicUpdated, TopicDeleted, TopicMoved, TopicFolderCreated, TopicFolderUpdated, TopicFolderDeleted}
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
