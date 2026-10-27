package pairing

import (
	"cmp"
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
	"github.com/otal-labs/nexul/internal/platform/logging"
	"github.com/otal-labs/nexul/internal/platform/permissions"
	shipped "github.com/otal-labs/nexul/internal/platform/skills"
)

// ProjectGate answers whether the caller may open a project; an unknown or hidden one is ErrNotFound (ADR 0097).
type ProjectGate interface {
	RequireProject(ctx context.Context, projectID string, action permissions.Action) error
}

// ComputerGate reads what a person holds on a computer through its owner's rules (ADR 0148).
type ComputerGate interface {
	ComputerPermissions(ctx context.Context, userID, computerID string) (ownerID string, held permissions.Set, err error)
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
	// Tunnels tears down what pairing through a tunnel left at Cloudflare.
	Tunnels Tunnels
	// Bus carries ephemeral pairing events; nil keeps them silent.
	Bus Publisher
	// Tokens mints and revokes each computer's MCP token.
	Tokens MCPTokens
	// Instance reads the instance URL the setup turns point each provider's MCP entry at.
	Instance InstanceSettings
	// Projects keeps a person's project links to the projects they can open.
	Projects ProjectGate
	// Runners mints a computer's personal runner code and reads its runner; nil pairs nothing.
	Runners Runners
	// Containers reads a computer's Docker containers and their logs through its runner; nil serves none.
	Containers Containers
	// ComputerAccess lets a person a computer is shared with read its logs; nil leaves them to the owner.
	ComputerAccess ComputerGate
	// Repositories finds a project's checkout on a computer by its git remote; nil finds only picked ones.
	Repositories ProjectRemotes
	// Cloner clones through a computer's runner with a token for the repository; nil leaves every clone to T3 Code.
	Cloner RepositoryCloner
}

// Service is the pairing use-case layer (ADR 0019); bearer tokens never leave it except encrypted at rest.
type Service struct {
	repo           Repo
	harnesses      harness.Registry
	key            []byte
	now            func() time.Time
	changed        func(userID string)
	tunnels        Tunnels
	bus            Publisher
	tokens         MCPTokens
	instance       InstanceSettings
	projects       ProjectGate
	runners        Runners
	containers     Containers
	computerAccess ComputerGate
	repositories   ProjectRemotes
	cloner         RepositoryCloner
	setupMu        sync.Mutex
	// settingUp holds a computer's drivers with a setup turn running, keyed by setupKey.
	settingUp map[string]bool
	setupRuns sync.WaitGroup
	// tokenMu keeps two setup runs starting at once from each minting the computer's token, which would revoke the other's.
	tokenMu sync.Mutex
	// pairMu guards pairFailures: why each computer's last pairing through its runner failed.
	pairMu       sync.Mutex
	pairFailures map[string]string
}

