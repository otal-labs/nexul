// Package pairing implements the user-level half of @Agent: paired computers, their harness sessions, and defaults.
package pairing

import (
	"fmt"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/otal-labs/nexul/internal/harness"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// Computer has no upstream refresh flow, so TokenExpiresAt governs expiry.
type Computer struct {
	ID             string       `json:"id"`
	UserID         string       `json:"-"`
	Kind           harness.Kind `json:"kind"`
	Name           string       `json:"name"`
	ServerURL      string       `json:"server_url"`
	TokenExpiresAt time.Time    `json:"token_expires_at"`
	HarnessVersion string       `json:"harness_version"`
	// SetupConfirmedAt is the overall setup confirmation (ADR 0063); nil means unconfirmed.
	SetupConfirmedAt *time.Time `json:"setup_confirmed_at"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
	// BearerToken is encrypted at rest; the `-` tag keeps it off every HTTP response.
	BearerToken string `json:"-"`
	// SetupMCPToken is the encrypted MCP token the setup turns wrote into the providers' configs.
	SetupMCPToken string `json:"-"`
	// Tunnel is nil for a computer paired by URL (ADR 0062).
	Tunnel *ComputerTunnel `json:"tunnel,omitempty"`
}

// ComputerTunnel is a computer's own tunnel on the instance's Cloudflare, with what teardown needs to remove it.
type ComputerTunnel struct {
	TunnelID    string `json:"tunnel_id"`
	Hostname    string `json:"hostname"`
	ZoneID      string `json:"-"`
	RecordID    string `json:"-"`
	AccessAppID string `json:"-"`
}

// TunnelStatus is a computer tunnel's two checks: Cloudflare's connector status and the harness answering through it.
type TunnelStatus struct {
	// Tunnel is Cloudflare's status: inactive, healthy, degraded, or down.
	Tunnel           string `json:"tunnel"`
	HarnessReachable bool   `json:"harness_reachable"`
	HarnessVersion   string `json:"harness_version,omitempty"`
}

// Connected reports both checks passing: the connector is up and the harness answers behind it.
func (t TunnelStatus) Connected() bool {
	return (t.Tunnel == "healthy" || t.Tunnel == "degraded") && t.HarnessReachable
}

// DefaultT3CodePort is the port T3 Code serves on unless started with another.
const DefaultT3CodePort = 3773

// PrerequisiteReason names what the instance needs before any computer can be reached through a tunnel.
type PrerequisiteReason string

const (
	// ReasonCloudflareNotConnected: the instance has no Cloudflare connection to create tunnels with.
	ReasonCloudflareNotConnected PrerequisiteReason = "cloudflare_not_connected"
	// ReasonZeroTrustDisabled: the Cloudflare account has never enabled Zero Trust, so no Access app can close a hostname.
	ReasonZeroTrustDisabled PrerequisiteReason = "zero_trust_disabled"
)

// PrerequisiteError unwraps to ErrInvalid; Reason tells the caller which fix to show.
type PrerequisiteError struct {
	Reason PrerequisiteReason
	Err    error
}

func (e *PrerequisiteError) Error() string { return e.Err.Error() }

func (e *PrerequisiteError) Unwrap() error { return apperrs.ErrInvalid }

// FieldError names the pairing input a failure belongs to (name, server_url, or token), so a form can show it on that field.
type FieldError struct {
	Field string
	Err   error
}

func (e *FieldError) Error() string { return e.Err.Error() }

func (e *FieldError) Unwrap() error { return e.Err }

// Paired reports whether the computer holds a harness session; a computer tunnel has none until T3 Code pairs over it.
func (c Computer) Paired() bool { return !c.TokenExpiresAt.IsZero() }

// address is where the harness answers: the tunnel hostname when the computer has one, else its server URL.
func (c Computer) address() string {
	if c.Tunnel != nil {
		return "https://" + c.Tunnel.Hostname
	}
	return c.ServerURL
}

// Session is the harness-facing view of a computer; only call it on a decrypted copy.
func (c Computer) Session() harness.Session {
	return harness.Session{Name: c.Name, ServerURL: c.ServerURL, BearerToken: c.BearerToken}
}

// Setup is a computer's setup confirmation, overall and per provider (ADR 0063).
type Setup struct {
	ComputerID  string          `json:"computer_id"`
	ConfirmedAt *time.Time      `json:"confirmed_at"`
	Providers   []ProviderSetup `json:"providers"`
	// Turns is each provider's newest setup turn, for a setup dialog opened mid-run.
	Turns []SetupTurnSummary `json:"turns"`
}

// SetupTurnSummary is one provider's newest setup turn without its transcript.
type SetupTurnSummary struct {
	RunID        string         `json:"run_id"`
	TurnID       string         `json:"turn_id"`
	Provider     string         `json:"provider"`
	ProviderName string         `json:"provider_name"`
	State        SetupTurnState `json:"state"`
	Status       string         `json:"status"`
	// Model is the model slug the turn ran on; empty means the provider's own default.
	Model     string    `json:"model"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ProviderSetup is one provider's confirmation on a computer, keyed by the provider's driver kind, never its instance id.
type ProviderSetup struct {
	Provider    string     `json:"provider"`
	ConfirmedAt *time.Time `json:"confirmed_at"`
	Skills      []string   `json:"skills"`
}

// SetupTurnState is where one provider's setup turn stands.
type SetupTurnState string

const (
	SetupTurnRunning   SetupTurnState = "running"
	SetupTurnConfirmed SetupTurnState = "confirmed"
	SetupTurnFailed    SetupTurnState = "failed"
)

// SetupTurn is one provider's setup on a computer: a prepare session and a confirm session, one transcript.
type SetupTurn struct {
	ID           string
	RunID        string
	ComputerID   string
	UserID       string
	Provider     string
	ProviderName string
	Model        string
	State        SetupTurnState
	// Status is the short line the setup dialog shows under the provider.
	Status     string
	Transcript []harness.Activity
	StartedAt  time.Time
	UpdatedAt  time.Time
	EndedAt    *time.Time
}

// appendStep replaces the step of the same tool call, so a call's start and finish are one step, else appends.
func (t *SetupTurn) appendStep(a harness.Activity) {
	if a.CallID != "" {
		for i := len(t.Transcript) - 1; i >= 0; i-- {
			if t.Transcript[i].CallID == a.CallID {
				t.Transcript[i] = a
				return
			}
		}
	}
	t.Transcript = append(t.Transcript, a)
}

// SetupRun is what starting setup hands back: the run and the providers it sets up, in order.
type SetupRun struct {
	RunID      string          `json:"run_id"`
	ComputerID string          `json:"computer_id"`
	Providers  []SetupProvider `json:"providers"`
}

// SetupProvider is one provider a setup run covers, by driver kind and display name, with the model its turn runs on.
type SetupProvider struct {
	Provider string `json:"provider"`
	Name     string `json:"name"`
	Model    string `json:"model,omitempty"`
}

// ProviderOption is a provider instance as the pickers list it: still selectable while it needs setup.
type ProviderOption struct {
	harness.Provider
	NeedsSetup bool `json:"needs_setup"`
}

// setupConfirmed is the gate's rule: the computer's overall confirmation and the driver's own are both set.
func setupConfirmed(c Computer, setups []ProviderSetup, driver string) bool {
	if c.SetupConfirmedAt == nil {
		return false
	}
	driver = strings.ToLower(strings.TrimSpace(driver))
	return slices.ContainsFunc(setups, func(p ProviderSetup) bool { return p.Provider == driver && p.ConfirmedAt != nil })
}

// Defaults are a user's pairing-settings defaults for non-project chat contexts; every field is optional.
type Defaults struct {
	UserID            string `json:"-"`
	DefaultComputerID string `json:"default_computer_id,omitempty"`
	FallbackProjectID string `json:"fallback_project_id,omitempty"`
	Provider          string `json:"provider,omitempty"`
	Model             string `json:"model,omitempty"`
}

// ProjectLink is the project-level pairing config; a zero value means unlinked, falling through to defaults.
type ProjectLink struct {
	ProjectID        string    `json:"project_id"`
	ComputerID       string    `json:"computer_id,omitempty"`
	HarnessProjectID string    `json:"harness_project_id,omitempty"`
	Provider         string    `json:"provider,omitempty"`
	Model            string    `json:"model,omitempty"`
	UpdatedAt        time.Time `json:"updated_at,omitempty"`
}

// NotConfiguredReason distinguishes why ResolveTarget failed, for a specific reply, not one generic message.
type NotConfiguredReason string

const (
	// ReasonUnpaired: no computer resolves at all.
	ReasonUnpaired NotConfiguredReason = "unpaired"
	// ReasonExpiredToken: a computer resolved, but its bearer session expired; the fix is re-pairing.
	ReasonExpiredToken NotConfiguredReason = "expired_token"
	// ReasonNoDefault: a valid computer resolved, but no harness project is configured for it.
	ReasonNoDefault NotConfiguredReason = "no_default"
	// ReasonNoDefaultComputer: several paired computers, none picked as default; the fix is choosing one.
	ReasonNoDefaultComputer NotConfiguredReason = "no_default_computer"
	// ReasonSetupRequired: the computer or the provider the run lands on has no setup confirmation (ADR 0063).
	ReasonSetupRequired NotConfiguredReason = "setup_required"
	// ReasonOffline: the resolved computer's harness did not answer, so the setup gate could not check its providers.
	ReasonOffline NotConfiguredReason = "offline"
)

// NotConfiguredError unwraps to ErrInvalid; Reason carries the specific fix.
type NotConfiguredError struct {
	Reason NotConfiguredReason
	// Provider and Computer are display names, set for ReasonSetupRequired and ReasonOffline so the refusal names them.
	Provider string
	Computer string
	// ComputerID and ProviderID carry the same refusal structurally, so a client can link to the fix.
	ComputerID string
	ProviderID string
	// Err is the harness failure behind ReasonOffline.
	Err error
}

// Error is the user-facing refusal for the gate's reasons, so chat, the play run dialog, and MCP all read the same line.
func (e *NotConfiguredError) Error() string {
	if e.Reason == ReasonSetupRequired {
		return fmt.Sprintf("@Agent can't use %s on %s until its setup is done — run setup for %s in Settings → T3 pairing.", e.Provider, e.Computer, e.Computer)
	}
	if e.Reason == ReasonOffline {
		return fmt.Sprintf("@Agent can't reach %s — is T3 Code running there?", e.Computer)
	}
	return fmt.Sprintf("pairing not configured: %s", e.Reason)
}

// RefusalDetails is a NotConfiguredError's machine-readable side, carried beside the message in the error envelope.
type RefusalDetails struct {
	Reason     NotConfiguredReason `json:"reason"`
	ComputerID string              `json:"computer_id,omitempty"`
	Computer   string              `json:"computer,omitempty"`
	ProviderID string              `json:"provider_id,omitempty"`
	Provider   string              `json:"provider,omitempty"`
}

// ErrorDetails satisfies httpx.DetailedError.
func (e *NotConfiguredError) ErrorDetails() any {
	return RefusalDetails{Reason: e.Reason, ComputerID: e.ComputerID, Computer: e.Computer, ProviderID: e.ProviderID, Provider: e.Provider}
}

func (e *NotConfiguredError) Unwrap() []error {
	if e.Err == nil {
		return []error{apperrs.ErrInvalid}
	}
	return []error{apperrs.ErrInvalid, e.Err}
}

// validateHarnessProjectID rejects a blank harness-side project id.
func validateHarnessProjectID(id string) (string, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return "", fmt.Errorf("%w: harness project id is required", apperrs.ErrInvalid)
	}
	return id, nil
}

// validateProvider normalizes a provider driver kind (claude, codex, opencode, …) and rejects a blank one.
func validateProvider(provider string) (string, error) {
	provider = strings.ToLower(strings.TrimSpace(provider))
	if provider == "" {
		return "", fmt.Errorf("%w: provider is required", apperrs.ErrInvalid)
	}
	return provider, nil
}

// validateSkills trims and de-duplicates the reported skills; a confirmation with none verified is rejected.
func validateSkills(skills []string) ([]string, error) {
	out := make([]string, 0, len(skills))
	for _, skill := range skills {
		skill = strings.TrimSpace(skill)
		if skill == "" || slices.Contains(out, skill) {
			continue
		}
		out = append(out, skill)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("%w: at least one reported skill is required", apperrs.ErrInvalid)
	}
	return out, nil
}

// validateName rejects a blank computer name.
func validateName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("%w: computer name is required", apperrs.ErrInvalid)
	}
	return name, nil
}

// validatePort rejects a local harness port outside the TCP range.
func validatePort(port int) error {
	if port < 1 || port > 65535 {
		return fmt.Errorf("%w: local harness port %d is not between 1 and 65535", apperrs.ErrInvalid, port)
	}
	return nil
}

// validateServerURL rejects non-http(s) origins and trims a trailing slash so clients can concatenate paths.
func validateServerURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("%w: server URL is required", apperrs.ErrInvalid)
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return "", fmt.Errorf("%w: server URL %q is not a valid http(s) URL", apperrs.ErrInvalid, raw)
	}
	return strings.TrimRight(raw, "/"), nil
}
