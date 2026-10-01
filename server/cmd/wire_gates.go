package main

import (
	"context"
	"errors"
	"time"

	"net/http"

	"github.com/otal-labs/nexul/internal/access"
	"github.com/otal-labs/nexul/internal/attachments"
	"github.com/otal-labs/nexul/internal/auth"
	"github.com/otal-labs/nexul/internal/automations"
	"github.com/otal-labs/nexul/internal/chat"
	"github.com/otal-labs/nexul/internal/connectors"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/githubapp"
	"github.com/otal-labs/nexul/internal/platform/permissions"
	"github.com/otal-labs/nexul/internal/platform/storage"
	"github.com/otal-labs/nexul/internal/plays"
	"github.com/otal-labs/nexul/internal/presence"
	"github.com/otal-labs/nexul/internal/roles"
	"github.com/otal-labs/nexul/internal/tenancy"
)

// allowlistGate adapts auth's sign-in allowlist to tenancy's AllowlistGate seam (ADR 0017: tenancy never imports auth).
type allowlistGate struct {
	svc *auth.Service
}

func (g allowlistGate) IsAllowlisted(ctx context.Context, login string) (bool, error) {
	return g.svc.IsLoginAllowlisted(ctx, login)
}

// userLookupGate adapts auth's user lookups to tenancy's UserLookupGate seam (ADR 0017: tenancy never imports auth).
type userLookupGate struct {
	svc *auth.Service
}

func (g userLookupGate) UserIDForLogin(ctx context.Context, login string) (string, bool, error) {
	return g.svc.UserIDForLogin(ctx, login)
}

func (g userLookupGate) LoginForUserID(ctx context.Context, userID string) (string, error) {
	u, err := g.svc.GetUserByID(ctx, userID)
	if err != nil {
		return "", err
	}
	return u.Login, nil
}

// accountGate adapts auth's registered accounts to tenancy's AccountGate seam (ADR 0017: tenancy never imports auth).
type accountGate struct {
	svc      *auth.Service
	presence *presence.Keeper
}

// Presence reads online from the live-events sockets, since last_active_at is only written once an hour.
func (g accountGate) Presence(ctx context.Context) (map[string]bool, map[string]time.Time, error) {
	seen, err := g.svc.LastSeen(ctx)
	if err != nil {
		return nil, nil, err
	}
	return g.presence.Online(), seen, nil
}

func (g accountGate) ListAccounts(ctx context.Context) ([]*tenancy.TeamAccount, error) {
	users, err := g.svc.ListUsers(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*tenancy.TeamAccount, 0, len(users))
	for _, u := range users {
		out = append(out, toTeamAccount(u))
	}
	return out, nil
}

func (g accountGate) Account(ctx context.Context, userID string) (*tenancy.TeamAccount, error) {
	u, err := g.svc.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return toTeamAccount(u), nil
}

func toTeamAccount(u *auth.User) *tenancy.TeamAccount {
	status := u.AccountStatus
	if status == "" {
		status = auth.AccountActive
	}
	a := &tenancy.TeamAccount{
		ID: u.ID, Login: u.Login, Name: u.Name, AvatarURL: u.AvatarURL,
		Status: string(status), CreatedAt: u.CreatedAt,
	}
	if u.DisplayName != nil {
		a.DisplayName = *u.DisplayName
	}
	if u.AvatarOverrideURL != nil {
		a.AvatarOverride = *u.AvatarOverrideURL
	}
	return a
}

// connectorAppSeederGate adapts connectors' app-config store to auth's seam (ADR 0017: auth never imports connectors).
type connectorAppSeederGate struct {
	store connectors.AppConfigStore
}

func (g connectorAppSeederGate) SeedGitHubApp(ctx context.Context, clientID, clientSecret, appSlug string) error {
	return g.store.SetAppConfig(ctx, connectors.AppConfig{
		ConnectorID:  "github",
		ClientID:     clientID,
		ClientSecret: clientSecret,
		AppSlug:      appSlug,
	})
}

// githubAppVerifierGate adapts platform/githubapp to auth's seam so Bootstrap never stores an App GitHub doesn't recognise.
type githubAppVerifierGate struct {
	hc *http.Client
}

func (g githubAppVerifierGate) VerifyGitHubApp(ctx context.Context, clientID, clientSecret, appSlug string) error {
	return githubapp.Verify(ctx, g.hc, githubapp.DefaultAPIBase, clientID, clientSecret, appSlug)
}

func (g githubAppVerifierGate) VerifyGitHubAppCheck(ctx context.Context, check, clientID, clientSecret, appSlug string) error {
	return githubapp.VerifyCheck(ctx, g.hc, githubapp.DefaultAPIBase, check, clientID, clientSecret, appSlug)
}

// pendingInviteResolverGate adapts tenancy's ResolvePendingInvites to auth's seam (ADR 0017: auth never imports tenancy).
type pendingInviteResolverGate struct {
	svc *tenancy.Service
}

