package storage

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/harness"
	"github.com/otal-labs/nexul/internal/pairing"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
)

func newTestComputer(id, userID, name string) pairing.Computer {
	return pairing.Computer{
		ID:             id,
		UserID:         userID,
		Kind:           harness.KindT3Code,
		Name:           name,
		ServerURL:      "https://" + name + ".example.com",
		BearerToken:    "encrypted-token-" + id,
		TokenExpiresAt: time.Unix(2_000_000_000, 0).UTC(),
		HarnessVersion: "0.0.34",
		CreatedAt:      time.Unix(1_000_000_000, 0).UTC(),
		UpdatedAt:      time.Unix(1_000_000_000, 0).UTC(),
	}
}

func TestPairingRepo_SaveAndGetComputer(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()
	_, _, err := s.Users.UpsertUser(ctx, newTestUser("u1", "42", "onik97"))
	require.NoError(t, err)

	c := newTestComputer("c1", "u1", "home")
	require.NoError(t, s.Pairing.SaveComputer(ctx, c))

	got, err := s.Pairing.GetComputer(ctx, "u1", "c1")
	require.NoError(t, err)
	assert.Equal(t, c.Name, got.Name)
	assert.Equal(t, c.ServerURL, got.ServerURL)
	assert.Equal(t, c.BearerToken, got.BearerToken)
	assert.Equal(t, c.TokenExpiresAt, got.TokenExpiresAt)
	assert.Equal(t, c.HarnessVersion, got.HarnessVersion)
	assert.Equal(t, harness.KindT3Code, got.Kind)
}

func TestPairingRepo_GetComputer_WrongOwnerNotFound(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()
	_, _, err := s.Users.UpsertUser(ctx, newTestUser("u1", "42", "onik97"))
	require.NoError(t, err)
	_, _, err = s.Users.UpsertUser(ctx, newTestUser("u2", "43", "other"))
	require.NoError(t, err)

	require.NoError(t, s.Pairing.SaveComputer(ctx, newTestComputer("c1", "u1", "home")))

	_, err = s.Pairing.GetComputer(ctx, "u2", "c1")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}

func TestPairingRepo_SaveComputer_UpsertUpdatesInPlace(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()
	_, _, err := s.Users.UpsertUser(ctx, newTestUser("u1", "42", "onik97"))
	require.NoError(t, err)

	require.NoError(t, s.Pairing.SaveComputer(ctx, newTestComputer("c1", "u1", "home")))
	updated := newTestComputer("c1", "u1", "home")
	updated.BearerToken = "encrypted-token-rotated"
	updated.HarnessVersion = "0.0.35"
	require.NoError(t, s.Pairing.SaveComputer(ctx, updated))

	got, err := s.Pairing.GetComputer(ctx, "u1", "c1")
	require.NoError(t, err)
	assert.Equal(t, "encrypted-token-rotated", got.BearerToken)
	assert.Equal(t, "0.0.35", got.HarnessVersion)

	all, err := s.Pairing.ListComputers(ctx, "u1")
	require.NoError(t, err)
	assert.Len(t, all, 1, "upsert must not create a duplicate row")
}

func TestPairingRepo_ListComputers_NewestFirst(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()
	_, _, err := s.Users.UpsertUser(ctx, newTestUser("u1", "42", "onik97"))
	require.NoError(t, err)

	older := newTestComputer("c1", "u1", "home")
	older.CreatedAt = time.Unix(1_000_000_000, 0).UTC()
	newer := newTestComputer("c2", "u1", "vps")
	newer.CreatedAt = time.Unix(2_000_000_000, 0).UTC()
	require.NoError(t, s.Pairing.SaveComputer(ctx, older))
	require.NoError(t, s.Pairing.SaveComputer(ctx, newer))

	list, err := s.Pairing.ListComputers(ctx, "u1")
	require.NoError(t, err)
	require.Len(t, list, 2)
	assert.Equal(t, "c2", list[0].ID)
	assert.Equal(t, "c1", list[1].ID)
}

