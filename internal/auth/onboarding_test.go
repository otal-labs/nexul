package auth

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

func TestMe_OwnerWizardOnlyBeforeAnyOwner(t *testing.T) {
	s, _, _, _ := newTestHarness(&fakeGitHub{user: ghUser("1", "a")})
	token, err := s.Login(context.Background(), "good-code")
	require.NoError(t, err)
	st, err := s.Me(context.Background(), mustVerify(t, s, token))
	require.NoError(t, err)
	assert.True(t, st.NeedsOwnerWizard)
	assert.False(t, st.NeedsFirstLoginWizard)
}

func TestMe_FirstLoginWizardForMember(t *testing.T) {
	s, _, _, _ := newTestHarness(&fakeGitHub{user: ghUser("1", "owner")})
	ownerToken, err := s.Login(context.Background(), "good-code")
	require.NoError(t, err)
	ownerID := mustVerify(t, s, ownerToken)
	require.NoError(t, s.CompleteOwnerWizard(context.Background(), ownerID, "https://deploy.example.com"))
	require.NoError(t, s.AddMember(context.Background(), ownerID, "member"))

	s.cfg.GitHub = &fakeGitHub{user: ghUser("2", "member")}
	token, err := s.Login(context.Background(), "good-code")
	require.NoError(t, err)
	st, err := s.Me(context.Background(), mustVerify(t, s, token))
	require.NoError(t, err)
	assert.False(t, st.NeedsOwnerWizard)
	assert.True(t, st.NeedsFirstLoginWizard)
}

func TestMe_OwnerAfterWizardNoWizardNeeded(t *testing.T) {
	s, _, _, _ := newTestHarness(&fakeGitHub{user: ghUser("1", "owner")})
	token, err := s.Login(context.Background(), "good-code")
	require.NoError(t, err)
	ownerID := mustVerify(t, s, token)
	require.NoError(t, s.CompleteOwnerWizard(context.Background(), ownerID, "https://deploy.example.com"))

	st, err := s.Me(context.Background(), ownerID)
	require.NoError(t, err)
	assert.False(t, st.NeedsOwnerWizard)
	assert.False(t, st.NeedsFirstLoginWizard)
	assert.True(t, st.User.CanCreateWorkspace)
}

