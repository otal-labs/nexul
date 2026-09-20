package pairing

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/otal-labs/nexul/internal/harness"
	"github.com/otal-labs/nexul/internal/platform/crypto"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
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
}

// Service is the pairing use-case layer (ADR 0019); bearer tokens never leave it except encrypted at rest.
type Service struct {
	repo      Repo
	harnesses harness.Registry
	key       []byte
	now       func() time.Time
	changed   func(userID string)
}

// NewService wires the pairing use-cases.
func NewService(cfg Config) *Service {
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	return &Service{repo: cfg.Repo, harnesses: cfg.Harnesses, key: cfg.EncryptionKey, now: cfg.Now, changed: cfg.OnComputersChanged}
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

// ListProviders fetches a paired computer's usable provider instances and models.
func (s *Service) ListProviders(ctx context.Context, userID, computerID string) ([]harness.Provider, error) {
	session, err := s.sessionComputer(ctx, userID, computerID)
	if err != nil {
		return nil, err
	}
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
	return s.pair(ctx, userID, "", kind, name, serverURL, secret)
}

// Repair re-runs the pairing flow against an existing computer row, updating it in place; the kind never changes.
func (s *Service) Repair(ctx context.Context, userID, id, name, serverURL, secret string) (*Computer, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, fmt.Errorf("%w: user is required", apperrs.ErrUnauthorized)
	}
	if strings.TrimSpace(id) == "" {
		return nil, fmt.Errorf("%w: computer id is required", apperrs.ErrInvalid)
	}
	existing, err := s.repo.GetComputer(ctx, userID, id)
	if err != nil {
		return nil, fmt.Errorf("get computer %s: %w", id, err)
	}
	return s.pair(ctx, userID, existing.ID, existing.Kind, name, serverURL, secret)
}

func (s *Service) pair(ctx context.Context, userID, id string, kind harness.Kind, name, serverURL, secret string) (*Computer, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, fmt.Errorf("%w: user is required", apperrs.ErrUnauthorized)
	}
	client, ok := s.harnesses[kind]
	if !ok {
		return nil, fmt.Errorf("%w: unsupported harness kind %q", apperrs.ErrInvalid, kind)
	}
	name, err := validateName(name)
	if err != nil {
		return nil, err
	}
	serverURL, err = validateServerURL(serverURL)
	if err != nil {
		return nil, err
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
		return nil, fmt.Errorf("pair %s: %w", kind, err)
	}
	encrypted, err := crypto.Encrypt(s.key, []byte(result.BearerToken))
	if err != nil {
		return nil, fmt.Errorf("encrypt bearer token: %w", err)
	}
	now := s.now().UTC()
	computer := Computer{
		ID:             id,
		UserID:         userID,
		Kind:           kind,
		Name:           name,
		ServerURL:      serverURL,
		BearerToken:    encrypted,
		TokenExpiresAt: now.Add(result.ExpiresIn),
		HarnessVersion: result.Version,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if computer.ID == "" {
		computer.ID = ids.New()
	}
	if err := s.repo.SaveComputer(ctx, computer); err != nil {
		return nil, fmt.Errorf("save computer: %w", err)
	}
	s.notifyComputersChanged(userID)
	computer.BearerToken = ""
	return &computer, nil
}

// DeleteComputer removes a paired computer; a mismatched id is ErrNotFound, never a permission leak.
func (s *Service) DeleteComputer(ctx context.Context, userID, id string) error {
	if strings.TrimSpace(userID) == "" {
		return fmt.Errorf("%w: user is required", apperrs.ErrUnauthorized)
	}
	if err := s.repo.DeleteComputer(ctx, userID, id); err != nil {
		return fmt.Errorf("delete computer %s: %w", id, err)
	}
	s.notifyComputersChanged(userID)
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

// ResolveTarget picks the computer/harness project/provider/model a mention runs against (resolution order).
func (s *Service) ResolveTarget(ctx context.Context, userID, projectID string) (*ResolvedTarget, error) {
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

// ResolveTargetOverride resolves a run's pinned computer, provider, and model (ticket 31): the computer must
// be the caller's own; the harness project and any blank provider/model come from the project link when it
// names this computer, else the caller's own fallback when this is their default computer — the same sources
// ResolveTarget's own resolution order reads, just checked against the caller's explicit choice of computer.
func (s *Service) ResolveTargetOverride(ctx context.Context, userID, projectID, computerID, provider, model string) (*ResolvedTarget, error) {
	computerID = strings.TrimSpace(computerID)
	if computerID == "" {
		return s.ResolveTarget(ctx, userID, projectID)
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
