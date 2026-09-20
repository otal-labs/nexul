package storage

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/auth"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

func newTestUser(id, providerID, login string) *auth.User {
	return &auth.User{
		ID:             id,
		Provider:       auth.ProviderGitHub,
		ProviderUserID: providerID,
		Login:          login,
		Name:           "Name " + login,
		AvatarURL:      "https://avatar/" + login,
	}
}

func TestUsersRepo_UpsertUser_Create(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	u, created, err := s.Users.UpsertUser(context.Background(), newTestUser("u1", "42", "onik97"))
	require.NoError(t, err)
	assert.True(t, created)
	assert.Equal(t, "onik97", u.Login)
	assert.False(t, u.CanCreateWorkspace)
	assert.False(t, u.CreatedAt.IsZero())
}

func TestUsersRepo_UpsertUser_UpdatesProviderFieldsNotFlags(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	_, _, err := s.Users.UpsertUser(context.Background(), newTestUser("u1", "42", "old-login"))
	require.NoError(t, err)
	require.NoError(t, s.Users.SetCanCreateWorkspace(context.Background(), "u1", true))
	require.NoError(t, s.Users.MarkFirstLoginDone(context.Background(), "u1"))

	renamed := newTestUser("", "42", "new-login")
	renamed.Name = "Renamed"
	u, created, err := s.Users.UpsertUser(context.Background(), renamed)
	require.NoError(t, err)
	assert.False(t, created)
	assert.Equal(t, "new-login", u.Login)
	assert.Equal(t, "Renamed", u.Name)
	assert.True(t, u.CanCreateWorkspace, "re-login must not reset owner flag")
	assert.True(t, u.FirstLoginDone, "re-login must not reset onboarding flag")

	byID, err := s.Users.GetUserByID(context.Background(), "u1")
	require.NoError(t, err)
	assert.Equal(t, "42", byID.ProviderUserID, "provider key is stable across login renames")
}

func TestUsersRepo_GetUserByID_NotFound(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	_, err := s.Users.GetUserByID(context.Background(), "missing")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}

