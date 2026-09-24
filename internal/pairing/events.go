package pairing

import "time"

// Topics published by the pairing domain.
const (
	TopicComputerPaired   = "computer.paired"
	TopicSetupConfirmed   = "computer.setup_confirmed"
	TopicSetupUnconfirmed = "computer.setup_unconfirmed"
	TopicTunnelCreated    = "computer.tunnel_created"
	TopicTunnelRemoved    = "computer.tunnel_removed"
	// TopicTunnelStatusChanged is ephemeral: published straight to the bus, never through the outbox.
	TopicTunnelStatusChanged = "computer.tunnel_status_changed"
)

// Topics returns every topic the pairing domain publishes.
func Topics() []string {
	return []string{TopicComputerPaired, TopicSetupConfirmed, TopicSetupUnconfirmed, TopicTunnelCreated, TopicTunnelRemoved, TopicTunnelStatusChanged}
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

// TunnelStatusChangedEvent carries a watched tunnel's two checks whole, so a consumer replaces its view instead of refetching.
type TunnelStatusChangedEvent struct {
	ComputerID       string `json:"computer_id"`
	UserID           string `json:"user_id"`
	Tunnel           string `json:"tunnel"`
	HarnessReachable bool   `json:"harness_reachable"`
	HarnessVersion   string `json:"harness_version,omitempty"`
}

// ComputerPairedEvent is a computer gaining a harness session, whether first paired, re-paired, or paired over its tunnel.
type ComputerPairedEvent struct {
	ComputerID     string    `json:"computer_id"`
	UserID         string    `json:"user_id"`
	ServerURL      string    `json:"server_url"`
	HarnessVersion string    `json:"harness_version,omitempty"`
	TokenExpiresAt time.Time `json:"token_expires_at"`
}
