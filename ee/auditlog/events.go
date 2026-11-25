package auditlog

import (
	"time"

	"github.com/otal-labs/nexul/internal/platform/eventbus"
)

// TopicSettingsChanged is published when the audit log's settings are saved or removed.
const TopicSettingsChanged = "instance.audit_settings_changed"

// Topics lists every topic this package publishes.
func Topics() []eventbus.Topic {
	return []eventbus.Topic{{Name: TopicSettingsChanged, Payload: SettingsChangedEvent{}}}
}

// SettingsChangedEvent is the instance.audit_settings_changed payload; the stream secret stays out.
type SettingsChangedEvent struct {
	RetentionDays int       `json:"retention_days" jsonschema:"How many days audit rows are now kept; 45 when the settings were removed."`
	StreamURL     string    `json:"stream_url" jsonschema:"Where each audit row is sent; empty when there is no destination."`
	StreamEnabled bool      `json:"stream_enabled" jsonschema:"True while the stream is switched on."`
	Removed       bool      `json:"removed" jsonschema:"True when the settings were removed and the instance holds none."`
	ChangedBy     string    `json:"changed_by" jsonschema:"The account that saved or removed the settings."`
	ChangedAt     time.Time `json:"changed_at"`
}