func TestPairingRepo_ListComputers_Empty(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()
	_, _, err := s.Users.UpsertUser(ctx, newTestUser("u1", "42", "onik97"))
	require.NoError(t, err)

	list, err := s.Pairing.ListComputers(ctx, "u1")
	require.NoError(t, err)
	assert.NotNil(t, list)
	assert.Empty(t, list)
}

func TestPairingRepo_DeleteComputer(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()
	_, _, err := s.Users.UpsertUser(ctx, newTestUser("u1", "42", "onik97"))
	require.NoError(t, err)
	require.NoError(t, s.Pairing.SaveComputer(ctx, newTestComputer("c1", "u1", "home")))

	require.NoError(t, s.Pairing.DeleteComputer(ctx, "u1", "c1"))

	_, err = s.Pairing.GetComputer(ctx, "u1", "c1")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}

func TestPairingRepo_DeleteComputer_NotFound(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()
	_, _, err := s.Users.UpsertUser(ctx, newTestUser("u1", "42", "onik97"))
	require.NoError(t, err)

	err = s.Pairing.DeleteComputer(ctx, "u1", "missing")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}

func TestPairingRepo_GetComputerByID_IgnoresOwner(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()
	_, _, err := s.Users.UpsertUser(ctx, newTestUser("u1", "42", "onik97"))
	require.NoError(t, err)
	require.NoError(t, s.Pairing.SaveComputer(ctx, newTestComputer("c1", "u1", "home")))

	got, err := s.Pairing.GetComputerByID(ctx, "c1")
	require.NoError(t, err)
	assert.Equal(t, "home", got.Name)
}

func TestPairingRepo_GetComputerByID_NotFound(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	_, err := s.Pairing.GetComputerByID(context.Background(), "missing")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}

func TestPairingRepo_ProjectLink_RoundTrip(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()
	_, _, err := s.Users.UpsertUser(ctx, newTestUser("u1", "42", "onik97"))
	require.NoError(t, err)
	require.NoError(t, s.Pairing.SaveComputer(ctx, newTestComputer("c1", "u1", "home")))
	require.NoError(t, s.Projects.Create(ctx, newTestProject("proj-1", "Project One", 0)))

	empty, err := s.Pairing.GetProjectLink(ctx, "proj-1")
	require.NoError(t, err)
	assert.Equal(t, pairing.ProjectLink{}, empty, "unlinked project is a zero value, not an error")

	link := pairing.ProjectLink{
		ProjectID: "proj-1", ComputerID: "c1", HarnessProjectID: "t3-proj-1",
		Provider: "claude", Model: "sonnet", UpdatedAt: time.Unix(1_000_000_000, 0).UTC(),
	}
	require.NoError(t, s.Pairing.SaveProjectLink(ctx, link))

	got, err := s.Pairing.GetProjectLink(ctx, "proj-1")
	require.NoError(t, err)
	assert.Equal(t, "c1", got.ComputerID)
	assert.Equal(t, "t3-proj-1", got.HarnessProjectID)
	assert.Equal(t, "claude", got.Provider)
	assert.Equal(t, "sonnet", got.Model)
}

func TestPairingRepo_SaveProjectLink_UpsertUpdatesInPlace(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()
	_, _, err := s.Users.UpsertUser(ctx, newTestUser("u1", "42", "onik97"))
	require.NoError(t, err)
	require.NoError(t, s.Pairing.SaveComputer(ctx, newTestComputer("c1", "u1", "home")))
	require.NoError(t, s.Pairing.SaveComputer(ctx, newTestComputer("c2", "u1", "vps")))
	require.NoError(t, s.Projects.Create(ctx, newTestProject("proj-1", "Project One", 0)))

	require.NoError(t, s.Pairing.SaveProjectLink(ctx, pairing.ProjectLink{ProjectID: "proj-1", ComputerID: "c1", HarnessProjectID: "p1"}))
	require.NoError(t, s.Pairing.SaveProjectLink(ctx, pairing.ProjectLink{ProjectID: "proj-1", ComputerID: "c2", HarnessProjectID: "p2"}))

	got, err := s.Pairing.GetProjectLink(ctx, "proj-1")
	require.NoError(t, err)
	assert.Equal(t, "c2", got.ComputerID)
	assert.Equal(t, "p2", got.HarnessProjectID)
}

