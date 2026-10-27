package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/harness"
	"github.com/otal-labs/nexul/internal/pairing"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/plays"
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

func TestPairingRepo_ListComputers_InTheirOwnersOrder(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()
	for _, u := range []string{"u1", "u2"} {
		_, _, err := s.Users.UpsertUser(ctx, newTestUser(u, u, u))
		require.NoError(t, err)
	}
	for _, c := range []pairing.Computer{newTestComputer("c1", "u1", "home"), newTestComputer("c2", "u1", "vps"), newTestComputer("c3", "u2", "theirs")} {
		require.NoError(t, s.Pairing.SaveComputer(ctx, c))
	}
	ids := func() []string {
		list, err := s.Pairing.ListComputers(ctx, "u1")
		require.NoError(t, err)
		out := []string{}
		for _, c := range list {
			out = append(out, c.ID)
		}
		return out
	}
	assert.Equal(t, []string{"c1", "c2"}, ids(), "a new computer goes last")

	require.NoError(t, s.Pairing.ReorderComputers(ctx, "u1", []string{"c2", "c1"}, time.Now()))
	assert.Equal(t, []string{"c2", "c1"}, ids())
	require.NoError(t, s.Pairing.SaveComputer(ctx, newTestComputer("c1", "u1", "home-renamed")))
	assert.Equal(t, []string{"c2", "c1"}, ids(), "saving a computer again never moves it")
	require.ErrorIs(t, s.Pairing.ReorderComputers(ctx, "u1", []string{"c3"}, time.Now()), apperrs.ErrNotFound, "another person's computer")
	theirs, err := s.Pairing.ListComputers(ctx, "u2")
	require.NoError(t, err)
	assert.Equal(t, 0, theirs[0].Position, "a refused order changes nothing")
}

func TestPairingRepo_SetComputerPlacement_OwnerOnly(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()
	_, _, err := s.Users.UpsertUser(ctx, newTestUser("u1", "42", "onik97"))
	require.NoError(t, err)
	require.NoError(t, s.Pairing.SaveComputer(ctx, newTestComputer("c1", "u1", "home")))

	got, err := s.Pairing.GetComputer(ctx, "u1", "c1")
	require.NoError(t, err)
	assert.Equal(t, []any{1, ""}, []any{got.RunsAtOnce, got.ChatProjectID}, "one run at once and no chat folder until set")

	require.NoError(t, s.Pairing.SetComputerPlacement(ctx, "u1", "c1", 4, "t3-chat", time.Now()))
	got, err = s.Pairing.GetComputer(ctx, "u1", "c1")
	require.NoError(t, err)
	assert.Equal(t, []any{4, "t3-chat"}, []any{got.RunsAtOnce, got.ChatProjectID})
	require.ErrorIs(t, s.Pairing.SetComputerPlacement(ctx, "u2", "c1", 2, "", time.Now()), apperrs.ErrNotFound)
}

