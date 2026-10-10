package pairing

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/otal-labs/nexul/internal/harness"
	"github.com/otal-labs/nexul/internal/platform/crypto"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/ids"
	"github.com/otal-labs/nexul/internal/platform/permissions"
	shipped "github.com/otal-labs/nexul/internal/platform/skills"
)

// ProjectGate answers whether the caller may open a project; an unknown or hidden one is ErrNotFound (ADR 0097).
type ProjectGate interface {
	RequireProject(ctx context.Context, projectID string, action permissions.Action) error
}

// Config wires the pairing use-cases.
type Config struct {
	// Repo persists computers and defaults.
	Repo Repo
	// Harnesses holds one client per supported harness kind; pairing, listing and presence all route through it.
	Harnesses harness.Registry
	// EncryptionKey seals stored bearer tokens at rest.
	EncryptionKey []byte
	// Now is overridable for tests.
	Now func() time.Time
	// OnComputersChanged lets the presence keeper reconcile without waiting for a browser reconnect. Optional.
	OnComputersChanged func(userID string)
	// Tunnels creates and removes computer tunnels; nil leaves only URL pairing working.
	Tunnels Tunnels
	// Bus carries the tunnel watch's status changes; nil keeps the watch silent.
	Bus Publisher
	// Tokens mints and revokes each computer's MCP token.
	Tokens MCPTokens
	// Instance reads the instance URL the setup turns point each provider's MCP entry at.
	Instance InstanceSettings
	// Projects keeps a person's project links to the projects they can open.
	Projects ProjectGate
}

// Service is the pairing use-case layer (ADR 0019); bearer tokens never leave it except encrypted at rest.
type Service struct {
	repo      Repo
	harnesses harness.Registry
	key       []byte
	now       func() time.Time
	changed   func(userID string)
	tunnels   Tunnels
	bus       Publisher
	tokens    MCPTokens
	instance  InstanceSettings
	projects  ProjectGate
	watchMu   sync.Mutex
	watching  map[string]tunnelWatch
	setupMu   sync.Mutex
	settingUp map[string]bool
	setupRuns sync.WaitGroup
}

// NewService wires the pairing use-cases.
func NewService(cfg Config) *Service {
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	return &Service{
		repo: cfg.Repo, harnesses: cfg.Harnesses, key: cfg.EncryptionKey, now: cfg.Now, changed: cfg.OnComputersChanged,
		tunnels: cfg.Tunnels, bus: cfg.Bus, tokens: cfg.Tokens, instance: cfg.Instance, projects: cfg.Projects, watching: map[string]tunnelWatch{}, settingUp: map[string]bool{},
	}
}

// client returns the registered client for kind; an unregistered kind on a stored computer is a wiring bug.
func (s *Service) client(kind harness.Kind) (harness.Client, error) {
	c, ok := s.harnesses[kind]
	if !ok {
		return nil, apperrs.Fatal(fmt.Errorf("%w: no harness client registered for kind %q", apperrs.ErrFatal, kind))
	}
	return c, nil
}

func (s *Service) notifyComputersChanged(userID string) {
	if s.changed != nil {
		s.changed(userID)
	}
}

// ListComputers returns the caller's paired computers, newest first.
func (s *Service) ListComputers(ctx context.Context, userID string) ([]Computer, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, fmt.Errorf("%w: user is required", apperrs.ErrUnauthorized)
	}
	cs, err := s.repo.ListComputers(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list computers: %w", err)
	}
	for i := range cs {
		cs[i].BearerToken = ""
	}
	return cs, nil
}

// sessionComputer decrypts a bearer token for a listing call only; it must never reach a response.
func (s *Service) sessionComputer(ctx context.Context, userID, computerID string) (Computer, error) {
	if strings.TrimSpace(userID) == "" {
		return Computer{}, fmt.Errorf("%w: user is required", apperrs.ErrUnauthorized)
	}
	if strings.TrimSpace(computerID) == "" {
		return Computer{}, fmt.Errorf("%w: computer id is required", apperrs.ErrInvalid)
	}
	computer, err := s.repo.GetComputer(ctx, userID, computerID)
	if err != nil {
		return Computer{}, fmt.Errorf("get computer %s: %w", computerID, err)
	}
	if !computer.Paired() {
		return Computer{}, fmt.Errorf("%w: this computer is still pairing — pair T3 Code on it first", apperrs.ErrInvalid)
	}
	if computer.TokenExpiresAt.Before(s.now()) {
		return Computer{}, fmt.Errorf("%w: this computer's session has expired — re-pair it first", apperrs.ErrInvalid)
	}
	decrypted, err := crypto.Decrypt(s.key, computer.BearerToken)
	if err != nil {
		return Computer{}, fmt.Errorf("decrypt bearer token for computer %s: %w", computer.ID, err)
	}
	session := *computer
	session.BearerToken = string(decrypted)
	return session, nil
}

// ListProjects fetches a paired computer's project registry for the settings UI's picker.
func (s *Service) ListProjects(ctx context.Context, userID, computerID string) ([]harness.Project, error) {
	session, err := s.sessionComputer(ctx, userID, computerID)
	if err != nil {
		return nil, err
	}
	client, err := s.client(session.Kind)
	if err != nil {
		return nil, err
	}
	projects, err := client.ListProjects(ctx, session.Session())
	if err != nil {
		return nil, fmt.Errorf("list projects on %s: %w", session.Name, err)
	}
	return projects, nil
}

