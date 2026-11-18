package imports

import (
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/ids"
)

// TopicProgress is an import's state, counts or report changing; the live socket pushes it to its creator alone.
const TopicProgress = "import.progress"

// ProgressEvent carries the import as import_run returns it, never its credential.
type ProgressEvent struct {
	// UserID is the person who started the import, the only one the frame reaches.
	UserID string `json:"user_id"`
	Import Import `json:"import"`
}

// Topics lists the topics this domain publishes, with their payload types.
func Topics() []eventbus.Topic {
	return []eventbus.Topic{{Name: TopicProgress, Payload: ProgressEvent{}}}
}

func progressEvent(imp *Import) eventbus.OutboxEvent {
	return eventbus.OutboxEvent{ID: ids.New(), Topic: TopicProgress, Payload: ProgressEvent{UserID: imp.CreatedBy, Import: *imp}}
}
