package storage

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/ee/sso"
	"github.com/otal-labs/nexul/internal/auth"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/permissions"
	"github.com/otal-labs/nexul/internal/roles"
	"github.com/otal-labs/nexul/internal/tenancy"
)

func seedDefaultRoles(t *testing.T, s *Store) {
	t.Helper()
	now := time.Now().UTC()
	for _, r := range []*roles.Role{
		{ID: "role-owner", WorkspaceID: tenancy.DefaultWorkspaceID, Name: "Owner", IsOwnerRole: true, CreatedAt: now, UpdatedAt: now},
		{ID: "role-member", WorkspaceID: tenancy.DefaultWorkspaceID, Name: "Member", Permissions: permissions.SetOf(permissions.DocsRead), CreatedAt: now, UpdatedAt: now},
	} {
		require.NoError(t, s.Roles.Create(t.Context(), r))
	}
}

func TestSSORepo_Settings_AreSealedAtRestAndRoundTrip(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := t.Context()
	got, err := s.SSO.Get(ctx)
	require.NoError(t, err)
	assert.Equal(t, sso.Settings{}, got, "a fresh instance has no provider")

	want := sso.Settings{
		IssuerURL: "https://login.example.com", ClientID: "client-1", ClientSecret: "op://vault/item/field", DisplayName: "Acme SSO",
		JoinRoleID: "role-member", Only: true, UpdatedBy: "u-1", UpdatedAt: time.Unix(1_000, 0).UTC(),
	}
	evt := eventbus.OutboxEvent{ID: "e1", Topic: sso.TopicChanged, Payload: sso.ChangedEvent{}}
	require.NoError(t, s.SSO.Set(ctx, want, evt))

	got, err = s.SSO.Get(ctx)
	require.NoError(t, err)
	assert.Equal(t, want, got)
	var raw string
	require.NoError(t, s.db.QueryRowContext(ctx, `SELECT client_secret FROM sso_settings`).Scan(&raw))
	assert.NotContains(t, raw, "vault", "the database holds the secret sealed, so a copied file carries none")

	want.DisplayName = "Acme"
	want.Only = false
	require.NoError(t, s.SSO.Set(ctx, want))
	got, err = s.SSO.Get(ctx)
	require.NoError(t, err)
	assert.Equal(t, want, got, "a second save replaces the one row")

	require.NoError(t, s.SSO.Delete(ctx, eventbus.OutboxEvent{ID: "e2", Topic: sso.TopicChanged, Payload: sso.ChangedEvent{Removed: true}}))
	got, err = s.SSO.Get(ctx)
	require.NoError(t, err)
	assert.Equal(t, sso.Settings{}, got)
	pending, err := s.Outbox.Unpublished(ctx, 10)
	require.NoError(t, err)
	assert.Len(t, pending, 2)
}

func TestSSORepo_Create_AdmitsLikeAnInvitationDoes(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := t.Context()
	seedDefaultRoles(t, s)
	id := sso.Identity{Key: "https://login.example.com#sub-1", Email: "alice@example.com", Name: "Alice", AvatarURL: "https://example.com/a.png"}

	require.NoError(t, s.SSO.Create(ctx, "u-alice", id, "role-member", time.Unix(500, 0)))

	acct, err := s.SSO.BySubject(ctx, id.Key)
	require.NoError(t, err)
	assert.Equal(t, sso.Account{UserID: "u-alice", Status: "active"}, acct)
	user, err := s.Users.GetUserByID(ctx, "u-alice")
	require.NoError(t, err)
	assert.Equal(t, "alice@example.com", user.Login)
	member, err := s.WorkspaceMembers.Member(ctx, tenancy.DefaultWorkspaceID, "u-alice")
	require.NoError(t, err)
	assert.Equal(t, []string{"role-member"}, member.RoleIDs)
	var topics []string
	pending, err := s.Outbox.Unpublished(ctx, 10)
	require.NoError(t, err)
	for _, e := range pending {
		topics = append(topics, e.Topic)
	}
	assert.ElementsMatch(t, []string{auth.TopicAccountAdmitted, tenancy.TopicWorkspaceMemberAdded}, topics)
}