// ListProviders fetches a paired computer's usable provider instances and models, each tagged with whether it needs setup.
func (s *Service) ListProviders(ctx context.Context, userID, computerID string) ([]ProviderOption, error) {
	session, err := s.sessionComputer(ctx, userID, computerID)
	if err != nil {
		return nil, err
	}
	providers, err := s.harnessProviders(ctx, session)
	if err != nil {
		return nil, err
	}
	setups, err := s.repo.ListProviderSetups(ctx, session.ID)
	if err != nil {
		return nil, fmt.Errorf("list provider setups for %s: %w", session.ID, err)
	}
	out := make([]ProviderOption, 0, len(providers))
	for _, p := range providers {
		out = append(out, ProviderOption{Provider: p, NeedsSetup: !setupConfirmed(session, setups, p.Driver)})
	}
	return out, nil
}

// harnessProviders asks a decrypted computer's harness for its provider instances.
func (s *Service) harnessProviders(ctx context.Context, session Computer) ([]harness.Provider, error) {
	client, err := s.client(session.Kind)
	if err != nil {
		return nil, err
	}
	providers, err := client.ListProviders(ctx, session.Session())
	if err != nil {
		return nil, fmt.Errorf("list providers on %s: %w", session.Name, err)
	}
	return providers, nil
}

// Pair trades a harness-specific secret (T3: the `t3 pair` token) for a bearer session; nothing persists until it succeeds.
func (s *Service) Pair(ctx context.Context, userID string, kind harness.Kind, name, serverURL, secret string) (*Computer, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, fmt.Errorf("%w: user is required", apperrs.ErrUnauthorized)
	}
	return s.pair(ctx, Computer{UserID: userID, Kind: kind, Name: name}, serverURL, secret)
}

// PairComputer pairs the harness at an existing computer's own address, over its tunnel hostname when it has one.
func (s *Service) PairComputer(ctx context.Context, userID, computerID, secret string) (*Computer, error) {
	existing, err := s.ownComputer(ctx, userID, computerID)
	if err != nil {
		return nil, err
	}
	return s.pair(ctx, *existing, existing.address(), secret)
}

// Repair re-runs the pairing flow against an existing computer row, updating it in place; the kind only moves forward.
func (s *Service) Repair(ctx context.Context, userID, id, name, serverURL, secret string) (*Computer, error) {
	existing, err := s.ownComputer(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	if existing.Tunnel != nil && strings.TrimRight(strings.TrimSpace(serverURL), "/") != existing.address() {
		return nil, &FieldError{Field: "server_url", Err: fmt.Errorf("%w: %s is reached through its tunnel at %s", apperrs.ErrInvalid, existing.Name, existing.address())}
	}
	existing.Name = name
	return s.pair(ctx, *existing, serverURL, secret)
}

// pair exchanges the secret at serverURL and saves base with the new session; base keeps its id, tunnel, and setup.
func (s *Service) pair(ctx context.Context, base Computer, serverURL, secret string) (*Computer, error) {
	client, ok := s.harnesses[base.Kind]
	if !ok {
		return nil, fmt.Errorf("%w: unsupported harness kind %q", apperrs.ErrInvalid, base.Kind)
	}
	name, err := validateName(base.Name)
	if err != nil {
		return nil, &FieldError{Field: "name", Err: err}
	}
	serverURL, err = validateServerURL(serverURL)
	if err != nil {
		return nil, &FieldError{Field: "server_url", Err: err}
	}
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return nil, &FieldError{Field: "token", Err: fmt.Errorf("%w: pairing token is required", apperrs.ErrInvalid)}
	}
	if len(s.key) == 0 {
		return nil, apperrs.Fatal(fmt.Errorf("%w: pairing encryption key is not configured", apperrs.ErrFatal))
	}
	result, err := client.Pair(ctx, serverURL, secret)
	if err != nil {
		return nil, pairFailure(serverURL, err)
	}
	kind := base.Kind
	if result.Kind != "" {
		kind = result.Kind
	}
	if base.ID != "" && kindStep(kind) < kindStep(base.Kind) {
		return nil, fmt.Errorf("%w: %s already moved forward to a newer harness protocol, and Nexul never moves a computer back", apperrs.ErrConflict, name)
	}
	encrypted, err := crypto.Encrypt(s.key, []byte(result.BearerToken))
	if err != nil {
		return nil, fmt.Errorf("encrypt bearer token: %w", err)
	}
	now := s.now().UTC()
	computer := base
	computer.Kind = kind
	computer.Name = name
	computer.ServerURL = serverURL
	computer.BearerToken = encrypted
	computer.TokenExpiresAt = now.Add(result.ExpiresIn)
	computer.HarnessVersion = result.Version
	computer.UpdatedAt = now
	if computer.ID == "" {
		computer.ID = ids.New()
		computer.CreatedAt = now
	}
	evts := []eventbus.OutboxEvent{{ID: ids.New(), Topic: TopicComputerPaired, Payload: ComputerPairedEvent{
		ComputerID: computer.ID, UserID: computer.UserID, ServerURL: serverURL, HarnessVersion: result.Version, TokenExpiresAt: computer.TokenExpiresAt,
	}}}
	if base.ID != "" && kindStep(kind) > kindStep(base.Kind) {
		evts = append(evts, eventbus.OutboxEvent{ID: ids.New(), Topic: TopicHarnessSwitched, Payload: HarnessSwitchedEvent{
			ComputerID: computer.ID, UserID: computer.UserID, FromKind: base.Kind, ToKind: kind, HarnessVersion: result.Version,
		}})
	}
	if err := s.repo.SaveComputer(ctx, computer, evts...); err != nil {
		return nil, fmt.Errorf("save computer: %w", err)
	}
	s.notifyComputersChanged(computer.UserID)
	computer.BearerToken = ""
	return &computer, nil
}