func TestCompleteOwnerWizard_GrantsOwnerAndSavesInstanceURL(t *testing.T) {
	s, users, _, settings := newTestHarness(&fakeGitHub{user: ghUser("1", "owner")})
	token, err := s.Login(context.Background(), "good-code")
	require.NoError(t, err)
	ownerID := mustVerify(t, s, token)

	require.NoError(t, s.CompleteOwnerWizard(context.Background(), ownerID, "https://deploy.example.com"))

	user, err := users.GetUserByID(context.Background(), ownerID)
	require.NoError(t, err)
	assert.True(t, user.CanCreateWorkspace)
	assert.True(t, user.FirstLoginDone)

	st, err := settings.Get(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "https://deploy.example.com", st.InstanceURL)
	assert.Equal(t, 2, st.SettingsVersion)

	exists, err := s.CanCreateWorkspaceExists(context.Background())
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestCompleteOwnerWizard_SameUserRedoIsIdempotent(t *testing.T) {
	s, _, _, _ := newTestHarness(&fakeGitHub{user: ghUser("1", "owner")})
	token, err := s.Login(context.Background(), "good-code")
	require.NoError(t, err)
	ownerID := mustVerify(t, s, token)

	require.NoError(t, s.CompleteOwnerWizard(context.Background(), ownerID, "https://deploy.example.com"))
	// A double-submitted finish (or retry after partial failure) from the admin themself must succeed, not 409: only a *different* user conflicts.
	require.NoError(t, s.CompleteOwnerWizard(context.Background(), ownerID, "https://deploy.example.com"))
}

func TestCompleteOwnerWizard_BindsDefaultWorkspaceOwner(t *testing.T) {
	s, _, _, _ := newTestHarness(&fakeGitHub{user: ghUser("1", "owner")})
	token, err := s.Login(context.Background(), "good-code")
	require.NoError(t, err)
	ownerID := mustVerify(t, s, token)

	binder := s.cfg.DefaultWorkspace.(*fakeDefaultWorkspace)
	require.Empty(t, binder.bound)

	require.NoError(t, s.CompleteOwnerWizard(context.Background(), ownerID, "https://deploy.example.com"))

	assert.Equal(t, []string{ownerID}, binder.bound)
}

func TestCompleteOwnerWizard_DefaultWorkspaceBindErrorPropagates(t *testing.T) {
	s, _, _, _ := newTestHarness(&fakeGitHub{user: ghUser("1", "owner")})
	token, err := s.Login(context.Background(), "good-code")
	require.NoError(t, err)
	ownerID := mustVerify(t, s, token)

	s.cfg.DefaultWorkspace.(*fakeDefaultWorkspace).err = apperrs.ErrNotFound
	err = s.CompleteOwnerWizard(context.Background(), ownerID, "https://deploy.example.com")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}

func TestGrantCanCreateWorkspace_RequiresPermissionAndUpdatesTarget(t *testing.T) {
	s, users, _, _ := newTestHarness(&fakeGitHub{user: ghUser("1", "owner")})
	token, err := s.Login(context.Background(), "good-code")
	require.NoError(t, err)
	ownerID := mustVerify(t, s, token)
	require.NoError(t, s.CompleteOwnerWizard(context.Background(), ownerID, "https://deploy.example.com"))

	target, _, err := users.UpsertUser(context.Background(), &User{ID: "u2", Provider: ProviderGitHub, ProviderUserID: "2", Login: "member"})
	require.NoError(t, err)
	require.False(t, target.CanCreateWorkspace)

	t.Run("non-admin forbidden", func(t *testing.T) {
		err := s.GrantCanCreateWorkspace(context.Background(), "u2", ownerID)
		require.ErrorIs(t, err, apperrs.ErrForbidden)
	})

	t.Run("admin grants the bit", func(t *testing.T) {
		require.NoError(t, s.GrantCanCreateWorkspace(context.Background(), ownerID, "u2"))
		got, err := users.GetUserByID(context.Background(), "u2")
		require.NoError(t, err)
		assert.True(t, got.CanCreateWorkspace)
	})

	t.Run("newly granted admin can grant/revoke too", func(t *testing.T) {
		target2, _, err := users.UpsertUser(context.Background(), &User{ID: "u3", Provider: ProviderGitHub, ProviderUserID: "3", Login: "member2"})
		require.NoError(t, err)
		require.False(t, target2.CanCreateWorkspace)

		require.NoError(t, s.GrantCanCreateWorkspace(context.Background(), "u2", "u3"))
		got, err := users.GetUserByID(context.Background(), "u3")
		require.NoError(t, err)
		assert.True(t, got.CanCreateWorkspace)

		require.NoError(t, s.RevokeCanCreateWorkspace(context.Background(), "u2", "u3"))
		got, err = users.GetUserByID(context.Background(), "u3")
		require.NoError(t, err)
		assert.False(t, got.CanCreateWorkspace)
	})
}

func TestRevokeCanCreateWorkspace_RequiresPermission(t *testing.T) {
	s, users, _, _ := newTestHarness(&fakeGitHub{user: ghUser("1", "owner")})
	token, err := s.Login(context.Background(), "good-code")
	require.NoError(t, err)
	ownerID := mustVerify(t, s, token)
	require.NoError(t, s.CompleteOwnerWizard(context.Background(), ownerID, "https://deploy.example.com"))

	target, _, err := users.UpsertUser(context.Background(), &User{ID: "u2", Provider: ProviderGitHub, ProviderUserID: "2", Login: "member"})
	require.NoError(t, err)
	require.NoError(t, s.GrantCanCreateWorkspace(context.Background(), ownerID, target.ID))
	nonAdmin, _, err := users.UpsertUser(context.Background(), &User{ID: "u3", Provider: ProviderGitHub, ProviderUserID: "3", Login: "plain-member"})
	require.NoError(t, err)
	require.False(t, nonAdmin.CanCreateWorkspace)

	t.Run("non-admin forbidden", func(t *testing.T) {
		err := s.RevokeCanCreateWorkspace(context.Background(), "u3", ownerID)
		require.ErrorIs(t, err, apperrs.ErrForbidden)
	})

	t.Run("admin revokes the bit", func(t *testing.T) {
		require.NoError(t, s.RevokeCanCreateWorkspace(context.Background(), ownerID, "u2"))
		got, err := users.GetUserByID(context.Background(), "u2")
		require.NoError(t, err)
		assert.False(t, got.CanCreateWorkspace)
	})
}

func TestCompleteOwnerWizard_InvalidInstanceURL(t *testing.T) {
	s, _, _, _ := newTestHarness(&fakeGitHub{user: ghUser("1", "owner")})
	token, err := s.Login(context.Background(), "good-code")
	require.NoError(t, err)
	for _, bad := range []string{"", "not-a-url", "ftp://host", "https://"} {
		err := s.CompleteOwnerWizard(context.Background(), mustVerify(t, s, token), bad)
		require.ErrorIs(t, err, apperrs.ErrInvalid, "url %q", bad)
	}
}

func TestCompleteOwnerWizard_SecondUserConflicts(t *testing.T) {
	s, users, _, _ := newTestHarness(&fakeGitHub{user: ghUser("1", "owner")})
	ownerToken, err := s.Login(context.Background(), "good-code")
	require.NoError(t, err)
	ownerID := mustVerify(t, s, ownerToken)
	require.NoError(t, s.CompleteOwnerWizard(context.Background(), ownerID, "https://deploy.example.com"))

	second, _, err := users.UpsertUser(context.Background(), &User{ID: "u2", Provider: ProviderGitHub, ProviderUserID: "2", Login: "intruder"})
	require.NoError(t, err)
	require.NotNil(t, second)

	err = s.CompleteOwnerWizard(context.Background(), "u2", "https://other.example.com")
	require.ErrorIs(t, err, apperrs.ErrConflict)
}

func TestCompleteFirstLogin_MarksDone(t *testing.T) {
	s, users, _, _ := newTestHarness(&fakeGitHub{user: ghUser("1", "owner")})
	token, err := s.Login(context.Background(), "good-code")
	require.NoError(t, err)
	userID := mustVerify(t, s, token)

	require.NoError(t, s.CompleteFirstLogin(context.Background(), userID))
	user, err := users.GetUserByID(context.Background(), userID)
	require.NoError(t, err)
	assert.True(t, user.FirstLoginDone)
}

func TestUpdateInstanceURL_OwnerOnly(t *testing.T) {
	s, _, _, settings := newTestHarness(&fakeGitHub{user: ghUser("1", "owner")})
	ownerToken, err := s.Login(context.Background(), "good-code")
	require.NoError(t, err)
	ownerID := mustVerify(t, s, ownerToken)
	require.NoError(t, s.CompleteOwnerWizard(context.Background(), ownerID, "https://deploy.example.com"))
	require.NoError(t, s.AddMember(context.Background(), ownerID, "member"))

	s.cfg.GitHub = &fakeGitHub{user: ghUser("2", "member")}
	memberToken, err := s.Login(context.Background(), "good-code")
	require.NoError(t, err)
	memberID := mustVerify(t, s, memberToken)

	t.Run("member forbidden", func(t *testing.T) {
		_, err := s.UpdateInstanceURL(context.Background(), memberID, "https://evil.example.com")
		require.ErrorIs(t, err, apperrs.ErrForbidden)
	})

	t.Run("owner updates and bumps version", func(t *testing.T) {
		st, err := s.UpdateInstanceURL(context.Background(), ownerID, "https://new.example.com")
		require.NoError(t, err)
		assert.Equal(t, "https://new.example.com", st.InstanceURL)
		got, err := settings.Get(context.Background())
		require.NoError(t, err)
		assert.Equal(t, st.SettingsVersion, got.SettingsVersion)
		assert.True(t, st.SettingsVersion > 2, "version should have been bumped")
	})
}

func TestListMembers_OwnerOnly(t *testing.T) {
	s, _, _, _ := newTestHarness(&fakeGitHub{user: ghUser("1", "owner")})
	token, err := s.Login(context.Background(), "good-code")
	require.NoError(t, err)
	ownerID := mustVerify(t, s, token)
	require.NoError(t, s.CompleteOwnerWizard(context.Background(), ownerID, "https://deploy.example.com"))
	require.NoError(t, s.AddMember(context.Background(), ownerID, "Bob"))
	require.NoError(t, s.AddMember(context.Background(), ownerID, "alice"))

	members, err := s.ListMembers(context.Background(), ownerID)
	require.NoError(t, err)
	assert.Equal(t, []string{"alice", "bob"}, members)
}

func TestAddMember_ValidationAndDuplicates(t *testing.T) {
	s, users, _, _ := newTestHarness(&fakeGitHub{user: ghUser("1", "owner")})
	token, err := s.Login(context.Background(), "good-code")
	require.NoError(t, err)
	ownerID := mustVerify(t, s, token)
	require.NoError(t, s.CompleteOwnerWizard(context.Background(), ownerID, "https://deploy.example.com"))

	t.Run("blank rejected", func(t *testing.T) {
		require.ErrorIs(t, s.AddMember(context.Background(), ownerID, "   "), apperrs.ErrInvalid)
	})

	t.Run("duplicate conflicts", func(t *testing.T) {
		require.NoError(t, s.AddMember(context.Background(), ownerID, "bob"))
		require.ErrorIs(t, s.AddMember(context.Background(), ownerID, "BOB"), apperrs.ErrConflict)
	})

	t.Run("non-owner forbidden", func(t *testing.T) {
		_, _, err := users.UpsertUser(context.Background(), &User{ID: "bob-id", Provider: ProviderGitHub, ProviderUserID: "2", Login: "bob"})
		require.NoError(t, err)
		require.ErrorIs(t, s.AddMember(context.Background(), "bob-id", "x"), apperrs.ErrForbidden)
	})
}

func TestRemoveMember_BlocksSignInButKeepsUserRow(t *testing.T) {
	s, users, allowlist, _ := newTestHarness(&fakeGitHub{user: ghUser("1", "owner")})
	ownerToken, err := s.Login(context.Background(), "good-code")
	require.NoError(t, err)
	ownerID := mustVerify(t, s, ownerToken)
	require.NoError(t, s.CompleteOwnerWizard(context.Background(), ownerID, "https://deploy.example.com"))
	require.NoError(t, s.AddMember(context.Background(), ownerID, "bob"))

	user, _, err := users.UpsertUser(context.Background(), &User{ID: "bob-id", Provider: ProviderGitHub, ProviderUserID: "2", Login: "bob"})
	require.NoError(t, err)
	require.NotNil(t, user)

	require.NoError(t, s.RemoveMember(context.Background(), ownerID, "Bob"))

	contains, err := allowlist.Contains(context.Background(), "bob")
	require.NoError(t, err)
	assert.False(t, contains)

	// The user row survives so an already-issued session lives until its TTL (ADR 0041).
	_, err = users.GetUserByID(context.Background(), "bob-id")
	require.NoError(t, err)

	t.Run("removing absent member", func(t *testing.T) {
		require.ErrorIs(t, s.RemoveMember(context.Background(), ownerID, "nobody"), apperrs.ErrNotFound)
	})

	s.cfg.GitHub = &fakeGitHub{user: ghUser("2", "bob")}
	_, err = s.Login(context.Background(), "good-code")
	require.ErrorIs(t, err, apperrs.ErrUnauthorized)
}

func TestGenerateConnectionToken_RequiresInstanceURLAndOwner(t *testing.T) {
	s, users, _, settings := newTestHarness(&fakeGitHub{user: ghUser("1", "owner")})
	token, err := s.Login(context.Background(), "good-code")
	require.NoError(t, err)
	ownerID := mustVerify(t, s, token)
	require.NoError(t, s.CompleteOwnerWizard(context.Background(), ownerID, "https://deploy.example.com"))

	t.Run("no instance url yet", func(t *testing.T) {
		settings.mu.Lock()
		settings.st.InstanceURL = ""
		settings.mu.Unlock()
		_, err := s.GenerateConnectionToken(context.Background(), ownerID)
		require.ErrorIs(t, err, apperrs.ErrInvalid)
	})
	settings.mu.Lock()
	settings.st.InstanceURL = "https://deploy.example.com"
	settings.mu.Unlock()

	t.Run("non-owner forbidden", func(t *testing.T) {
		_, _, err := users.UpsertUser(context.Background(), &User{ID: "bob-id", Provider: ProviderGitHub, ProviderUserID: "2", Login: "bob"})
		require.NoError(t, err)
		_, err = s.GenerateConnectionToken(context.Background(), "bob-id")
		require.ErrorIs(t, err, apperrs.ErrForbidden)
	})

	t.Run("owner gets a token with current settings", func(t *testing.T) {
		ct, err := s.GenerateConnectionToken(context.Background(), ownerID)
		require.NoError(t, err)
		assert.Equal(t, "https://deploy.example.com", ct.InstanceURL)
		claims, err := s.ParseConnectionToken(ct.Token)
		require.NoError(t, err)
		assert.Equal(t, "https://deploy.example.com", claims.InstanceURL)
		assert.Equal(t, 2, claims.Version)
	})

	t.Run("regenerates after url change", func(t *testing.T) {
		_, err := s.UpdateInstanceURL(context.Background(), ownerID, "https://new.example.com")
		require.NoError(t, err)
		ct, err := s.GenerateConnectionToken(context.Background(), ownerID)
		require.NoError(t, err)
		claims, err := s.ParseConnectionToken(ct.Token)
		require.NoError(t, err)
		assert.Equal(t, "https://new.example.com", claims.InstanceURL)
		assert.Equal(t, 3, claims.Version)
	})
}

func TestParseConnectionToken_RejectsBad(t *testing.T) {
	s, _, _, _ := newTestHarness(&fakeGitHub{user: ghUser("1", "owner")})
	token, err := s.Login(context.Background(), "good-code")
	require.NoError(t, err)
	ownerID := mustVerify(t, s, token)
	require.NoError(t, s.CompleteOwnerWizard(context.Background(), ownerID, "https://deploy.example.com"))

	ct, err := s.GenerateConnectionToken(context.Background(), ownerID)
	require.NoError(t, err)

	for _, tok := range []string{"", "a.b", "a.b.c.d", ct.Token + "x"} {
		_, err := s.ParseConnectionToken(tok)
		require.ErrorIs(t, err, apperrs.ErrUnauthorized, "token %q", tok)
	}

	t.Run("expired", func(t *testing.T) {
		ct2, err := s.GenerateConnectionToken(context.Background(), ownerID)
		require.NoError(t, err)
		now := s.cfg.Now
		s.cfg.Now = func() time.Time { return now().Add(31 * 24 * time.Hour) }
		_, err = s.ParseConnectionToken(ct2.Token)
		require.ErrorIs(t, err, apperrs.ErrUnauthorized)
	})
}
