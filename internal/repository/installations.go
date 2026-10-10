package repository

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	apperrors "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/githubapp"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// installStatePrefix marks the state an install link carries, so the sign-in callback can tell it from its own.
const installStatePrefix = "install."

// installStateTTL is how long an unused install link works; claiming it uses it up.
const installStateTTL = 24 * time.Hour

// Config wires the repository use-cases.
type Config struct {
	Gate          Gate
	Scanner       Scanner
	Installations InstallationLister
	Accounts      AccountResolver
	Store         InstallationStore
	States        InstallStateStore
	Installers    InstallerChecker
	Now           func() time.Time
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

// Scan confines a wizard scan to the named workspace, or the caller's writable workspaces when omitted.
func (s *Service) Scan(ctx context.Context, workspaceID, owner, name, ref string) (*ScanResult, error) {
	if err := githubapp.ValidateRepository(owner, name); err != nil {
		return nil, err
	}
	workspaces, err := s.wizardWorkspaces(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	if err := s.requireAssigned(ctx, workspaces, owner, name); err != nil {
		return nil, err
	}
	return Scan(ctx, s.cfg.Gate, s.cfg.Scanner, owner, name, ref)
}

// RequireAssigned holds an operation on owner/name to its installation being assigned to workspaceID, once Nexul
// reads GitHub as its App; background work passes through it too.
func (s *Service) RequireAssigned(ctx context.Context, workspaceID, owner, name string) error {
	return s.requireAssigned(ctx, []string{workspaceID}, owner, name)
}

func (s *Service) requireAssigned(ctx context.Context, workspaces []string, owner, name string) error {
	visible, err := s.visibleAccounts(ctx, workspaces)
	if err != nil || visible == nil {
		return err
	}
	notAssigned := fmt.Errorf("%w: repository is not assigned to this workspace", apperrors.ErrNotFound)
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
		if visible != nil && !visible[r.AccountID] {
			continue
		}
		if q != "" && !strings.Contains(strings.ToLower(r.FullName), q) {
			continue
		}
		matches = append(matches, r)
	}
	return matches, nil
}

// visibleAccounts is the ids of the accounts workspaces see, or nil for all while Nexul reads GitHub as the
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

// resolveAccounts records the account id of every assignment made before ids were kept, the first time one is read.
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

// sync brings the assignments in line with the installations the App lists: ids recorded, renames followed, and an
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

// ListInstallations lists the installations assigned to the workspaces where the caller holds connectors:read, and to
// someone holding connectors:write the unassigned ones, with only the workspaces the caller may read.
func (s *Service) ListInstallations(ctx context.Context) ([]Installation, error) {
	readable, all, manager, err := s.installationViewer(ctx)
	if err != nil {
		return nil, err
	}
	installs, err := s.cfg.Installations.ListInstallations(ctx)
	if err != nil {
		return nil, fmt.Errorf("list installations: %w", err)
	}
	asApp, err := s.cfg.Installations.ReadsAsApp(ctx)
	if err != nil {
		return nil, fmt.Errorf("read the git provider's mode: %w", err)
	}
	if asApp {
		if err := s.sync(ctx, installs); err != nil {
			return nil, err
		}
	}
	visible, err := s.visibleAssignments(ctx, readable, all)
	if err != nil {
		return nil, err
	}
	assigned, err := s.cfg.Store.AssignedAccounts(ctx)
	if err != nil {
		return nil, err
	}
	out := []Installation{}
	for _, inst := range installs {
		inst.Workspaces = workspacesOf(visible, inst.AccountID, inst.AccountLogin, false)
		if len(inst.Workspaces) == 0 && (!manager || slices.Contains(assigned, inst.AccountID)) {
			continue
		}
		out = append(out, inst)
	}
	return append(out, goneInstallations(visible)...), nil
}

// installationViewer is the workspaces whose installations the caller reads and whether they manage unassigned ones.
func (s *Service) installationViewer(ctx context.Context) (readable []string, all, manager bool, err error) {
	if s.cfg.Gate == nil || identity.Internal(ctx) {
		return nil, true, true, permissions.Ungated(ctx)
	}
	readable, err = s.cfg.Gate.WorkspacesWith(ctx, permissions.ConnectorsRead)
	if err != nil {
		return nil, false, false, err
	}
	manager = s.cfg.Gate.RequireAnywhere(ctx, permissions.ConnectorsWrite) == nil
	if len(readable) == 0 && !manager {
		return nil, false, false, fmt.Errorf("%w: %s required", apperrors.ErrForbidden, permissions.ConnectorsRead)
	}
	return readable, false, manager, nil
}

func (s *Service) visibleAssignments(ctx context.Context, readable []string, all bool) ([]Assignment, error) {
	if all {
		return s.cfg.Store.ListAssignments(ctx)
	}
	return s.cfg.Store.AssignmentsIn(ctx, readable)
}

func workspacesOf(rows []Assignment, id int64, login string, gone bool) []InstallationWorkspace {
	var out []InstallationWorkspace
	for _, r := range rows {
		if r.Gone == gone && r.sameAccount(id, login) {
			out = append(out, InstallationWorkspace{ID: r.WorkspaceID, Name: r.WorkspaceName})
		}
	}
	return out
}

// goneInstallations lists each uninstalled account still assigned, so its workspaces can be cleared.
func goneInstallations(rows []Assignment) []Installation {
	var out []Installation
	for _, r := range rows {
		if !r.Gone || slices.ContainsFunc(out, func(i Installation) bool { return r.sameAccount(i.AccountID, i.AccountLogin) }) {
			continue
		}
		out = append(out, Installation{
			AccountID: r.AccountID, AccountLogin: r.AccountLogin, Gone: true,
			Workspaces: workspacesOf(rows, r.AccountID, r.AccountLogin, true),
		})
	}
	return out
}

// AssignInstallation lets workspaceID list account's repositories. It takes connectors:write there, the
// instance-level connectors:write for an unassigned installation, and connectors:write in a workspace already holding
// it for one assigned elsewhere, which stays hidden from anyone else.
func (s *Service) AssignInstallation(ctx context.Context, account, workspaceID string) error {
	login, err := s.requireAssigner(ctx, account, workspaceID)
	if err != nil {
		return err
	}
	inst, err := s.installedAccount(ctx, login)
	if err != nil {
		return err
	}
	rows, err := s.cfg.Store.ListAssignments(ctx)
	if err != nil {
		return err
	}
	holders := holdersOf(rows, inst.AccountID, login)
	if slices.Contains(holders, workspaceID) {
		return nil
	}
	if err := s.requireReassign(ctx, holders); err != nil {
		return err
	}
	a := Assignment{AccountID: inst.AccountID, AccountLogin: login, WorkspaceID: workspaceID}
	_, err = s.cfg.Store.AssignInstallation(ctx, a, installationEvent(TopicInstallationAssigned, a, actorID(ctx)))
	return err
}

func (s *Service) requireReassign(ctx context.Context, holders []string) error {
	if s.cfg.Gate == nil {
		return permissions.Ungated(ctx)
	}
	if len(holders) == 0 {
		return s.cfg.Gate.RequireAnywhere(ctx, permissions.ConnectorsWrite)
	}
	for _, id := range holders {
		if s.cfg.Gate.Require(ctx, id, permissions.ConnectorsWrite) == nil {
			return nil
		}
	}
	return fmt.Errorf("%w: no installation on that account", apperrors.ErrNotFound)
}

// installedAccount is the installation on login, ErrNotFound when the App is not installed there.
func (s *Service) installedAccount(ctx context.Context, login string) (Installation, error) {
	installs, err := s.cfg.Accounts.InstallationAccounts(ctx)
	if err != nil {
		return Installation{}, fmt.Errorf("list installations: %w", err)
	}
	for _, inst := range installs {
		if strings.EqualFold(inst.AccountLogin, login) {
			return inst, nil
		}
	}
	return Installation{}, fmt.Errorf("%w: the GitHub App is not installed on %s", apperrors.ErrNotFound, login)
}

// holdersOf lists the workspaces account is assigned to and not gone.
func holdersOf(rows []Assignment, id int64, login string) []string {
	var out []string
	for _, r := range rows {
		if !r.Gone && r.sameAccount(id, login) {
			out = append(out, r.WorkspaceID)
		}
	}
	return out
}

// UnassignInstallation stops workspaceID seeing the repositories of the installation on account; it takes
// connectors:write there.
func (s *Service) UnassignInstallation(ctx context.Context, account, workspaceID string) error {
	login, err := s.requireAssigner(ctx, account, workspaceID)
	if err != nil {
		return err
	}
	rows, err := s.cfg.Store.AssignmentsIn(ctx, []string{workspaceID})
	if err != nil {
		return err
	}
	for _, r := range rows {
		if r.AccountLogin != login {
			continue
		}
		r.Gone = false
		_, err := s.cfg.Store.UnassignInstallation(ctx, r, installationEvent(TopicInstallationUnassigned, r, actorID(ctx)))
		return err
	}
	return nil
}

func (s *Service) requireAssigner(ctx context.Context, account, workspaceID string) (string, error) {
	account = strings.ToLower(strings.TrimSpace(account))
	if account == "" || workspaceID == "" {
		return "", fmt.Errorf("%w: an account and a workspace are required", apperrors.ErrInvalid)
	}
	if s.cfg.Gate == nil {
		return account, permissions.Ungated(ctx)
	}
	return account, s.cfg.Gate.Require(ctx, workspaceID, permissions.ConnectorsWrite)
}

func actorID(ctx context.Context) string {
	a, _ := identity.ActorFromCtx(ctx)
	return a.ID
}

// InstallURL is GitHub's install page carrying a state for workspaceID, made for a signed-in caller holding
// projects:write there; the state works once, for a day.
func (s *Service) InstallURL(ctx context.Context, workspaceID string) (string, error) {
	if workspaceID == "" {
		return "", fmt.Errorf("%w: workspace_id is required", apperrors.ErrInvalid)
	}
	if _, err := s.wizardWorkspaces(ctx, workspaceID); err != nil {
		return "", err
	}
	userID := actorID(ctx)
	if userID == "" {
		return "", fmt.Errorf("%w: an install link is made for a signed-in person", apperrors.ErrUnauthorized)
	}
	base, err := s.cfg.Installations.InstallURL(ctx)
	if err != nil {
		return "", err
	}
	if base == "" {
		return "", fmt.Errorf("%w: the GitHub App has no slug registered", apperrors.ErrNotFound)
	}
	state := installStatePrefix + rand.Text()
	now := s.cfg.Now()
	st := InstallState{WorkspaceID: workspaceID, UserID: userID, ExpiresAt: now.Add(installStateTTL)}
	if err := s.cfg.States.SaveInstallState(ctx, stateHash(state), st, now); err != nil {
		return "", err
	}
	return base + "?" + url.Values{"state": {state}}.Encode(), nil
}

func stateHash(state string) string {
	sum := sha256.Sum256([]byte(state))
	return hex.EncodeToString(sum[:])
}

// ClaimsState reports whether state is one an install link carried.
func (s *Service) ClaimsState(state string) bool {
	return strings.HasPrefix(state, installStatePrefix)
}

// ClaimInstallation assigns the installation an install link led to the link's workspace. The link works once and
// only while its maker may still add projects there; the installer must be the account itself or an admin of the
// organisation; and an account already assigned elsewhere is left to a connector manager.
func (s *Service) ClaimInstallation(ctx context.Context, state, code, installationID string) (string, error) {
	st, err := s.consumeState(ctx, state)
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
	maker := identity.WithActor(ctx, identity.Actor{ID: st.UserID})
	if _, err := s.wizardWorkspaces(maker, st.WorkspaceID); err != nil {
		return "", fmt.Errorf("the install link's maker can no longer add projects to its workspace: %w", err)
	}
	inst, err := s.cfg.Installers.Installer(ctx, code, id)
	if err != nil {
		return "", fmt.Errorf("confirm installation %d: %w", id, err)
	}
	if !inst.Admin {
		return "", fmt.Errorf("%w: the installer is not shown to administer %s, so a connector manager assigns it", apperrors.ErrForbidden, inst.AccountLogin)
	}
	if err := s.claim(ctx, inst, st); err != nil {
		return "", err
	}
	return st.WorkspaceID, nil
}

func (s *Service) consumeState(ctx context.Context, state string) (InstallState, error) {
	if !s.ClaimsState(state) {
		return InstallState{}, fmt.Errorf("%w: not an install link's state", apperrors.ErrInvalid)
	}
	st, err := s.cfg.States.ConsumeInstallState(ctx, stateHash(state))
	if errors.Is(err, apperrors.ErrNotFound) {
		return InstallState{}, fmt.Errorf("%w: the install link was already used or not made by this instance; open a fresh one from the project wizard", apperrors.ErrInvalid)
	}
	if err != nil {
		return InstallState{}, err
	}
	if !s.cfg.Now().Before(st.ExpiresAt) {
		return InstallState{}, fmt.Errorf("%w: the install link has expired; open a fresh one from the project wizard", apperrors.ErrInvalid)
	}
	return st, nil
}

func (s *Service) claim(ctx context.Context, inst Installer, st InstallState) error {
	login := strings.ToLower(inst.AccountLogin)
	rows, err := s.cfg.Store.ListAssignments(ctx)
	if err != nil {
		return err
	}
	holders := holdersOf(rows, inst.AccountID, login)
	if slices.Contains(holders, st.WorkspaceID) {
		return nil
	}
	if len(holders) > 0 {
		return fmt.Errorf("%w: %s is already assigned to another workspace; a connector manager reassigns it", apperrors.ErrConflict, login)
	}
	a := Assignment{AccountID: inst.AccountID, AccountLogin: login, WorkspaceID: st.WorkspaceID}
	_, err = s.cfg.Store.AssignInstallation(ctx, a, installationEvent(TopicInstallationAssigned, a, st.UserID))
	return err
}
