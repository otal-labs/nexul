package pairing

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/harness"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/ids"
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

	c, err := pairAt(svc, context.Background(), "u1", harness.KindT3Code, "  Home  ", "https://home.example.com", "one-time-token")
	require.NoError(t, err)
	assert.Equal(t, "Home", c.Name)
	assert.Equal(t, "https://home.example.com", c.ServerURL)
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
		{"missing name", "u1", "  ", "https://h.example.com", "tok"},
		{"missing token", "u1", "Home", "https://h.example.com", "  "},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := pairAt(svc, context.Background(), tt.userID, harness.KindT3Code, tt.compName, tt.serverURL, tt.token)
			require.Error(t, err)
		})
	}
}

func TestService_Pair_ExchangeError(t *testing.T) {
	t.Parallel()
	svc := newTestService(newFakeRepo(), &fakeExchanger{exchangeErr: errBoom})
	_, err := pairAt(svc, context.Background(), "u1", harness.KindT3Code, "Home", "https://h.example.com", "tok")
	require.ErrorIs(t, err, errBoom)
}

func TestService_Pair_VersionError(t *testing.T) {
	t.Parallel()
	svc := newTestService(newFakeRepo(), &fakeExchanger{result: harness.PairResult{BearerToken: "b"}, versionErr: errBoom})
	_, err := pairAt(svc, context.Background(), "u1", harness.KindT3Code, "Home", "https://h.example.com", "tok")
	require.ErrorIs(t, err, errBoom)
}

func TestService_Pair_NoEncryptionKey(t *testing.T) {
	t.Parallel()
	svc := NewService(Config{Repo: newFakeRepo(), Harnesses: registry(&fakeExchanger{result: harness.PairResult{BearerToken: "b"}, version: "0.0.34"})})
	_, err := pairAt(svc, context.Background(), "u1", harness.KindT3Code, "Home", "https://h.example.com", "tok")
	require.ErrorIs(t, err, apperrs.ErrFatal)
}

func TestService_ListComputers_StripsBearerToken(t *testing.T) {
	t.Parallel()
	repo := newFakeRepo()
	svc := newTestService(repo, &fakeExchanger{result: harness.PairResult{BearerToken: "secret"}, version: "0.0.34"})
	_, err := pairAt(svc, context.Background(), "u1", harness.KindT3Code, "Home", "https://h.example.com", "tok")
	require.NoError(t, err)

	list, err := svc.ListComputers(context.Background(), "u1")
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Empty(t, list[0].BearerToken)
}

func TestService_DeleteComputer(t *testing.T) {
	t.Parallel()
	repo := newFakeRepo()
	svc := newTestService(repo, &fakeExchanger{result: harness.PairResult{BearerToken: "b"}, version: "0.0.34"})
	c, err := pairAt(svc, context.Background(), "u1", harness.KindT3Code, "Home", "https://h.example.com", "tok")
	require.NoError(t, err)

	require.NoError(t, svc.DeleteComputer(context.Background(), "u1", c.ID))

	_, err = repo.GetComputer(context.Background(), "u1", c.ID)
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}

func TestService_ActiveSessions_DecryptsAndSkipsExpiredAndUnmoved(t *testing.T) {
	t.Parallel()
	repo := newFakeRepo()
	exch := &fakeExchanger{result: harness.PairResult{BearerToken: "bearer-live", ExpiresIn: 30 * 24 * time.Hour}, version: "0.0.34"}
	svc := newTestService(repo, exch)

	onRunner := func(name string) *Computer {
		id := ids.New()
		c, err := svc.pair(context.Background(), Computer{ID: id, UserID: "u1", Kind: harness.KindT3Code, Name: name}, runnerAddress(id), "tok")
		require.NoError(t, err)
		return c
	}
	live := onRunner("Live")
	_, err := pairAt(svc, context.Background(), "u1", harness.KindT3Code, "Unmoved", "https://unmoved.example.com", "tok")
	require.NoError(t, err)
	// A third computer whose session expires immediately.
	exch.result = harness.PairResult{BearerToken: "bearer-dead", ExpiresIn: -time.Hour}
	onRunner("Dead")

	sessions, err := svc.ActiveSessions(context.Background(), "u1")
	require.NoError(t, err)
	require.Len(t, sessions, 1, "the expired computer and the one paired before the Nexul app hold nothing open")
	assert.Equal(t, live.ID, sessions[0].ID)
	assert.Equal(t, "bearer-live", sessions[0].BearerToken, "token returned decrypted")
}

func TestService_OnComputersChanged_FiresOnPairAndDelete(t *testing.T) {
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

	c, err := pairAt(svc, context.Background(), "u1", harness.KindT3Code, "Home", "https://h.example.com", "tok")
	require.NoError(t, err)
	require.NoError(t, svc.DeleteComputer(context.Background(), "u1", c.ID))

	assert.Equal(t, []string{"u1", "u1"}, changed)
}

func TestService_DeleteComputer_WrongOwner(t *testing.T) {
	t.Parallel()
	repo := newFakeRepo()
	svc := newTestService(repo, &fakeExchanger{result: harness.PairResult{BearerToken: "b"}, version: "0.0.34"})
	c, err := pairAt(svc, context.Background(), "u1", harness.KindT3Code, "Home", "https://h.example.com", "tok")
	require.NoError(t, err)

	err = svc.DeleteComputer(context.Background(), "u2", c.ID)
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}

func TestService_ReadsAndResolves_RequireAUser(t *testing.T) {
	t.Parallel()
	svc := newTestService(newFakeRepo(), &fakeExchanger{})
	ctx := context.Background()
	calls := map[string]func() error{
		"ListComputers":         func() error { _, err := svc.ListComputers(ctx, ""); return err },
		"ActiveSessions":        func() error { _, err := svc.ActiveSessions(ctx, " "); return err },
		"ResolveTarget":         func() error { _, err := svc.ResolveTarget(ctx, "", "proj-1"); return err },
		"ResolveTargetOverride": func() error { _, err := svc.ResolveTargetOverride(ctx, "", "proj-1", "c-1", "", "", nil); return err },
	}
	for name, call := range calls {
		t.Run(name, func(t *testing.T) {
			require.ErrorIs(t, call(), apperrs.ErrUnauthorized)
		})
	}
}
