package repository

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"

	apperrors "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/githubapp"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// Config wires the repository use-cases.
type Config struct {
	Gate Gate
	// People reads GitHub as the person asking, with their own token: every list, search and scan (ADR 0147).
	People        People
	Installations InstallationLister
	// Accounts reads GitHub as the App, for background work on attached repositories only.
	Accounts AccountResolver
	Store    InstallationStore
}

// Service is the repository use-case layer: the wizard's list and scan, what attaching a repository links, and the
// installations a person sees.
type Service struct {
	cfg Config
}

// NewService wires the repository use-cases.
func NewService(cfg Config) *Service {
	return &Service{cfg: cfg}
}

// view is GitHub as the caller's own token shows it; a caller with no GitHub link gets the error saying how to connect.
func (s *Service) view(ctx context.Context) (GitHubView, error) {
	return s.cfg.People.GitHubView(ctx, actorID(ctx))
}

// Scan reads owner/name with the caller's own GitHub token, so a scan reaches only what they can open themselves.
func (s *Service) Scan(ctx context.Context, workspaceID, owner, name, ref string) (*ScanResult, error) {
	if err := githubapp.ValidateRepository(owner, name); err != nil {
		return nil, err
	}
	if _, err := s.wizardWorkspaces(ctx, workspaceID); err != nil {
		return nil, err
	}
	v, err := s.view(ctx)
	if err != nil {
		return nil, err
	}
	return Scan(ctx, s.cfg.Gate, v, owner, name, ref)
}

// RequireAssigned holds background work on owner/name to its installation being linked to workspaceID, once Nexul
// reads GitHub as its App.
func (s *Service) RequireAssigned(ctx context.Context, workspaceID, owner, name string) error {
	visible, err := s.visibleAccounts(ctx, []string{workspaceID})
	if err != nil || visible == nil {
		return err
	}
	notAssigned := fmt.Errorf("%w: repository is not linked to this workspace", apperrors.ErrNotFound)
	if len(visible) == 0 {
		return notAssigned
	}
	id, err := s.cfg.Accounts.AccountOf(ctx, owner, name)
	if errors.Is(err, apperrors.ErrNotFound) {
		return notAssigned
	}
	if err != nil {
		return err
	}
	if !visible[id] {
		return notAssigned
	}
	return nil
}

// RequireAttach lets a person attach owner/name to a project in workspaceID only when their own GitHub token lists it,
// and links its installation's account to the workspace, so background work may read it as the App (ADR 0147). The
// server's own attaches, which have no person, keep RequireAssigned's check.
func (s *Service) RequireAttach(ctx context.Context, workspaceID, owner, name string) error {
	if identity.Internal(ctx) {
		return s.RequireAssigned(ctx, workspaceID, owner, name)
	}
	repo, err := s.personRepo(ctx, owner, name)
	if err != nil {
		return err
	}
	a := Assignment{AccountID: repo.AccountID, AccountLogin: strings.ToLower(repo.Owner), WorkspaceID: workspaceID}
	_, err = s.cfg.Store.AssignInstallation(ctx, a, installationEvent(TopicInstallationAssigned, a, actorID(ctx)))
	return err
}

