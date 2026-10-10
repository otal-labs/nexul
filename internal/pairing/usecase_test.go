package pairing

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/harness"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

var testEncKey = []byte("0123456789abcdef0123456789abcdef")

func newTestService(repo *fakeRepo, exch *fakeExchanger) *Service {
	return NewService(Config{
		Repo:          repo,
		Harnesses:     registry(exch),
		EncryptionKey: testEncKey,
		Now:           func() time.Time { return time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC) },
		Tokens:        newFakeTokens(),
		Projects:      fakeProjects{},
	})
}

func TestService_Pair_Success(t *testing.T) {
	t.Parallel()
	repo := newFakeRepo()
	exch := &fakeExchanger{result: harness.PairResult{BearerToken: "bearer-abc", ExpiresIn: 30 * 24 * time.Hour}, version: "0.0.34"}
	svc := newTestService(repo, exch)

	c, err := svc.Pair(context.Background(), "u1", harness.KindT3Code, "  Home  ", "https://home.example.com/", "one-time-token")
	require.NoError(t, err)
	assert.Equal(t, "Home", c.Name)
	assert.Equal(t, "https://home.example.com", c.ServerURL) // trailing slash trimmed
	assert.Equal(t, "0.0.34", c.HarnessVersion)
	assert.Equal(t, time.Date(2026, 9, 25, 12, 0, 0, 0, time.UTC), c.TokenExpiresAt)
	assert.Empty(t, c.BearerToken, "bearer token never returned to the caller")
	assert.NotEmpty(t, c.ID)

	// The stored row keeps the encrypted token, not the plaintext.
	stored, err := repo.GetComputer(context.Background(), "u1", c.ID)
	require.NoError(t, err)
	assert.NotEqual(t, "bearer-abc", stored.BearerToken)
	assert.NotEmpty(t, stored.BearerToken)
}

func TestService_Pair_ValidatesInput(t *testing.T) {
	t.Parallel()
	svc := newTestService(newFakeRepo(), &fakeExchanger{})

	tests := []struct {
		name      string
		userID    string
		compName  string
		serverURL string
		token     string
	}{
		{"missing user", "", "Home", "https://h.example.com", "tok"},
		{"missing name", "u1", "  ", "https://h.example.com", "tok"},
		{"missing server url", "u1", "Home", "", "tok"},
		{"bad server url scheme", "u1", "Home", "ftp://h.example.com", "tok"},
		{"missing token", "u1", "Home", "https://h.example.com", "  "},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.Pair(context.Background(), tt.userID, harness.KindT3Code, tt.compName, tt.serverURL, tt.token)
			require.Error(t, err)
		})
	}
}

func TestService_Pair_ExchangeError(t *testing.T) {
	t.Parallel()
	svc := newTestService(newFakeRepo(), &fakeExchanger{exchangeErr: errBoom})
	_, err := svc.Pair(context.Background(), "u1", harness.KindT3Code, "Home", "https://h.example.com", "tok")
	require.ErrorIs(t, err, errBoom)
}

func TestService_Pair_VersionError(t *testing.T) {
	t.Parallel()
	svc := newTestService(newFakeRepo(), &fakeExchanger{result: harness.PairResult{BearerToken: "b"}, versionErr: errBoom})
	_, err := svc.Pair(context.Background(), "u1", harness.KindT3Code, "Home", "https://h.example.com", "tok")
	require.ErrorIs(t, err, errBoom)
}

func TestService_Pair_NoEncryptionKey(t *testing.T) {
	t.Parallel()
	svc := NewService(Config{Repo: newFakeRepo(), Harnesses: registry(&fakeExchanger{result: harness.PairResult{BearerToken: "b"}, version: "0.0.34"})})
	_, err := svc.Pair(context.Background(), "u1", harness.KindT3Code, "Home", "https://h.example.com", "tok")
	require.ErrorIs(t, err, apperrs.ErrFatal)
}

func TestService_Repair_UpdatesExistingRow(t *testing.T) {
	t.Parallel()
	repo := newFakeRepo()
	exch := &fakeExchanger{result: harness.PairResult{BearerToken: "bearer-1", ExpiresIn: time.Hour}, version: "0.0.33"}
	svc := newTestService(repo, exch)

	created, err := svc.Pair(context.Background(), "u1", harness.KindT3Code, "Home", "https://h.example.com", "tok1")
	require.NoError(t, err)

	exch.result = harness.PairResult{BearerToken: "bearer-2", ExpiresIn: 2 * time.Hour}
	exch.version = "0.0.34"
	updated, err := svc.Repair(context.Background(), "u1", created.ID, "Home", "https://h.example.com", "tok2")
	require.NoError(t, err)

	assert.Equal(t, created.ID, updated.ID, "re-pair updates the same row, not a new one")
	assert.Equal(t, "0.0.34", updated.HarnessVersion)

	all, err := repo.ListComputers(context.Background(), "u1")
	require.NoError(t, err)
	assert.Len(t, all, 1, "repair must not create a second row")
}

func TestService_Repair_WrongOwnerIsNotFound(t *testing.T) {
	t.Parallel()
	repo := newFakeRepo()
	svc := newTestService(repo, &fakeExchanger{result: harness.PairResult{BearerToken: "b"}, version: "0.0.34"})
	created, err := svc.Pair(context.Background(), "u1", harness.KindT3Code, "Home", "https://h.example.com", "tok")
	require.NoError(t, err)

	_, err = svc.Repair(context.Background(), "u2", created.ID, "Home", "https://h.example.com", "tok2")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}

