package storage

import (
	"database/sql"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/auth"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/permissions"
	"github.com/otal-labs/nexul/internal/roles"
	"github.com/otal-labs/nexul/internal/tenancy"
)

func seedInvitationWorkspace(t *testing.T, s *Store, workspaceID, name, roleID string, owner bool, actorID string) {
	t.Helper()
	require.NoError(t, s.Workspaces.Create(t.Context(), newTestWorkspace(workspaceID, name)))
	_, _, err := s.Users.UpsertUser(t.Context(), newTestUser(actorID, actorID, actorID))
	require.NoError(t, err)
	role := &roles.Role{ID: roleID, WorkspaceID: workspaceID, Name: "Editors", IsOwnerRole: owner, Permissions: permissions.SetOf(permissions.MembersWrite), CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
	require.NoError(t, s.Roles.Create(t.Context(), role))
	require.NoError(t, s.WorkspaceMembers.AddMember(t.Context(), &tenancy.Member{UserID: actorID, WorkspaceID: workspaceID, RoleID: roleID, CreatedAt: time.Now().UTC()}))
}

func newInvitation(id, actorID, workspaceID, roleID string, createdAt time.Time, lifetime time.Duration) *tenancy.Invitation {
	return &tenancy.Invitation{
		ID: id, InvitedBy: actorID, CreatedAt: createdAt, ExpiresAt: createdAt.Add(lifetime),
		Grants: []*tenancy.InvitationGrant{{WorkspaceID: workspaceID, RoleID: roleID}},
	}
}

func invitationToken(t *testing.T, suffix string) (string, string) {
	t.Helper()
	raw := "4d2f3f29-2a43-4ae7-b2d4-0b6f1a7f4c" + suffix
	hash, err := tenancy.HashInvitationToken(raw)
	require.NoError(t, err)
	return raw, hash
}

func createAcceptanceHandoff(t *testing.T, s *Store, invitationID string, now time.Time, identity tenancy.InvitationIdentity, suffix string) string {
	t.Helper()
	_, stateHash := invitationToken(t, suffix)
	_, acceptanceHash := invitationToken(t, "f"+suffix[1:])
	handoffClock := time.Now().UTC()
	handoff := &auth.OAuthHandoff{ID: "handoff-" + invitationID + "-" + suffix, InvitationID: invitationID, OAuthStateHash: stateHash, Provider: auth.Provider(identity.Provider), CreatedAt: handoffClock, ExpiresAt: handoffClock.Add(10 * time.Minute)}
	require.NoError(t, s.OAuthHandoffs.StartOAuthHandoff(t.Context(), handoff))
	_, err := s.OAuthHandoffs.CompleteOAuthCallback(t.Context(), stateHash, acceptanceHash, auth.OAuthHandoffIdentity{Provider: auth.Provider(identity.Provider), ProviderUserID: identity.ProviderUserID, Login: identity.Login, Name: identity.Name, AvatarURL: identity.AvatarURL}, handoffClock.Add(10*time.Minute), handoffClock)
	require.NoError(t, err)
	return acceptanceHash
}

func TestInvitationsRepo_CreateAndGet_DoesNotExposeRawToken(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := t.Context()
	now := time.Date(2026, 9, 20, 12, 0, 0, 0, time.UTC)
	seedInvitationWorkspace(t, s, "ws-1", "Acme", "role-editor", false, "actor")

	const rawToken = "4d2f3f29-2a43-4ae7-b2d4-0b6f1a7f4c11"
	tokenHash, err := tenancy.HashInvitationToken(rawToken)
	require.NoError(t, err)
	want := &tenancy.Invitation{
		ID:        "inv-1",
		InvitedBy: "actor",
		CreatedAt: now,
		ExpiresAt: now.Add(7 * 24 * time.Hour),
		Grants: []*tenancy.InvitationGrant{{
			WorkspaceID: "ws-1",
			RoleID:      "role-editor",
			Allow:       permissions.SetOf(permissions.DocsRead),
		}},
	}

	require.NoError(t, s.Invitations.Create(ctx, want, tokenHash, eventbus.OutboxEvent{ID: "evt-1", Topic: "invitation.created", Payload: map[string]string{"id": want.ID}}))

	var storedHash string
	require.NoError(t, s.db.QueryRowContext(ctx, `SELECT token_hash FROM invitations WHERE id = ?`, want.ID).Scan(&storedHash))
	assert.NotEqual(t, rawToken, storedHash)
	assert.Equal(t, tokenHash, storedHash)

	got, err := s.Invitations.GetByTokenHash(ctx, tokenHash, now)
	require.NoError(t, err)
	assert.Equal(t, want.ID, got.ID)
	assert.Equal(t, want.ExpiresAt, got.ExpiresAt)
	assert.Equal(t, permissions.SetOf(permissions.DocsRead), got.Grants[0].Allow)

	_, err = s.Invitations.GetByTokenHash(ctx, rawToken, now)
	assert.ErrorIs(t, err, apperrs.ErrNotFound)
}

func TestInvitationsRepo_GetByTokenHash_RejectsInvalidTokenShape(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	_, err := s.Invitations.GetByTokenHash(t.Context(), "not-a-uuid", time.Now())
	assert.ErrorIs(t, err, apperrs.ErrNotFound)
}

func TestUsersRepo_AccountStatus_RoundTrip(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := t.Context()
	_, _, err := s.Users.UpsertUser(ctx, newTestUser("u-1", "provider-1", "alice"))
	require.NoError(t, err)

	user, err := s.Users.GetUserByID(ctx, "u-1")
	require.NoError(t, err)
	assert.Equal(t, auth.AccountActive, user.AccountStatus)

	require.NoError(t, s.Users.SetAccountStatus(ctx, "u-1", auth.AccountDisabled))
	user, err = s.Users.GetUserByProvider(ctx, auth.ProviderGitHub, "provider-1")
	require.NoError(t, err)
	assert.Equal(t, auth.AccountDisabled, user.AccountStatus)
}

func TestMigration_PrivateInvitationFoundation_IsPresentAndBackfillsActiveUsers(t *testing.T) {
	t.Parallel()
	db := newTestDB(t)
	ctx := t.Context()
	_, err := db.ExecContext(ctx, `INSERT INTO users (id, provider, provider_user_id, login, created_at, updated_at) VALUES ('u-1', 'github', 'provider-1', 'alice', 1, 1)`)
	require.NoError(t, err)
	var status string
	require.NoError(t, db.QueryRowContext(ctx, `SELECT account_status FROM users WHERE id = 'u-1'`).Scan(&status))
	assert.Equal(t, string(auth.AccountActive), status)
	_, err = db.ExecContext(ctx, `UPDATE users SET account_status = 'unknown' WHERE id = 'u-1'`)
	assert.Error(t, err)
	for _, table := range []string{"invitations", "invitation_grants", "allowlist", "workspace_invites"} {
		var count int
		require.NoError(t, db.QueryRowContext(ctx, `SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = ?`, table).Scan(&count))
		assert.Equal(t, 1, count, table)
	}
}

func TestInvitationsRepo_Create_RollsBackOutboxWithInvitation(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := t.Context()
	now := time.Now().UTC()
	seedInvitationWorkspace(t, s, "ws-1", "Acme", "role-editor", false, "actor")
	tokenHash, err := tenancy.HashInvitationToken("4d2f3f29-2a43-4ae7-b2d4-0b6f1a7f4c11")
	require.NoError(t, err)
	want := &tenancy.Invitation{ID: "inv-1", InvitedBy: "actor", CreatedAt: now, ExpiresAt: now.Add(time.Hour), Grants: []*tenancy.InvitationGrant{{WorkspaceID: "ws-1", RoleID: "role-editor"}}}

	err = s.Invitations.Create(ctx, want, tokenHash, eventbus.OutboxEvent{ID: "evt-1", Topic: "invitation.created", Payload: make(chan int)})
	require.Error(t, err)

	var count int
	require.NoError(t, s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM invitations`).Scan(&count))
	assert.Zero(t, count)
	require.NoError(t, s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM outbox`).Scan(&count))
	assert.Zero(t, count)
	assert.False(t, errors.Is(err, apperrs.ErrNotFound))
}