// NewService wires the pairing use-cases.
func NewService(cfg Config) *Service {
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	return &Service{
		repo: cfg.Repo, harnesses: cfg.Harnesses, key: cfg.EncryptionKey, now: cfg.Now, changed: cfg.OnComputersChanged,
		tunnels: cfg.Tunnels, bus: cfg.Bus, tokens: cfg.Tokens, instance: cfg.Instance, projects: cfg.Projects, runners: cfg.Runners, containers: cfg.Containers, computerAccess: cfg.ComputerAccess, repositories: cfg.Repositories, cloner: cfg.Cloner, settingUp: map[string]bool{},
		pairFailures: map[string]string{},
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

// ListComputers returns the caller's computers, then those shared with them by name, by name and runner connection only.
func (s *Service) ListComputers(ctx context.Context, userID string) ([]Computer, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, fmt.Errorf("%w: user is required", apperrs.ErrUnauthorized)
	}
	cs, err := s.repo.ListComputersFor(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list computers: %w", err)
	}
	for i := range cs {
		cs[i].BearerToken = ""
	}
	cs, err = s.withRunners(ctx, cs)
	if err != nil {
		return nil, err
	}
	if cs, err = s.withRunning(ctx, cs); err != nil {
		return nil, err
	}
	outdated, err := s.repo.ListOutdatedSkillsComputers(ctx, userID, shipped.SetVersion)
	if err != nil {
		return nil, fmt.Errorf("list computers: %w", err)
	}
	for i := range cs {
		cs[i].SkillsOutdated = slices.Contains(outdated, cs[i].ID)
		if cs[i].Shared != nil {
			cs[i] = sharedView(cs[i])
		}
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
		out = append(out, ProviderOption{Provider: p, NeedsSetup: !setupConfirmed(session, setups, p.Driver), Skipped: switchedOff(session, p.Driver)})
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

// PairComputer pairs one of the caller's computers now, through its runner (ADR 0146).
func (s *Service) PairComputer(ctx context.Context, userID, computerID string) (*Computer, error) {
	existing, err := s.ownComputer(ctx, userID, computerID)
	if err != nil {
		return nil, err
	}
	paired, err := s.pairThroughRunner(ctx, *existing)
	if err != nil {
		return nil, err
	}
	if err := s.recordFacts(ctx, userID, paired.ID, nil); err != nil {
		logging.FromCtx(ctx).Warn("computer facts not read after pairing", "computer_id", paired.ID, "error", err)
	}
	return paired, nil
}

// pair exchanges the secret at serverURL and saves base with the new session; base keeps its id and setup.
func (s *Service) pair(ctx context.Context, base Computer, serverURL, secret string) (*Computer, error) {
	client, ok := s.harnesses[base.Kind]
	if !ok {
		return nil, fmt.Errorf("%w: unsupported harness kind %q", apperrs.ErrInvalid, base.Kind)
	}
	name, err := validateName(base.Name)
	if err != nil {
		return nil, &FieldError{Field: "name", Err: err}
	}
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return nil, fmt.Errorf("%w: pairing token is required", apperrs.ErrInvalid)
	}
	if len(s.key) == 0 {
		return nil, apperrs.Fatal(fmt.Errorf("%w: pairing encryption key is not configured", apperrs.ErrFatal))
	}
	result, err := client.Pair(ctx, serverURL, secret)
	if err != nil {
		return nil, pairFailure(err)
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
		ComputerID: computer.ID, UserID: computer.UserID, ServerURL: serverURL, HarnessVersion: result.Version, TokenExpiresAt: computer.TokenExpiresAt, MembersOnly: true,
	}}}
	// A computer's first pairing lands on whatever kind answers; only a paired one switches.
	if base.Paired() && kindStep(kind) > kindStep(base.Kind) {
		evts = append(evts, eventbus.OutboxEvent{ID: ids.New(), Topic: TopicHarnessSwitched, Payload: HarnessSwitchedEvent{
			ComputerID: computer.ID, UserID: computer.UserID, FromKind: base.Kind, ToKind: kind, HarnessVersion: result.Version, MembersOnly: true,
		}})
	}
	if err := s.repo.SaveComputer(ctx, computer, evts...); err != nil {
		return nil, fmt.Errorf("save computer: %w", err)
	}
	s.notifyComputersChanged(computer.UserID)
	computer.BearerToken = ""
	return &computer, nil
}

