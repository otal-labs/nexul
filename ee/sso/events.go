package sso

import (
	"time"

	"github.com/otal-labs/nexul/internal/platform/eventbus"
)

// TopicChanged is published when the instance's single sign-on settings are saved or removed.
const TopicChanged = "instance.sso_changed"

// Topics lists every topic this package publishes.
func Topics() []eventbus.Topic {
	return []eventbus.Topic{{Name: TopicChanged, Payload: ChangedEvent{}}}
}

// ChangedEvent is the instance.sso_changed payload; the client secret stays out.
type ChangedEvent struct {
	DisplayName string    `json:"display_name" jsonschema:"The provider's name on the sign-in button; empty when the settings were removed."`
	IssuerURL   string    `json:"issuer_url" jsonschema:"The provider's issuer URL; empty when the settings were removed."`
	Only        bool      `json:"only" jsonschema:"True while single sign-on is the only way in besides the instance Owner's passkey."`
	Removed     bool      `json:"removed" jsonschema:"True when the settings were removed and the instance holds none."`
	ChangedBy   string    `json:"changed_by" jsonschema:"The account that saved or removed the settings."`
	ChangedAt   time.Time `json:"changed_at"`
}
