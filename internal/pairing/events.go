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
	TopicSetupTurnChanged    = "computer.setup_turn_changed"
	TopicSetupFinished       = "computer.setup_finished"
	// TopicSetupTurnActivity is ephemeral: the running turn's latest step, never written to the outbox.
	TopicSetupTurnActivity = "computer.setup_turn_activity"
)

// Topics returns every topic the pairing domain publishes.
func Topics() []string {
	return []string{
		TopicComputerPaired, TopicSetupConfirmed, TopicSetupUnconfirmed, TopicTunnelCreated, TopicTunnelRemoved, TopicTunnelStatusChanged,
		TopicSetupTurnChanged, TopicSetupFinished, TopicSetupTurnActivity,
	}
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

// SetupTurnChangedEvent is one provider's setup turn starting, confirming, or failing, with its short status line.
type SetupTurnChangedEvent struct {
	ComputerID   string         `json:"computer_id"`
	UserID       string         `json:"user_id"`
	RunID        string         `json:"run_id"`
	TurnID       string         `json:"turn_id"`
	Provider     string         `json:"provider"`
	ProviderName string         `json:"provider_name"`
	Model        string         `json:"model,omitempty"`
	State        SetupTurnState `json:"state"`
	Status       string         `json:"status"`
	StartedAt    time.Time      `json:"started_at"`
	EndedAt      *time.Time     `json:"ended_at,omitempty"`
}

// SetupFinishedEvent closes a setup run: each provider's outcome and whether the computer ended confirmed overall.
type SetupFinishedEvent struct {
	ComputerID string             `json:"computer_id"`
	UserID     string             `json:"user_id"`
	RunID      string             `json:"run_id"`
	Confirmed  bool               `json:"confirmed"`
	Providers  []SetupTurnOutcome `json:"providers"`
}

// SetupTurnOutcome is one provider's end state in a finished setup run.
type SetupTurnOutcome struct {
	Provider string         `json:"provider"`
	State    SetupTurnState `json:"state"`
	Status   string         `json:"status"`
}

// SetupTurnActivityEvent is the running turn's latest step as one line, for the commentary under its row.
type SetupTurnActivityEvent struct {
	ComputerID string `json:"computer_id"`
	UserID     string `json:"user_id"`
	RunID      string `json:"run_id"`
	TurnID     string `json:"turn_id"`
	Provider   string `json:"provider"`
	Status     string `json:"status"`
	// CallID names the tool call the step belongs to, so a consumer updates that step's line instead of adding one.
	CallID string `json:"call_id,omitempty"`
}