func TestInvitationsRepo_Redeem_NewUserIsAtomicAndWritesOutbox(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := t.Context()
	now := time.Date(2026, 9, 20, 12, 0, 0, 0, time.UTC)
	seedInvitationWorkspace(t, s, "ws-1", "Acme", "role-editor", false, "actor")
	_, tokenHash := invitationToken(t, "12")
	invitation := newInvitation("inv-1", "actor", "ws-1", "role-editor", now, 7*24*time.Hour)
	invitation.Grants[0].Allow = permissions.SetOf(permissions.DocsRead)
	require.NoError(t, s.Invitations.Create(ctx, invitation, tokenHash))

	acceptanceHash := createAcceptanceHandoff(t, s, invitation.ID, now, tenancy.InvitationIdentity{ID: "user-new", Provider: "github", ProviderUserID: "provider-new", Login: "new-user"}, "18")
	admission, err := s.Invitations.Redeem(ctx, acceptanceHash, tenancy.InvitationIdentity{ID: "user-new", Provider: "github", ProviderUserID: "provider-new", Login: "new-user"}, now, eventbus.OutboxEvent{ID: "evt-redeemed", Topic: "invitation.redeemed", Payload: map[string]string{"invitation_id": invitation.ID}})
	require.NoError(t, err)
	assert.True(t, admission.Created)

	user, err := s.Users.GetUserByProvider(ctx, auth.ProviderGitHub, "provider-new")
	require.NoError(t, err)
	assert.Equal(t, "user-new", user.ID)
	roleID, err := s.WorkspaceMembers.RoleIDFor(ctx, "ws-1", "user-new")
	require.NoError(t, err)
	assert.Equal(t, "role-editor", roleID)
	overwrite, err := s.Access.Get(ctx, "workspace", "ws-1", "user-new")
	require.NoError(t, err)
	assert.Equal(t, permissions.SetOf(permissions.DocsRead), overwrite.Allow)
	_, err = s.Invitations.GetByTokenHash(ctx, tokenHash, now)
	assert.ErrorIs(t, err, apperrs.ErrNotFound)

	var outboxCount int
	require.NoError(t, s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM outbox WHERE id = ?`, "evt-redeemed").Scan(&outboxCount))
	assert.Equal(t, 1, outboxCount)
}

func TestInvitationsRepo_Redeem_ExistingMembershipPreservesRoleAndOverwrite(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := t.Context()
	now := time.Date(2026, 9, 20, 12, 0, 0, 0, time.UTC)
	seedInvitationWorkspace(t, s, "ws-1", "Acme", "role-editor", false, "actor")
	_, _, err := s.Users.UpsertUser(ctx, newTestUser("user-existing", "provider-existing", "existing"))
	require.NoError(t, err)
	role := &roles.Role{ID: "role-existing", WorkspaceID: "ws-1", Name: "Existing", Permissions: permissions.SetOf(permissions.ProjectsWrite), CreatedAt: now, UpdatedAt: now}
	require.NoError(t, s.Roles.Create(ctx, role))
	require.NoError(t, s.WorkspaceMembers.AddMember(ctx, &tenancy.Member{UserID: "user-existing", WorkspaceID: "ws-1", RoleID: role.ID, CreatedAt: now}))
	require.NoError(t, s.Access.Set(ctx, "workspace", "ws-1", "user-existing", permissions.SetOf(permissions.ProjectsWrite), permissions.SetOf(permissions.DocsDelete)))
	_, tokenHash := invitationToken(t, "13")
	invitation := newInvitation("inv-1", "actor", "ws-1", "role-editor", now, 24*time.Hour)
	invitation.Grants[0].Allow = permissions.SetOf(permissions.DocsRead)
	require.NoError(t, s.Invitations.Create(ctx, invitation, tokenHash))

	acceptanceHash := createAcceptanceHandoff(t, s, invitation.ID, now, tenancy.InvitationIdentity{ID: "ignored", Provider: "github", ProviderUserID: "provider-existing", Login: "existing"}, "19")
	admission, err := s.Invitations.Redeem(ctx, acceptanceHash, tenancy.InvitationIdentity{ID: "ignored", Provider: "github", ProviderUserID: "provider-existing", Login: "existing"}, now)
	require.NoError(t, err)
	assert.False(t, admission.Created)
	roleID, err := s.WorkspaceMembers.RoleIDFor(ctx, "ws-1", "user-existing")
	require.NoError(t, err)
	assert.Equal(t, "role-existing", roleID)
	overwrite, err := s.Access.Get(ctx, "workspace", "ws-1", "user-existing")
	require.NoError(t, err)
	assert.Equal(t, permissions.SetOf(permissions.ProjectsWrite), overwrite.Allow)
	assert.Equal(t, permissions.SetOf(permissions.DocsDelete), overwrite.Deny)
}

func TestInvitationsRepo_Redeem_InvalidGrantRollsBackNewUser(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := t.Context()
	now := time.Date(2026, 9, 20, 12, 0, 0, 0, time.UTC)
	seedInvitationWorkspace(t, s, "ws-1", "Acme", "role-editor", false, "actor")
	seedInvitationWorkspace(t, s, "ws-2", "Beta", "role-manager", false, "actor")
	require.NoError(t, s.Roles.Create(ctx, &roles.Role{ID: "role-reviewer", WorkspaceID: "ws-2", Name: "Reviewers", CreatedAt: now, UpdatedAt: now}))
	_, tokenHash := invitationToken(t, "14")
	invitation := newInvitation("inv-1", "actor", "ws-1", "role-editor", now, 7*24*time.Hour)
	invitation.Grants = append(invitation.Grants, &tenancy.InvitationGrant{WorkspaceID: "ws-2", RoleID: "role-reviewer"})
	require.NoError(t, s.Invitations.Create(ctx, invitation, tokenHash))
	require.NoError(t, s.Roles.Delete(ctx, "role-reviewer"))

	acceptanceHash := createAcceptanceHandoff(t, s, invitation.ID, now, tenancy.InvitationIdentity{ID: "user-new", Provider: "github", ProviderUserID: "provider-new", Login: "new-user"}, "20")
	_, err := s.Invitations.Redeem(ctx, acceptanceHash, tenancy.InvitationIdentity{ID: "user-new", Provider: "github", ProviderUserID: "provider-new", Login: "new-user"}, now)
	assert.ErrorIs(t, err, apperrs.ErrNotFound)
	_, err = s.Users.GetUserByProvider(ctx, auth.ProviderGitHub, "provider-new")
	assert.ErrorIs(t, err, apperrs.ErrNotFound)
	_, err = s.WorkspaceMembers.RoleIDFor(ctx, "ws-1", "user-new")
	assert.ErrorIs(t, err, apperrs.ErrNotFound)
	var invitationCount int
	require.NoError(t, s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM invitations WHERE id = ?`, invitation.ID).Scan(&invitationCount))
	assert.Zero(t, invitationCount)
}