func TestPairingRepo_DeleteProjectLink(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()
	_, _, err := s.Users.UpsertUser(ctx, newTestUser("u1", "42", "onik97"))
	require.NoError(t, err)
	require.NoError(t, s.Pairing.SaveComputer(ctx, newTestComputer("c1", "u1", "home")))
	require.NoError(t, s.Projects.Create(ctx, newTestProject("proj-1", "Project One", 0)))
	require.NoError(t, s.Pairing.SaveProjectLink(ctx, pairing.ProjectLink{ProjectID: "proj-1", ComputerID: "c1", HarnessProjectID: "p1"}))

	require.NoError(t, s.Pairing.DeleteProjectLink(ctx, "proj-1"))

	got, err := s.Pairing.GetProjectLink(ctx, "proj-1")
	require.NoError(t, err)
	assert.Equal(t, pairing.ProjectLink{}, got)
}

func TestPairingRepo_DeleteProjectLink_NeverLinkedIsNoop(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	require.NoError(t, s.Pairing.DeleteProjectLink(context.Background(), "proj-missing"))
}

func TestPairingRepo_Defaults_RoundTrip(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()
	_, _, err := s.Users.UpsertUser(ctx, newTestUser("u1", "42", "onik97"))
	require.NoError(t, err)
	require.NoError(t, s.Pairing.SaveComputer(ctx, newTestComputer("c1", "u1", "home")))

	empty, err := s.Pairing.GetDefaults(ctx, "u1")
	require.NoError(t, err)
	assert.Equal(t, pairing.Defaults{}, empty, "no row yet is a zero value, not an error")

	d := pairing.Defaults{UserID: "u1", DefaultComputerID: "c1", FallbackProjectID: "proj-1", Provider: "claude", Model: "sonnet"}
	require.NoError(t, s.Pairing.SaveDefaults(ctx, d))

	got, err := s.Pairing.GetDefaults(ctx, "u1")
	require.NoError(t, err)
	assert.Equal(t, "c1", got.DefaultComputerID)
	assert.Equal(t, "proj-1", got.FallbackProjectID)
	assert.Equal(t, "claude", got.Provider)
	assert.Equal(t, "sonnet", got.Model)
}

func TestPairingRepo_SaveDefaults_UpsertUpdatesInPlace(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()
	_, _, err := s.Users.UpsertUser(ctx, newTestUser("u1", "42", "onik97"))
	require.NoError(t, err)

	require.NoError(t, s.Pairing.SaveDefaults(ctx, pairing.Defaults{UserID: "u1", Provider: "claude"}))
	require.NoError(t, s.Pairing.SaveDefaults(ctx, pairing.Defaults{UserID: "u1", Provider: "opencode"}))

	got, err := s.Pairing.GetDefaults(ctx, "u1")
	require.NoError(t, err)
	assert.Equal(t, "opencode", got.Provider)
}

func newSetupTestStore(t *testing.T) *Store {
	t.Helper()
	s := newTestStore(t)
	_, _, err := s.Users.UpsertUser(t.Context(), newTestUser("u1", "42", "onik97"))
	require.NoError(t, err)
	require.NoError(t, s.Pairing.SaveComputer(t.Context(), newTestComputer("c1", "u1", "home")))
	return s
}

func setupEvt(id string) eventbus.OutboxEvent {
	return eventbus.OutboxEvent{ID: id, Topic: pairing.TopicSetupConfirmed, Payload: pairing.SetupChangedEvent{ComputerID: "c1", UserID: "u1"}}
}

func TestPairingRepo_SetSetupConfirmedAt_WrongOwner_NotFoundAndNoEvent(t *testing.T) {
	t.Parallel()
	s := newSetupTestStore(t)
	at := time.Unix(1_700_000_000, 0).UTC()

	err := s.Pairing.SetSetupConfirmedAt(t.Context(), "u2", "c1", &at, setupEvt("e1"))
	require.ErrorIs(t, err, apperrs.ErrNotFound)

	entries, err := s.Outbox.Unpublished(t.Context(), 10)
	require.NoError(t, err)
	assert.Empty(t, entries, "the outbox write rolls back with the missed update")
}