func TestPairingRepo_CountRunning_CountsTheRunsGoingOnEachComputer(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()
	seedPlays(t, s, "play-1")
	for i, tc := range []struct {
		computer string
		state    plays.TrailState
	}{{"c1", plays.TrailRunning}, {"c1", plays.TrailWaiting}, {"c1", plays.TrailDone}, {"c2", plays.TrailStarting}, {"c3", plays.TrailRunning}} {
		require.NoError(t, s.PlayTrails.CreateTrail(ctx, &plays.Trail{
			ID: fmt.Sprintf("tr-%d", i), WorkspaceID: "ws-1", PlayID: "play-1", TargetType: plays.TargetTicket, TargetID: "t-1",
			StarterID: "u1", Via: plays.ViaWeb, SelectedMemoryIDs: []string{}, State: tc.state, StartedAt: time.Now(), ComputerID: tc.computer,
		}))
	}

	running, err := s.Pairing.CountRunning(ctx, []string{"c1", "c2", "c-idle"})

	require.NoError(t, err)
	assert.Equal(t, map[string]int{"c1": 2, "c2": 1}, running, "an ended run frees its room, and another computer's runs are not counted")
	none, err := s.Pairing.CountRunning(ctx, nil)
	require.NoError(t, err)
	assert.Empty(t, none)
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

func TestPairingRepo_ProjectLink_RoundTrip(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()
	_, _, err := s.Users.UpsertUser(ctx, newTestUser("u1", "42", "onik97"))
	require.NoError(t, err)
	require.NoError(t, s.Pairing.SaveComputer(ctx, newTestComputer("c1", "u1", "home")))
	require.NoError(t, s.Pairing.SaveComputer(ctx, newTestComputer("c2", "u1", "vps")))
	require.NoError(t, s.Projects.Create(ctx, newTestProject("proj-1", "Project One", 0)))

	empty, err := s.Pairing.GetProjectLink(ctx, "u1", "proj-1")
	require.NoError(t, err)
	assert.Equal(t, pairing.ProjectLink{UserID: "u1", ProjectID: "proj-1"}, empty, "unlinked project is a zero value, not an error")

	link := pairing.ProjectLink{
		UserID: "u1", ProjectID: "proj-1", Provider: "claude", Model: "sonnet", UpdatedAt: time.Unix(1_000_000_000, 0).UTC(),
		ModelOptions: []harness.OptionSetting{{ID: "effort", Value: "high"}, {ID: "fastMode", Value: true}},
		StartIn:      pairing.StartInWorktree, Placement: pairing.PlacementSpread, Computers: []string{"c2", "c1"},
		Checkouts: []pairing.Checkout{{ComputerID: "c1", HarnessProjectID: "t3-home"}},
	}
	require.NoError(t, s.Pairing.SaveProjectLink(ctx, link))

	got, err := s.Pairing.GetProjectLink(ctx, "u1", "proj-1")
	require.NoError(t, err)
	assert.Equal(t, link, got, "a switch reads back as a bool, a choice as a string, the computers in their order")

	link.Computers = []string{"c1"}
	require.NoError(t, s.Pairing.SaveProjectLink(ctx, link))
	got, err = s.Pairing.GetProjectLink(ctx, "u1", "proj-1")
	require.NoError(t, err)
	assert.Equal(t, []string{"c1"}, got.Computers, "a saved list replaces the old one")
}

func TestPairingRepo_ProjectLinks_AreKeptPerPerson(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()
	for _, u := range []string{"u1", "u2"} {
		_, _, err := s.Users.UpsertUser(ctx, newTestUser(u, u, u))
		require.NoError(t, err)
	}
	require.NoError(t, s.Pairing.SaveComputer(ctx, newTestComputer("c1", "u1", "home")))
	require.NoError(t, s.Pairing.SaveComputer(ctx, newTestComputer("c2", "u1", "vps")))
	require.NoError(t, s.Pairing.SaveComputer(ctx, newTestComputer("c3", "u2", "theirs")))
	require.NoError(t, s.Projects.Create(ctx, newTestProject("proj-1", "Project One", 0)))
	require.NoError(t, s.Projects.Create(ctx, newTestProject("proj-2", "Project Two", 1)))
	require.NoError(t, s.Projects.Create(ctx, newTestProject("proj-3", "Project Three", 2)))

	require.NoError(t, s.Pairing.SaveProjectLink(ctx, pairing.ProjectLink{UserID: "u1", ProjectID: "proj-2", Model: "sonnet"}))
	require.NoError(t, s.Pairing.SaveProjectLink(ctx, pairing.ProjectLink{UserID: "u1", ProjectID: "proj-1", Computers: []string{"c1"}}))
	require.NoError(t, s.Pairing.SaveProjectLink(ctx, pairing.ProjectLink{UserID: "u2", ProjectID: "proj-1", Computers: []string{"c3"}}))
	require.NoError(t, s.Pairing.SaveCheckout(ctx, "u1", "proj-3", "c2", "t3-three"))

	links, err := s.Pairing.ListProjectLinks(ctx, "u1")
	require.NoError(t, err)
	require.Len(t, links, 3)
	assert.Equal(t, []string{"proj-1", "proj-2", "proj-3"}, []string{links[0].ProjectID, links[1].ProjectID, links[2].ProjectID})
	assert.Equal(t, []string{"c1"}, links[0].Computers)
	assert.Equal(t, []pairing.Checkout{{ComputerID: "c2", HarnessProjectID: "t3-three"}}, links[2].Checkouts, "a picked checkout alone is listed too")

	require.NoError(t, s.Pairing.SaveCheckout(ctx, "u1", "proj-1", "c1", "t3-one"))
	require.NoError(t, s.Pairing.DeleteProjectLink(ctx, "u1", "proj-1"))
	mine, err := s.Pairing.GetProjectLink(ctx, "u1", "proj-1")
	require.NoError(t, err)
	assert.Empty(t, mine.Computers, "clearing drops the project's own order")
	assert.Equal(t, []pairing.Checkout{{ComputerID: "c1", HarnessProjectID: "t3-one"}}, mine.Checkouts, "and keeps where the code is")
	theirs, err := s.Pairing.GetProjectLink(ctx, "u2", "proj-1")
	require.NoError(t, err)
	assert.Equal(t, []string{"c3"}, theirs.Computers, "neither a save nor a clear touches another person's link")
}

func TestPairingRepo_DeleteProjectLink_NeverLinkedIsNoop(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	require.NoError(t, s.Pairing.DeleteProjectLink(context.Background(), "u1", "proj-missing"))
}

func TestPairingRepo_Defaults_RoundTrip(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()
	_, _, err := s.Users.UpsertUser(ctx, newTestUser("u1", "42", "onik97"))
	require.NoError(t, err)

	empty, err := s.Pairing.GetDefaults(ctx, "u1")
	require.NoError(t, err)
	assert.Equal(t, pairing.Defaults{}, empty, "no row yet is a zero value, not an error")

	d := pairing.Defaults{UserID: "u1", Provider: "claude", Model: "sonnet", ModelOptions: []harness.OptionSetting{{ID: "contextWindow", Value: "1m"}},
		StartIn: pairing.StartInWorktree, Placement: pairing.PlacementSpread, CloneAutomatically: true}
	require.NoError(t, s.Pairing.SaveDefaults(ctx, d))

	got, err := s.Pairing.GetDefaults(ctx, "u1")
	require.NoError(t, err)
	d.UserID = ""
	assert.Equal(t, d, got)
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

func TestPairingRepo_SaveSetupChoices_RoundTripsAndSurvivesRePair(t *testing.T) {
	t.Parallel()
	s := newSetupTestStore(t)
	got, err := s.Pairing.GetComputer(t.Context(), "u1", "c1")
	require.NoError(t, err)
	empty := pairing.SetupChoices{Skipped: []string{}, Models: map[string]string{}, ModelOptions: map[string][]harness.OptionSetting{}}
	assert.Equal(t, empty, got.SetupChoices, "a new computer skips nothing and has no saved picks")

	choices := pairing.SetupChoices{
		Skipped:      []string{"codex"},
		Models:       map[string]string{"claudeagent": "claude-opus-5-5", "opencode": ""},
		ModelOptions: map[string][]harness.OptionSetting{"claudeagent": {{ID: "effort", Value: "high"}, {ID: "fastMode", Value: true}}},
		Folder:       "/home/me/app",
	}
	require.ErrorIs(t, s.Pairing.SaveSetupChoices(t.Context(), "u2", "c1", choices), apperrs.ErrNotFound)
	require.NoError(t, s.Pairing.SaveSetupChoices(t.Context(), "u1", "c1", choices))
	require.NoError(t, s.Pairing.SaveComputer(t.Context(), newTestComputer("c1", "u1", "renamed")))
	got, err = s.Pairing.GetComputer(t.Context(), "u1", "c1")
	require.NoError(t, err)
	assert.Equal(t, choices, got.SetupChoices, "re-pairing keeps the saved choices")

	require.NoError(t, s.Pairing.SaveSetupChoices(t.Context(), "u1", "c1", empty))
	got, err = s.Pairing.GetComputer(t.Context(), "u1", "c1")
	require.NoError(t, err)
	assert.Equal(t, empty, got.SetupChoices)
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

func TestPairingRepo_ListOutdatedSkillsComputers_OwnConfirmedProvidersOnAnotherVersion(t *testing.T) {
	t.Parallel()
	s := newSetupTestStore(t)
	ctx := t.Context()
	at := time.Unix(1_700_000_000, 0).UTC()
	require.NoError(t, s.Pairing.SaveComputer(ctx, newTestComputer("c2", "u1", "laptop")))
	_, _, err := s.Users.UpsertUser(ctx, newTestUser("u2", "43", "lena"))
	require.NoError(t, err)
	require.NoError(t, s.Pairing.SaveComputer(ctx, newTestComputer("c3", "u2", "lenas")))
	require.NoError(t, s.Pairing.SaveProviderSetup(ctx, "c1", pairing.ProviderSetup{Provider: "codex", ConfirmedAt: &at, Skills: []string{}, SkillsVersion: "v1"}, at, setupEvt("e1")))
	require.NoError(t, s.Pairing.SaveProviderSetup(ctx, "c1", pairing.ProviderSetup{Provider: "claude", ConfirmedAt: &at, Skills: []string{}, SkillsVersion: "v1"}, at, setupEvt("e2")))
	require.NoError(t, s.Pairing.SaveProviderSetup(ctx, "c2", pairing.ProviderSetup{Provider: "codex", ConfirmedAt: &at, Skills: []string{}, SkillsVersion: "v2"}, at, setupEvt("e3")))
	require.NoError(t, s.Pairing.SaveProviderSetup(ctx, "c2", pairing.ProviderSetup{Provider: "claude", Skills: []string{}, SkillsVersion: "v1"}, at, setupEvt("e4")))
	require.NoError(t, s.Pairing.SaveProviderSetup(ctx, "c3", pairing.ProviderSetup{Provider: "codex", ConfirmedAt: &at, Skills: []string{}, SkillsVersion: "v1"}, at, setupEvt("e5")))

	got, err := s.Pairing.ListOutdatedSkillsComputers(ctx, "u1", "v2")
	require.NoError(t, err)
	assert.Equal(t, []string{"c1"}, got, "c1 once for both its providers; c2's old provider is unconfirmed; c3 is another person's")
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

func withTunnel(c pairing.Computer, hostname string) pairing.Computer {
	c.Tunnel = &pairing.ComputerTunnel{TunnelID: "tun-" + c.ID, Hostname: hostname, ZoneID: "z1", RecordID: "rec-" + c.ID, AccessAppID: "app-" + c.ID}
	return c
}

func TestPairingRepo_ComputerTunnel_SurvivesRepairAndMarksItsHostname(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := t.Context()
	_, _, err := s.Users.UpsertUser(ctx, newTestUser("u1", "42", "onik97"))
	require.NoError(t, err)
	c := withTunnel(newTestComputer("c1", "u1", "laptop"), "laptop-ab12cd34.example.com")
	created := eventbus.OutboxEvent{ID: "evt-created", Topic: pairing.TopicTunnelCreated, Payload: map[string]string{"computer_id": "c1"}}
	require.NoError(t, s.Pairing.SaveComputer(ctx, c, created))

	repaired := newTestComputer("c1", "u1", "laptop")
	repaired.BearerToken = "encrypted-new"
	require.NoError(t, s.Pairing.SaveComputer(ctx, repaired))

	got, err := s.Pairing.GetComputer(ctx, "u1", "c1")
	require.NoError(t, err)
	assert.Equal(t, "encrypted-new", got.BearerToken)
	assert.Equal(t, c.Tunnel, got.Tunnel, "re-pairing never drops the tunnel")

	plain := newTestComputer("c2", "u1", "vps")
	require.NoError(t, s.Pairing.SaveComputer(ctx, plain))
	got, err = s.Pairing.GetComputer(ctx, "u1", "c2")
	require.NoError(t, err)
	assert.Nil(t, got.Tunnel)

	onTunnel, err := s.Pairing.ListTunnelComputers(ctx)
	require.NoError(t, err)
	require.Len(t, onTunnel, 1, "only the computer holding a tunnel, whoever owns it")
	assert.Equal(t, "c1", onTunnel[0].ID)
	assert.Equal(t, c.Tunnel, onTunnel[0].Tunnel)

	removed := eventbus.OutboxEvent{ID: "evt-removed", Topic: pairing.TopicTunnelRemoved, Payload: map[string]string{"computer_id": "c1"}}
	require.NoError(t, s.Pairing.DeleteComputer(ctx, "u1", "c1", removed))
	onTunnel, err = s.Pairing.ListTunnelComputers(ctx)
	require.NoError(t, err)
	assert.Empty(t, onTunnel)

	pending, err := s.Outbox.Unpublished(ctx, 10)
	require.NoError(t, err)
	topics := make([]string, 0, len(pending))
	for _, p := range pending {
		topics = append(topics, p.Topic)
	}
	assert.Equal(t, []string{pairing.TopicTunnelCreated, pairing.TopicTunnelRemoved}, topics)
}

func TestPairingRepo_ComputerTunnel_HostnameIsUnique(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := t.Context()
	_, _, err := s.Users.UpsertUser(ctx, newTestUser("u1", "42", "onik97"))
	require.NoError(t, err)
	require.NoError(t, s.Pairing.SaveComputer(ctx, withTunnel(newTestComputer("c1", "u1", "a"), "same.example.com")))

	err = s.Pairing.SaveComputer(ctx, withTunnel(newTestComputer("c2", "u1", "b"), "same.example.com"))
	require.ErrorIs(t, err, apperrs.ErrConflict)
}

func TestPairingRepo_SetSetupMCPToken_SurvivesRePairAndChecksOwner(t *testing.T) {
	t.Parallel()
	s := newSetupTestStore(t)

	require.ErrorIs(t, s.Pairing.SetSetupMCPToken(t.Context(), "u2", "c1", "sealed"), apperrs.ErrNotFound)
	require.NoError(t, s.Pairing.SetSetupMCPToken(t.Context(), "u1", "c1", "sealed"))
	require.NoError(t, s.Pairing.SaveComputer(t.Context(), newTestComputer("c1", "u1", "renamed")))

	got, err := s.Pairing.GetComputer(t.Context(), "u1", "c1")
	require.NoError(t, err)
	assert.Equal(t, "sealed", got.SetupMCPToken)
}

func TestPairingRepo_SaveSetupTurn_UpsertsWithItsEvents(t *testing.T) {
	t.Parallel()
	s := newSetupTestStore(t)
	started := time.Unix(1_700_000_000, 0).UTC()
	turn := pairing.SetupTurn{
		ID: "t1", RunID: "r1", ComputerID: "c1", UserID: "u1", Provider: "codex", ProviderName: "Codex",
		State: pairing.SetupTurnRunning, Status: "Installing", StartedAt: started,
	}
	evt := eventbus.OutboxEvent{ID: "e1", Topic: pairing.TopicSetupTurnChanged, Payload: pairing.SetupTurnChangedEvent{TurnID: "t1"}}
	require.NoError(t, s.Pairing.SaveSetupTurn(t.Context(), turn, evt))

	ended := started.Add(time.Minute)
	turn.State, turn.Status, turn.EndedAt = pairing.SetupTurnConfirmed, "Confirmed", &ended
	turn.Transcript = []harness.Activity{{Kind: harness.ActivityText, Summary: "done"}}
	evt.ID = "e2"
	require.NoError(t, s.Pairing.SaveSetupTurn(t.Context(), turn, evt))

	var state, transcript string
	var endedAt int64
	row := s.db.QueryRowContext(t.Context(), `SELECT state, transcript, ended_at FROM pairing_setup_turns WHERE id = 't1'`)
	require.NoError(t, row.Scan(&state, &transcript, &endedAt))
	assert.Equal(t, "confirmed", state)
	assert.Contains(t, transcript, `"Summary":"done"`)
	assert.Equal(t, ended.Unix(), endedAt)
	entries, err := s.Outbox.Unpublished(t.Context(), 10)
	require.NoError(t, err)
	assert.Len(t, entries, 2)

	turn.ComputerID = "missing"
	turn.ID = "t2"
	require.Error(t, s.Pairing.SaveSetupTurn(t.Context(), turn), "a turn needs its computer")
}

func TestPairingRepo_ListLatestSetupTurns_NewestPerProvider(t *testing.T) {
	t.Parallel()
	s := newSetupTestStore(t)
	at := time.Unix(1_700_000_000, 0).UTC()
	save := func(id, run, provider string, state pairing.SetupTurnState, started time.Time) {
		t.Helper()
		require.NoError(t, s.Pairing.SaveSetupTurn(t.Context(), pairing.SetupTurn{
			ID: id, RunID: run, ComputerID: "c1", UserID: "u1", Provider: provider, ProviderName: provider,
			State: state, Status: string(state), StartedAt: started, UpdatedAt: started.Add(time.Second),
		}))
	}
	save("t1", "r1", "codex", pairing.SetupTurnFailed, at)
	save("t2", "r1", "claudeagent", pairing.SetupTurnConfirmed, at.Add(time.Minute))
	save("t3", "r2", "codex", pairing.SetupTurnConfirmed, at.Add(time.Hour))
	picked := pairing.SetupTurn{ID: "t4", RunID: "r3", ComputerID: "c1", UserID: "u1", Provider: "opencode", ProviderName: "OpenCode", Model: "big-pickle",
		Kind: pairing.SetupTurnSkills, State: pairing.SetupTurnRunning, StartedAt: at, UpdatedAt: at}
	require.NoError(t, s.Pairing.SaveSetupTurn(t.Context(), picked))

	got, err := s.Pairing.ListLatestSetupTurns(t.Context(), "c1")
	require.NoError(t, err)
	require.Len(t, got, 3)
	assert.Equal(t, "big-pickle", got[2].Model, "the picked model reads back with the turn")
	assert.Equal(t, pairing.SetupTurnSkills, got[2].Kind, "a skills update reads back as one")
	assert.Equal(t, pairing.SetupTurnSummary{RunID: "r1", TurnID: "t2", Provider: "claudeagent", ProviderName: "claudeagent", State: pairing.SetupTurnConfirmed, Status: "confirmed", UpdatedAt: at.Add(time.Minute + time.Second)}, got[0])
	assert.Equal(t, "t3", got[1].TurnID)
	assert.Equal(t, "r2", got[1].RunID)

	none, err := s.Pairing.ListLatestSetupTurns(t.Context(), "other")
	require.NoError(t, err)
	assert.Empty(t, none)
}

func switchedEvt(userID string) eventbus.OutboxEvent {
	return eventbus.OutboxEvent{ID: uuid.NewString(), Topic: pairing.TopicHarnessSwitched, Payload: pairing.HarnessSwitchedEvent{ComputerID: "c1", UserID: userID}}
}

func TestPairingRepo_SwitchComputerKind_MovesOnlyAComputerStillOnTheOldKind(t *testing.T) {
	t.Parallel()
	s := newSetupTestStore(t)
	at := time.Unix(1_800_000_000, 0).UTC()

	require.NoError(t, s.Pairing.SwitchComputerKind(t.Context(), "gone", harness.KindT3Code, harness.KindT3CodeV2, "0.0.46", at, switchedEvt))
	require.NoError(t, s.Pairing.SwitchComputerKind(t.Context(), "c1", harness.KindT3Code, harness.KindT3CodeV2, "0.0.46", at, switchedEvt))
	require.NoError(t, s.Pairing.SwitchComputerKind(t.Context(), "c1", harness.KindT3Code, harness.KindT3CodeV2, "0.0.47", at, switchedEvt))

	got, err := s.Pairing.GetComputer(t.Context(), "u1", "c1")
	require.NoError(t, err)
	assert.Equal(t, harness.KindT3CodeV2, got.Kind)
	assert.Equal(t, "0.0.46", got.HarnessVersion, "a computer already moved keeps its row")
	assert.Equal(t, at, got.UpdatedAt)
	assert.Equal(t, "encrypted-token-c1", got.BearerToken, "no re-pair: the bearer session carries over")
	entries, err := s.Outbox.Unpublished(t.Context(), 10)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	var payload pairing.HarnessSwitchedEvent
	require.NoError(t, json.Unmarshal(entries[0].Payload, &payload))
	assert.Equal(t, "u1", payload.UserID, "the event names the owner the update returned")
}

func TestPairingRepo_SwitchComputerKind_ConcurrentSwitchesWriteOneEvent(t *testing.T) {
	t.Parallel()
	s := newSetupTestStore(t)
	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			assert.NoError(t, s.Pairing.SwitchComputerKind(t.Context(), "c1", harness.KindT3Code, harness.KindT3CodeV2, "0.0.46", time.Now(), switchedEvt))
		})
	}
	wg.Wait()

	entries, err := s.Outbox.Unpublished(t.Context(), 20)
	require.NoError(t, err)
	assert.Len(t, entries, 1, "every caller that found the computer already moved wrote nothing")
}

// TestPairingRepo_SetFacts_RoundTripsOwnerOnlyAndSurvivesRePair: facts read back whole with their time and their event
// in the outbox, another person's write is not found and leaves nothing, and a re-pair never drops them.
func TestPairingRepo_SetFacts_RoundTripsOwnerOnlyAndSurvivesRePair(t *testing.T) {
	t.Parallel()
	s := newSetupTestStore(t)
	got, err := s.Pairing.GetComputer(t.Context(), "u1", "c1")
	require.NoError(t, err)
	assert.Nil(t, got.Facts, "no facts before the runner reports")
	assert.Nil(t, got.FactsAt)

	at := time.Unix(1_700_000_000, 0).UTC()
	facts := pairing.Facts{Hostname: "alice-laptop", GitEmail: "alice@example.com", T3: pairing.T3Facts{State: "answering", Port: 3773},
		Providers: []pairing.ProviderFacts{{ID: "codex", Driver: "codex", Name: "Codex", SignIn: harness.SignedOut}}}
	evt := eventbus.OutboxEvent{ID: "e1", Topic: pairing.TopicFactsChanged, Payload: pairing.FactsChangedEvent{ComputerID: "c1", UserID: "u1", FactsAt: at, MembersOnly: true}}

	require.ErrorIs(t, s.Pairing.SetFacts(t.Context(), "u2", "c1", facts, at, evt), apperrs.ErrNotFound)
	entries, err := s.Outbox.Unpublished(t.Context(), 10)
	require.NoError(t, err)
	assert.Empty(t, entries, "the outbox write rolls back with the missed update")

	require.NoError(t, s.Pairing.SetFacts(t.Context(), "u1", "c1", facts, at, evt))
	require.NoError(t, s.Pairing.SaveComputer(t.Context(), newTestComputer("c1", "u1", "renamed")))
	got, err = s.Pairing.GetComputer(t.Context(), "u1", "c1")
	require.NoError(t, err)
	require.NotNil(t, got.Facts, "a re-pair keeps the facts")
	assert.Equal(t, "alice-laptop", got.Facts.Hostname)
	assert.Equal(t, harness.SignedOut, got.Facts.Providers[0].SignIn)
	assert.Empty(t, got.Facts.Projects)
	assert.Equal(t, at, *got.FactsAt)
	entries, err = s.Outbox.Unpublished(t.Context(), 10)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, pairing.TopicFactsChanged, entries[0].Topic)
}