func TestService_ListComputers_StripsBearerToken(t *testing.T) {
	t.Parallel()
	repo := newFakeRepo()
	svc := newTestService(repo, &fakeExchanger{result: harness.PairResult{BearerToken: "secret"}, version: "0.0.34"})
	_, err := svc.Pair(context.Background(), "u1", harness.KindT3Code, "Home", "https://h.example.com", "tok")
	require.NoError(t, err)

	list, err := svc.ListComputers(context.Background(), "u1")
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Empty(t, list[0].BearerToken)
}

func TestService_ListComputers_RequiresUser(t *testing.T) {
	t.Parallel()
	svc := newTestService(newFakeRepo(), &fakeExchanger{})
	_, err := svc.ListComputers(context.Background(), "")
	require.ErrorIs(t, err, apperrs.ErrUnauthorized)
}

func TestService_DeleteComputer(t *testing.T) {
	t.Parallel()
	repo := newFakeRepo()
	svc := newTestService(repo, &fakeExchanger{result: harness.PairResult{BearerToken: "b"}, version: "0.0.34"})
	c, err := svc.Pair(context.Background(), "u1", harness.KindT3Code, "Home", "https://h.example.com", "tok")
	require.NoError(t, err)

	require.NoError(t, svc.DeleteComputer(context.Background(), "u1", c.ID))

	_, err = repo.GetComputer(context.Background(), "u1", c.ID)
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}

func TestService_ActiveSessions_DecryptsAndSkipsExpired(t *testing.T) {
	t.Parallel()
	repo := newFakeRepo()
	exch := &fakeExchanger{result: harness.PairResult{BearerToken: "bearer-live", ExpiresIn: 30 * 24 * time.Hour}, version: "0.0.34"}
	svc := newTestService(repo, exch)

	live, err := svc.Pair(context.Background(), "u1", harness.KindT3Code, "Live", "https://live.example.com", "tok")
	require.NoError(t, err)
	// A second computer whose session expires immediately.
	exch.result = harness.PairResult{BearerToken: "bearer-dead", ExpiresIn: -time.Hour}
	_, err = svc.Pair(context.Background(), "u1", harness.KindT3Code, "Dead", "https://dead.example.com", "tok")
	require.NoError(t, err)

	sessions, err := svc.ActiveSessions(context.Background(), "u1")
	require.NoError(t, err)
	require.Len(t, sessions, 1, "expired computer skipped")
	assert.Equal(t, live.ID, sessions[0].ID)
	assert.Equal(t, "bearer-live", sessions[0].BearerToken, "token returned decrypted")
}

func TestService_ActiveSessions_RequiresUser(t *testing.T) {
	t.Parallel()
	svc := newTestService(newFakeRepo(), &fakeExchanger{})
	_, err := svc.ActiveSessions(context.Background(), " ")
	require.ErrorIs(t, err, apperrs.ErrUnauthorized)
}

func TestService_OnComputersChanged_FiresOnPairRepairDelete(t *testing.T) {
	t.Parallel()
	repo := newFakeRepo()
	exch := &fakeExchanger{result: harness.PairResult{BearerToken: "b", ExpiresIn: time.Hour}, version: "0.0.34"}
	var changed []string
	svc := NewService(Config{
		Repo:               repo,
		Harnesses:          registry(exch),
		EncryptionKey:      testEncKey,
		OnComputersChanged: func(userID string) { changed = append(changed, userID) },
		Tokens:             newFakeTokens(),
	})

	c, err := svc.Pair(context.Background(), "u1", harness.KindT3Code, "Home", "https://h.example.com", "tok")
	require.NoError(t, err)
	_, err = svc.Repair(context.Background(), "u1", c.ID, "Home", "https://h.example.com", "tok2")
	require.NoError(t, err)
	require.NoError(t, svc.DeleteComputer(context.Background(), "u1", c.ID))

	assert.Equal(t, []string{"u1", "u1", "u1"}, changed)
}

func TestService_DeleteComputer_WrongOwner(t *testing.T) {
	t.Parallel()
	repo := newFakeRepo()
	svc := newTestService(repo, &fakeExchanger{result: harness.PairResult{BearerToken: "b"}, version: "0.0.34"})
	c, err := svc.Pair(context.Background(), "u1", harness.KindT3Code, "Home", "https://h.example.com", "tok")
	require.NoError(t, err)

	err = svc.DeleteComputer(context.Background(), "u2", c.ID)
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}

func TestService_Defaults_RoundTrip(t *testing.T) {
	t.Parallel()
	repo := newFakeRepo()
	svc := newTestService(repo, &fakeExchanger{result: harness.PairResult{BearerToken: "b"}, version: "0.0.34"})
	c, err := svc.Pair(context.Background(), "u1", harness.KindT3Code, "Home", "https://h.example.com", "tok")
	require.NoError(t, err)

	empty, err := svc.GetDefaults(context.Background(), "u1")
	require.NoError(t, err)
	assert.Empty(t, empty.DefaultComputerID, "unset defaults are a zero value, not an error")

	saved, err := svc.SetDefaults(context.Background(), "u1", Defaults{
		DefaultComputerID: c.ID,
		FallbackProjectID: "t3-proj-1",
		Provider:          "claude",
		Model:             "claude-sonnet-4-5",
	})
	require.NoError(t, err)
	assert.Equal(t, c.ID, saved.DefaultComputerID)

	got, err := svc.GetDefaults(context.Background(), "u1")
	require.NoError(t, err)
	assert.Equal(t, saved, got)
}