func TestInvitationsRepo_Redeem_ConcurrentConsumptionHasOneWinner(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := t.Context()
	now := time.Date(2026, 9, 20, 12, 0, 0, 0, time.UTC)
	seedInvitationWorkspace(t, s, "ws-1", "Acme", "role-editor", false, "actor")
	_, tokenHash := invitationToken(t, "15")
	invitation := newInvitation("inv-1", "actor", "ws-1", "role-editor", now, 7*24*time.Hour)
	require.NoError(t, s.Invitations.Create(ctx, invitation, tokenHash))
	firstIdentity := tenancy.InvitationIdentity{ID: "user-1", Provider: "github", ProviderUserID: "provider-1", Login: "one"}
	acceptanceHash := createAcceptanceHandoff(t, s, invitation.ID, now, firstIdentity, "21")

	var wg sync.WaitGroup
	results := make(chan error, 2)
	for _, identity := range []tenancy.InvitationIdentity{
		firstIdentity,
		{ID: "user-2", Provider: "github", ProviderUserID: "provider-2", Login: "two"},
	} {
		wg.Go(func() {
			_, err := s.Invitations.Redeem(ctx, acceptanceHash, identity, now)
			results <- err
		})
	}
	wg.Wait()
	close(results)

	winners := 0
	for err := range results {
		if err == nil {
			winners++
			continue
		}
		assert.ErrorIs(t, err, apperrs.ErrNotFound)
	}
	assert.Equal(t, 1, winners)
}

