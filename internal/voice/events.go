package voice

// TopicOccupancyChanged is published whenever a voice channel's occupants change, from either feed.
const TopicOccupancyChanged = "voice.occupancy.changed"

// Topics returns every topic the voice domain publishes.
func Topics() []string {
	return []string{TopicOccupancyChanged}
}

// OccupancyChangedEvent is TopicOccupancyChanged's payload: the full occupant list, replacing the consumer's view.
type OccupancyChangedEvent struct {
	ConversationID string     `json:"conversation_id"`
	Occupants      []Occupant `json:"occupants"`
}