// pairFailure blames the address for a harness it could not reach or follow, and the token when the harness refused it.
func pairFailure(serverURL string, err error) error {
	if errors.Is(err, harness.ErrProtocol) {
		return &FieldError{Field: "server_url", Err: err}
	}
	if errors.Is(err, apperrs.ErrRetryable) {
		return &FieldError{Field: "server_url", Err: fmt.Errorf("couldn't reach the harness at %s: %w", serverURL, err)}
	}
	return &FieldError{Field: "token", Err: fmt.Errorf("the harness refused this token, get a fresh one from T3 Code on the computer: %w", err)}
}

// kindOrder is the order a computer's kind moves in, one step at a time and never back (ADR 0113).
var kindOrder = []harness.Kind{harness.KindT3Code, harness.KindT3CodeV2}

func kindStep(kind harness.Kind) int {
	return slices.Index(kindOrder, kind)
}

// SwitchHarness is harness.Forward's moved callback: sess's computer moves up to kind to, once (ADR 0113).
func (s *Service) SwitchHarness(ctx context.Context, sess harness.Session, to harness.Kind) error {
	step := kindStep(to)
	if step < 1 {
		return fmt.Errorf("%w: no computer moves forward to harness kind %q", apperrs.ErrInvalid, to)
	}
	client, err := s.client(to)
	if err != nil {
		return err
	}
	version, err := client.Version(ctx, sess.ServerURL)
	if err != nil {
		return err
	}
	from := kindOrder[step-1]
	// The repo asks for the event only when the row changed, so a computer already moved notifies nobody.
	owner := ""
	evt := func(userID string) eventbus.OutboxEvent {
		owner = userID
		return eventbus.OutboxEvent{ID: ids.New(), Topic: TopicHarnessSwitched, Payload: HarnessSwitchedEvent{
			ComputerID: sess.ComputerID, UserID: userID, FromKind: from, ToKind: to, HarnessVersion: version,
		}}
	}
	if err := s.repo.SwitchComputerKind(ctx, sess.ComputerID, from, to, version, s.now().UTC(), evt); err != nil {
		return fmt.Errorf("switch computer %s to %s: %w", sess.ComputerID, to, err)
	}
	if owner != "" {
		s.notifyComputersChanged(owner)
	}
	return nil
}

// ActiveSessions hands out plaintext bearer tokens for the presence keeper; never expose on a response.
func (s *Service) ActiveSessions(ctx context.Context, userID string) ([]Computer, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, fmt.Errorf("%w: user is required", apperrs.ErrUnauthorized)
	}
	if len(s.key) == 0 {
		return nil, apperrs.Fatal(fmt.Errorf("%w: pairing encryption key is not configured", apperrs.ErrFatal))
	}
	cs, err := s.repo.ListComputers(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list computers: %w", err)
	}
	now := s.now()
	out := make([]Computer, 0, len(cs))
	for _, c := range cs {
		if !c.TokenExpiresAt.After(now) {
			continue
		}
		decrypted, err := crypto.Decrypt(s.key, c.BearerToken)
		if err != nil {
			return nil, fmt.Errorf("decrypt bearer token for computer %s: %w", c.ID, err)
		}
		c.BearerToken = string(decrypted)
		out = append(out, c)
	}
	return out, nil
}

// GetDefaults returns the caller's pairing defaults, zero-valued when unset.
func (s *Service) GetDefaults(ctx context.Context, userID string) (Defaults, error) {
	if strings.TrimSpace(userID) == "" {
		return Defaults{}, fmt.Errorf("%w: user is required", apperrs.ErrUnauthorized)
	}
	d, err := s.repo.GetDefaults(ctx, userID)
	if err != nil {
		return Defaults{}, fmt.Errorf("get defaults: %w", err)
	}
	d.UserID = userID
	return d, nil
}

// SetDefaults stores the caller's pairing defaults; a default computer must be one of their own.
func (s *Service) SetDefaults(ctx context.Context, userID string, d Defaults) (Defaults, error) {
	if strings.TrimSpace(userID) == "" {
		return Defaults{}, fmt.Errorf("%w: user is required", apperrs.ErrUnauthorized)
	}
	d.UserID = userID
	d.DefaultComputerID = strings.TrimSpace(d.DefaultComputerID)
	if d.DefaultComputerID != "" {
		if _, err := s.repo.GetComputer(ctx, userID, d.DefaultComputerID); err != nil {
			return Defaults{}, fmt.Errorf("default computer %s: %w", d.DefaultComputerID, err)
		}
	}
	d.FallbackProjectID = strings.TrimSpace(d.FallbackProjectID)
	d.Provider = strings.TrimSpace(d.Provider)
	d.Model = strings.TrimSpace(d.Model)
	options, err := harness.CleanOptions(d.ModelOptions)
	if err != nil {
		return Defaults{}, err
	}
	d.ModelOptions = options
	if d.StartIn, err = validateStartIn(d.StartIn); err != nil {
		return Defaults{}, err
	}
	if err := s.repo.SaveDefaults(ctx, d); err != nil {
		return Defaults{}, fmt.Errorf("save defaults: %w", err)
	}
	return d, nil
}

