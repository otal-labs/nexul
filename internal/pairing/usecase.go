package pairing

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/otal-labs/nexul/internal/harness"
	"github.com/otal-labs/nexul/internal/platform/crypto"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/ids"
)

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
		tunnels: cfg.Tunnels, bus: cfg.Bus, tokens: cfg.Tokens, instance: cfg.Instance, watching: map[string]tunnelWatch{}, settingUp: map[string]bool{},
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

// Repair re-runs the pairing flow against an existing computer row, updating it in place; the kind never changes.
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
	encrypted, err := crypto.Encrypt(s.key, []byte(result.BearerToken))
	if err != nil {
		return nil, fmt.Errorf("encrypt bearer token: %w", err)
	}
	now := s.now().UTC()
	computer := base
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
	evt := eventbus.OutboxEvent{ID: ids.New(), Topic: TopicComputerPaired, Payload: ComputerPairedEvent{
		ComputerID: computer.ID, UserID: computer.UserID, ServerURL: serverURL, HarnessVersion: result.Version, TokenExpiresAt: computer.TokenExpiresAt,
	}}
	if err := s.repo.SaveComputer(ctx, computer, evt); err != nil {
		return nil, fmt.Errorf("save computer: %w", err)
	}
	s.notifyComputersChanged(computer.UserID)
	computer.BearerToken = ""
	return &computer, nil
}

// pairFailure blames the address when the harness could not be reached, and the token when it answered and refused.
func pairFailure(serverURL string, err error) error {
	if errors.Is(err, apperrs.ErrRetryable) {
		return &FieldError{Field: "server_url", Err: fmt.Errorf("couldn't reach the harness at %s: %w", serverURL, err)}
	}
	return &FieldError{Field: "token", Err: fmt.Errorf("the harness refused this token, run t3 pair for a fresh one: %w", err)}
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
	if err := s.repo.SaveDefaults(ctx, d); err != nil {
		return Defaults{}, fmt.Errorf("save defaults: %w", err)
	}
	return d, nil
}

// GetProjectLink returns project's pairing link, zero-valued when unset.
func (s *Service) GetProjectLink(ctx context.Context, userID, projectID string) (ProjectLink, error) {
	if strings.TrimSpace(userID) == "" {
		return ProjectLink{}, fmt.Errorf("%w: user is required", apperrs.ErrUnauthorized)
	}
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return ProjectLink{}, fmt.Errorf("%w: project id is required", apperrs.ErrInvalid)
	}
	link, err := s.repo.GetProjectLink(ctx, projectID)
	if err != nil {
		return ProjectLink{}, fmt.Errorf("get project link %s: %w", projectID, err)
	}
	link.ProjectID = projectID
	return link, nil
}

// SetProjectLink links project to a computer + harness project + provider/model; computer must be caller's own.
func (s *Service) SetProjectLink(ctx context.Context, userID, projectID string, link ProjectLink) (ProjectLink, error) {
	if strings.TrimSpace(userID) == "" {
		return ProjectLink{}, fmt.Errorf("%w: user is required", apperrs.ErrUnauthorized)
	}
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return ProjectLink{}, fmt.Errorf("%w: project id is required", apperrs.ErrInvalid)
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
	link.ProjectID = projectID
	link.HarnessProjectID = harnessProjectID
	link.Provider = strings.TrimSpace(link.Provider)
	link.Model = strings.TrimSpace(link.Model)
	link.UpdatedAt = s.now().UTC()
	if err := s.repo.SaveProjectLink(ctx, link); err != nil {
		return ProjectLink{}, fmt.Errorf("save project link %s: %w", projectID, err)
	}
	return link, nil
}

// ClearProjectLink removes project's link; future mentions fall back to the mentioning user's own defaults.
func (s *Service) ClearProjectLink(ctx context.Context, userID, projectID string) error {
	if strings.TrimSpace(userID) == "" {
		return fmt.Errorf("%w: user is required", apperrs.ErrUnauthorized)
	}
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return fmt.Errorf("%w: project id is required", apperrs.ErrInvalid)
	}
	if err := s.repo.DeleteProjectLink(ctx, projectID); err != nil {
		return fmt.Errorf("clear project link %s: %w", projectID, err)
	}
	return nil
}

// ResolvedTarget is what a chat mention needs to run a turn; BearerToken is plaintext here.
type ResolvedTarget struct {
	Computer         Computer
	HarnessProjectID string
	Provider         string
	Model            string
}

// ResolveTarget picks the computer/harness project/provider/model a mention runs against, behind the setup gate.
func (s *Service) ResolveTarget(ctx context.Context, userID, projectID string) (*ResolvedTarget, error) {
	target, err := s.resolveTarget(ctx, userID, projectID)
	if err != nil {
		return nil, err
	}
	return s.requireSetup(ctx, target)
}