// TestPairingRepo_ClearTunnel_ForgetsTheTunnelAndKeepsTheComputer: clearing a moved computer's tunnel empties the tunnel
// columns with its event, keeps the row's session and setup, and frees the hostname; another owner's computer is not
// found.
func TestPairingRepo_ClearTunnel_ForgetsTheTunnelAndKeepsTheComputer(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := t.Context()
	_, _, err := s.Users.UpsertUser(ctx, newTestUser("u1", "42", "alice"))
	require.NoError(t, err)
	c := newTestComputer("c1", "u1", "laptop")
	c.Tunnel = &pairing.ComputerTunnel{TunnelID: "tun-1", Hostname: "laptop-ab12cd34.example.com", ZoneID: "z1", RecordID: "rec-1", AccessAppID: "app-1"}
	require.NoError(t, s.Pairing.SaveComputer(ctx, c))
	at := time.Unix(1_900_000_000, 0).UTC()

	err = s.Pairing.ClearTunnel(ctx, "u-other", "c1", at, eventbus.OutboxEvent{ID: "evt-0", Topic: pairing.TopicTunnelRemoved, Payload: map[string]string{}})
	require.ErrorIs(t, err, apperrs.ErrNotFound)
	require.NoError(t, s.Pairing.ClearTunnel(ctx, "u1", "c1", at, eventbus.OutboxEvent{ID: "evt-1", Topic: pairing.TopicTunnelRemoved, Payload: map[string]string{}}))

	got, err := s.Pairing.GetComputer(ctx, "u1", "c1")
	require.NoError(t, err)
	assert.Nil(t, got.Tunnel)
	assert.Equal(t, c.BearerToken, got.BearerToken)
	assert.Equal(t, at, got.UpdatedAt)
	onTunnel, err := s.Pairing.ListTunnelComputers(ctx)
	require.NoError(t, err)
	assert.Empty(t, onTunnel, "the next boot's cleanup finds nothing left")
	assert.Equal(t, 1, count(t, s.db, `SELECT COUNT(*) FROM outbox WHERE id = 'evt-1'`))
	assert.Zero(t, count(t, s.db, `SELECT COUNT(*) FROM outbox WHERE id = 'evt-0'`))
}
