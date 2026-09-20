package gitprovider

import (
	"encoding/json"
	"time"
)

const (
	// TopicProviderEvent is published for every accepted delivery as one provider-agnostic envelope, all types.
	TopicProviderEvent = "git.provider_event"
)

// RepositoryRef identifies the repository a delivery concerns, shaped for future providers to map onto.
type RepositoryRef struct {
	ID            int64  `json:"id"`
	Name          string `json:"name"`
	FullName      string `json:"full_name"`
	Owner         string `json:"owner"`
	HTMLURL       string `json:"html_url"`
	DefaultBranch string `json:"default_branch"`
}

// ProviderEvent is the envelope on TopicProviderEvent, additive-only; Payload carries the provider's typed JSON.
type ProviderEvent struct {
	Provider   string          `json:"provider"`
	EventType  string          `json:"event_type"`
	DeliveryID string          `json:"delivery_id"`
	Action     string          `json:"action,omitempty"`
	Repository *RepositoryRef  `json:"repository,omitempty"`
	Payload    json.RawMessage `json:"payload"`
	ReceivedAt time.Time       `json:"received_at"`
}