func TestPairingRepo_SetSetupConfirmedAt_SetClearAndSurvivesRePair(t *testing.T) {
	t.Parallel()
	s := newSetupTestStore(t)
	at := time.Unix(1_700_000_000, 0).UTC()

	require.NoError(t, s.Pairing.SetSetupConfirmedAt(t.Context(), "u1", "c1", &at, setupEvt("e1")))
	require.NoError(t, s.Pairing.SaveComputer(t.Context(), newTestComputer("c1", "u1", "renamed")))
	got, err := s.Pairing.GetComputer(t.Context(), "u1", "c1")
	require.NoError(t, err)
	require.NotNil(t, got.SetupConfirmedAt, "a re-pair never withdraws the confirmation")
	assert.Equal(t, at, *got.SetupConfirmedAt)

	require.NoError(t, s.Pairing.SetSetupConfirmedAt(t.Context(), "u1", "c1", nil, setupEvt("e2")))
	got, err = s.Pairing.GetComputer(t.Context(), "u1", "c1")
	require.NoError(t, err)
	assert.Nil(t, got.SetupConfirmedAt)

	entries, err := s.Outbox.Unpublished(t.Context(), 10)
	require.NoError(t, err)
	require.Len(t, entries, 2)
	assert.Equal(t, pairing.TopicSetupConfirmed, entries[0].Topic)
}

func TestPairingRepo_ProviderSetups_UpsertListAndCascade(t *testing.T) {
	t.Parallel()
	s := newSetupTestStore(t)
	ctx := t.Context()
	at := time.Unix(1_700_000_000, 0).UTC()

	empty, err := s.Pairing.ListProviderSetups(ctx, "c1")
	require.NoError(t, err)
	assert.Empty(t, empty)

	require.NoError(t, s.Pairing.SaveProviderSetup(ctx, "c1", pairing.ProviderSetup{Provider: "codex", ConfirmedAt: &at, Skills: []string{"tdd"}}, at, setupEvt("e1")))
	require.NoError(t, s.Pairing.SaveProviderSetup(ctx, "c1", pairing.ProviderSetup{Provider: "claude", ConfirmedAt: &at, Skills: []string{"tdd", "diagnose"}}, at, setupEvt("e2")))
	require.NoError(t, s.Pairing.SaveProviderSetup(ctx, "c1", pairing.ProviderSetup{Provider: "codex", Skills: []string{}}, at, setupEvt("e3")))

	got, err := s.Pairing.ListProviderSetups(ctx, "c1")
	require.NoError(t, err)
	assert.Equal(t, []pairing.ProviderSetup{
		{Provider: "claude", ConfirmedAt: &at, Skills: []string{"tdd", "diagnose"}},
		{Provider: "codex", Skills: []string{}},
	}, got)

	entries, err := s.Outbox.Unpublished(ctx, 10)
	require.NoError(t, err)
	assert.Len(t, entries, 3)

	require.NoError(t, s.Pairing.DeleteComputer(ctx, "u1", "c1"))
	got, err = s.Pairing.ListProviderSetups(ctx, "c1")
	require.NoError(t, err)
	assert.Empty(t, got, "unpairing drops the computer's provider rows")
}

func TestPairingRepo_SaveProviderSetup_UnknownComputerRejected(t *testing.T) {
	t.Parallel()
	s := newSetupTestStore(t)
	err := s.Pairing.SaveProviderSetup(t.Context(), "missing", pairing.ProviderSetup{Provider: "claude", Skills: []string{}}, time.Now(), setupEvt("e1"))
	require.Error(t, err)
}

func TestPairingRepo_ListProviderSetups_CorruptSkills_Error(t *testing.T) {
	t.Parallel()
	s := newSetupTestStore(t)
	_, err := s.db.Exec(`INSERT INTO pairing_provider_setups (computer_id, provider, skills_json, updated_at) VALUES ('c1', 'claude', 'not json', 0)`)
	require.NoError(t, err)
	_, err = s.Pairing.ListProviderSetups(t.Context(), "c1")
	require.Error(t, err)
}