// GetProjectLink returns the caller's own link for a project they can open, zero-valued when they never set one.
func (s *Service) GetProjectLink(ctx context.Context, userID, projectID string) (ProjectLink, error) {
	projectID, err := s.openProject(ctx, userID, projectID)
	if err != nil {
		return ProjectLink{}, err
	}
	link, err := s.repo.GetProjectLink(ctx, userID, projectID)
	if err != nil {
		return ProjectLink{}, fmt.Errorf("get project link %s: %w", projectID, err)
	}
	link.UserID, link.ProjectID = userID, projectID
	return link, nil
}

// ListProjectLinks returns the caller's own links for every project they can still open.
func (s *Service) ListProjectLinks(ctx context.Context, userID string) ([]ProjectLink, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, fmt.Errorf("%w: user is required", apperrs.ErrUnauthorized)
	}
	links, err := s.repo.ListProjectLinks(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list project links: %w", err)
	}
	out := make([]ProjectLink, 0, len(links))
	for _, link := range links {
		_, err := s.openProject(ctx, userID, link.ProjectID)
		if errors.Is(err, apperrs.ErrNotFound) || errors.Is(err, apperrs.ErrForbidden) {
			continue
		}
		if err != nil {
			return nil, err
		}
		out = append(out, link)
	}
	return out, nil
}

// SetProjectLink sets the caller's own link for a project they can open; the computer must be one of theirs.
func (s *Service) SetProjectLink(ctx context.Context, userID, projectID string, link ProjectLink) (ProjectLink, error) {
	projectID, err := s.openProject(ctx, userID, projectID)
	if err != nil {
		return ProjectLink{}, err
	}
	link.ComputerID = strings.TrimSpace(link.ComputerID)
	if link.ComputerID == "" {
		return ProjectLink{}, fmt.Errorf("%w: computer is required", apperrs.ErrInvalid)
	}
	if _, err := s.repo.GetComputer(ctx, userID, link.ComputerID); err != nil {
		return ProjectLink{}, fmt.Errorf("link computer %s: %w", link.ComputerID, err)
	}
	harnessProjectID, err := validateHarnessProjectID(link.HarnessProjectID)
	if err != nil {
		return ProjectLink{}, err
	}
	link.UserID, link.ProjectID = userID, projectID
	link.HarnessProjectID = harnessProjectID
	link.Provider = strings.TrimSpace(link.Provider)
	link.Model = strings.TrimSpace(link.Model)
	link.ModelOptions, err = harness.CleanOptions(link.ModelOptions)
	if err != nil {
		return ProjectLink{}, err
	}
	if link.StartIn, err = validateStartIn(link.StartIn); err != nil {
		return ProjectLink{}, err
	}
	link.UpdatedAt = s.now().UTC()
	if err := s.repo.SaveProjectLink(ctx, link); err != nil {
		return ProjectLink{}, fmt.Errorf("save project link %s: %w", projectID, err)
	}
	return link, nil
}

// ClearProjectLink removes the caller's own link; their turns in that project fall back to their defaults.
func (s *Service) ClearProjectLink(ctx context.Context, userID, projectID string) error {
	projectID, err := s.openProject(ctx, userID, projectID)
	if err != nil {
		return err
	}
	if err := s.repo.DeleteProjectLink(ctx, userID, projectID); err != nil {
		return fmt.Errorf("clear project link %s: %w", projectID, err)
	}
	return nil
}

// openProject checks the caller and that they may open projectID, returning it trimmed.
func (s *Service) openProject(ctx context.Context, userID, projectID string) (string, error) {
	if strings.TrimSpace(userID) == "" {
		return "", fmt.Errorf("%w: user is required", apperrs.ErrUnauthorized)
	}
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return "", fmt.Errorf("%w: project id is required", apperrs.ErrInvalid)
	}
	if s.projects == nil {
		return "", apperrs.Fatal(fmt.Errorf("%w: project access is not wired into pairing", apperrs.ErrFatal))
	}
	if err := s.projects.RequireProject(ctx, projectID, permissions.Member); err != nil {
		return "", fmt.Errorf("project %s: %w", projectID, err)
	}
	return projectID, nil
}

// ResolvedTarget is what a chat mention needs to run a turn; BearerToken is plaintext here.
type ResolvedTarget struct {
	Computer         Computer
	HarnessProjectID string
	Provider         string
	Model            string
	ModelOptions     []harness.OptionSetting
	// Worktree starts a new harness thread in a fresh git worktree instead of the T3 project's folder.
	Worktree bool
}

// modelPick keeps a model's options beside it: they only mean something for the model they were picked with.
type modelPick struct {
	provider string
	model    string
	options  []harness.OptionSetting
}

// ResolveTarget picks the computer/harness project/provider/model a mention runs against, behind the setup gate.
func (s *Service) ResolveTarget(ctx context.Context, userID, projectID string) (*ResolvedTarget, error) {
	target, err := s.resolveTarget(ctx, userID, projectID)
	if err != nil {
		return nil, err
	}
	return s.requireSetup(ctx, target)
}

