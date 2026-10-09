package voice

import "github.com/otal-labs/nexul/internal/platform/eventbus"

// TopicOccupancyChanged is published whenever a voice channel's occupants change, from either feed.
const TopicOccupancyChanged = "voice.occupancy.changed"

// Topics returns every topic the voice domain publishes.
func Topics() []eventbus.Topic {
	return []eventbus.Topic{{Name: TopicOccupancyChanged, Payload: OccupancyChangedEvent{}}}
}

// OccupancyChangedEvent is TopicOccupancyChanged's payload: the full occupant list, replacing the consumer's view.
type OccupancyChangedEvent struct {
	ConversationID string     `json:"conversation_id"`
	Occupants      []Occupant `json:"occupants"`
	MembersOnly    bool       `json:"members_only,omitempty" jsonschema:"Set on a private voice channel's call, which is never delivered to integrations or automations."`
}
