package license

import (
	"time"

	"github.com/otal-labs/nexul/internal/platform/eventbus"
)

// TopicChanged is published when the instance's license key is pasted, replaced or removed.
const TopicChanged = "instance.license_changed"

// Topics lists every topic this package publishes.
func Topics() []eventbus.Topic {
	return []eventbus.Topic{{Name: TopicChanged, Payload: ChangedEvent{}}}
}

// ChangedEvent is the instance.license_changed payload; the key's text stays out.
type ChangedEvent struct {
	Licensee  string    `json:"licensee" jsonschema:"Who the new key is for; empty when the key was removed."`
	Features  []Feature `json:"features" jsonschema:"The enterprise features the new key unlocks; empty when the key was removed."`
	ExpiresAt time.Time `json:"expires_at,omitzero" jsonschema:"When the new key stops unlocking its features."`
	Removed   bool      `json:"removed" jsonschema:"True when the key was removed and the instance holds none."`
	ChangedBy string    `json:"changed_by" jsonschema:"The account that pasted or removed the key."`
	ChangedAt time.Time `json:"changed_at"`
}