func TestService_ProjectLink_RoundTrip(t *testing.T) {
	t.Parallel()
	repo := newFakeRepo()
	svc := newTestService(repo, &fakeExchanger{result: harness.PairResult{BearerToken: "b"}, version: "0.0.34"})
	c, err := svc.Pair(context.Background(), "u1", harness.KindT3Code, "Home", "https://h.example.com", "tok")
	require.NoError(t, err)

	empty, err := svc.GetProjectLink(context.Background(), "u1", "proj-1")
	require.NoError(t, err)
	assert.Empty(t, empty.ComputerID, "unlinked project is a zero value, not an error")

	saved, err := svc.SetProjectLink(context.Background(), "u1", "proj-1", ProjectLink{
		ComputerID: c.ID, HarnessProjectID: "t3-proj-1", Provider: "claude", Model: "claude-sonnet-4-5",
	})
	require.NoError(t, err)
	assert.Equal(t, "proj-1", saved.ProjectID)
	assert.Equal(t, c.ID, saved.ComputerID)

	got, err := svc.GetProjectLink(context.Background(), "u1", "proj-1")
	require.NoError(t, err)
	assert.Equal(t, saved.ComputerID, got.ComputerID)
	assert.Equal(t, "t3-proj-1", got.HarnessProjectID)
}

func TestService_SetProjectLink_RejectsForeignComputer(t *testing.T) {
	t.Parallel()
	repo := newFakeRepo()
	svc := newTestService(repo, &fakeExchanger{result: harness.PairResult{BearerToken: "b"}, version: "0.0.34"})
	other, err := svc.Pair(context.Background(), "u2", harness.KindT3Code, "Their box", "https://h.example.com", "tok")
	require.NoError(t, err)

	_, err = svc.SetProjectLink(context.Background(), "u1", "proj-1", ProjectLink{ComputerID: other.ID, HarnessProjectID: "p"})
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}

func TestService_SetProjectLink_RequiresComputerAndProject(t *testing.T) {
	t.Parallel()
	repo := newFakeRepo()
	svc := newTestService(repo, &fakeExchanger{result: harness.PairResult{BearerToken: "b"}, version: "0.0.34"})
	c, err := svc.Pair(context.Background(), "u1", harness.KindT3Code, "Home", "https://h.example.com", "tok")
	require.NoError(t, err)

	_, err = svc.SetProjectLink(context.Background(), "u1", "proj-1", ProjectLink{HarnessProjectID: "p"})
	require.ErrorIs(t, err, apperrs.ErrInvalid, "missing computer")

	_, err = svc.SetProjectLink(context.Background(), "u1", "proj-1", ProjectLink{ComputerID: c.ID})
	require.ErrorIs(t, err, apperrs.ErrInvalid, "missing T3 project id")
}

func TestService_ClearProjectLink(t *testing.T) {
	t.Parallel()
	repo := newFakeRepo()
	svc := newTestService(repo, &fakeExchanger{result: harness.PairResult{BearerToken: "b"}, version: "0.0.34"})
	c, err := svc.Pair(context.Background(), "u1", harness.KindT3Code, "Home", "https://h.example.com", "tok")
	require.NoError(t, err)
	_, err = svc.SetProjectLink(context.Background(), "u1", "proj-1", ProjectLink{ComputerID: c.ID, HarnessProjectID: "t3-proj-1"})
	require.NoError(t, err)

	require.NoError(t, svc.ClearProjectLink(context.Background(), "u1", "proj-1"))

	got, err := svc.GetProjectLink(context.Background(), "u1", "proj-1")
	require.NoError(t, err)
	assert.Empty(t, got.ComputerID)
}

func TestService_ResolveTarget_UsesProjectLinkOverDefaults(t *testing.T) {
	t.Parallel()
	repo := newFakeRepo()
	svc := newTestService(repo, &fakeExchanger{result: harness.PairResult{BearerToken: "secret-token"}, version: "0.0.34"})
	linked, err := svc.Pair(context.Background(), "u1", harness.KindT3Code, "Linked", "https://h.example.com", "tok1")
	require.NoError(t, err)
	fallback, err := svc.Pair(context.Background(), "u1", harness.KindT3Code, "Fallback", "https://f.example.com", "tok2")
	require.NoError(t, err)
	_, err = svc.SetDefaults(context.Background(), "u1", Defaults{
		DefaultComputerID: fallback.ID, FallbackProjectID: "default-proj", Provider: "opencode", Model: "gpt",
	})
	require.NoError(t, err)
	_, err = svc.SetProjectLink(context.Background(), "u1", "proj-1", ProjectLink{
		ComputerID: linked.ID, HarnessProjectID: "linked-proj", Provider: "claude", Model: "sonnet",
	})
	require.NoError(t, err)

	target, err := svc.resolveTarget(context.Background(), "u1", "proj-1")
	require.NoError(t, err)
	assert.Equal(t, linked.ID, target.Computer.ID)
	assert.Equal(t, "linked-proj", target.HarnessProjectID)
	assert.Equal(t, "claude", target.Provider)
	assert.Equal(t, "secret-token", target.Computer.BearerToken, "resolution decrypts the bearer token for internal use")
}

func TestService_ResolveTarget_StartIn_ProjectLinkOverridesDefaults(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		defaults     StartIn
		link         StartIn
		pinned       bool
		wantWorktree bool
	}{
		{"nothing set starts in the folder", "", "", false, false},
		{"defaults ask for a worktree", StartInWorktree, "", false, true},
		{"the link keeps a project in its folder", StartInWorktree, StartInFolder, false, false},
		{"the link asks for a worktree", "", StartInWorktree, false, true},
		{"a run pinned to a computer keeps the link's choice", "", StartInWorktree, true, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			svc := newTestService(newFakeRepo(), &fakeExchanger{result: harness.PairResult{BearerToken: "b"}, version: "0.0.34"})
			c, err := svc.Pair(t.Context(), "u1", harness.KindT3Code, "Home", "https://h.example.com", "tok")
			require.NoError(t, err)
			_, err = svc.SetDefaults(t.Context(), "u1", Defaults{DefaultComputerID: c.ID, FallbackProjectID: "default-proj", StartIn: tt.defaults})
			require.NoError(t, err)
			_, err = svc.SetProjectLink(t.Context(), "u1", "proj-1", ProjectLink{ComputerID: c.ID, HarnessProjectID: "linked-proj", StartIn: tt.link})
			require.NoError(t, err)

			target, err := svc.resolveTarget(t.Context(), "u1", "proj-1")
			if tt.pinned {
				target, err = svc.resolveTargetOverride(t.Context(), "u1", "proj-1", c.ID, modelPick{})
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantWorktree, target.Worktree)
		})
	}
}

