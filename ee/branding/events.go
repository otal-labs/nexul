package branding

import (
	"time"

	"github.com/otal-labs/nexul/internal/platform/eventbus"
)

// TopicChanged is published when the instance's branding is saved, its logo replaced or removed, or everything cleared.
const TopicChanged = "instance.branding_changed"

// Topics lists every topic this package publishes.
func Topics() []eventbus.Topic {
	return []eventbus.Topic{{Name: TopicChanged, Payload: ChangedEvent{}}}
}

// ChangedEvent is the instance.branding_changed payload; it names no colour, name or image, so a page refetches the look.
type ChangedEvent struct {
	Cleared   bool      `json:"cleared" jsonschema:"True when all the branding was removed and the instance shows the default look."`
	ChangedBy string    `json:"changed_by" jsonschema:"The account that changed the branding."`
	ChangedAt time.Time `json:"changed_at"`
}