// personRepo finds owner/name in the caller's own list, asking GitHub again once before refusing.
func (s *Service) personRepo(ctx context.Context, owner, name string) (Repo, error) {
	v, err := s.view(ctx)
	if err != nil {
		return Repo{}, err
	}
	for _, refresh := range []bool{false, true} {
		repos, err := v.Repos(ctx, refresh)
		if err != nil {
			return Repo{}, fmt.Errorf("list your repositories: %w", err)
		}
		if i := slices.IndexFunc(repos, func(r Repo) bool { return strings.EqualFold(r.FullName, owner+"/"+name) }); i >= 0 {
			return repos[i], nil
		}
	}
	return Repo{}, fmt.Errorf("%w: your GitHub account cannot open %s/%s where the GitHub App is installed; repository_list lists the ones it can", apperrors.ErrNotFound, owner, name)
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

// ListRepos lists the repositories the caller's own GitHub account can open where the App is installed, whose
// owner/name contains q; it never reads as the App or the connector (ADR 0147).
func (s *Service) ListRepos(ctx context.Context, workspaceID, q string, refresh bool) ([]Repo, error) {
	if _, err := s.wizardWorkspaces(ctx, workspaceID); err != nil {
		return nil, err
	}
	q = strings.ToLower(strings.TrimSpace(q))
	if q != "" && utf8.RuneCountInString(q) < MinSearchLength {
		return nil, fmt.Errorf("%w: q needs at least %d characters", apperrors.ErrInvalid, MinSearchLength)
	}
	v, err := s.view(ctx)
	if err != nil {
		return nil, err
	}
	repos, err := v.Repos(ctx, refresh)
	if err != nil {
		return nil, fmt.Errorf("list your repositories: %w", err)
	}
	matches := []Repo{}
	for _, r := range repos {
		if q != "" && !strings.Contains(strings.ToLower(r.FullName), q) {
			continue
		}
		matches = append(matches, r)
	}
	return matches, nil
}

// visibleAccounts is the ids of the accounts linked to workspaces, or nil for all while Nexul reads GitHub as the
// connected account.
func (s *Service) visibleAccounts(ctx context.Context, workspaces []string) (map[int64]bool, error) {
	asApp, err := s.cfg.Installations.ReadsAsApp(ctx)
	if err != nil {
		return nil, fmt.Errorf("read the git provider's mode: %w", err)
	}
	if !asApp {
		return nil, nil
	}
	if err := s.resolveAccounts(ctx); err != nil {
		return nil, err
	}
	accounts, err := s.cfg.Store.AssignedAccountsIn(ctx, workspaces)
	if err != nil {
		return nil, err
	}
	visible := make(map[int64]bool, len(accounts))
	for _, id := range accounts {
		visible[id] = true
	}
	return visible, nil
}

// resolveAccounts records the account id of every link made before ids were kept, the first time one is read.
func (s *Service) resolveAccounts(ctx context.Context) error {
	pending, err := s.cfg.Store.HasUnresolvedAssignments(ctx)
	if err != nil || !pending {
		return err
	}
	live, err := s.cfg.Accounts.InstallationAccounts(ctx)
	if err != nil {
		return fmt.Errorf("resolve installation accounts: %w", err)
	}
	return s.sync(ctx, live)
}

// sync brings the links in line with the installations the App lists: ids recorded, renames followed, and an
// account no longer listed marked gone, with an unassigned event per workspace that loses it.
func (s *Service) sync(ctx context.Context, live []Installation) error {
	rows, err := s.cfg.Store.ListAssignments(ctx)
	if err != nil {
		return err
	}
	plan := planSync(rows, live)
	if plan.empty() {
		return nil
	}
	events := make([]eventbus.OutboxEvent, 0, len(plan.Gone))
	for _, a := range plan.Gone {
		events = append(events, installationEvent(TopicInstallationUnassigned, a, ""))
	}
	return s.cfg.Store.SyncAccounts(ctx, plan, events...)
}

func planSync(rows []Assignment, live []Installation) AccountSync {
	byID := make(map[int64]string, len(live))
	byLogin := make(map[string]int64, len(live))
	for _, inst := range live {
		login := strings.ToLower(inst.AccountLogin)
		byID[inst.AccountID] = login
		byLogin[login] = inst.AccountID
	}
	plan := AccountSync{Resolved: map[string]int64{}, Renamed: map[int64]string{}}
	for _, r := range rows {
		login, listed := byID[r.AccountID]
		switch {
		case r.AccountID == 0 && r.Gone:
		case r.AccountID == 0 && byLogin[r.AccountLogin] != 0:
			plan.Resolved[r.AccountLogin] = byLogin[r.AccountLogin]
		case r.Gone && listed:
			plan.Reinstalled = append(plan.Reinstalled, r.AccountID)
		case r.Gone:
		case !listed:
			r.Gone = true
			plan.Gone = append(plan.Gone, r)
		case login != r.AccountLogin:
			plan.Renamed[r.AccountID] = login
		}
	}
	return plan
}

// ListInstallations lists the installations the caller's own GitHub token sees, so another person's account never
// appears, each with the caller's workspaces that use it: a project there attaches one of its repositories.
func (s *Service) ListInstallations(ctx context.Context) ([]Installation, error) {
	v, err := s.view(ctx)
	if err != nil {
		return nil, err
	}
	installs, err := v.Installations(ctx)
	if err != nil {
		return nil, fmt.Errorf("list your installations: %w", err)
	}
	if err := s.syncAsApp(ctx); err != nil {
		return nil, err
	}
	readable, manageable, err := s.installationViewer(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := s.cfg.Store.AssignmentsIn(ctx, readable)
	if err != nil {
		return nil, err
	}
	out := make([]Installation, 0, len(installs))
	for _, inst := range installs {
		inst.Workspaces = usedBy(rows, inst, manageable)
		out = append(out, inst)
	}
	return out, nil
}

// syncAsApp marks uninstalled accounts gone, which only the App's own list can tell.
func (s *Service) syncAsApp(ctx context.Context) error {
	asApp, err := s.cfg.Installations.ReadsAsApp(ctx)
	if err != nil || !asApp {
		return err
	}
	live, err := s.cfg.Accounts.InstallationAccounts(ctx)
	if err != nil {
		return fmt.Errorf("list the App's installations: %w", err)
	}
	return s.sync(ctx, live)
}

// installationViewer is the caller's workspaces whose use of an account they may see, and those they may detach it from.
func (s *Service) installationViewer(ctx context.Context) (readable, manageable []string, err error) {
	if s.cfg.Gate == nil {
		return nil, nil, permissions.Ungated(ctx)
	}
	readable, err = s.cfg.Gate.WorkspacesWith(ctx, permissions.ProjectsRead)
	if err != nil {
		return nil, nil, err
	}
	manageable, err = s.cfg.Gate.WorkspacesWith(ctx, permissions.ProjectsWrite)
	return readable, manageable, err
}

// usedBy is the workspaces among rows where a project attaches one of inst's repositories.
func usedBy(rows []Assignment, inst Installation, manageable []string) []InstallationWorkspace {
	out := []InstallationWorkspace{}
	for _, r := range rows {
		if r.Gone || !r.Attached || !r.sameAccount(inst.AccountID, inst.AccountLogin) {
			continue
		}
		out = append(out, InstallationWorkspace{ID: r.WorkspaceID, Name: r.WorkspaceName, CanDetach: slices.Contains(manageable, r.WorkspaceID)})
	}
	return out
}

// UnassignInstallation detaches the account from workspaceID: background work there stops reading its repositories
// as the App until someone attaches one of them again. It takes projects:write there.
func (s *Service) UnassignInstallation(ctx context.Context, account, workspaceID string) error {
	account = strings.ToLower(strings.TrimSpace(account))
	if account == "" || workspaceID == "" {
		return fmt.Errorf("%w: an account and a workspace are required", apperrors.ErrInvalid)
	}
	if err := s.requireDetacher(ctx, workspaceID); err != nil {
		return err
	}
	rows, err := s.cfg.Store.AssignmentsIn(ctx, []string{workspaceID})
	if err != nil {
		return err
	}
	for _, r := range rows {
		if r.AccountLogin != account {
			continue
		}
		r.Gone = false
		_, err := s.cfg.Store.UnassignInstallation(ctx, r, installationEvent(TopicInstallationUnassigned, r, actorID(ctx)))
		return err
	}
	return nil
}

func (s *Service) requireDetacher(ctx context.Context, workspaceID string) error {
	if s.cfg.Gate == nil {
		return permissions.Ungated(ctx)
	}
	return s.cfg.Gate.Require(ctx, workspaceID, permissions.ProjectsWrite)
}

func actorID(ctx context.Context) string {
	a, _ := identity.ActorFromCtx(ctx)
	return a.ID
}

// InstallURL is GitHub's page for installing the App on an account the caller owns or administers. It links
// nothing: a workspace uses the account once someone attaches one of its repositories.
func (s *Service) InstallURL(ctx context.Context) (string, error) {
	base, err := s.cfg.Installations.InstallURL(ctx)
	if err != nil {
		return "", err
	}
	if base == "" {
		return "", fmt.Errorf("%w: the GitHub App has no slug registered", apperrors.ErrNotFound)
	}
	return base, nil
}