func TestSSORepo_Create_ASecondSignInOfTheSameSubject_LeavesNoHalfMadeAccount(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := t.Context()
	seedDefaultRoles(t, s)
	id := sso.Identity{Key: "https://login.example.com#sub-1", Email: "alice@example.com"}
	require.NoError(t, s.SSO.Create(ctx, "u-1", id, "role-member", time.Unix(500, 0)))

	err := s.SSO.Create(ctx, "u-2", id, "role-member", time.Unix(600, 0))

	require.ErrorIs(t, err, apperrs.ErrConflict, "two sign-ins racing to make the same person leave one account")
	_, err = s.Users.GetUserByID(ctx, "u-2")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}

func TestSSORepo_ByEmail_MatchesAVerifiedEmailSignInOnly(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := t.Context()
	for _, u := range []*auth.Identity{
		{UserID: "u-google", Provider: auth.ProviderGoogle, ProviderUserID: "g1", Login: "Alice@Example.com"},
		{UserID: "u-discord", Provider: auth.ProviderDiscord, ProviderUserID: "d1", Login: "bob@example.com"},
		{UserID: "u-github", Provider: auth.ProviderGitHub, ProviderUserID: "gh1", Login: "carol@example.com"},
	} {
		_, _, err := s.Users.UpsertUser(ctx, u)
		require.NoError(t, err)
	}
	require.NoError(t, s.Users.SetAccountStatus(ctx, "u-discord", auth.AccountDisabled))

	for email, want := range map[string]sso.Account{
		"alice@example.com": {UserID: "u-google", Status: "active"},
		"ALICE@EXAMPLE.COM": {UserID: "u-google", Status: "active"},
		"bob@example.com":   {UserID: "u-discord", Status: "disabled"},
	} {
		got, err := s.SSO.ByEmail(ctx, email)
		require.NoError(t, err, email)
		assert.Equal(t, want, got, email)
	}
	_, err := s.SSO.ByEmail(ctx, "carol@example.com")
	require.ErrorIs(t, err, apperrs.ErrNotFound, "a GitHub login is a handle someone chose, not an email the provider verified")
	_, err = s.SSO.ByEmail(ctx, "nobody@example.com")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}

func TestSSORepo_Link_LetsTheAccountSignInThroughTheProvider(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := t.Context()
	_, _, err := s.Users.UpsertUser(ctx, &auth.Identity{UserID: "u-google", Provider: auth.ProviderGoogle, ProviderUserID: "g1", Login: "alice@example.com"})
	require.NoError(t, err)
	id := sso.Identity{Key: "https://login.example.com#sub-1", Email: "alice@example.com", Name: "Alice"}

	require.NoError(t, s.SSO.Link(ctx, "u-google", id))

	acct, err := s.SSO.BySubject(ctx, id.Key)
	require.NoError(t, err)
	assert.Equal(t, "u-google", acct.UserID)
	identities, err := s.Users.ListIdentities(ctx, "u-google")
	require.NoError(t, err)
	assert.Len(t, identities, 2, "the Google sign-in stays")
	_, _, err = s.Users.UpsertUser(ctx, &auth.Identity{UserID: "u-other", Provider: auth.ProviderGoogle, ProviderUserID: "g2", Login: "bob@example.com"})
	require.NoError(t, err)
	require.ErrorIs(t, s.SSO.Link(ctx, "u-other", id), apperrs.ErrConflict, "a subject belongs to one account")
}

func TestSSORepo_CanJoinAs_IsARoleOfTheDefaultWorkspaceThatIsNotItsOwner(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := t.Context()
	seedDefaultRoles(t, s)
	require.NoError(t, s.Workspaces.Create(ctx, newTestWorkspace("ws-2", "Other")))
	require.NoError(t, s.Roles.Create(ctx, &roles.Role{ID: "role-elsewhere", WorkspaceID: "ws-2", Name: "Member", CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}))

	for roleID, want := range map[string]bool{"role-member": true, "role-owner": false, "role-elsewhere": false, "role-gone": false} {
		got, err := s.SSO.CanJoinAs(ctx, roleID)
		require.NoError(t, err)
		assert.Equal(t, want, got, roleID)
	}
}