// ResolveTargetOverride resolves a run's pinned computer, provider, model and its options, behind the same setup gate.
func (s *Service) ResolveTargetOverride(ctx context.Context, userID, projectID, computerID, provider, model string, options []harness.OptionSetting) (*ResolvedTarget, error) {
	target, err := s.resolveTargetOverride(ctx, userID, projectID, computerID, modelPick{provider: provider, model: model, options: options})
	if err != nil {
		return nil, err
	}
	return s.requireSetup(ctx, target)
}

// PreviewTarget is ResolveTarget without the gate or bearer token, so readiness never greys a play that refuses on press.
func (s *Service) PreviewTarget(ctx context.Context, userID, projectID string) (*ResolvedTarget, error) {
	target, err := s.resolveTarget(ctx, userID, projectID)
	if err != nil {
		return nil, err
	}
	target.Computer.BearerToken = ""
	return target, nil
}

// ResolveSetupTurnTarget skips the setup gate for the wizard's setup turn only; TestResolveSetupTurnTarget_OnlyTheWizardCallsIt guards it.
// A picked projectID wins over the defaults.
func (s *Service) ResolveSetupTurnTarget(ctx context.Context, userID, computerID, provider, projectID string) (*ResolvedTarget, error) {
	if strings.TrimSpace(computerID) == "" {
		return nil, fmt.Errorf("%w: a setup turn needs its computer", apperrs.ErrInvalid)
	}
	if projectID != "" {
		session, err := s.sessionComputer(ctx, userID, computerID)
		if err != nil {
			return nil, err
		}
		return &ResolvedTarget{Computer: session, HarnessProjectID: projectID, Provider: strings.TrimSpace(provider)}, nil
	}
	target, err := s.resolveTargetOverride(ctx, userID, "", computerID, modelPick{provider: provider})
	var nc *NotConfiguredError
	if !errors.As(err, &nc) || nc.Reason != ReasonNoDefault {
		return target, err
	}
	return s.setupTurnInFirstProject(ctx, userID, computerID, provider)
}

// setupTurnInFirstProject runs a setup turn in the harness's first project when none is linked: it only touches user-level files.
func (s *Service) setupTurnInFirstProject(ctx context.Context, userID, computerID, provider string) (*ResolvedTarget, error) {
	session, err := s.sessionComputer(ctx, userID, computerID)
	if err != nil {
		return nil, err
	}
	client, err := s.client(session.Kind)
	if err != nil {
		return nil, err
	}
	projects, err := client.ListProjects(ctx, session.Session())
	if err != nil {
		return nil, fmt.Errorf("list projects on %s: %w", session.Name, err)
	}
	if len(projects) == 0 {
		return nil, fmt.Errorf("%w: open any project in T3 Code on %s first; setup runs its turns inside one", apperrs.ErrInvalid, session.Name)
	}
	return &ResolvedTarget{Computer: session, HarnessProjectID: projects[0].ID, Provider: strings.TrimSpace(provider)}, nil
}

// requireSetup is the setup gate (ADR 0063); an empty provider is pinned to the harness's first so the checked one runs.
func (s *Service) requireSetup(ctx context.Context, target *ResolvedTarget) (*ResolvedTarget, error) {
	client, err := s.client(target.Computer.Kind)
	if err != nil {
		return nil, err
	}
	providers, err := client.ListProviders(ctx, target.Computer.Session())
	if errors.Is(err, harness.ErrProtocol) {
		return nil, err
	}
	if err != nil {
		return nil, &NotConfiguredError{Reason: ReasonOffline, Computer: target.Computer.Name, ComputerID: target.Computer.ID, Err: err}
	}
	// The call may have moved the computer to a newer kind; the turn and its version-drift check read what it holds now.
	stored, err := s.repo.GetComputer(ctx, target.Computer.UserID, target.Computer.ID)
	if err != nil {
		return nil, fmt.Errorf("get computer %s: %w", target.Computer.ID, err)
	}
	target.Computer.Kind, target.Computer.HarnessVersion = stored.Kind, stored.HarnessVersion
	provider, err := pickProvider(providers, target.Provider)
	if err != nil {
		return nil, err
	}
	setups, err := s.repo.ListProviderSetups(ctx, target.Computer.ID)
	if err != nil {
		return nil, fmt.Errorf("list provider setups for %s: %w", target.Computer.ID, err)
	}
	if !setupConfirmed(target.Computer, setups, provider.Driver) {
		return nil, &NotConfiguredError{
			Reason: ReasonSetupRequired, Provider: provider.Name, Computer: target.Computer.Name, ProviderID: provider.ID, ComputerID: target.Computer.ID,
		}
	}
	target.Provider = provider.ID
	return target, nil
}

// pickProvider finds the stored instance id among the harness's providers, or the harness default for an empty one.
func pickProvider(providers []harness.Provider, id string) (harness.Provider, error) {
	for _, p := range providers {
		if id == "" || p.ID == id {
			return p, nil
		}
	}
	if id == "" {
		return harness.Provider{}, fmt.Errorf("%w: the harness lists no usable provider", apperrs.ErrInvalid)
	}
	return harness.Provider{}, fmt.Errorf("%w: provider %s is not available on this computer", apperrs.ErrInvalid, id)
}