// pairFailure keeps a harness's own protocol message, else says whether T3 Code was unreachable or refused the token.
func pairFailure(err error) error {
	if errors.Is(err, harness.ErrProtocol) {
		return err
	}
	if errors.Is(err, apperrs.ErrRetryable) {
		return fmt.Errorf("couldn't reach T3 Code through the Nexul app: %w", err)
	}
	return fmt.Errorf("T3 Code refused the pairing token the Nexul app made: %w", err)
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
			ComputerID: sess.ComputerID, UserID: userID, FromKind: from, ToKind: to, HarnessVersion: version, MembersOnly: true,
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
		// A computer paired the old way is reached only once it moves to its runner, so nothing holds it open.
		if !c.TokenExpiresAt.After(now) || !onRunner(c) {
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
	d.Placement = cmp.Or(d.Placement, PlacementInOrder)
	return d, nil
}

// SetDefaults stores the caller's pairing defaults and wakes their waiting runs, which may now land somewhere.
func (s *Service) SetDefaults(ctx context.Context, userID string, d Defaults) (Defaults, error) {
	if strings.TrimSpace(userID) == "" {
		return Defaults{}, fmt.Errorf("%w: user is required", apperrs.ErrUnauthorized)
	}
	d.UserID = userID
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
	if d.Placement, err = validatePlacement(d.Placement); err != nil {
		return Defaults{}, err
	}
	d.Placement = cmp.Or(d.Placement, PlacementInOrder)
	if err := s.repo.SaveDefaults(ctx, d); err != nil {
		return Defaults{}, fmt.Errorf("save defaults: %w", err)
	}
	s.placementChanged(ctx, userID)
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

// SetProjectLink sets the caller's own link for a project they can open; its computers and checkouts must be theirs.
func (s *Service) SetProjectLink(ctx context.Context, userID, projectID string, link ProjectLink) (ProjectLink, error) {
	projectID, err := s.openProject(ctx, userID, projectID)
	if err != nil {
		return ProjectLink{}, err
	}
	link.UserID, link.ProjectID = userID, projectID
	if link.Computers, link.Checkouts, err = s.ownLinkComputers(ctx, userID, link.Computers, link.Checkouts); err != nil {
		return ProjectLink{}, err
	}
	link.Provider = strings.TrimSpace(link.Provider)
	link.Model = strings.TrimSpace(link.Model)
	if link.ModelOptions, err = harness.CleanOptions(link.ModelOptions); err != nil {
		return ProjectLink{}, err
	}
	if link.StartIn, err = validateStartIn(link.StartIn); err != nil {
		return ProjectLink{}, err
	}
	if link.Placement, err = validatePlacement(link.Placement); err != nil {
		return ProjectLink{}, err
	}
	link.UpdatedAt = s.now().UTC()
	if err := s.repo.SaveProjectLink(ctx, link); err != nil {
		return ProjectLink{}, fmt.Errorf("save project link %s: %w", projectID, err)
	}
	s.placementChanged(ctx, userID)
	return link, nil
}

// ownLinkComputers trims a link's computers and checkouts, refusing a computer that is not the caller's or named twice.
func (s *Service) ownLinkComputers(ctx context.Context, userID string, ids []string, checkouts []Checkout) ([]string, []Checkout, error) {
	own := func(id string) error {
		_, err := s.repo.GetComputer(ctx, userID, id)
		if err != nil {
			return fmt.Errorf("link computer %s: %w", id, err)
		}
		return nil
	}
	outIDs := make([]string, 0, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if slices.Contains(outIDs, id) {
			return nil, nil, fmt.Errorf("%w: computer %s is listed twice", apperrs.ErrInvalid, id)
		}
		if err := own(id); err != nil {
			return nil, nil, err
		}
		outIDs = append(outIDs, id)
	}
	outCheckouts := make([]Checkout, 0, len(checkouts))
	for _, c := range checkouts {
		c.ComputerID = strings.TrimSpace(c.ComputerID)
		harnessProjectID, err := validateHarnessProjectID(c.HarnessProjectID)
		if err != nil {
			return nil, nil, err
		}
		if err := own(c.ComputerID); err != nil {
			return nil, nil, err
		}
		outCheckouts = append(outCheckouts, Checkout{ComputerID: c.ComputerID, HarnessProjectID: harnessProjectID})
	}
	return outIDs, outCheckouts, nil
}

// ClearProjectLink removes the caller's own link and project order; the checkouts they picked stay where the code is.
func (s *Service) ClearProjectLink(ctx context.Context, userID, projectID string) error {
	projectID, err := s.openProject(ctx, userID, projectID)
	if err != nil {
		return err
	}
	if err := s.repo.DeleteProjectLink(ctx, userID, projectID); err != nil {
		return fmt.Errorf("clear project link %s: %w", projectID, err)
	}
	s.placementChanged(ctx, userID)
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
	// PassedOver are the computers before this one that could not take the turn, and why.
	PassedOver []PassOver
}

// modelPick keeps a model's options beside it: they only mean something for the model they were picked with.
type modelPick struct {
	provider string
	model    string
	options  []harness.OptionSetting
}

// ResolveTarget lands a chat turn on the caller's first ready computer, behind the setup gate.
func (s *Service) ResolveTarget(ctx context.Context, userID, projectID string) (*ResolvedTarget, error) {
	return s.ResolveRun(ctx, RunRequest{UserID: userID, ProjectID: projectID, Chat: true})
}

// ResolveTargetOverride lands a chat turn on one computer, with a provider, model and options, behind the same gate.
func (s *Service) ResolveTargetOverride(ctx context.Context, userID, projectID, computerID, provider, model string, options []harness.OptionSetting) (*ResolvedTarget, error) {
	req := RunRequest{UserID: userID, ProjectID: projectID, Provider: provider, Model: model, ModelOptions: options, Chat: true}
	if computerID = strings.TrimSpace(computerID); computerID != "" {
		req.ComputerIDs = []string{computerID}
	}
	return s.ResolveRun(ctx, req)
}

// ResolveConfirmedTarget rechecks the same computer and setup without reading mutable location or model settings.
func (s *Service) ResolveConfirmedTarget(ctx context.Context, userID string, target ResolvedTarget) (*ResolvedTarget, error) {
	resolved, err := s.resolveConfirmedTarget(ctx, userID, target)
	if err != nil {
		return nil, err
	}
	return s.requireSetup(ctx, resolved)
}

func (s *Service) resolveConfirmedTarget(ctx context.Context, userID string, target ResolvedTarget) (*ResolvedTarget, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, fmt.Errorf("%w: user is required", apperrs.ErrUnauthorized)
	}
	if len(s.key) == 0 {
		return nil, apperrs.Fatal(fmt.Errorf("%w: pairing encryption key is not configured", apperrs.ErrFatal))
	}
	computer, err := s.fetchTargetComputer(ctx, userID, target.Computer.ID)
	if err != nil {
		return nil, err
	}
	decrypted, err := crypto.Decrypt(s.key, computer.BearerToken)
	if err != nil {
		return nil, fmt.Errorf("decrypt bearer token for computer %s: %w", computer.ID, err)
	}
	target.Computer = *computer
	target.Computer.BearerToken = string(decrypted)
	return &target, nil
}

func (s *Service) requireListedProject(ctx context.Context, userID, computerID, harnessProjectID string) error {
	target, err := s.resolveConfirmedTarget(ctx, userID, ResolvedTarget{Computer: Computer{ID: computerID}})
	if err != nil {
		return err
	}
	client, err := s.client(target.Computer.Kind)
	if err != nil {
		return err
	}
	projects, err := client.ListProjects(ctx, target.Computer.Session())
	if errors.Is(err, harness.ErrProtocol) {
		return err
	}
	if err != nil {
		return s.offline(ctx, target.Computer, err)
	}
	if !slices.ContainsFunc(projects, func(p harness.Project) bool { return p.ID == harnessProjectID }) {
		return fmt.Errorf("%w: %s is not one of the T3 projects on %s", apperrs.ErrInvalid, harnessProjectID, target.Computer.Name)
	}
	return nil
}

// PreviewTarget is ResolveTarget without the gate or bearer token, so readiness never greys a play that refuses on press.

// PreviewTarget is where a play pressed in projectID would land now, without the gate or bearer token, so readiness
// never greys a play that would start on press.
func (s *Service) PreviewTarget(ctx context.Context, userID, projectID string) (*ResolvedTarget, error) {
	return s.PreviewRun(ctx, RunRequest{UserID: userID, ProjectID: projectID, Room: RoomPreferred, Person: true})
}

// ResolveSetupTurnTarget skips the setup gate for the wizard's setup turn only; TestResolveSetupTurnTarget_OnlyTheWizardCallsIt guards it.
// A picked projectID wins over the computer's chat folder, which wins over its first T3 project.
func (s *Service) ResolveSetupTurnTarget(ctx context.Context, userID, computerID, provider, projectID string) (*ResolvedTarget, error) {
	if strings.TrimSpace(computerID) == "" {
		return nil, fmt.Errorf("%w: a setup turn needs its computer", apperrs.ErrInvalid)
	}
	if _, err := s.fetchTargetComputer(ctx, userID, computerID); err != nil {
		return nil, err
	}
	session, err := s.sessionComputer(ctx, userID, computerID)
	if err != nil {
		return nil, err
	}
	if projectID = cmp.Or(projectID, session.ChatProjectID); projectID != "" {
		return &ResolvedTarget{Computer: session, HarnessProjectID: projectID, Provider: strings.TrimSpace(provider)}, nil
	}
	return s.setupTurnInFirstProject(ctx, userID, computerID, provider)
}

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
		return nil, fmt.Errorf("%w: %s has no T3 project and setup runs its turns inside one; create one under Settings → Computers → Projects → Create empty project, or run computer_project_create with a name", apperrs.ErrInvalid, session.Name)
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
		return nil, s.offline(ctx, target.Computer, err)
	}
	if listsMore(target.Computer.Facts, providers) {
		if err := s.recordFacts(ctx, target.Computer.UserID, target.Computer.ID, nil); err != nil {
			logging.FromCtx(ctx).Warn("computer facts not read again after T3 Code listed more providers", "computer_id", target.Computer.ID, "error", err)
		}
	}
	// The call may have moved the computer to a newer kind; the turn and its version-drift check read what it holds now.
	stored, err := s.repo.GetComputer(ctx, target.Computer.UserID, target.Computer.ID)
	if err != nil {
		return nil, fmt.Errorf("get computer %s: %w", target.Computer.ID, err)
	}
	target.Computer.Kind, target.Computer.HarnessVersion = stored.Kind, stored.HarnessVersion
	setups, err := s.repo.ListProviderSetups(ctx, target.Computer.ID)
	if err != nil {
		return nil, fmt.Errorf("list provider setups for %s: %w", target.Computer.ID, err)
	}
	provider, err := pickProvider(providers, target.Provider, func(p harness.Provider) bool { return setupConfirmed(target.Computer, setups, p.Driver) })
	if err != nil {
		return nil, err
	}
	if !setupConfirmed(target.Computer, setups, provider.Driver) {
		nc := &NotConfiguredError{
			Reason: ReasonSetupRequired, Provider: provider.Name, Computer: target.Computer.Name, ProviderID: provider.ID, ComputerID: target.Computer.ID,
		}
		if switchedOff(target.Computer, provider.Driver) {
			nc.Why = "it's switched off"
		}
		return nil, nc
	}
	target.Provider = provider.ID
	return target, nil
}