func TestUsersRepo_CanCreateWorkspaceExists(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	exists, err := s.Users.CanCreateWorkspaceExists(context.Background())
	require.NoError(t, err)
	assert.False(t, exists)

	_, _, err = s.Users.UpsertUser(context.Background(), newTestUser("u1", "1", "a"))
	require.NoError(t, err)
	exists, err = s.Users.CanCreateWorkspaceExists(context.Background())
	require.NoError(t, err)
	assert.False(t, exists)

	require.NoError(t, s.Users.SetCanCreateWorkspace(context.Background(), "u1", true))
	exists, err = s.Users.CanCreateWorkspaceExists(context.Background())
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestUsersRepo_SetCanCreateWorkspace_MissingUser(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	err := s.Users.SetCanCreateWorkspace(context.Background(), "ghost", true)
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}

func TestUsersRepo_SetAccountStatus_LastActiveAdmin_IsRejectedAtomically(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := t.Context()
	_, _, err := s.Users.UpsertUser(ctx, newTestUser("u1", "1", "owner"))
	require.NoError(t, err)
	require.NoError(t, s.Users.SetCanCreateWorkspace(ctx, "u1", true))

	err = s.Users.SetAccountStatus(ctx, "u1", auth.AccountDisabled)
	require.ErrorIs(t, err, apperrs.ErrConflict)
	user, err := s.Users.GetUserByID(ctx, "u1")
	require.NoError(t, err)
	assert.Equal(t, auth.AccountActive, user.AccountStatus)
}

func TestUsersRepo_SetAccountStatus_AllowsChangingWhenAnotherAdminIsActive(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := t.Context()
	for _, id := range []string{"u1", "u2"} {
		_, _, err := s.Users.UpsertUser(ctx, newTestUser(id, id, id))
		require.NoError(t, err)
		require.NoError(t, s.Users.SetCanCreateWorkspace(ctx, id, true))
	}
	require.NoError(t, s.Users.SetAccountStatus(ctx, "u1", auth.AccountDisabled))
	count, err := s.Users.CountActiveAdmins(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestUsersRepo_MarkFirstLoginDone_MissingUser(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	err := s.Users.MarkFirstLoginDone(context.Background(), "ghost")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}

func TestUsersRepo_SetProfileOverride_RoundTrip(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()
	_, _, err := s.Users.UpsertUser(ctx, newTestUser("u1", "42", "onik97"))
	require.NoError(t, err)

	name := "Onik"
	avatar := "data:image/png;base64,aGVsbG8="
	require.NoError(t, s.Users.SetProfileOverride(ctx, "u1", &name, &avatar))

	u, err := s.Users.GetUserByID(ctx, "u1")
	require.NoError(t, err)
	require.NotNil(t, u.DisplayName)
	require.NotNil(t, u.AvatarOverrideURL)
	assert.Equal(t, name, *u.DisplayName)
	assert.Equal(t, avatar, *u.AvatarOverrideURL)
	// Provider-sourced columns are untouched by the override.
	assert.Equal(t, "onik97", u.Login)
	assert.Equal(t, "Name onik97", u.Name)
}

func TestUsersRepo_SetProfileOverride_ClearsWithNil(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()
	_, _, err := s.Users.UpsertUser(ctx, newTestUser("u1", "42", "onik97"))
	require.NoError(t, err)
	name := "Onik"
	require.NoError(t, s.Users.SetProfileOverride(ctx, "u1", &name, nil))

	require.NoError(t, s.Users.SetProfileOverride(ctx, "u1", nil, nil))
	u, err := s.Users.GetUserByID(ctx, "u1")
	require.NoError(t, err)
	assert.Nil(t, u.DisplayName)
	assert.Nil(t, u.AvatarOverrideURL)
}

func TestUsersRepo_SetProfileOverride_MissingUser(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	name := "Onik"
	err := s.Users.SetProfileOverride(context.Background(), "ghost", &name, nil)
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}

// TestUsersRepo_UpsertUser_NeverTouchesProfileOverride is the important
// regression test for the optional manual profile override: a manual
// override must survive every future GitHub sign-in without any change to
// the sync logic, because UpsertUser's INSERT/UPDATE never reference
// display_name/avatar_override_url.
func TestUsersRepo_UpsertUser_NeverTouchesProfileOverride(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()
	_, _, err := s.Users.UpsertUser(ctx, newTestUser("u1", "42", "onik97"))
	require.NoError(t, err)

	name := "Manual Name"
	avatar := "data:image/png;base64,aGVsbG8="
	require.NoError(t, s.Users.SetProfileOverride(ctx, "u1", &name, &avatar))

	// Simulate a re-login: UpsertUser is called again with fresh
	// provider-sourced data, exactly like Service.Login does.
	relogin := newTestUser("", "42", "onik97")
	relogin.Name = "Renamed On GitHub"
	relogin.AvatarURL = "https://avatar/renamed"
	u, created, err := s.Users.UpsertUser(ctx, relogin)
	require.NoError(t, err)
	assert.False(t, created)
	assert.Equal(t, "Renamed On GitHub", u.Name, "provider-sourced name still syncs")

	require.NotNil(t, u.DisplayName, "manual override must survive a re-login")
	require.NotNil(t, u.AvatarOverrideURL, "manual override must survive a re-login")
	assert.Equal(t, name, *u.DisplayName)
	assert.Equal(t, avatar, *u.AvatarOverrideURL)
}

func TestAllowlistRepo_Add_Remove_Contains_List(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()

	require.NoError(t, s.Allowlist.Add(ctx, "alice"))
	require.NoError(t, s.Allowlist.Add(ctx, "bob"))

	t.Run("duplicate add conflicts", func(t *testing.T) {
		err := s.Allowlist.Add(ctx, "alice")
		require.ErrorIs(t, err, apperrs.ErrConflict)
	})

	t.Run("contains", func(t *testing.T) {
		ok, err := s.Allowlist.Contains(ctx, "alice")
		require.NoError(t, err)
		assert.True(t, ok)
		ok, err = s.Allowlist.Contains(ctx, "nobody")
		require.NoError(t, err)
		assert.False(t, ok)
	})

	t.Run("list sorted", func(t *testing.T) {
		got, err := s.Allowlist.List(ctx)
		require.NoError(t, err)
		assert.Equal(t, []string{"alice", "bob"}, got)
	})

	t.Run("remove", func(t *testing.T) {
		require.NoError(t, s.Allowlist.Remove(ctx, "alice"))
		ok, err := s.Allowlist.Contains(ctx, "alice")
		require.NoError(t, err)
		assert.False(t, ok)
	})

	t.Run("remove absent member", func(t *testing.T) {
		err := s.Allowlist.Remove(ctx, "ghost")
		require.ErrorIs(t, err, apperrs.ErrNotFound)
	})
}

func TestSettingsRepo_Get_Defaults(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	st, err := s.Settings.Get(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "", st.InstanceURL)
	assert.Equal(t, 1, st.SettingsVersion)
	assert.Equal(t, "{ticket.Ticket} {ticket.Status}", st.MentionChipTemplate, "default must render identically to today's hardcoded chip (spec.md section 6)")
}

// TestSettingsRepo_SetMentionChipTemplate_RoundTrip proves the mention-chip
// layout template (spec.md section 6) round-trips through
// SetMentionChipTemplate/Get and doesn't disturb SettingsVersion or the
// instance URL (unlike Set, which bumps the version).
func TestSettingsRepo_SetMentionChipTemplate_RoundTrip(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()

	_, err := s.Settings.Set(ctx, "https://deploy.example.com")
	require.NoError(t, err)

	st, err := s.Settings.SetMentionChipTemplate(ctx, "{ticket.Project} {ticket.Ticket}")
	require.NoError(t, err)
	assert.Equal(t, "{ticket.Project} {ticket.Ticket}", st.MentionChipTemplate)
	assert.Equal(t, "https://deploy.example.com", st.InstanceURL, "instance url untouched")
	assert.Equal(t, 2, st.SettingsVersion, "settings version untouched (only the instance url bumps it)")

	got, err := s.Settings.Get(ctx)
	require.NoError(t, err)
	assert.Equal(t, "{ticket.Project} {ticket.Ticket}", got.MentionChipTemplate)
}

func TestSettingsRepo_Set_BumpsVersion(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()

	st, err := s.Settings.Set(ctx, "https://deploy.example.com")
	require.NoError(t, err)
	assert.Equal(t, "https://deploy.example.com", st.InstanceURL)
	assert.Equal(t, 2, st.SettingsVersion)

	got, err := s.Settings.Get(ctx)
	require.NoError(t, err)
	assert.Equal(t, "https://deploy.example.com", got.InstanceURL)
	assert.Equal(t, 2, got.SettingsVersion)

	st, err = s.Settings.Set(ctx, "https://new.example.com")
	require.NoError(t, err)
	assert.Equal(t, 3, st.SettingsVersion)
}

// TestSettingsRepo_SetGitHubOAuth_RoundTrip proves the GitHub OAuth App
// client ID/secret (T1) round-trip through SetGitHubOAuth/Get, that the
// secret is genuinely encrypted at rest (not merely round-tripped), and that
// SetGitHubOAuth does not disturb the instance URL / settings version.
func TestSettingsRepo_SetGitHubOAuth_RoundTrip(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()

	before, err := s.Settings.Get(ctx)
	require.NoError(t, err)
	assert.False(t, before.Configured())
	assert.Equal(t, "", before.GitHubOAuthClientID)
	assert.Equal(t, "", before.GitHubOAuthClientSecret)

	_, err = s.Settings.Set(ctx, "https://deploy.example.com")
	require.NoError(t, err)

	const clientID = "Iv1.client-id-123"
	const clientSecret = "super-secret-value"

	st, err := s.Settings.SetGitHubOAuth(ctx, clientID, clientSecret)
	require.NoError(t, err)
	assert.Equal(t, clientID, st.GitHubOAuthClientID)
	assert.Equal(t, clientSecret, st.GitHubOAuthClientSecret)
	assert.True(t, st.Configured())
	assert.Equal(t, "https://deploy.example.com", st.InstanceURL, "instance url untouched")

	got, err := s.Settings.Get(ctx)
	require.NoError(t, err)
	assert.Equal(t, clientID, got.GitHubOAuthClientID)
	assert.Equal(t, clientSecret, got.GitHubOAuthClientSecret)
	assert.True(t, got.Configured())

	// Verify the secret is genuinely encrypted at rest: the raw column must
	// not contain the plaintext.
	var rawSecret string
	require.NoError(t, s.db.QueryRowContext(ctx,
		`SELECT github_oauth_client_secret FROM instance_settings WHERE id = 1`).Scan(&rawSecret))
	assert.NotEqual(t, clientSecret, rawSecret, "secret must not be stored as plaintext")
	assert.NotEmpty(t, rawSecret)
}

// TestSettingsRepo_SetGoogleOAuth_RoundTrip mirrors the GitHub case for the
// optional second provider (ADR 0040): encrypted at rest, independent of the GitHub
// columns, and clearable back to "not offered".
func TestSettingsRepo_SetGoogleOAuth_RoundTrip(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()

	_, err := s.Settings.SetGitHubOAuth(ctx, "gh-id", "gh-secret")
	require.NoError(t, err)

	st, err := s.Settings.SetProviderOAuth(ctx, auth.ProviderGoogle, "g-id", "g-secret")
	require.NoError(t, err)
	assert.True(t, st.ProviderConfigured(auth.ProviderGoogle))
	assert.Equal(t, "gh-id", st.GitHubOAuthClientID, "github columns untouched")
	assert.Equal(t, "gh-secret", st.GitHubOAuthClientSecret)

	got, err := s.Settings.Get(ctx)
	require.NoError(t, err)
	assert.Equal(t, "g-id", got.GoogleOAuthClientID)
	assert.Equal(t, "g-secret", got.GoogleOAuthClientSecret)

	var rawSecret string
	require.NoError(t, s.db.QueryRowContext(ctx,
		`SELECT google_oauth_client_secret FROM instance_settings WHERE id = 1`).Scan(&rawSecret))
	assert.NotEqual(t, "g-secret", rawSecret, "secret must not be stored as plaintext")

	_, err = s.Settings.SetProviderOAuth(ctx, auth.ProviderGoogle, "", "")
	require.NoError(t, err)
	got, err = s.Settings.Get(ctx)
	require.NoError(t, err)
	assert.False(t, got.ProviderConfigured(auth.ProviderGoogle))
	assert.True(t, got.Configured(), "clearing google leaves github alone")
}

func TestSettingsRepo_SetProviderOAuth_Discord(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()

	_, err := s.Settings.SetProviderOAuth(ctx, auth.ProviderGitHub, "x", "y")
	assert.ErrorIs(t, err, apperrs.ErrInvalid, "github has no optional-provider columns")

	st, err := s.Settings.SetProviderOAuth(ctx, auth.ProviderDiscord, "d-id", "d-secret")
	require.NoError(t, err)
	assert.True(t, st.ProviderConfigured(auth.ProviderDiscord))
	assert.False(t, st.ProviderConfigured(auth.ProviderGoogle))

	got, err := s.Settings.Get(ctx)
	require.NoError(t, err)
	assert.Equal(t, "d-secret", got.DiscordOAuthClientSecret)
	var raw string
	require.NoError(t, s.db.QueryRowContext(ctx, `SELECT discord_oauth_client_secret FROM instance_settings WHERE id = 1`).Scan(&raw))
	assert.NotEqual(t, "d-secret", raw)
}