// resolveTarget is the resolution order with no setup gate; only the gated and preview entry points call it.
func (s *Service) resolveTarget(ctx context.Context, userID, projectID string) (*ResolvedTarget, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, fmt.Errorf("%w: user is required", apperrs.ErrUnauthorized)
	}
	if len(s.key) == 0 {
		return nil, apperrs.Fatal(fmt.Errorf("%w: pairing encryption key is not configured", apperrs.ErrFatal))
	}

	computerID, harnessProjectID, pick, err := s.resolveTargetSource(ctx, userID, projectID)
	if err != nil {
		return nil, err
	}

	computer, err := s.fetchTargetComputer(ctx, userID, computerID)
	if err != nil {
		return nil, err
	}
	if harnessProjectID == "" {
		return nil, &NotConfiguredError{Reason: ReasonNoDefault}
	}

	decrypted, err := crypto.Decrypt(s.key, computer.BearerToken)
	if err != nil {
		return nil, fmt.Errorf("decrypt bearer token for computer %s: %w", computer.ID, err)
	}
	worktree, err := s.startsInWorktree(ctx, userID, projectID)
	if err != nil {
		return nil, err
	}
	resolved := *computer
	resolved.BearerToken = string(decrypted)
	return &ResolvedTarget{
		Computer: resolved, HarnessProjectID: harnessProjectID, Provider: pick.provider, Model: pick.model, ModelOptions: pick.options, Worktree: worktree,
	}, nil
}

// startsInWorktree is the caller's project link's start_in, else their defaults'; a run's pinned computer keeps it.
func (s *Service) startsInWorktree(ctx context.Context, userID, projectID string) (bool, error) {
	if projectID = strings.TrimSpace(projectID); projectID != "" {
		link, err := s.repo.GetProjectLink(ctx, userID, projectID)
		if err != nil {
			return false, fmt.Errorf("get project link %s: %w", projectID, err)
		}
		if link.StartIn != "" {
			return link.StartIn == StartInWorktree, nil
		}
	}
	defaults, err := s.repo.GetDefaults(ctx, userID)
	if err != nil {
		return false, fmt.Errorf("get defaults: %w", err)
	}
	return defaults.StartIn == StartInWorktree, nil
}

// resolveTargetSource: the user's own project link, then their defaults, then their sole paired computer if unambiguous.
func (s *Service) resolveTargetSource(ctx context.Context, userID, projectID string) (computerID, harnessProjectID string, pick modelPick, err error) {
	if projectID = strings.TrimSpace(projectID); projectID != "" {
		link, err := s.repo.GetProjectLink(ctx, userID, projectID)
		if err != nil {
			return "", "", modelPick{}, fmt.Errorf("get project link %s: %w", projectID, err)
		}
		if link.ComputerID != "" {
			return link.ComputerID, link.HarnessProjectID, modelPick{link.Provider, link.Model, link.ModelOptions}, nil
		}
	}

	defaults, err := s.repo.GetDefaults(ctx, userID)
	if err != nil {
		return "", "", modelPick{}, fmt.Errorf("get defaults: %w", err)
	}
	computerID, harnessProjectID = defaults.DefaultComputerID, defaults.FallbackProjectID
	pick = modelPick{defaults.Provider, defaults.Model, defaults.ModelOptions}
	if computerID != "" {
		return computerID, harnessProjectID, pick, nil
	}

	computers, err := s.repo.ListComputers(ctx, userID)
	if err != nil {
		return "", "", modelPick{}, fmt.Errorf("list computers: %w", err)
	}
	if len(computers) == 0 {
		return "", "", modelPick{}, &NotConfiguredError{Reason: ReasonUnpaired}
	}
	if len(computers) > 1 {
		return "", "", modelPick{}, &NotConfiguredError{Reason: ReasonNoDefaultComputer}
	}
	return computers[0].ID, harnessProjectID, pick, nil
}

// resolveTargetOverride resolves a pinned computer, provider, and model with no setup gate: the computer must
// be the caller's own; the harness project and any blank provider/model come from their own project link when it
// names this computer, else the caller's own fallback when this is their default computer — the same sources
// resolveTarget's own resolution order reads, just checked against the caller's explicit choice of computer.
func (s *Service) resolveTargetOverride(ctx context.Context, userID, projectID, computerID string, pick modelPick) (*ResolvedTarget, error) {
	computerID = strings.TrimSpace(computerID)
	if computerID == "" {
		return s.resolveTarget(ctx, userID, projectID)
	}
	if strings.TrimSpace(userID) == "" {
		return nil, fmt.Errorf("%w: user is required", apperrs.ErrUnauthorized)
	}
	if len(s.key) == 0 {
		return nil, apperrs.Fatal(fmt.Errorf("%w: pairing encryption key is not configured", apperrs.ErrFatal))
	}
	computer, err := s.fetchTargetComputer(ctx, userID, computerID)
	if err != nil {
		return nil, err
	}
	pick.provider, pick.model = strings.TrimSpace(pick.provider), strings.TrimSpace(pick.model)
	harnessProjectID, pick, err := s.overrideProjectAndModel(ctx, userID, strings.TrimSpace(projectID), computerID, pick)
	if err != nil {
		return nil, err
	}
	if harnessProjectID == "" {
		return nil, &NotConfiguredError{Reason: ReasonNoDefault}
	}
	decrypted, err := crypto.Decrypt(s.key, computer.BearerToken)
	if err != nil {
		return nil, fmt.Errorf("decrypt bearer token for computer %s: %w", computer.ID, err)
	}
	worktree, err := s.startsInWorktree(ctx, userID, projectID)
	if err != nil {
		return nil, err
	}
	resolved := *computer
	resolved.BearerToken = string(decrypted)
	return &ResolvedTarget{
		Computer: resolved, HarnessProjectID: harnessProjectID, Provider: pick.provider, Model: pick.model, ModelOptions: pick.options, Worktree: worktree,
	}, nil
}