func TestService_SetStartIn_RejectsAnUnknownPlace(t *testing.T) {
	t.Parallel()
	svc := newTestService(newFakeRepo(), &fakeExchanger{result: harness.PairResult{BearerToken: "b"}, version: "0.0.34"})
	c, err := svc.Pair(t.Context(), "u1", harness.KindT3Code, "Home", "https://h.example.com", "tok")
	require.NoError(t, err)

	_, err = svc.SetDefaults(t.Context(), "u1", Defaults{StartIn: "docker"})
	require.ErrorIs(t, err, apperrs.ErrInvalid)
	_, err = svc.SetProjectLink(t.Context(), "u1", "proj-1", ProjectLink{ComputerID: c.ID, HarnessProjectID: "p", StartIn: "docker"})
	require.ErrorIs(t, err, apperrs.ErrInvalid)
}

func TestService_ResolveTarget_FallsBackToUserDefaults(t *testing.T) {
	t.Parallel()
	repo := newFakeRepo()
	svc := newTestService(repo, &fakeExchanger{result: harness.PairResult{BearerToken: "b"}, version: "0.0.34"})
	fallback, err := svc.Pair(context.Background(), "u1", harness.KindT3Code, "Fallback", "https://f.example.com", "tok")
	require.NoError(t, err)
	_, err = svc.SetDefaults(context.Background(), "u1", Defaults{
		DefaultComputerID: fallback.ID, FallbackProjectID: "default-proj",
	})
	require.NoError(t, err)

	target, err := svc.resolveTarget(context.Background(), "u1", "")
	require.NoError(t, err)
	assert.Equal(t, fallback.ID, target.Computer.ID)
	assert.Equal(t, "default-proj", target.HarnessProjectID)
}

func TestService_ResolvePersonRun_AsksWhereOnceThenUsesTheLink(t *testing.T) {
	t.Parallel()
	exch := &fakeExchanger{result: harness.PairResult{BearerToken: "b"}, version: "0.0.34"}
	exch.ListProvidersFn = func(context.Context, harness.Session) ([]harness.Provider, error) {
		return []harness.Provider{{ID: "claude", Driver: "claude", Name: "Provider"}}, nil
	}
	exch.ListProjectsFn = listsT3Projects("t3-app", "t3-web")
	svc := newTestService(newFakeRepo(), exch)
	ctx := t.Context()
	home, err := svc.Pair(ctx, "u1", harness.KindT3Code, "Home", "https://h.example.com", "tok")
	require.NoError(t, err)
	_, err = svc.ConfirmSetup(ctx, "u1", home.ID)
	require.NoError(t, err)
	_, err = svc.ConfirmProviderSetup(ctx, "u1", home.ID, "claude", []string{"tdd"})
	require.NoError(t, err)
	_, err = svc.SetDefaults(ctx, "u1", Defaults{DefaultComputerID: home.ID, FallbackProjectID: "default-proj", Provider: "claude", Model: "sonnet"})
	require.NoError(t, err)

	_, err = svc.ResolvePersonRun(ctx, "u1", "proj-1", "", "", "", "", nil)
	var nc *NotConfiguredError
	require.ErrorAs(t, err, &nc, "the defaults would resolve, but a person's run never falls back to them")
	assert.Equal(t, ReasonNeedsLocation, nc.Reason)
	require.ErrorIs(t, err, apperrs.ErrInvalid)

	target, err := svc.ResolvePersonRun(ctx, "u1", "proj-1", home.ID, "t3-app", "claude", "opus", nil)
	require.NoError(t, err)
	assert.Equal(t, "t3-app", target.HarnessProjectID)
	assert.Equal(t, "opus", target.Model, "the run takes the model picked for it")
	link, err := svc.GetProjectLink(ctx, "u1", "proj-1")
	require.NoError(t, err)
	assert.Equal(t, home.ID, link.ComputerID, "the answer is saved as their link")
	assert.Equal(t, "t3-app", link.HarnessProjectID)
	assert.Equal(t, "claude/sonnet", link.Provider+"/"+link.Model, "the model was for this run only; the link takes their default")

	_, err = svc.ResolvePersonRun(ctx, "u1", "proj-1", home.ID, "t3-web", "claude", "haiku", nil)
	require.NoError(t, err)
	link, err = svc.GetProjectLink(ctx, "u1", "proj-1")
	require.NoError(t, err)
	assert.Equal(t, "t3-web", link.HarnessProjectID, "Change saves where")
	assert.Equal(t, "claude/sonnet", link.Provider+"/"+link.Model, "and keeps the link's model")

	target, err = svc.ResolvePersonRun(ctx, "u1", "proj-1", "", "", "", "", nil)
	require.NoError(t, err, "a linked project runs without asking")
	assert.Equal(t, home.ID, target.Computer.ID)
	assert.Equal(t, "t3-web", target.HarnessProjectID)

	require.NoError(t, svc.ClearProjectLink(ctx, "u1", "proj-1"))
	_, err = svc.ResolvePersonRun(ctx, "u1", "proj-1", "", "", "", "", nil)
	require.ErrorAs(t, err, &nc)
	assert.Equal(t, ReasonNeedsLocation, nc.Reason)
}