// notConnected is why a computer whose Nexul app has no live connection is offline.
const notConnected = "it isn't connected to Nexul"

// offline is the refusal for a computer whose harness did not answer; a run aimed at it never moves to another computer.
func (s *Service) offline(ctx context.Context, c Computer, err error) *NotConfiguredError {
	nc := &NotConfiguredError{Reason: ReasonOffline, Computer: c.Name, ComputerID: c.ID, Err: err}
	if s.runners == nil {
		return nc
	}
	if r, rerr := s.runners.ComputerRunner(ctx, c.ID); rerr == nil && !r.Connected {
		nc.Why = notConnected
	}
	return nc
}

// pickProvider finds the stored instance id among the harness's providers; an empty one takes the first usable, else the harness default.
// listsMore is whether T3 Code lists a provider facts lack, as it does once it checks a provider it listed unchecked.
func listsMore(facts *Facts, providers []harness.Provider) bool {
	return facts != nil && slices.ContainsFunc(providers, func(p harness.Provider) bool {
		return !slices.ContainsFunc(facts.Providers, func(f ProviderFacts) bool { return f.ID == p.ID })
	})
}

func pickProvider(providers []harness.Provider, id string, usable func(harness.Provider) bool) (harness.Provider, error) {
	if i := slices.IndexFunc(providers, usable); id == "" && i >= 0 {
		return providers[i], nil
	}
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
	return s.saveProviderSetup(ctx, userID, computerID, ProviderSetup{Provider: provider, ConfirmedAt: &now, Skills: skills, SkillsVersion: shipped.SetVersion})
}

// RecordSkillsVersion records that the skill folders on the caller's computer hold version, for every confirmed provider since
// they all read the same folders; it changes no confirmation. Only the version skill_get returns is accepted.
func (s *Service) RecordSkillsVersion(ctx context.Context, userID, computerID, version string) (Setup, error) {
	if version != shipped.SetVersion {
		return Setup{}, fmt.Errorf("%w: skills_version %q is not the current skills version; the skills update instructions name it", apperrs.ErrInvalid, version)
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
	e.MembersOnly = true
	return eventbus.OutboxEvent{ID: ids.New(), Topic: topic, Payload: e}
}