func (g pendingInviteResolverGate) ResolvePendingInvites(ctx context.Context, login, userID string) error {
	return g.svc.ResolvePendingInvites(ctx, login, userID)
}

// defaultWorkspaceGate adapts tenancy's BindDefaultWorkspaceOwner to auth's seam (ADR 0017: auth never imports tenancy).
type defaultWorkspaceGate struct {
	svc *tenancy.Service
}

func (g defaultWorkspaceGate) BindDefaultWorkspaceOwner(ctx context.Context, userID string) error {
	return g.svc.BindDefaultWorkspaceOwner(ctx, userID)
}

// workspaceGate adapts tenancy's Get to workspace's WorkspaceGate seam (ADR 0017); mirrors deployProjectStore's pattern.
type workspaceGate struct {
	svc *tenancy.Service
}

func (g workspaceGate) WorkspaceExists(ctx context.Context, id string) (bool, error) {
	_, err := g.svc.Get(ctx, id)
	if err != nil {
		if errors.Is(err, apperrs.ErrNotFound) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// roleGate adapts roles' CreateOwnerRole to tenancy's consumer-side RoleGate seam (ADR 0017: tenancy never imports roles).
type roleGate struct {
	svc *roles.Service
}

func (g roleGate) CreateOwnerRole(ctx context.Context, workspaceID string) (string, error) {
	r, err := g.svc.CreateOwnerRole(ctx, workspaceID)
	if err != nil {
		return "", err
	}
	return r.ID, nil
}

// roleNameGate backs the account row's role badge via tenancy's RoleNameGate seam (ADR 0017).
type roleNameGate struct {
	svc *roles.Service
}

func (g roleNameGate) RoleName(ctx context.Context, workspaceID, roleID string) (string, error) {
	r, err := g.svc.Get(ctx, workspaceID, roleID)
	if err != nil {
		return "", err
	}
	return r.Name, nil
}

func (g roleNameGate) IsOwnerRole(ctx context.Context, workspaceID, roleID string) (bool, error) {
	r, err := g.svc.Get(ctx, workspaceID, roleID)
	if err != nil {
		return false, err
	}
	return r.IsOwnerRole, nil
}

func (g roleNameGate) RolePermissions(ctx context.Context, workspaceID, roleID string) (permissions.Set, error) {
	r, err := g.svc.Get(ctx, workspaceID, roleID)
	if err != nil {
		return nil, err
	}
	return r.Permissions, nil
}

// workspacePermissionGate adapts access's HasPermission to tenancy's seam (ADR 0017); backs the `/me` permissions field.
type workspacePermissionGate struct {
	svc *access.Service
}

func (g workspacePermissionGate) WorkspacePermissions(ctx context.Context, userID, workspaceID string) []string {
	return g.svc.WorkspacePermissions(ctx, userID, workspaceID)
}

// memoriesProjectLookup reads a project's workspace from storage for memories' ProjectLookup seam (ADR 0017), so a
// new memory can denormalize it.
type memoriesProjectLookup struct {
	projects *storage.ProjectsRepo
}

func (g memoriesProjectLookup) WorkspaceForProject(ctx context.Context, projectID string) (string, error) {
	p, err := g.projects.Get(ctx, projectID)
	if err != nil {
		return "", err
	}
	return p.WorkspaceID, nil
}

// channelGate adapts chat's EnsureGeneralChannel to tenancy's ChannelGate seam (ADR 0017: tenancy never imports chat).
type channelGate struct {
	svc *chat.Service
}

func (g channelGate) CreateGeneralChannel(ctx context.Context, workspaceID, creatorUserID string) error {
	return g.svc.EnsureGeneralChannel(ctx, workspaceID, creatorUserID)
}

// roleMemberGate adapts tenancy's MemberRoleID to roles' MemberGate seam (ADR 0017: roles never imports tenancy).
type roleMemberGate struct {
	svc *tenancy.Service
}

func (g roleMemberGate) MemberRoleID(ctx context.Context, workspaceID, userID string) (string, error) {
	return g.svc.MemberRoleID(ctx, workspaceID, userID)
}

// workspaceDefaultsGate seeds a workspace's default plays and automations for tenancy's DefaultsGate seam (ADR 0017).
type workspaceDefaultsGate struct {
	plays       *plays.Service
	automations *automations.Seeder
}

func (g workspaceDefaultsGate) SeedWorkspaceDefaults(ctx context.Context, workspaceID string) error {
	if err := g.plays.SeedDefaults(ctx, workspaceID); err != nil {
		return err
	}
	return g.automations.SeedWorkspace(ctx, workspaceID)
}

// playsPermissionGate adapts access's HasPermission to plays' seam (ADR 0017).
type playsPermissionGate struct {
	svc *access.Service
}

func (g playsPermissionGate) HasPermission(ctx context.Context, userID, workspaceID string, action permissions.Action, resourceType, resourceID string) bool {
	return g.svc.HasPermission(ctx, userID, workspaceID, action, resourceType, resourceID)
}

// notificationPermissionGate adapts access to workspace notifications' PermissionChecker seam (ADR 0017), so a
// notice only reaches someone who may read its subject.
type notificationPermissionGate struct {
	svc *access.Service
}

func (g notificationPermissionGate) HasPermission(ctx context.Context, userID, workspaceID string, action permissions.Action) bool {
	return g.svc.HasPermission(ctx, userID, workspaceID, action, "", "")
}

func (g notificationPermissionGate) CanReadDoc(ctx context.Context, userID, docID string) bool {
	ok, err := g.svc.Can(ctx, userID, docID, permissions.DocsRead)
	return err == nil && ok
}

// memoriesAttachmentsGate adapts attachments' owner-copy API to memories' AttachmentsCopier seam (ADR 0017:
// memories never imports attachments), for Clone.
type memoriesAttachmentsGate struct {
	svc *attachments.Service
}

func (g memoriesAttachmentsGate) ListOwnerIDs(ctx context.Context, memoryID string) ([]string, error) {
	return g.svc.ListOwnerAttachmentIDs(ctx, attachments.Owner{MemoryID: memoryID})
}

func (g memoriesAttachmentsGate) CopyOwnerWithIDs(ctx context.Context, fromMemoryID, toMemoryID string, idMap map[string]string) error {
	return g.svc.CopyAttachmentsWithIDs(ctx, attachments.Owner{MemoryID: fromMemoryID}, attachments.Owner{MemoryID: toMemoryID}, idMap)
}

// docsAttachmentsGate is memoriesAttachmentsGate for docs' Clone, keyed by doc id.
type docsAttachmentsGate struct {
	svc *attachments.Service
}

func (g docsAttachmentsGate) ListOwnerIDs(ctx context.Context, docID string) ([]string, error) {
	return g.svc.ListOwnerAttachmentIDs(ctx, attachments.Owner{DocID: docID})
}

func (g docsAttachmentsGate) CopyOwnerWithIDs(ctx context.Context, fromDocID, toDocID string, idMap map[string]string) error {
	return g.svc.CopyAttachmentsWithIDs(ctx, attachments.Owner{DocID: fromDocID}, attachments.Owner{DocID: toDocID}, idMap)
}

// membershipGate adapts tenancy's raw membership store to chat's membership seam (a DM's participants), ADR 0017.
type membershipGate struct {
	members *storage.WorkspaceMembersRepo
}

func (g membershipGate) IsMember(ctx context.Context, userID, workspaceID string) (bool, error) {
	_, err := g.members.RoleIDFor(ctx, workspaceID, userID)
	if err != nil {
		if errors.Is(err, apperrs.ErrNotFound) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (g membershipGate) MemberIDs(ctx context.Context, workspaceID string) ([]string, error) {
	members, err := g.members.ListByWorkspace(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	ids := make([]string, len(members))
	for i, m := range members {
		ids[i] = m.UserID
	}
	return ids, nil
}

// chatStanding answers chat's Owner and Restricted member lookups from the person's membership.
type chatStanding struct {
	roles accessRoleResolver
}

func (c chatStanding) IsOwner(ctx context.Context, userID, workspaceID string) (bool, error) {
	info, err := c.roles.MemberRole(ctx, workspaceID, userID)
	if errors.Is(err, apperrs.ErrNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return info.IsOwnerRole, nil
}

// IsRestricted answers no until restricted memberships exist (.scratch/project-access ticket 08).
func (chatStanding) IsRestricted(context.Context, string, string) (bool, error) {
	return false, nil
}

// chatThreadGate checks a ticket thread through its ticket's project and an interview thread through its project.
type chatThreadGate struct {
	projectEntityGate
}

func (g chatThreadGate) RequireProject(ctx context.Context, projectID string, action permissions.Action) error {
	return g.access.RequireProject(ctx, projectID, action)
}

// chatAttachmentConversations lets attachments read a conversation-owned file through reading the conversation.
type chatAttachmentConversations struct {
	svc *chat.Service
}

func (g chatAttachmentConversations) RequireRead(ctx context.Context, conversationID string) error {
	_, err := g.svc.GetConversation(ctx, conversationID)
	return err
}

// memoryAttachmentsAccessGate checks a memory's file through its project for attachments' MemoryAccessChecker
// seam (ADR 0017, ADR 0099); the actor comes from ctx, the same one attachments read the user id from.
type memoryAttachmentsAccessGate struct {
	memories *storage.MemoriesRepo
	access   *access.Service
}

func (g memoryAttachmentsAccessGate) Can(ctx context.Context, _, memoryID string, action permissions.Action) (bool, error) {
	m, err := g.memories.GetByID(ctx, memoryID)
	if err != nil {
		return false, err
	}
	return g.access.RequireProject(ctx, m.ProjectID, action) == nil, nil
}
