// Package pairing implements the user-level half of @Agent: paired computers, their harness sessions, and defaults.
package pairing

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/otal-labs/nexul/internal/harness"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/permissions"
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
	// Shared is set on another person's computer shared with the caller by name (ADR 0148).
	Shared *SharedComputer `json:"shared,omitempty"`
	// Position is the computer's place in its owner's order, first runs first (ADR 0165).
	Position int `json:"position"`
	// RunsAtOnce is how many play runs the computer takes at a time (ADR 0166).
	RunsAtOnce int `json:"runs_at_once"`
	// ChatProjectID is the T3 project a chat outside any project runs in on this computer.
	ChatProjectID string `json:"chat_project_id"`
	// Running is how many play runs are going on it; set on a list only.
	Running int `json:"running"`
}

// SharedComputer is whose a shared computer is and what the caller holds on it.
type SharedComputer struct {
	OwnerID     string          `json:"owner_id"`
	OwnerName   string          `json:"owner_name"`
	Permissions permissions.Set `json:"permissions"`
}

// SharedName is how a person a computer is shared with reads it: "<owner>'s <computer> (shared)".
func (c Computer) SharedName() string {
	if c.Shared == nil {
		return c.Name
	}
	return fmt.Sprintf("%s's %s (shared)", c.Shared.OwnerName, c.Name)
}