func TestService_ResolvePersonRun_Where(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name              string
		otherPicked       bool
		replacedAfterSave bool
		harnessProj       string
		wantReason        NotConfiguredReason
		wantInvalid       bool
		wantStartIn       StartIn
		wantHarnProj      string
	}{
		{name: "another computer without its T3 project asks again", otherPicked: true, wantReason: ReasonNeedsLocation},
		{name: "a T3 project without its computer is refused", harnessProj: "t3-other", wantInvalid: true},
		{name: "changing where keeps the link's start-in", otherPicked: true, harnessProj: "t3-other", wantStartIn: StartInWorktree, wantHarnProj: "t3-other"},
		{name: "another tab replaces the link while the pick is saved", otherPicked: true, harnessProj: "t3-other", replacedAfterSave: true, wantStartIn: StartInFolder, wantHarnProj: "t3-other"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			exch := &fakeExchanger{result: harness.PairResult{BearerToken: "b"}, version: "0.0.34"}
			exch.ListProvidersFn = func(context.Context, harness.Session) ([]harness.Provider, error) {
				return []harness.Provider{{ID: "claude", Driver: "claude", Name: "Provider"}}, nil
			}
			exch.ListProjectsFn = listsT3Projects("t3-other")
			svc := newTestService(newFakeRepo(), exch)
			ctx := t.Context()
			home, err := svc.Pair(ctx, "u1", harness.KindT3Code, "Home", "https://h.example.com", "tok")
			require.NoError(t, err)
			other, err := svc.Pair(ctx, "u1", harness.KindT3Code, "Other", "https://o.example.com", "tok")
			require.NoError(t, err)
			_, err = svc.ConfirmProviderSetup(ctx, "u1", other.ID, "claude", []string{"tdd"})
			require.NoError(t, err)
			_, err = svc.ConfirmSetup(ctx, "u1", other.ID)
			require.NoError(t, err)
			_, err = svc.SetProjectLink(ctx, "u1", "proj-1", ProjectLink{ComputerID: home.ID, HarnessProjectID: "t3-app", StartIn: StartInWorktree})
			require.NoError(t, err)
			computerID := ""
			if tt.otherPicked {
				computerID = other.ID
			}

			if tt.replacedAfterSave {
				svc.repo = replacedRunLocationRepo{svc.repo}
			}

			target, err := svc.ResolvePersonRun(ctx, "u1", "proj-1", computerID, tt.harnessProj, "claude", "", nil)

			if tt.wantInvalid {
				require.ErrorIs(t, err, apperrs.ErrInvalid)
				return
			}
			var nc *NotConfiguredError
			if tt.wantReason != "" {
				require.ErrorAs(t, err, &nc)
				assert.Equal(t, tt.wantReason, nc.Reason)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantHarnProj, target.HarnessProjectID)
			if tt.replacedAfterSave {
				assert.True(t, target.Worktree, "the run keeps the start-in captured before the competing save")
				assert.Empty(t, target.Model, "the competing model is not this run's choice")
			}
			link, err := svc.GetProjectLink(ctx, "u1", "proj-1")
			require.NoError(t, err)
			assert.Equal(t, other.ID, link.ComputerID)
			assert.Equal(t, tt.wantStartIn, link.StartIn)
		})
	}
}

// TestService_ProjectLink_EachPersonResolvesTheirOwn guards ADR 0102: a teammate's link never sends someone else's
// turns to the teammate's computer, and two people's links for one project sit side by side.
func TestService_ProjectLink_EachPersonResolvesTheirOwn(t *testing.T) {
	t.Parallel()
	repo := newFakeRepo()
	svc := newTestService(repo, &fakeExchanger{result: harness.PairResult{BearerToken: "b"}, version: "0.0.34"})
	ctx := t.Context()
	theirs, err := svc.Pair(ctx, "u2", harness.KindT3Code, "Their box", "https://h.example.com", "tok")
	require.NoError(t, err)
	mine, err := svc.Pair(ctx, "u1", harness.KindT3Code, "My box", "https://m.example.com", "tok")
	require.NoError(t, err)
	_, err = svc.SetDefaults(ctx, "u1", Defaults{DefaultComputerID: mine.ID, FallbackProjectID: "my-fallback"})
	require.NoError(t, err)
	_, err = svc.SetProjectLink(ctx, "u2", "proj-1", ProjectLink{ComputerID: theirs.ID, HarnessProjectID: "their-proj"})
	require.NoError(t, err)

	target, err := svc.resolveTarget(ctx, "u1", "proj-1")
	require.NoError(t, err)
	assert.Equal(t, mine.ID, target.Computer.ID, "with no link of their own, a person falls back to their own defaults")
	assert.Equal(t, "my-fallback", target.HarnessProjectID)
	got, err := svc.GetProjectLink(ctx, "u1", "proj-1")
	require.NoError(t, err)
	assert.Empty(t, got.ComputerID, "a teammate's link is not read back as theirs")

	_, err = svc.SetProjectLink(ctx, "u1", "proj-1", ProjectLink{ComputerID: mine.ID, HarnessProjectID: "my-proj"})
	require.NoError(t, err)
	target, err = svc.resolveTarget(ctx, "u2", "proj-1")
	require.NoError(t, err)
	assert.Equal(t, theirs.ID, target.Computer.ID, "setting one person's link leaves the other's in place")
	assert.Equal(t, "their-proj", target.HarnessProjectID)
}

