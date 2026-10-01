package templates

import "time"

// TopicUpdated is published when an instance template is edited, cloned into, or reset to its code default.
const TopicUpdated = "instance_template.updated"

// Topics lists every topic this domain publishes.
func Topics() []string {
	return []string{TopicUpdated}
}

// UpdatedEvent is the instance_template.updated payload; the body stays out, like the interview template's event.
type UpdatedEvent struct {
	Kind      string    `json:"kind"`
	Key       string    `json:"key"`
	AuthorID  string    `json:"author_id"`
	UpdatedAt time.Time `json:"updated_at"`
	// Reset is true when the template went back to its code default.
	Reset bool `json:"reset"`
}