// sharedView is c with only its name, who shares it and its runner's connection: whether it is connected, and since when it last was.
func sharedView(c Computer) Computer {
	out := Computer{ID: c.ID, Name: c.Name, Shared: c.Shared, RunsAtOnce: c.RunsAtOnce}
	if c.Runner != nil {
		out.Runner = &ComputerRunner{Connected: c.Runner.Connected, LastSeen: c.Runner.LastSeen}
	}
	return out
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
	Docker        *DockerFacts      `json:"docker,omitempty"`
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

// DockerFacts is Docker on the computer as its runner found it; nil from a runner older than container logs.
type DockerFacts struct {
	Available bool   `json:"available"`
	Version   string `json:"version,omitempty"`
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

// FieldError names the input a failure belongs to, so a form can show it on that field.
type FieldError struct {
	Field string
	Err   error
}

func (e *FieldError) Error() string { return e.Err.Error() }

func (e *FieldError) Unwrap() error { return e.Err }

// Paired reports whether the computer holds a harness session.
func (c Computer) Paired() bool { return !c.TokenExpiresAt.IsZero() }

// Session is the harness-facing view of a decrypted computer; one added with a runner is reached through it (ADR 0146).
func (c Computer) Session() harness.Session {
	serverURL := c.ServerURL
	if serverURL == "" {
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
	// SkillsVersion is the shipped skills version the confirming setup installed; empty for confirmations older than versioning.
	SkillsVersion string `json:"skills_version"`
	// SkillsOutdated is a signal, never a gate (ADR 0063): a skills update or a setup re-run refreshes the skill.
	SkillsOutdated bool `json:"skills_outdated"`
}

// outdated is a confirmed provider whose recorded skills are not the ones this release ships.
func (p ProviderSetup) outdated() bool {
	return p.ConfirmedAt != nil && p.SkillsVersion != shipped.SetVersion
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

// ProviderOption is a provider instance as the pickers list it: shown but not runnable while it needs setup, hidden once switched off.
type ProviderOption struct {
	harness.Provider
	NeedsSetup bool `json:"needs_setup"`
	// Skipped is a driver its owner switched off in the Set up step, so pickers leave it out.
	Skipped bool `json:"skipped"`
}

// setupConfirmed is the gate's rule: the computer's overall confirmation and the driver's own are both set, and the driver is switched on.
func setupConfirmed(c Computer, setups []ProviderSetup, driver string) bool {
	if c.SetupConfirmedAt == nil {
		return false
	}
	driver = strings.ToLower(strings.TrimSpace(driver))
	if switchedOff(c, driver) {
		return false
	}
	return slices.ContainsFunc(setups, func(p ProviderSetup) bool { return p.Provider == driver && p.ConfirmedAt != nil })
}

// switchedOff is whether the Set up step switched driver off on c.
func switchedOff(c Computer, driver string) bool {
	return slices.ContainsFunc(c.SetupChoices.Skipped, func(d string) bool { return strings.EqualFold(d, strings.TrimSpace(driver)) })
}

// Defaults are what a person's runs use wherever a project link says nothing; every field is optional.
type Defaults struct {
	UserID   string `json:"-"`
	Provider string `json:"provider,omitempty"`
	Model    string `json:"model,omitempty"`
	// ModelOptions are the options picked with Model; unset ones are the harness's defaults.
	ModelOptions []harness.OptionSetting `json:"model_options,omitempty"`
	// StartIn is where new harness threads start; empty is the T3 project's folder.
	StartIn StartIn `json:"start_in,omitempty"`
	// Placement is how a run picks among ready computers; empty is in order.
	Placement Placement `json:"placement"`
	// CloneAutomatically clones a project's repository onto a computer that has no checkout of it, before a run there.
	CloneAutomatically bool `json:"clone_automatically"`
}

// ProjectLink is one person's settings for one project (ADRs 0102, 0165); a zero value follows their defaults and order.
type ProjectLink struct {
	UserID       string                  `json:"-"`
	ProjectID    string                  `json:"project_id"`
	Provider     string                  `json:"provider,omitempty"`
	Model        string                  `json:"model,omitempty"`
	ModelOptions []harness.OptionSetting `json:"model_options,omitempty"`
	// StartIn overrides the defaults' StartIn for this project; empty falls through to them.
	StartIn StartIn `json:"start_in,omitempty"`
	// Placement overrides the defaults' for a project with its own Computers; empty follows them.
	Placement Placement `json:"placement,omitempty"`
	// Computers is the project's own order of the person's computers; empty follows their order.
	Computers []string `json:"computers"`
	// Checkouts are the T3 projects the person picked for this project, one per computer.
	Checkouts []Checkout `json:"checkouts"`
	UpdatedAt time.Time  `json:"updated_at,omitempty"`
}

// Checkout is the T3 project a person picked for a project on one computer.
type Checkout struct {
	ComputerID       string `json:"computer_id"`
	HarnessProjectID string `json:"harness_project_id"`
}

// checkoutOn is the checkout picked on computerID, "" when none was.
func (l ProjectLink) checkoutOn(computerID string) string {
	for _, c := range l.Checkouts {
		if c.ComputerID == computerID {
			return c.HarnessProjectID
		}
	}
	return ""
}

// Placement is how a run picks among a person's ready computers (ADR 0165).
type Placement string

const (
	// PlacementInOrder takes the first ready computer with room.
	PlacementInOrder Placement = "in_order"
	// PlacementSpread takes the ready computer least busy for its runs at once, ties by order.
	PlacementSpread Placement = "spread"
)

// validatePlacement accepts the two placements and empty, which defers to the next level down.
func validatePlacement(p Placement) (Placement, error) {
	p = Placement(strings.TrimSpace(string(p)))
	if p == "" || p == PlacementInOrder || p == PlacementSpread {
		return p, nil
	}
	return "", fmt.Errorf("%w: placement must be %q or %q", apperrs.ErrInvalid, PlacementInOrder, PlacementSpread)
}

// MinRunsAtOnce and MaxRunsAtOnce bound a computer's runs at once (ADR 0166).
const (
	MinRunsAtOnce = 1
	MaxRunsAtOnce = 5
)

// PassOverReason is why a computer could not take a run.
type PassOverReason string

const (
	PassOverOffline           PassOverReason = "offline"
	PassOverExpiredToken      PassOverReason = "expired_token"
	PassOverSetupRequired     PassOverReason = "setup_required"
	PassOverNoModel           PassOverReason = "no_model"
	PassOverNoCheckout        PassOverReason = "no_checkout"
	PassOverAmbiguousCheckout PassOverReason = "ambiguous_checkout"
	PassOverNoChatFolder      PassOverReason = "no_chat_folder"
	PassOverFull              PassOverReason = "full"
	PassOverCloneFailed       PassOverReason = "clone_failed"
)

// PassOver is one computer a run passed over and why; Text says it in a line every client shows as is.
type PassOver struct {
	ComputerID string         `json:"computer_id"`
	Name       string         `json:"name"`
	Why        PassOverReason `json:"why"`
	Text       string         `json:"text"`
}

// missingProject is whether the computer was passed over only for not having the project.
func (p PassOver) missingProject() bool {
	return p.Why == PassOverNoCheckout || p.Why == PassOverAmbiguousCheckout || p.Why == PassOverCloneFailed
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
	// ReasonNoComputerReady: none of the person's computers can take the run; PassedOver says why for each (ADR 0165).
	ReasonNoComputerReady NotConfiguredReason = "no_computer_ready"
	// ReasonSetupRequired: the computer or the provider the run lands on has no setup confirmation (ADR 0063).
	ReasonSetupRequired NotConfiguredReason = "setup_required"
	// ReasonOffline: the resolved computer's harness did not answer, so the setup gate could not check its providers.
	ReasonOffline NotConfiguredReason = "offline"
	// ReasonNeedsLocation: no computer of the person's has the project, so their run asks where (ADR 0165).
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
	// Why says what is offline for ReasonOffline when Nexul knows more than that T3 Code did not answer, and for
	// ReasonSetupRequired that the provider is switched off rather than not set up.
	Why string
	// PassedOver names every computer the run could not land on and why.
	PassedOver []PassOver
}

// Error is the user-facing refusal for the gate's reasons, so chat, the play run dialog, and MCP all read the same line.
func (e *NotConfiguredError) Error() string {
	if e.Reason == ReasonSetupRequired && e.Why != "" {
		return fmt.Sprintf("@Agent can't use %s on %s: %s. Switch it on in its setup in Settings → Computers.", e.Provider, e.Computer, e.Why)
	}
	if e.Reason == ReasonSetupRequired {
		return fmt.Sprintf("@Agent can't use %s on %s until its setup is done — run setup for %s in Settings → Computers.", e.Provider, e.Computer, e.Computer)
	}
	if e.Reason == ReasonOffline {
		why := e.Why
		if why == "" {
			why = "T3 Code isn't answering there"
		}
		return fmt.Sprintf("%s is offline: %s.", e.Computer, why)
	}
	if e.Reason == ReasonNeedsLocation {
		return "None of your computers has this project. Pick a computer and its T3 project; it's remembered for this project, " +
			"and Settings → Computers → Projects can change it."
	}
	if e.Reason == ReasonNoComputerReady {
		return "none of your computers can take it: " + passOverLines(e.PassedOver)
	}
	if text, ok := notConfiguredText[e.Reason]; ok {
		return text
	}
	return fmt.Sprintf("pairing not configured: %s", e.Reason)
}

// notConfiguredText reads after "didn't run:" on a ticket and alone in the run dialog alike, so it names no person.
var notConfiguredText = map[NotConfiguredReason]string{
	ReasonUnpaired:     "no computer is paired to run it on; add one in Settings → Computers",
	ReasonExpiredToken: "the paired computer's session expired; pair it again in Settings → Computers",
}

// passOverLines joins each computer's reason into one sentence's worth: "laptop is offline; build-box is full (2 of 2)."
func passOverLines(ps []PassOver) string {
	lines := make([]string, 0, len(ps))
	for _, p := range ps {
		lines = append(lines, p.Text)
	}
	return strings.Join(lines, "; ") + "."
}

// RefusalDetails is a NotConfiguredError's machine-readable side, carried beside the message in the error envelope.
type RefusalDetails struct {
	Reason     NotConfiguredReason `json:"reason"`
	ComputerID string              `json:"computer_id,omitempty"`
	Computer   string              `json:"computer,omitempty"`
	ProviderID string              `json:"provider_id,omitempty"`
	Provider   string              `json:"provider,omitempty"`
	PassedOver []PassOver          `json:"passed_over,omitempty"`
}

// ErrorDetails satisfies httpx.DetailedError.
func (e *NotConfiguredError) ErrorDetails() any {
	return RefusalDetails{Reason: e.Reason, ComputerID: e.ComputerID, Computer: e.Computer, ProviderID: e.ProviderID, Provider: e.Provider, PassedOver: e.PassedOver}
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
