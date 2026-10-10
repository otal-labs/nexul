package repository

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	apperrors "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// installStatePrefix marks the state an install link carries, so the sign-in callback can tell it from its own.
const installStatePrefix = "install."

// installStateTTL is how long an install link stays good: long enough to send to a client who installs it later.
const installStateTTL = 7 * 24 * time.Hour

// InstallerChecker returns the account of installationID as the installer's own token sees it, ErrForbidden if unseen.
type InstallerChecker interface {
	InstallerAccount(ctx context.Context, code string, installationID int64) (string, error)
}

// Config wires the repository use-cases.
type Config struct {
	Gate          Gate
	Scanner       Scanner
	Installations InstallationLister
	Store         InstallationStore
	Installers    InstallerChecker
	// StateKey signs the workspace an install link carries.
	StateKey []byte
	Now      func() time.Time
}

// Service is the repository use-case layer: the wizard's list and scan, and which workspaces see each installation.
type Service struct {
	cfg Config
}

// NewService wires the repository use-cases.
func NewService(cfg Config) *Service {
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	return &Service{cfg: cfg}
}

// Scan is Scan over the wired gate and scanner.
func (s *Service) Scan(ctx context.Context, owner, name, ref string) (*ScanResult, error) {
	return Scan(ctx, s.cfg.Gate, s.cfg.Scanner, owner, name, ref)
}

// wizardWorkspaces checks projects:write in workspaceID, or in any workspace when none is named, and returns them.
func (s *Service) wizardWorkspaces(ctx context.Context, workspaceID string) ([]string, error) {
	if s.cfg.Gate == nil {
		return []string{workspaceID}, permissions.Ungated(ctx)
	}
	if workspaceID != "" {
		return []string{workspaceID}, s.cfg.Gate.Require(ctx, workspaceID, permissions.ProjectsWrite)
	}
	if err := s.cfg.Gate.RequireAnywhere(ctx, permissions.ProjectsWrite); err != nil {
		return nil, err
	}
	return s.cfg.Gate.WorkspacesWith(ctx, permissions.ProjectsWrite)
}