// ResolveTargetOverride resolves a run's pinned computer, provider, and model, behind the same setup gate.
func (s *Service) ResolveTargetOverride(ctx context.Context, userID, projectID, computerID, provider, model string) (*ResolvedTarget, error) {
	target, err := s.resolveTargetOverride(ctx, userID, projectID, computerID, provider, model)
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
func (s *Service) ResolveSetupTurnTarget(ctx context.Context, userID, computerID, provider string) (*ResolvedTarget, error) {
	if strings.TrimSpace(computerID) == "" {
		return nil, fmt.Errorf("%w: a setup turn needs its computer", apperrs.ErrInvalid)
	}
	target, err := s.resolveTargetOverride(ctx, userID, "", computerID, provider, "")
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
	if err != nil {
		return nil, &NotConfiguredError{Reason: ReasonOffline, Computer: target.Computer.Name, ComputerID: target.Computer.ID, Err: err}
	}
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

	computerID, harnessProjectID, provider, model, fromLink, err := s.resolveTargetSource(ctx, userID, projectID)
	if err != nil {
		return nil, err
	}

	computer, err := s.fetchTargetComputer(ctx, userID, computerID, fromLink)
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
	resolved := *computer
	resolved.BearerToken = string(decrypted)
	return &ResolvedTarget{Computer: resolved, HarnessProjectID: harnessProjectID, Provider: provider, Model: model}, nil
}

// resolveTargetSource: project link, then user defaults, then the user's sole paired computer if unambiguous.
func (s *Service) resolveTargetSource(ctx context.Context, userID, projectID string) (computerID, harnessProjectID, provider, model string, fromLink bool, err error) {
	if projectID = strings.TrimSpace(projectID); projectID != "" {
		link, err := s.repo.GetProjectLink(ctx, projectID)
		if err != nil {
			return "", "", "", "", false, fmt.Errorf("get project link %s: %w", projectID, err)
		}
		if link.ComputerID != "" {
			return link.ComputerID, link.HarnessProjectID, link.Provider, link.Model, true, nil
		}
	}

	defaults, err := s.repo.GetDefaults(ctx, userID)
	if err != nil {
		return "", "", "", "", false, fmt.Errorf("get defaults: %w", err)
	}
	computerID, harnessProjectID, provider, model = defaults.DefaultComputerID, defaults.FallbackProjectID, defaults.Provider, defaults.Model
	if computerID != "" {
		return computerID, harnessProjectID, provider, model, false, nil
	}

	computers, err := s.repo.ListComputers(ctx, userID)
	if err != nil {
		return "", "", "", "", false, fmt.Errorf("list computers: %w", err)
	}
	if len(computers) == 0 {
		return "", "", "", "", false, &NotConfiguredError{Reason: ReasonUnpaired}
	}
	if len(computers) > 1 {
		return "", "", "", "", false, &NotConfiguredError{Reason: ReasonNoDefaultComputer}
	}
	return computers[0].ID, harnessProjectID, provider, model, false, nil
}

// resolveTargetOverride resolves a pinned computer, provider, and model with no setup gate: the computer must
// be the caller's own; the harness project and any blank provider/model come from the project link when it
// names this computer, else the caller's own fallback when this is their default computer — the same sources
// resolveTarget's own resolution order reads, just checked against the caller's explicit choice of computer.
func (s *Service) resolveTargetOverride(ctx context.Context, userID, projectID, computerID, provider, model string) (*ResolvedTarget, error) {
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
	computer, err := s.fetchTargetComputer(ctx, userID, computerID, false)
	if err != nil {
		return nil, err
	}
	harnessProjectID, provider, model, err := s.overrideProjectAndModel(ctx, userID, strings.TrimSpace(projectID), computerID, strings.TrimSpace(provider), strings.TrimSpace(model))
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
	resolved := *computer
	resolved.BearerToken = string(decrypted)
	return &ResolvedTarget{Computer: resolved, HarnessProjectID: harnessProjectID, Provider: provider, Model: model}, nil
}

// overrideProjectAndModel fills in the harness project and any blank provider/model from the project link
// (when it names computerID) and the caller's own defaults (when computerID is their default computer).
func (s *Service) overrideProjectAndModel(ctx context.Context, userID, projectID, computerID, provider, model string) (harnessProjectID, resolvedProvider, resolvedModel string, err error) {
	resolvedProvider, resolvedModel = provider, model
	if projectID != "" {
		link, err := s.repo.GetProjectLink(ctx, projectID)
		if err != nil {
			return "", "", "", fmt.Errorf("get project link %s: %w", projectID, err)
		}
		if link.ComputerID == computerID {
			harnessProjectID = link.HarnessProjectID
			if resolvedProvider == "" {
				resolvedProvider = link.Provider
			}
			if resolvedModel == "" {
				resolvedModel = link.Model
			}
		}
	}
	if harnessProjectID != "" && resolvedProvider != "" && resolvedModel != "" {
		return harnessProjectID, resolvedProvider, resolvedModel, nil
	}
	defaults, err := s.repo.GetDefaults(ctx, userID)
	if err != nil {
		return "", "", "", fmt.Errorf("get defaults: %w", err)
	}
	if defaults.DefaultComputerID != computerID {
		return harnessProjectID, resolvedProvider, resolvedModel, nil
	}
	if harnessProjectID == "" {
		harnessProjectID = defaults.FallbackProjectID
	}
	if resolvedProvider == "" {
		resolvedProvider = defaults.Provider
	}
	if resolvedModel == "" {
		resolvedModel = defaults.Model
	}
	return harnessProjectID, resolvedProvider, resolvedModel, nil
}

func (s *Service) fetchTargetComputer(ctx context.Context, userID, computerID string, fromLink bool) (*Computer, error) {
	computer, err := s.getComputerFor(ctx, userID, computerID, fromLink)
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

func (s *Service) getComputerFor(ctx context.Context, userID, computerID string, fromLink bool) (*Computer, error) {
	if fromLink {
		return s.repo.GetComputerByID(ctx, computerID)
	}
	return s.repo.GetComputer(ctx, userID, computerID)
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
	return Setup{ComputerID: computer.ID, ConfirmedAt: computer.SetupConfirmedAt, Providers: providers, Turns: turns}, nil
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
	return s.saveProviderSetup(ctx, userID, computerID, ProviderSetup{Provider: provider, ConfirmedAt: &now, Skills: skills})
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