func TestInvitationsRepo_GetByTokenHash_ExpiryDeletesActiveRow(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := t.Context()
	now := time.Date(2026, 9, 20, 12, 0, 0, 0, time.UTC)
	seedInvitationWorkspace(t, s, "ws-1", "Acme", "role-editor", false, "actor")
	_, oneDayHash := invitationToken(t, "16")
	_, sevenDayHash := invitationToken(t, "17")
	require.NoError(t, s.Invitations.Create(ctx, newInvitation("inv-1", "actor", "ws-1", "role-editor", now, 24*time.Hour), oneDayHash))
	require.NoError(t, s.Invitations.Create(ctx, newInvitation("inv-2", "actor", "ws-1", "role-editor", now, 7*24*time.Hour), sevenDayHash))

	_, err := s.Invitations.GetByTokenHash(ctx, oneDayHash, now.Add(24*time.Hour))
	assert.ErrorIs(t, err, apperrs.ErrNotFound)
	var expiredCount int
	require.NoError(t, s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM invitations WHERE token_hash = ?`, oneDayHash).Scan(&expiredCount))
	assert.Zero(t, expiredCount)
	got, err := s.Invitations.GetByTokenHash(ctx, sevenDayHash, now.Add(6*24*time.Hour))
	require.NoError(t, err)
	assert.Equal(t, "inv-2", got.ID)
}

func TestOAuthHandoffs_StateBoundAcceptance_IsolatedAndHashed(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := t.Context()
	now := time.Now().UTC()
	seedInvitationWorkspace(t, s, "ws-1", "Acme", "role-editor", false, "actor")
	_, tokenHash := invitationToken(t, "22")
	invitation := newInvitation("inv-1", "actor", "ws-1", "role-editor", now, 7*24*time.Hour)
	require.NoError(t, s.Invitations.Create(ctx, invitation, tokenHash))
	first := tenancy.InvitationIdentity{ID: "user-1", Provider: "github", ProviderUserID: "provider-1", Login: "one"}
	second := tenancy.InvitationIdentity{ID: "user-2", Provider: "google", ProviderUserID: "provider-2", Login: "two"}
	firstAcceptance := createAcceptanceHandoff(t, s, invitation.ID, now, first, "23")
	secondAcceptance := createAcceptanceHandoff(t, s, invitation.ID, now, second, "24")

	assert.NotEqual(t, firstAcceptance, secondAcceptance)
	row, err := s.OAuthHandoffs.GetOAuthHandoffByAcceptanceHash(ctx, firstAcceptance, now)
	require.NoError(t, err)
	assert.Equal(t, first.ProviderUserID, row.ProviderUserID)
	assert.NotContains(t, row.AcceptanceHash, "4d2f3f29")
	var storedState, storedAcceptance sql.NullString
	require.NoError(t, s.db.QueryRowContext(ctx, `SELECT oauth_state_hash, acceptance_hash FROM invitation_oauth_handoffs WHERE id = ?`, row.ID).Scan(&storedState, &storedAcceptance))
	assert.False(t, storedState.Valid)
	assert.Equal(t, firstAcceptance, storedAcceptance.String)
}

func TestInvitationsRepo_Redeem_DisabledAfterOAuthIsRejected(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := t.Context()
	now := time.Date(2026, 9, 20, 12, 0, 0, 0, time.UTC)
	seedInvitationWorkspace(t, s, "ws-1", "Acme", "role-editor", false, "actor")
	_, _, err := s.Users.UpsertUser(ctx, newTestUser("user-existing", "provider-existing", "existing"))
	require.NoError(t, err)
	_, tokenHash := invitationToken(t, "25")
	invitation := newInvitation("inv-1", "actor", "ws-1", "role-editor", now, 7*24*time.Hour)
	require.NoError(t, s.Invitations.Create(ctx, invitation, tokenHash))
	identity := tenancy.InvitationIdentity{ID: "ignored", Provider: "github", ProviderUserID: "provider-existing", Login: "existing"}
	acceptanceHash := createAcceptanceHandoff(t, s, invitation.ID, now, identity, "26")
	require.NoError(t, s.Users.SetAccountStatus(ctx, "user-existing", auth.AccountDisabled))

	_, err = s.Invitations.Redeem(ctx, acceptanceHash, identity, now)
	assert.ErrorIs(t, err, apperrs.ErrNotFound)
	var redeemedAt sql.NullInt64
	require.NoError(t, s.db.QueryRowContext(ctx, `SELECT redeemed_at FROM invitations WHERE id = ?`, invitation.ID).Scan(&redeemedAt))
	assert.False(t, redeemedAt.Valid)
}

func TestInvitationsRepo_Redeem_ConsumeBeforeWriteRollsBackOnOutboxFailure(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := t.Context()
	now := time.Date(2026, 9, 20, 12, 0, 0, 0, time.UTC)
	seedInvitationWorkspace(t, s, "ws-1", "Acme", "role-editor", false, "actor")
	_, tokenHash := invitationToken(t, "27")
	invitation := newInvitation("inv-1", "actor", "ws-1", "role-editor", now, 7*24*time.Hour)
	require.NoError(t, s.Invitations.Create(ctx, invitation, tokenHash))
	identity := tenancy.InvitationIdentity{ID: "user-new", Provider: "github", ProviderUserID: "provider-new", Login: "new-user"}
	acceptanceHash := createAcceptanceHandoff(t, s, invitation.ID, now, identity, "28")

	_, err := s.Invitations.Redeem(ctx, acceptanceHash, identity, now, eventbus.OutboxEvent{ID: "bad", Topic: "invitation.redeemed", Payload: make(chan int)})
	require.Error(t, err)
	var redeemedAt sql.NullInt64
	require.NoError(t, s.db.QueryRowContext(ctx, `SELECT redeemed_at FROM invitations WHERE id = ?`, invitation.ID).Scan(&redeemedAt))
	assert.False(t, redeemedAt.Valid)
	_, err = s.Users.GetUserByProvider(ctx, auth.ProviderGitHub, identity.ProviderUserID)
	assert.ErrorIs(t, err, apperrs.ErrNotFound)
	_, err = s.WorkspaceMembers.RoleIDFor(ctx, "ws-1", identity.ID)
	assert.ErrorIs(t, err, apperrs.ErrNotFound)
}

func TestInvitationsRepo_Redeem_RetainsReceiptAndRetriesSameIdentity(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := t.Context()
	now := time.Date(2026, 9, 20, 12, 0, 0, 0, time.UTC)
	seedInvitationWorkspace(t, s, "ws-1", "Acme", "role-editor", false, "actor")
	_, tokenHash := invitationToken(t, "29")
	invitation := newInvitation("inv-1", "actor", "ws-1", "role-editor", now, 7*24*time.Hour)
	require.NoError(t, s.Invitations.Create(ctx, invitation, tokenHash))
	identity := tenancy.InvitationIdentity{ID: "user-new", Provider: "github", ProviderUserID: "provider-new", Login: "new-user"}
	acceptanceHash := createAcceptanceHandoff(t, s, invitation.ID, now, identity, "30")

	first, err := s.Invitations.Redeem(ctx, acceptanceHash, identity, now)
	require.NoError(t, err)
	second, err := s.Invitations.Redeem(ctx, acceptanceHash, identity, now)
	require.NoError(t, err)
	assert.Equal(t, first.UserID, second.UserID)
	assert.False(t, second.Created)
	var redeemedAt sql.NullInt64
	var redeemedBy string
	require.NoError(t, s.db.QueryRowContext(ctx, `SELECT redeemed_at, redeemed_by FROM invitations WHERE id = ?`, invitation.ID).Scan(&redeemedAt, &redeemedBy))
	assert.True(t, redeemedAt.Valid)
	assert.Equal(t, identity.ID, redeemedBy)
	active, err := s.Invitations.List(ctx, "actor", now)
	require.NoError(t, err)
	assert.Empty(t, active)
}

func TestInvitationsRepo_Redeem_DifferentIdentityCannotUseCompletedHandoff(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := t.Context()
	now := time.Date(2026, 9, 20, 12, 0, 0, 0, time.UTC)
	seedInvitationWorkspace(t, s, "ws-1", "Acme", "role-editor", false, "actor")
	_, tokenHash := invitationToken(t, "31")
	invitation := newInvitation("inv-1", "actor", "ws-1", "role-editor", now, 7*24*time.Hour)
	require.NoError(t, s.Invitations.Create(ctx, invitation, tokenHash))
	identity := tenancy.InvitationIdentity{ID: "user-new", Provider: "github", ProviderUserID: "provider-new", Login: "new-user"}
	acceptanceHash := createAcceptanceHandoff(t, s, invitation.ID, now, identity, "32")
	require.NoError(t, func() error { _, err := s.Invitations.Redeem(ctx, acceptanceHash, identity, now); return err }())

	_, err := s.Invitations.Redeem(ctx, acceptanceHash, tenancy.InvitationIdentity{ID: "other", Provider: "github", ProviderUserID: "other-provider", Login: "other"}, now)
	assert.ErrorIs(t, err, apperrs.ErrNotFound)
}
