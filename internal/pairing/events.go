package pairing

import (
	"time"

	"github.com/otal-labs/nexul/internal/harness"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
)

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
	TopicHarnessSwitched   = "computer.harness_switched"
)

// Topics returns every topic the pairing domain publishes.
func Topics() []eventbus.Topic {
	return []eventbus.Topic{
		{Name: TopicComputerPaired, Payload: ComputerPairedEvent{}},
		{Name: TopicSetupConfirmed, Payload: SetupChangedEvent{}},
		{Name: TopicSetupUnconfirmed, Payload: SetupChangedEvent{}},
		{Name: TopicTunnelCreated, Payload: TunnelChangedEvent{}},
		{Name: TopicTunnelRemoved, Payload: TunnelChangedEvent{}},
		{Name: TopicTunnelStatusChanged, Payload: TunnelStatusChangedEvent{}},
		{Name: TopicSetupTurnChanged, Payload: SetupTurnChangedEvent{}},
		{Name: TopicSetupFinished, Payload: SetupFinishedEvent{}},
		{Name: TopicSetupTurnActivity, Payload: SetupTurnActivityEvent{}},
		{Name: TopicHarnessSwitched, Payload: HarnessSwitchedEvent{}},
	}
}

// SetupChangedEvent is the payload for both setup topics; an empty Provider means the overall confirmation.
type SetupChangedEvent struct {
	ComputerID  string     `json:"computer_id"`
	UserID      string     `json:"user_id"`
	Provider    string     `json:"provider,omitempty"`
	ConfirmedAt *time.Time `json:"confirmed_at,omitempty"`
	Skills      []string   `json:"skills,omitempty"`
	MembersOnly bool       `json:"members_only" jsonschema:"Always true: a person's computer is never delivered to integrations or automations."`
}

// TunnelChangedEvent is the payload for both tunnel topics; it never carries the connector token.
type TunnelChangedEvent struct {
	ComputerID  string `json:"computer_id"`
	UserID      string `json:"user_id"`
	TunnelID    string `json:"tunnel_id"`
	Hostname    string `json:"hostname"`
	MembersOnly bool   `json:"members_only" jsonschema:"Always true: a person's computer is never delivered to integrations or automations."`
}

// TunnelStatusChangedEvent carries a watched tunnel's two checks whole, so a consumer replaces its view instead of refetching.
type TunnelStatusChangedEvent struct {
	ComputerID       string `json:"computer_id"`
	UserID           string `json:"user_id"`
	Tunnel           string `json:"tunnel"`
	HarnessReachable bool   `json:"harness_reachable"`
	HarnessVersion   string `json:"harness_version,omitempty"`
	MembersOnly      bool   `json:"members_only" jsonschema:"Always true: a person's computer is never delivered to integrations or automations."`
}

// ComputerPairedEvent is a computer gaining a harness session, whether first paired, re-paired, or paired over its tunnel.
type ComputerPairedEvent struct {
	ComputerID     string    `json:"computer_id"`
	UserID         string    `json:"user_id"`
	ServerURL      string    `json:"server_url"`
	HarnessVersion string    `json:"harness_version,omitempty"`
	TokenExpiresAt time.Time `json:"token_expires_at"`
	MembersOnly    bool      `json:"members_only" jsonschema:"Always true: a person's computer is never delivered to integrations or automations."`
}

// HarnessSwitchedEvent is a computer's stored kind moving forward, because its harness moved on (ADR 0113).
type HarnessSwitchedEvent struct {
	ComputerID     string       `json:"computer_id"`
	UserID         string       `json:"user_id"`
	FromKind       harness.Kind `json:"from_kind" jsonschema:"The harness kind the computer was stored under, for example t3code."`
	ToKind         harness.Kind `json:"to_kind" jsonschema:"The harness kind it moved forward to, for example t3code-v2; a computer never moves back."`
	HarnessVersion string       `json:"harness_version" jsonschema:"The harness version read when the computer moved."`
	MembersOnly    bool         `json:"members_only" jsonschema:"Always true: a person's computer is never delivered to integrations or automations."`
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
	State        SetupTurnState `json:"state" enum:"running,confirmed,failed"`
	Status       string         `json:"status"`
	StartedAt    time.Time      `json:"started_at"`
	EndedAt      *time.Time     `json:"ended_at,omitempty"`
	MembersOnly  bool           `json:"members_only" jsonschema:"Always true: a person's computer is never delivered to integrations or automations."`
}

// SetupFinishedEvent closes a setup run: each provider's outcome and whether the computer ended confirmed overall.
type SetupFinishedEvent struct {
	ComputerID  string             `json:"computer_id"`
	UserID      string             `json:"user_id"`
	RunID       string             `json:"run_id"`
	Confirmed   bool               `json:"confirmed"`
	Providers   []SetupTurnOutcome `json:"providers"`
	MembersOnly bool               `json:"members_only" jsonschema:"Always true: a person's computer is never delivered to integrations or automations."`
}

// SetupTurnOutcome is one provider's end state in a finished setup run.
type SetupTurnOutcome struct {
	Provider string         `json:"provider"`
	State    SetupTurnState `json:"state" enum:"running,confirmed,failed"`
	Status   string         `json:"status"`
}

// SetupTurnActivityEvent is the running turn's latest step as one line, for the setup dialog's transcript.
type SetupTurnActivityEvent struct {
	ComputerID string `json:"computer_id"`
	UserID     string `json:"user_id"`
	RunID      string `json:"run_id"`
	TurnID     string `json:"turn_id"`
	Provider   string `json:"provider"`
	Status     string `json:"status"`
	// CallID names the tool call the step belongs to, so a consumer updates that step's line instead of adding one.
	CallID string `json:"call_id,omitempty"`
	// Kind is the step's harness.ActivityKind; tool_call means the call is still open.
	Kind string `json:"kind,omitempty" enum:"tool_call,tool_result,text,question,other,note,user_message"`
	// Tool names the tool a call ran, so the dialog tells a command from any other tool.
	Tool string `json:"tool,omitempty"`
	// Text is a text step's whole message, which the dialog shows as prose; Status stays its one-line preview.
	Text        string    `json:"text,omitempty"`
	At          time.Time `json:"at"`
	MembersOnly bool      `json:"members_only" jsonschema:"Always true: a person's computer is never delivered to integrations or automations."`
}