// TestService_ProjectLink_HiddenProjectIsNotFound guards ADR 0097 on the link routes: a project the caller can't open
// reads, writes, and clears as not found, and nothing is stored.
func TestService_ProjectLink_HiddenProjectIsNotFound(t *testing.T) {
	t.Parallel()
	repo := newFakeRepo()
	svc := NewService(Config{Repo: repo, Harnesses: registry(&fakeExchanger{result: harness.PairResult{BearerToken: "b"}, version: "0.0.34"}),
		EncryptionKey: testEncKey, Projects: fakeProjects{hidden: []string{"proj-hidden"}}})
	c, err := svc.Pair(t.Context(), "u1", harness.KindT3Code, "Home", "https://h.example.com", "tok")
	require.NoError(t, err)

	calls := map[string]func() error{
		"get": func() error { _, err := svc.GetProjectLink(t.Context(), "u1", "proj-hidden"); return err },
		"set": func() error {
			_, err := svc.SetProjectLink(t.Context(), "u1", "proj-hidden", ProjectLink{ComputerID: c.ID, HarnessProjectID: "p"})
			return err
		},
		"clear": func() error { return svc.ClearProjectLink(t.Context(), "u1", "proj-hidden") },
	}
	for name, call := range calls {
		t.Run(name, func(t *testing.T) {
			require.ErrorIs(t, call(), apperrs.ErrNotFound)
		})
	}
	assert.Empty(t, repo.projectLinks)
}

// TestService_ListProjectLinks_OwnLinksOnProjectsTheyCanOpen guards the Projects tab's read: a teammate's links and a
// link on a project since hidden from the caller stay out.
func TestService_ListProjectLinks_OwnLinksOnProjectsTheyCanOpen(t *testing.T) {
	t.Parallel()
	repo := newFakeRepo()
	svc := NewService(Config{Repo: repo, Projects: fakeProjects{hidden: []string{"proj-hidden"}}})
	for _, link := range []ProjectLink{
		{UserID: "u1", ProjectID: "proj-2", ComputerID: "c1", HarnessProjectID: "p2"},
		{UserID: "u1", ProjectID: "proj-1", ComputerID: "c1", HarnessProjectID: "p1"},
		{UserID: "u1", ProjectID: "proj-hidden", ComputerID: "c1", HarnessProjectID: "ph"},
		{UserID: "u2", ProjectID: "proj-1", ComputerID: "c2", HarnessProjectID: "theirs"},
	} {
		require.NoError(t, repo.SaveProjectLink(t.Context(), link))
	}

	links, err := svc.ListProjectLinks(t.Context(), "u1")
	require.NoError(t, err)
	require.Len(t, links, 2)
	assert.Equal(t, []string{"proj-1", "proj-2"}, []string{links[0].ProjectID, links[1].ProjectID})
	assert.Equal(t, "p1", links[0].HarnessProjectID)
}

func TestService_ResolveTarget_NotConfiguredReasons(t *testing.T) {
	t.Parallel()

	t.Run("unpaired: nothing configured at all", func(t *testing.T) {
		t.Parallel()
		svc := newTestService(newFakeRepo(), &fakeExchanger{})
		_, err := svc.ResolveTarget(context.Background(), "u1", "")
		var nc *NotConfiguredError
		require.ErrorAs(t, err, &nc)
		assert.Equal(t, ReasonUnpaired, nc.Reason)
	})

	t.Run("a single paired computer is used implicitly without defaults", func(t *testing.T) {
		t.Parallel()
		repo := newFakeRepo()
		svc := newTestService(repo, &fakeExchanger{result: harness.PairResult{BearerToken: "b"}, version: "0.0.34"})
		_, err := svc.Pair(context.Background(), "u1", harness.KindT3Code, "Home", "https://h.example.com", "tok")
		require.NoError(t, err)
		_, err = svc.ResolveTarget(context.Background(), "u1", "")
		var nc *NotConfiguredError
		require.ErrorAs(t, err, &nc)
		assert.Equal(t, ReasonNoDefault, nc.Reason, "the computer must resolve implicitly — only the T3 project is missing, never unpaired")
	})

	t.Run("several computers without a default ask for one", func(t *testing.T) {
		t.Parallel()
		repo := newFakeRepo()
		svc := newTestService(repo, &fakeExchanger{result: harness.PairResult{BearerToken: "b"}, version: "0.0.34"})
		_, err := svc.Pair(context.Background(), "u1", harness.KindT3Code, "Home", "https://h.example.com", "tok")
		require.NoError(t, err)
		_, err = svc.Pair(context.Background(), "u1", harness.KindT3Code, "VPS", "https://v.example.com", "tok2")
		require.NoError(t, err)
		_, err = svc.ResolveTarget(context.Background(), "u1", "")
		var nc *NotConfiguredError
		require.ErrorAs(t, err, &nc)
		assert.Equal(t, ReasonNoDefaultComputer, nc.Reason)
	})

	t.Run("no_default: computer resolves but no T3 project", func(t *testing.T) {
		t.Parallel()
		repo := newFakeRepo()
		svc := newTestService(repo, &fakeExchanger{result: harness.PairResult{BearerToken: "b"}, version: "0.0.34"})
		c, err := svc.Pair(context.Background(), "u1", harness.KindT3Code, "Home", "https://h.example.com", "tok")
		require.NoError(t, err)
		_, err = svc.SetDefaults(context.Background(), "u1", Defaults{DefaultComputerID: c.ID})
		require.NoError(t, err)

		_, err = svc.ResolveTarget(context.Background(), "u1", "")
		var nc *NotConfiguredError
		require.ErrorAs(t, err, &nc)
		assert.Equal(t, ReasonNoDefault, nc.Reason)
	})

	t.Run("expired_token: computer resolves but token is expired", func(t *testing.T) {
		t.Parallel()
		repo := newFakeRepo()
		exch := &fakeExchanger{result: harness.PairResult{BearerToken: "b", ExpiresIn: time.Hour}, version: "0.0.34"}
		svc := newTestService(repo, exch)
		c, err := svc.Pair(context.Background(), "u1", harness.KindT3Code, "Home", "https://h.example.com", "tok")
		require.NoError(t, err)
		_, err = svc.SetDefaults(context.Background(), "u1", Defaults{DefaultComputerID: c.ID, FallbackProjectID: "p"})
		require.NoError(t, err)

		// newTestService's clock is fixed at 2026-08-26T12:00:00Z; Pair set
		// expiry to +1h from that instant, so a later service with a clock
		// further ahead sees it as expired.
		later := NewService(Config{
			Repo: repo, Harnesses: registry(exch), EncryptionKey: testEncKey,
			Now: func() time.Time { return time.Date(2026, 8, 26, 14, 0, 0, 0, time.UTC) },
		})
		_, err = later.ResolveTarget(context.Background(), "u1", "")
		var nc *NotConfiguredError
		require.ErrorAs(t, err, &nc)
		assert.Equal(t, ReasonExpiredToken, nc.Reason)
	})

	t.Run("dangling default computer id is unpaired", func(t *testing.T) {
		t.Parallel()
		repo := newFakeRepo()
		svc := newTestService(repo, &fakeExchanger{})
		require.NoError(t, repo.SaveDefaults(context.Background(), Defaults{UserID: "u1", DefaultComputerID: "gone", FallbackProjectID: "p"}))

		_, err := svc.ResolveTarget(context.Background(), "u1", "")
		var nc *NotConfiguredError
		require.ErrorAs(t, err, &nc)
		assert.Equal(t, ReasonUnpaired, nc.Reason)
	})
}

