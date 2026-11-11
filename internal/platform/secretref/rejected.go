package secretref

import (
	"slices"

	"github.com/otal-labs/nexul/internal/platform/eventbus"
)

// TopicTokenRejected fires once each time a secrets manager's state turns "token rejected" on a runner, an
// automations host or the server, so the people who can fix the token hear about it.
const TopicTokenRejected = "secret_manager.token_rejected"

// Sources a TokenRejectedEvent names.
const (
	SourceRunner = "runner"
	SourceHost   = "automations_host"
	SourceServer = "server"
)

// TokenRejectedEvent names the process whose token a manager refused; it never carries a token.
type TokenRejectedEvent struct {
	Source    string `json:"source" jsonschema:"What holds the rejected token: a machine runner, an automations host or the server." enum:"runner,automations_host,server"`
	Name      string `json:"name,omitempty" jsonschema:"The runner's or the automations host's name; empty for the server."`
	Machine   string `json:"machine,omitempty" jsonschema:"The machine the runner or host sits on; the people who manage it are told. Empty for the server, whose Owner is told."`
	Manager   string `json:"manager" jsonschema:"The secrets manager, named by the scheme of the references it resolves, such as op."`
	Workspace string `json:"workspace,omitempty" jsonschema:"The workspace slug whose token the host holds; empty for a runner and the server."`
	// WorkspaceID names the workspace for good; Workspace is its slug at the time, for display.
	WorkspaceID string `json:"workspace_id,omitempty" jsonschema:"The id of the workspace whose token the host holds, which a rename never changes; empty for a runner and the server."`
}

// Topics returns the topics the secrets manager seam publishes.
func Topics() []eventbus.Topic {
	return []eventbus.Topic{{Name: TopicTokenRejected, Payload: TokenRejectedEvent{}, Description: "A secrets manager's state turned token rejected on a runner, an automations host or the server."}}
}

// NewlyRejected is the managers in next that are token rejected now and were not in prev, in next's order.
func NewlyRejected(prev, next []ManagerState) []ManagerState {
	var out []ManagerState
	for _, m := range next {
		if m.State != ManagerTokenRejected {
			continue
		}
		if !slices.Contains(prev, m) {
			out = append(out, m)
		}
	}
	return out
}