// ListRepos lists the repositories a workspace can make a project from whose owner/name contains q (ADR 0144).
func (s *Service) ListRepos(ctx context.Context, workspaceID, q string, refresh bool) ([]Repo, error) {
	workspaces, err := s.wizardWorkspaces(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	q = strings.ToLower(strings.TrimSpace(q))
	if q != "" && utf8.RuneCountInString(q) < MinSearchLength {
		return nil, fmt.Errorf("%w: q needs at least %d characters", apperrors.ErrInvalid, MinSearchLength)
	}
	visible, err := s.visibleAccounts(ctx, workspaces)
	if err != nil {
		return nil, err
	}
	repos, err := s.cfg.Scanner.ListInstallationRepos(ctx, refresh)
	if err != nil {
		return nil, fmt.Errorf("list installation repositories: %w", err)
	}
	matches := []Repo{}
	for _, r := range repos {
		if visible != nil && !visible[strings.ToLower(r.Owner)] {
			continue
		}
		if q != "" && !strings.Contains(strings.ToLower(r.FullName), q) {
			continue
		}
		matches = append(matches, r)
	}
	return matches, nil
}

// visibleAccounts is the accounts workspaces see, or nil for all while Nexul reads GitHub as the connected account.
func (s *Service) visibleAccounts(ctx context.Context, workspaces []string) (map[string]bool, error) {
	asApp, err := s.cfg.Installations.ReadsAsApp(ctx)
	if err != nil {
		return nil, fmt.Errorf("read the git provider's mode: %w", err)
	}
	if !asApp {
		return nil, nil
	}
	accounts, err := s.cfg.Store.InstallationAccountsIn(ctx, workspaces)
	if err != nil {
		return nil, err
	}
	visible := make(map[string]bool, len(accounts))
	for _, a := range accounts {
		visible[a] = true
	}
	return visible, nil
}

// ListInstallations lists the App's installations with their workspaces; it takes connectors:read anywhere.
func (s *Service) ListInstallations(ctx context.Context) ([]Installation, error) {
	if err := s.requireAnywhere(ctx, permissions.ConnectorsRead); err != nil {
		return nil, err
	}
	installs, err := s.cfg.Installations.ListInstallations(ctx)
	if err != nil {
		return nil, fmt.Errorf("list installations: %w", err)
	}
	assigned, err := s.cfg.Store.ListInstallationWorkspaces(ctx)
	if err != nil {
		return nil, err
	}
	for i := range installs {
		installs[i].Workspaces = assigned[strings.ToLower(installs[i].AccountLogin)]
	}
	return installs, nil
}

func (s *Service) requireAnywhere(ctx context.Context, action permissions.Action) error {
	if s.cfg.Gate == nil {
		return permissions.Ungated(ctx)
	}
	return s.cfg.Gate.RequireAnywhere(ctx, action)
}

// AssignInstallation lets workspaceID list account's repositories; it takes connectors:write and membership there.
func (s *Service) AssignInstallation(ctx context.Context, account, workspaceID string) error {
	account, err := s.requireAssigner(ctx, account, workspaceID)
	if err != nil {
		return err
	}
	if s.cfg.Gate != nil {
		if err := s.cfg.Gate.Require(ctx, workspaceID, ""); err != nil {
			return err
		}
	}
	return s.cfg.Store.AssignInstallation(ctx, account, workspaceID)
}

// UnassignInstallation stops workspaceID seeing the repositories of the installation on account.
func (s *Service) UnassignInstallation(ctx context.Context, account, workspaceID string) error {
	account, err := s.requireAssigner(ctx, account, workspaceID)
	if err != nil {
		return err
	}
	return s.cfg.Store.UnassignInstallation(ctx, account, workspaceID)
}

func (s *Service) requireAssigner(ctx context.Context, account, workspaceID string) (string, error) {
	account = strings.ToLower(strings.TrimSpace(account))
	if account == "" || workspaceID == "" {
		return "", fmt.Errorf("%w: an account and a workspace are required", apperrors.ErrInvalid)
	}
	return account, s.requireAnywhere(ctx, permissions.ConnectorsWrite)
}

// InstallURL is GitHub's install page carrying workspaceID, signed, for a caller holding projects:write there.
func (s *Service) InstallURL(ctx context.Context, workspaceID string) (string, error) {
	if workspaceID == "" {
		return "", fmt.Errorf("%w: workspace_id is required", apperrors.ErrInvalid)
	}
	if _, err := s.wizardWorkspaces(ctx, workspaceID); err != nil {
		return "", err
	}
	base, err := s.cfg.Installations.InstallURL(ctx)
	if err != nil {
		return "", err
	}
	if base == "" {
		return "", fmt.Errorf("%w: the GitHub App has no slug registered", apperrors.ErrNotFound)
	}
	return base + "?" + url.Values{"state": {s.signState(workspaceID)}}.Encode(), nil
}

// ClaimsState reports whether state is one an install link carried.
func (s *Service) ClaimsState(state string) bool {
	return strings.HasPrefix(state, installStatePrefix)
}

// ClaimInstallation assigns the installation an install link led to its workspace once the installer is confirmed.
func (s *Service) ClaimInstallation(ctx context.Context, state, code, installationID string) (string, error) {
	workspaceID, err := s.verifyState(state)
	if err != nil {
		return "", err
	}
	id, err := strconv.ParseInt(installationID, 10, 64)
	if err != nil {
		return "", fmt.Errorf("%w: installation_id %q is not a number", apperrors.ErrInvalid, installationID)
	}
	if code == "" {
		return "", fmt.Errorf("%w: GitHub sent no code to confirm the installer, so the installation stays unassigned", apperrors.ErrInvalid)
	}
	account, err := s.cfg.Installers.InstallerAccount(ctx, code, id)
	if err != nil {
		return "", fmt.Errorf("confirm installation %d: %w", id, err)
	}
	if err := s.cfg.Store.AssignInstallation(ctx, account, workspaceID); err != nil {
		return "", err
	}
	return workspaceID, nil
}

func (s *Service) signState(workspaceID string) string {
	payload := installStatePrefix + workspaceID + "." + strconv.FormatInt(s.cfg.Now().Add(installStateTTL).Unix(), 10)
	return payload + "." + s.stateMAC(payload)
}

func (s *Service) stateMAC(payload string) string {
	mac := hmac.New(sha256.New, s.cfg.StateKey)
	mac.Write([]byte(payload))
	return hex.EncodeToString(mac.Sum(nil))
}

func (s *Service) verifyState(state string) (string, error) {
	cut := strings.LastIndex(state, ".")
	if !s.ClaimsState(state) || cut < 0 || !hmac.Equal([]byte(state[cut+1:]), []byte(s.stateMAC(state[:cut]))) {
		return "", fmt.Errorf("%w: the install link was not signed by this instance", apperrors.ErrInvalid)
	}
	workspaceID, expiry, ok := strings.Cut(strings.TrimPrefix(state[:cut], installStatePrefix), ".")
	exp, err := strconv.ParseInt(expiry, 10, 64)
	if !ok || err != nil || s.cfg.Now().Unix() > exp {
		return "", fmt.Errorf("%w: the install link has expired; open a fresh one from the project wizard", apperrors.ErrInvalid)
	}
	return workspaceID, nil
}