// overrideProjectAndModel fills in the harness project and any blank provider/model from the caller's project link
// (when it names computerID) and the caller's own defaults (when computerID is their default computer).
func (s *Service) overrideProjectAndModel(ctx context.Context, userID, projectID, computerID string, pick modelPick) (string, modelPick, error) {
	harnessProjectID := ""
	if projectID != "" {
		link, err := s.repo.GetProjectLink(ctx, userID, projectID)
		if err != nil {
			return "", modelPick{}, fmt.Errorf("get project link %s: %w", projectID, err)
		}
		if link.ComputerID == computerID {
			harnessProjectID = link.HarnessProjectID
			pick = fillPick(pick, modelPick{link.Provider, link.Model, link.ModelOptions})
		}
	}
	if harnessProjectID != "" && pick.provider != "" && pick.model != "" {
		return harnessProjectID, pick, nil
	}
	defaults, err := s.repo.GetDefaults(ctx, userID)
	if err != nil {
		return "", modelPick{}, fmt.Errorf("get defaults: %w", err)
	}
	if defaults.DefaultComputerID != computerID {
		return harnessProjectID, pick, nil
	}
	if harnessProjectID == "" {
		harnessProjectID = defaults.FallbackProjectID
	}
	return harnessProjectID, fillPick(pick, modelPick{defaults.Provider, defaults.Model, defaults.ModelOptions}), nil
}

// fillPick fills a blank provider or model from fallback; a filled-in model brings its own options along.
func fillPick(pick, fallback modelPick) modelPick {
	if pick.provider == "" {
		pick.provider = fallback.provider
	}
	if pick.model == "" {
		pick.model, pick.options = fallback.model, fallback.options
	}
	return pick
}

func (s *Service) fetchTargetComputer(ctx context.Context, userID, computerID string) (*Computer, error) {
	computer, err := s.repo.GetComputer(ctx, userID, computerID)
	if err != nil {
		if errors.Is(err, apperrs.ErrNotFound) {
			return nil, &NotConfiguredError{Reason: ReasonUnpaired}
		}
		return nil, fmt.Errorf("get computer %s: %w", computerID, err)
	}
	if !computer.Paired() {
		return nil, &NotConfiguredError{Reason: ReasonUnpaired}
	}
	if computer.TokenExpiresAt.Before(s.now()) {
		return nil, &NotConfiguredError{Reason: ReasonExpiredToken}
	}
	return computer, nil
}

// ownComputer returns one of userID's own computers; another user's id is ErrNotFound, never a permission leak.
func (s *Service) ownComputer(ctx context.Context, userID, computerID string) (*Computer, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, fmt.Errorf("%w: user is required", apperrs.ErrUnauthorized)
	}
	computerID = strings.TrimSpace(computerID)
	if computerID == "" {
		return nil, fmt.Errorf("%w: computer id is required", apperrs.ErrInvalid)
	}
	computer, err := s.repo.GetComputer(ctx, userID, computerID)
	if err != nil {
		return nil, fmt.Errorf("get computer %s: %w", computerID, err)
	}
	return computer, nil
}

// GetSetup returns the caller's own computer's overall and per-provider setup confirmation.
func (s *Service) GetSetup(ctx context.Context, userID, computerID string) (Setup, error) {
	computer, err := s.ownComputer(ctx, userID, computerID)
	if err != nil {
		return Setup{}, err
	}
	providers, err := s.repo.ListProviderSetups(ctx, computer.ID)
	if err != nil {
		return Setup{}, fmt.Errorf("list provider setups for %s: %w", computer.ID, err)
	}
	turns, err := s.repo.ListLatestSetupTurns(ctx, computer.ID)
	if err != nil {
		return Setup{}, fmt.Errorf("list setup turns for %s: %w", computer.ID, err)
	}
	for i, p := range providers {
		providers[i].SkillsOutdated = p.outdated()
	}
	return Setup{ComputerID: computer.ID, ConfirmedAt: computer.SetupConfirmedAt, Providers: providers, SetupChoices: computer.SetupChoices.withEmpty(), Turns: turns}, nil
}

// SaveSetupChoices keeps the Set up step's choices on the caller's own computer without running setup, and returns its setup.
func (s *Service) SaveSetupChoices(ctx context.Context, userID, computerID string, choices SetupChoices) (Setup, error) {
	computer, err := s.ownComputer(ctx, userID, computerID)
	if err != nil {
		return Setup{}, err
	}
	choices, err = choices.clean()
	if err != nil {
		return Setup{}, err
	}
	if err := s.repo.SaveSetupChoices(ctx, userID, computer.ID, choices); err != nil {
		return Setup{}, fmt.Errorf("save setup choices for %s: %w", computer.ID, err)
	}
	return s.GetSetup(ctx, userID, computer.ID)
}

