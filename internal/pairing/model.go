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
	shipped "github.com/otal-labs/nexul/internal/platform/skills"
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
	// SetupChoices are the Set up step's saved choices for this computer.
	SetupChoices SetupChoices `json:"-"`
	// Tunnel is nil for a computer paired by URL (ADR 0062).
	Tunnel *ComputerTunnel `json:"tunnel,omitempty"`
	// Runner is the personal runner that reaches the computer (ADR 0146), nil until one enrolls; set on a list only.
	Runner *ComputerRunner `json:"runner,omitempty"`
	// PairError is why the last pairing through the runner failed, cleared when one succeeds; set on a list only.
	PairError string `json:"pair_error,omitempty"`
	// Facts are nil until the computer's runner first reports; FactsAt is when they last changed. Its owner's alone.
	Facts   *Facts     `json:"facts,omitempty"`
	FactsAt *time.Time `json:"facts_at,omitempty"`
}

// Facts is what a computer's owner reads about it (ADR 0146): its runner's report, and what its T3 Code last listed
// through the runner, kept while T3 Code is not answering.
type Facts struct {
	Hostname      string            `json:"hostname,omitempty"`
	OS            string            `json:"os,omitempty"`
	Arch          string            `json:"arch,omitempty"`
	RunnerVersion string            `json:"runner_version,omitempty"`
	T3            T3Facts           `json:"t3"`
	Cloudflared   string            `json:"cloudflared,omitempty"`
	GitName       string            `json:"git_name,omitempty"`
	GitEmail      string            `json:"git_email,omitempty"`
	FreeDiskBytes int64             `json:"free_disk_bytes,omitempty"`
	Providers     []ProviderFacts   `json:"providers"`
	Projects      []harness.Project `json:"projects"`
}

// T3Facts is T3 Code on the computer as its runner found it.
type T3Facts struct {
	// State is answering, not_running, missing or not_loopback.
	State string `json:"state"`
	// Install is service, command_line or desktop_app, "" when unknown.
	Install string `json:"install,omitempty"`
	Port    int    `json:"port,omitempty"`
	Version string `json:"version,omitempty"`
	// RestartedAt is when the runner last restarted T3 Code's background service; RestartError why it could not.
	RestartedAt  *time.Time `json:"restarted_at,omitempty"`
	RestartError string     `json:"restart_error,omitempty"`
}

// t3Answering is the T3Facts state of a T3 Code answering on the computer's loopback.
const t3Answering = "answering"

// ProviderFacts is one provider T3 Code lists on the computer: its CLI's version, whether it is signed in, its models.
type ProviderFacts struct {
	ID      string         `json:"id"`
	Driver  string         `json:"driver"`
	Name    string         `json:"name"`
	Version string         `json:"version,omitempty"`
	SignIn  harness.SignIn `json:"sign_in"`
	Models  []ModelFacts   `json:"models"`
}

// ModelFacts is one model a provider offers, named as T3 Code names it.
type ModelFacts struct {
	Slug string `json:"slug"`
	Name string `json:"name"`
}

func providerFacts(providers []harness.Provider) []ProviderFacts {
	out := make([]ProviderFacts, 0, len(providers))
	for _, p := range providers {
		models := make([]ModelFacts, 0, len(p.Models))
		for _, m := range p.Models {
			models = append(models, ModelFacts{Slug: m.Slug, Name: m.Name})
		}
		out = append(out, ProviderFacts{ID: p.ID, Driver: p.Driver, Name: p.Name, Version: p.Version, SignIn: p.SignIn, Models: models})
	}
	return out
}

// ComputerHostSuffix names a computer reached through its personal runner (ADR 0146); .invalid never resolves.
const ComputerHostSuffix = ".nexul-computer.invalid"

