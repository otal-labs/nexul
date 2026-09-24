package pairing

import "time"

// Topics published by the pairing domain.
const (
	TopicSetupConfirmed   = "computer.setup_confirmed"
	TopicSetupUnconfirmed = "computer.setup_unconfirmed"
	TopicTunnelCreated    = "computer.tunnel_created"
	TopicTunnelRemoved    = "computer.tunnel_removed"
)

// Topics returns every topic the pairing domain publishes.
func Topics() []string {
	return []string{TopicSetupConfirmed, TopicSetupUnconfirmed, TopicTunnelCreated, TopicTunnelRemoved}
}

// SetupChangedEvent is the payload for both setup topics; an empty Provider means the overall confirmation.
type SetupChangedEvent struct {
	ComputerID  string     `json:"computer_id"`
	UserID      string     `json:"user_id"`
	Provider    string     `json:"provider,omitempty"`
	ConfirmedAt *time.Time `json:"confirmed_at,omitempty"`
	Skills      []string   `json:"skills,omitempty"`
}

// TunnelChangedEvent is the payload for both tunnel topics; it never carries the connector token.
type TunnelChangedEvent struct {
	ComputerID string `json:"computer_id"`
	UserID     string `json:"user_id"`
	TunnelID   string `json:"tunnel_id"`
	Hostname   string `json:"hostname"`
}