// ConfirmSetup records the caller's computer as set up overall; independent of any provider's confirmation.
func (s *Service) ConfirmSetup(ctx context.Context, userID, computerID string) (Setup, error) {
	now := s.now().UTC()
	return s.setOverallSetup(ctx, userID, computerID, &now)
}

// UnconfirmSetup withdraws the caller's computer's overall confirmation and revokes its MCP token first.
func (s *Service) UnconfirmSetup(ctx context.Context, userID, computerID string) (Setup, error) {
	computer, err := s.ownComputer(ctx, userID, computerID)
	if err != nil {
		return Setup{}, err
	}
	if err := s.revokeMCPToken(ctx, *computer); err != nil {
		return Setup{}, err
	}
	return s.setOverallSetup(ctx, userID, computer.ID, nil)
}

func (s *Service) setOverallSetup(ctx context.Context, userID, computerID string, at *time.Time) (Setup, error) {
	computer, err := s.ownComputer(ctx, userID, computerID)
	if err != nil {
		return Setup{}, err
	}
	evt := setupEvent(SetupChangedEvent{ComputerID: computer.ID, UserID: userID, ConfirmedAt: at})
	if err := s.repo.SetSetupConfirmedAt(ctx, userID, computer.ID, at, evt); err != nil {
		return Setup{}, fmt.Errorf("set setup confirmation for %s: %w", computer.ID, err)
	}
	return s.GetSetup(ctx, userID, computer.ID)
}

// ConfirmProviderSetup records a provider as set up on the caller's computer, with the skills its harness reported.
func (s *Service) ConfirmProviderSetup(ctx context.Context, userID, computerID, provider string, skills []string) (Setup, error) {
	provider, err := validateProvider(provider)
	if err != nil {
		return Setup{}, err
	}
	skills, err = validateSkills(skills)
	if err != nil {
		return Setup{}, err
	}
	now := s.now().UTC()
	// The setup that confirms has just written the current skill, so that is the version this provider now holds.
	return s.saveProviderSetup(ctx, userID, computerID, ProviderSetup{Provider: provider, ConfirmedAt: &now, Skills: skills, SkillsVersion: shipped.NexulMemory.Version})
}

// RecordSkillsVersion records that the skill folders on the caller's computer hold version, for every confirmed provider since
// they all read the same folders; it changes no confirmation. Only the version skill_get returns is accepted.
func (s *Service) RecordSkillsVersion(ctx context.Context, userID, computerID, version string) (Setup, error) {
	if version != shipped.NexulMemory.Version {
		return Setup{}, fmt.Errorf("%w: skills_version %q is not the current nexul-memory version; skill_get returns it", apperrs.ErrInvalid, version)
	}
	computer, err := s.ownComputer(ctx, userID, computerID)
	if err != nil {
		return Setup{}, err
	}
	setups, err := s.repo.ListProviderSetups(ctx, computer.ID)
	if err != nil {
		return Setup{}, fmt.Errorf("list provider setups for %s: %w", computer.ID, err)
	}
	if !slices.ContainsFunc(setups, func(p ProviderSetup) bool { return p.ConfirmedAt != nil }) {
		return Setup{}, fmt.Errorf("%w: no provider on %s is confirmed; run setup to confirm one first", apperrs.ErrInvalid, computer.Name)
	}
	for _, p := range setups {
		if !p.outdated() {
			continue
		}
		p.SkillsVersion = version
		evt := setupEvent(SetupChangedEvent{ComputerID: computer.ID, UserID: userID, Provider: p.Provider, ConfirmedAt: p.ConfirmedAt, Skills: p.Skills})
		if err := s.repo.SaveProviderSetup(ctx, computer.ID, p, s.now().UTC(), evt); err != nil {
			return Setup{}, fmt.Errorf("save %s skills version for %s: %w", p.Provider, computer.ID, err)
		}
	}
	return s.GetSetup(ctx, userID, computer.ID)
}

// UnconfirmProviderSetup withdraws a provider's confirmation on the caller's computer and clears its skills list.
func (s *Service) UnconfirmProviderSetup(ctx context.Context, userID, computerID, provider string) (Setup, error) {
	provider, err := validateProvider(provider)
	if err != nil {
		return Setup{}, err
	}
	return s.saveProviderSetup(ctx, userID, computerID, ProviderSetup{Provider: provider, Skills: []string{}})
}

func (s *Service) saveProviderSetup(ctx context.Context, userID, computerID string, p ProviderSetup) (Setup, error) {
	computer, err := s.ownComputer(ctx, userID, computerID)
	if err != nil {
		return Setup{}, err
	}
	evt := setupEvent(SetupChangedEvent{ComputerID: computer.ID, UserID: userID, Provider: p.Provider, ConfirmedAt: p.ConfirmedAt, Skills: p.Skills})
	if err := s.repo.SaveProviderSetup(ctx, computer.ID, p, s.now().UTC(), evt); err != nil {
		return Setup{}, fmt.Errorf("save %s setup for %s: %w", p.Provider, computer.ID, err)
	}
	return s.GetSetup(ctx, userID, computer.ID)
}

func setupEvent(e SetupChangedEvent) eventbus.OutboxEvent {
	topic := TopicSetupConfirmed
	if e.ConfirmedAt == nil {
		topic = TopicSetupUnconfirmed
	}
	return eventbus.OutboxEvent{ID: ids.New(), Topic: topic, Payload: e}
}