// runnerAddress is the address the harness client reaches computerID's T3 Code at, through its runner.
func runnerAddress(computerID string) string {
	return "http://" + computerID + ComputerHostSuffix
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

// Session is the harness-facing view of a decrypted computer; one added with a runner is reached through it (ADR 0146).
func (c Computer) Session() harness.Session {
	serverURL := c.ServerURL
	if serverURL == "" && c.Tunnel == nil {
		serverURL = runnerAddress(c.ID)
	}
	return harness.Session{ComputerID: c.ID, Name: c.Name, ServerURL: serverURL, BearerToken: c.BearerToken}
}

// Setup is a computer's setup confirmation, overall and per provider (ADR 0063).
type Setup struct {
	ComputerID  string          `json:"computer_id"`
	ConfirmedAt *time.Time      `json:"confirmed_at"`
	Providers   []ProviderSetup `json:"providers"`
	// SetupChoices seed the Set up step when it opens.
	SetupChoices
	// Turns is each provider's newest setup turn, for a setup dialog opened mid-run.
	Turns []SetupTurnSummary `json:"turns"`
}

// SetupChoices are what the Set up step last saved for a computer, by lower-case driver kind: the providers switched off,
// each provider's model ("" for its own default) with that model's options, and the folder the turns run in.
type SetupChoices struct {
	Skipped      []string                           `json:"skipped_providers"`
	Models       map[string]string                  `json:"models"`
	ModelOptions map[string][]harness.OptionSetting `json:"model_options"`
	Folder       string                             `json:"folder"`
}

// withEmpty swaps a nil map for an empty one, so the wire never carries null for one.
func (c SetupChoices) withEmpty() SetupChoices {
	if c.Models == nil {
		c.Models = map[string]string{}
	}
	if c.ModelOptions == nil {
		c.ModelOptions = map[string][]harness.OptionSetting{}
	}
	return c
}

// clean lower-cases the driver keys and checks the options; a nil list or map is stored as empty.
func (c SetupChoices) clean() (SetupChoices, error) {
	out := SetupChoices{Skipped: []string{}, Models: map[string]string{}, ModelOptions: map[string][]harness.OptionSetting{}, Folder: strings.TrimSpace(c.Folder)}
	for _, driver := range c.Skipped {
		if driver = strings.ToLower(strings.TrimSpace(driver)); driver != "" && !slices.Contains(out.Skipped, driver) {
			out.Skipped = append(out.Skipped, driver)
		}
	}
	for driver, model := range c.Models {
		out.Models[strings.ToLower(strings.TrimSpace(driver))] = strings.TrimSpace(model)
	}
	for driver, options := range c.ModelOptions {
		cleaned, err := harness.CleanOptions(options)
		if err != nil {
			return SetupChoices{}, err
		}
		if len(cleaned) > 0 {
			out.ModelOptions[strings.ToLower(strings.TrimSpace(driver))] = cleaned
		}
	}
	return out, nil
}

// SetupTurnSummary is one provider's newest setup turn without its transcript.
type SetupTurnSummary struct {
	RunID        string         `json:"run_id"`
	TurnID       string         `json:"turn_id"`
	Provider     string         `json:"provider"`
	ProviderName string         `json:"provider_name"`
	Kind         SetupTurnKind  `json:"kind"`
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
	// SkillsVersion is the nexul-memory version the confirming setup installed; empty for confirmations older than versioning.
	SkillsVersion string `json:"skills_version"`
	// SkillsOutdated is a signal, never a gate (ADR 0063): a skills update or a setup re-run refreshes the skill.
	SkillsOutdated bool `json:"skills_outdated"`
}

// outdated is a confirmed provider whose recorded skills are not the ones this release ships.
func (p ProviderSetup) outdated() bool {
	return p.ConfirmedAt != nil && p.SkillsVersion != shipped.NexulMemory.Version
}

// SetupTurnState is where one provider's setup turn stands.
type SetupTurnState string

const (
	SetupTurnRunning   SetupTurnState = "running"
	SetupTurnConfirmed SetupTurnState = "confirmed"
	SetupTurnFailed    SetupTurnState = "failed"
)

// SetupTurnKind is the job a turn did, so a failed one retries as the same job.
type SetupTurnKind string

const (
	// SetupTurnSetup connects Nexul's MCP server, installs the skills, and confirms the provider.
	SetupTurnSetup SetupTurnKind = "setup"
	// SetupTurnSkills only rewrites Nexul's skills in the folders every provider on the computer reads.
	SetupTurnSkills SetupTurnKind = "skills"
)

// SetupTurn is one provider's turn on a computer: a setup (a prepare session and a confirm session) or a skills update, one transcript.
type SetupTurn struct {
	ID           string
	RunID        string
	ComputerID   string
	UserID       string
	Provider     string
	ProviderName string
	Model        string
	Kind         SetupTurnKind
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
	Folder     string          `json:"folder,omitempty"`
	Providers  []SetupProvider `json:"providers"`
	// projectID is the harness project Folder resolved to; empty leaves the choice to ResolveSetupTurnTarget.
	projectID string
}

// SetupProvider is one provider a setup run covers, by driver kind and display name, with the model its turn runs on.
type SetupProvider struct {
	Provider     string                  `json:"provider"`
	Name         string                  `json:"name"`
	Model        string                  `json:"model,omitempty"`
	ModelOptions []harness.OptionSetting `json:"model_options,omitempty"`
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
	// ModelOptions are the options picked with Model; unset ones are the harness's defaults.
	ModelOptions []harness.OptionSetting `json:"model_options,omitempty"`
	// StartIn is where new harness threads start; empty is the T3 project's folder.
	StartIn StartIn `json:"start_in,omitempty"`
}

// ProjectLink is one person's pairing config for one project (ADR 0102); a zero value falls through to their defaults.
type ProjectLink struct {
	UserID           string                  `json:"-"`
	ProjectID        string                  `json:"project_id"`
	ComputerID       string                  `json:"computer_id,omitempty"`
	HarnessProjectID string                  `json:"harness_project_id,omitempty"`
	Provider         string                  `json:"provider,omitempty"`
	Model            string                  `json:"model,omitempty"`
	ModelOptions     []harness.OptionSetting `json:"model_options,omitempty"`
	// StartIn overrides the defaults' StartIn for this project; empty falls through to them.
	StartIn   StartIn   `json:"start_in,omitempty"`
	UpdatedAt time.Time `json:"updated_at,omitempty"`
}

// StartIn is where a new harness thread starts: the T3 project's own folder, or a fresh git worktree of it.
type StartIn string

const (
	StartInFolder   StartIn = "folder"
	StartInWorktree StartIn = "worktree"
)

// validateStartIn accepts the two places and empty, which defers to the next level down.
func validateStartIn(s StartIn) (StartIn, error) {
	s = StartIn(strings.TrimSpace(string(s)))
	if s == "" || s == StartInFolder || s == StartInWorktree {
		return s, nil
	}
	return "", fmt.Errorf("%w: start_in must be %q or %q", apperrs.ErrInvalid, StartInFolder, StartInWorktree)
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
	// ReasonNeedsLocation: a person's run in a project they never linked asks where to run instead of using their defaults (ADR 0145).
	ReasonNeedsLocation NotConfiguredReason = "needs_location"
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
	// Why says what is offline for ReasonOffline when Nexul knows more than that T3 Code did not answer.
	Why string
}

// Error is the user-facing refusal for the gate's reasons, so chat, the play run dialog, and MCP all read the same line.
func (e *NotConfiguredError) Error() string {
	if e.Reason == ReasonSetupRequired {
		return fmt.Sprintf("@Agent can't use %s on %s until its setup is done — run setup for %s in Settings → T3 Code Setup.", e.Provider, e.Computer, e.Computer)
	}
	if e.Reason == ReasonOffline {
		why := e.Why
		if why == "" {
			why = "T3 Code isn't answering there"
		}
		return fmt.Sprintf("%s is offline: %s.", e.Computer, why)
	}
	if e.Reason == ReasonNeedsLocation {
		return "Pick where plays run in this project: a computer and its T3 project. It's saved as your link for this project, " +
			"which Settings → T3 Code Setup → Projects can change."
	}
	if text, ok := notConfiguredText[e.Reason]; ok {
		return text
	}
	return fmt.Sprintf("pairing not configured: %s", e.Reason)
}

// notConfiguredText reads after "didn't run:" on a ticket and alone in the run dialog alike, so it names no person.
var notConfiguredText = map[NotConfiguredReason]string{
	ReasonUnpaired:          "no computer is paired to run it on; pair one in Settings → T3 Code Setup",
	ReasonExpiredToken:      "the paired computer's session expired; pair it again in Settings → T3 Code Setup",
	ReasonNoDefault:         "no T3 project is picked for this project on the paired computer; link one in Settings → T3 Code Setup → Projects, or set a fallback under Defaults",
	ReasonNoDefaultComputer: "several computers are paired and none is the default; pick one under Defaults in Settings → T3 Code Setup",
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

// pairingSecret reads the token out of a pasted pairing link (`<origin>/pair#token=<token>`, what `t3 pair` prints and T3 Code's Share panel copies); anything else is the token itself.
func pairingSecret(raw string) string {
	raw = strings.TrimSpace(raw)
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return raw
	}
	fragment, _ := url.ParseQuery(u.Fragment)
	return strings.TrimSpace(fragment.Get("token"))
}