func TestService_ResolveTarget_RequiresUser(t *testing.T) {
	t.Parallel()
	svc := newTestService(newFakeRepo(), &fakeExchanger{})
	_, err := svc.ResolveTarget(context.Background(), "", "proj-1")
	require.ErrorIs(t, err, apperrs.ErrUnauthorized)
}

func TestService_ResolveTargetOverride_EmptyComputerFallsBackToResolveTarget(t *testing.T) {
	t.Parallel()
	repo := newFakeRepo()
	svc := newTestService(repo, &fakeExchanger{result: harness.PairResult{BearerToken: "b"}, version: "0.0.34"})
	c, err := svc.Pair(context.Background(), "u1", harness.KindT3Code, "Home", "https://h.example.com", "tok")
	require.NoError(t, err)
	_, err = svc.SetDefaults(context.Background(), "u1", Defaults{DefaultComputerID: c.ID, FallbackProjectID: "p", Provider: "opencode", Model: "gpt"})
	require.NoError(t, err)

	target, err := svc.resolveTargetOverride(context.Background(), "u1", "", "", modelPick{})
	require.NoError(t, err)
	assert.Equal(t, c.ID, target.Computer.ID)
	assert.Equal(t, "opencode", target.Provider)
}

func TestService_ResolveTargetOverride_PinnedComputerWithExplicitProviderAndModel(t *testing.T) {
	t.Parallel()
	repo := newFakeRepo()
	svc := newTestService(repo, &fakeExchanger{result: harness.PairResult{BearerToken: "secret"}, version: "0.0.34"})
	c, err := svc.Pair(context.Background(), "u1", harness.KindT3Code, "Home", "https://h.example.com", "tok")
	require.NoError(t, err)
	_, err = svc.SetProjectLink(context.Background(), "u1", "proj-1", ProjectLink{
		ComputerID: c.ID, HarnessProjectID: "linked-proj", Model: "opus", ModelOptions: []harness.OptionSetting{{ID: "effort", Value: "low"}},
	})
	require.NoError(t, err)

	picked := []harness.OptionSetting{{ID: "fastMode", Value: true}}
	target, err := svc.resolveTargetOverride(context.Background(), "u1", "proj-1", c.ID, modelPick{"claude", "sonnet-5", picked})
	require.NoError(t, err)
	assert.Equal(t, c.ID, target.Computer.ID)
	assert.Equal(t, "linked-proj", target.HarnessProjectID)
	assert.Equal(t, "claude", target.Provider)
	assert.Equal(t, "sonnet-5", target.Model)
	assert.Equal(t, picked, target.ModelOptions, "the link's options belong to its own model, never to the picked one")
	assert.Equal(t, "secret", target.Computer.BearerToken)
}

func TestService_ResolveTargetOverride_BlankProviderAndModelFillFromTheMatchingProjectLink(t *testing.T) {
	t.Parallel()
	repo := newFakeRepo()
	svc := newTestService(repo, &fakeExchanger{result: harness.PairResult{BearerToken: "b"}, version: "0.0.34"})
	c, err := svc.Pair(context.Background(), "u1", harness.KindT3Code, "Home", "https://h.example.com", "tok")
	require.NoError(t, err)
	_, err = svc.SetProjectLink(context.Background(), "u1", "proj-1", ProjectLink{
		ComputerID: c.ID, HarnessProjectID: "linked-proj", Provider: "claude", Model: "sonnet", ModelOptions: []harness.OptionSetting{{ID: "effort", Value: "high"}},
	})
	require.NoError(t, err)

	target, err := svc.resolveTargetOverride(context.Background(), "u1", "proj-1", c.ID, modelPick{})
	require.NoError(t, err)
	assert.Equal(t, "linked-proj", target.HarnessProjectID)
	assert.Equal(t, "claude", target.Provider)
	assert.Equal(t, "sonnet", target.Model)
	assert.Equal(t, []harness.OptionSetting{{ID: "effort", Value: "high"}}, target.ModelOptions, "a filled-in model brings its options")
}

