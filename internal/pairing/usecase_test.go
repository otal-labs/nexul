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

func TestService_ResolveTarget_ProjectLinkedComputerNotOwnedByCaller(t *testing.T) {
	t.Parallel()
	repo := newFakeRepo()
	svc := newTestService(repo, &fakeExchanger{result: harness.PairResult{BearerToken: "b"}, version: "0.0.34"})
	// u2 owns and links the computer to proj-1; u1 (a teammate) mentions @Agent there.
	linked, err := svc.Pair(context.Background(), "u2", harness.KindT3Code, "Shared", "https://h.example.com", "tok")
	require.NoError(t, err)
	_, err = svc.SetProjectLink(context.Background(), "u2", "proj-1", ProjectLink{ComputerID: linked.ID, HarnessProjectID: "proj"})
	require.NoError(t, err)

	target, err := svc.resolveTarget(context.Background(), "u1", "proj-1")
	require.NoError(t, err, "a project-linked computer resolves for any mentioning user, not just its owner")
	assert.Equal(t, linked.ID, target.Computer.ID)
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

	target, err := svc.resolveTargetOverride(context.Background(), "u1", "", "", "", "")
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
	_, err = svc.SetProjectLink(context.Background(), "u1", "proj-1", ProjectLink{ComputerID: c.ID, HarnessProjectID: "linked-proj"})
	require.NoError(t, err)

	target, err := svc.resolveTargetOverride(context.Background(), "u1", "proj-1", c.ID, "claude", "sonnet-5")
	require.NoError(t, err)
	assert.Equal(t, c.ID, target.Computer.ID)
	assert.Equal(t, "linked-proj", target.HarnessProjectID)
	assert.Equal(t, "claude", target.Provider)
	assert.Equal(t, "sonnet-5", target.Model)
	assert.Equal(t, "secret", target.Computer.BearerToken)
}

func TestService_ResolveTargetOverride_BlankProviderAndModelFillFromTheMatchingProjectLink(t *testing.T) {
	t.Parallel()
	repo := newFakeRepo()
	svc := newTestService(repo, &fakeExchanger{result: harness.PairResult{BearerToken: "b"}, version: "0.0.34"})
	c, err := svc.Pair(context.Background(), "u1", harness.KindT3Code, "Home", "https://h.example.com", "tok")
	require.NoError(t, err)
	_, err = svc.SetProjectLink(context.Background(), "u1", "proj-1", ProjectLink{ComputerID: c.ID, HarnessProjectID: "linked-proj", Provider: "claude", Model: "sonnet"})
	require.NoError(t, err)

	target, err := svc.resolveTargetOverride(context.Background(), "u1", "proj-1", c.ID, "", "")
	require.NoError(t, err)
	assert.Equal(t, "linked-proj", target.HarnessProjectID)
	assert.Equal(t, "claude", target.Provider)
	assert.Equal(t, "sonnet", target.Model)
}

func TestService_ResolveTargetOverride_BlankFieldsFillFromDefaultsWhenTheComputerIsTheirDefault(t *testing.T) {
	t.Parallel()
	repo := newFakeRepo()
	svc := newTestService(repo, &fakeExchanger{result: harness.PairResult{BearerToken: "b"}, version: "0.0.34"})
	c, err := svc.Pair(context.Background(), "u1", harness.KindT3Code, "Home", "https://h.example.com", "tok")
	require.NoError(t, err)
	_, err = svc.SetDefaults(context.Background(), "u1", Defaults{DefaultComputerID: c.ID, FallbackProjectID: "default-proj", Provider: "opencode", Model: "gpt"})
	require.NoError(t, err)

	target, err := svc.resolveTargetOverride(context.Background(), "u1", "", c.ID, "", "")
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

	_, err = svc.ResolveTargetOverride(context.Background(), "u1", "", c.ID, "claude", "sonnet")
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

	_, err = svc.ResolveTargetOverride(context.Background(), "u1", "", theirs.ID, "claude", "sonnet")
	var nc *NotConfiguredError
	require.ErrorAs(t, err, &nc)
	assert.Equal(t, ReasonUnpaired, nc.Reason)
}

func TestService_ResolveTargetOverride_RequiresUser(t *testing.T) {
	t.Parallel()
	svc := newTestService(newFakeRepo(), &fakeExchanger{})
	_, err := svc.ResolveTargetOverride(context.Background(), "", "proj-1", "c-1", "", "")
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