func TestService_ResolveTargetOverride_BlankFieldsFillFromDefaultsWhenTheComputerIsTheirDefault(t *testing.T) {
	t.Parallel()
	repo := newFakeRepo()
	svc := newTestService(repo, &fakeExchanger{result: harness.PairResult{BearerToken: "b"}, version: "0.0.34"})
	c, err := svc.Pair(context.Background(), "u1", harness.KindT3Code, "Home", "https://h.example.com", "tok")
	require.NoError(t, err)
	_, err = svc.SetDefaults(context.Background(), "u1", Defaults{DefaultComputerID: c.ID, FallbackProjectID: "default-proj", Provider: "opencode", Model: "gpt"})
	require.NoError(t, err)

	target, err := svc.resolveTargetOverride(context.Background(), "u1", "", c.ID, modelPick{})
	require.NoError(t, err)
	assert.Equal(t, "default-proj", target.HarnessProjectID)
	assert.Equal(t, "opencode", target.Provider)
	assert.Equal(t, "gpt", target.Model)
}

func TestService_ResolveTargetOverride_NoProjectResolvesIsNoDefault(t *testing.T) {
	t.Parallel()
	repo := newFakeRepo()
	svc := newTestService(repo, &fakeExchanger{result: harness.PairResult{BearerToken: "b"}, version: "0.0.34"})
	c, err := svc.Pair(context.Background(), "u1", harness.KindT3Code, "Home", "https://h.example.com", "tok")
	require.NoError(t, err)

	_, err = svc.ResolveTargetOverride(context.Background(), "u1", "", c.ID, "claude", "sonnet", nil)
	var nc *NotConfiguredError
	require.ErrorAs(t, err, &nc)
	assert.Equal(t, ReasonNoDefault, nc.Reason)
}

func TestService_ResolveTargetOverride_RefusesAComputerNotOwnedByTheCaller(t *testing.T) {
	t.Parallel()
	repo := newFakeRepo()
	svc := newTestService(repo, &fakeExchanger{result: harness.PairResult{BearerToken: "b"}, version: "0.0.34"})
	theirs, err := svc.Pair(context.Background(), "u2", harness.KindT3Code, "Their box", "https://h.example.com", "tok")
	require.NoError(t, err)

	_, err = svc.ResolveTargetOverride(context.Background(), "u1", "", theirs.ID, "claude", "sonnet", nil)
	var nc *NotConfiguredError
	require.ErrorAs(t, err, &nc)
	assert.Equal(t, ReasonUnpaired, nc.Reason)
}

func TestService_ResolveTargetOverride_RequiresUser(t *testing.T) {
	t.Parallel()
	svc := newTestService(newFakeRepo(), &fakeExchanger{})
	_, err := svc.ResolveTargetOverride(context.Background(), "", "proj-1", "c-1", "", "", nil)
	require.ErrorIs(t, err, apperrs.ErrUnauthorized)
}

func TestService_SetDefaults_RejectsForeignComputer(t *testing.T) {
	t.Parallel()
	repo := newFakeRepo()
	svc := newTestService(repo, &fakeExchanger{result: harness.PairResult{BearerToken: "b"}, version: "0.0.34"})
	other, err := svc.Pair(context.Background(), "u2", harness.KindT3Code, "Their box", "https://h.example.com", "tok")
	require.NoError(t, err)

	_, err = svc.SetDefaults(context.Background(), "u1", Defaults{DefaultComputerID: other.ID})
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}

type replacedRunLocationRepo struct{ Repo }

func (r replacedRunLocationRepo) SaveProjectLink(ctx context.Context, link ProjectLink) error {
	if err := r.Repo.SaveProjectLink(ctx, link); err != nil {
		return err
	}
	link.HarnessProjectID, link.Model, link.StartIn = "t3-replaced", "other-model", StartInFolder
	return r.Repo.SaveProjectLink(ctx, link)
}

func listsT3Projects(ids ...string) func(context.Context, harness.Session) ([]harness.Project, error) {
	return func(context.Context, harness.Session) ([]harness.Project, error) {
		projects := make([]harness.Project, 0, len(ids))
		for _, id := range ids {
			projects = append(projects, harness.Project{ID: id, Title: id})
		}
		return projects, nil
	}
}

func TestService_ResolvePersonRun_AT3ProjectTheComputerDoesNotList_IsRefusedAndNotSaved(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		list       func(context.Context, harness.Session) ([]harness.Project, error)
		wantReason NotConfiguredReason
	}{
		{name: "not one of the computer's T3 projects", list: listsT3Projects("t3-app")},
		{name: "the computer cannot say", list: func(context.Context, harness.Session) ([]harness.Project, error) {
			return nil, errBoom
		}, wantReason: ReasonOffline},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			exch := &fakeExchanger{result: harness.PairResult{BearerToken: "b"}, version: "0.0.34"}
			exch.ListProjectsFn = tt.list
			svc := newTestService(newFakeRepo(), exch)
			ctx := t.Context()
			home, err := svc.Pair(ctx, "u1", harness.KindT3Code, "Home", "https://h.example.com", "tok")
			require.NoError(t, err)

			_, err = svc.ResolvePersonRun(ctx, "u1", "proj-1", home.ID, "t3-typo", "", "", nil)

			require.ErrorIs(t, err, apperrs.ErrInvalid)
			var nc *NotConfiguredError
			if tt.wantReason != "" {
				require.ErrorAs(t, err, &nc)
				assert.Equal(t, tt.wantReason, nc.Reason)
			}
			link, err := svc.GetProjectLink(ctx, "u1", "proj-1")
			require.NoError(t, err)
			assert.Empty(t, link.ComputerID, "an unchecked T3 project is never saved as the link")
		})
	}
}
