package storage

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/auth"
	"github.com/otal-labs/nexul/internal/integrations"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

type holdsEverywhere struct{}

func (holdsEverywhere) HoldsAnywhere(context.Context, string, permissions.Action) (bool, error) {
	return true, nil
}

// statementsPerCall runs call 100 times and returns the first call's cost and the cost of each later call.
func statementsPerCall(t *testing.T, st interface {
	Count() int64
	Reset()
}, call func()) (first, rest int64) {
	t.Helper()
	st.Reset()
	call()
	first = st.Count()
	st.Reset()
	for range 99 {
		call()
	}
	t.Logf("first call: %d statements, next 99 calls: %d statements", first, st.Count())
	return first, st.Count()
}

func TestAuthenticatePAT_TouchesLastUsedOncePerInterval(t *testing.T) {
	s, st := newCountedStore(t)
	ctx := context.Background()
	_, _, err := s.Users.UpsertUser(ctx, newTestUser("u1", "42", "alice"))
	require.NoError(t, err)
	svc := auth.NewService(auth.Config{
		Secret: []byte("secret"), Users: s.Users, PATs: s.PATs, Now: time.Now,
	})
	raw, _, err := svc.MintPAT(ctx, "u1", "ci")
	require.NoError(t, err)

	first, rest := statementsPerCall(t, st, func() {
		_, err := svc.AuthenticatePAT(ctx, raw)
		require.NoError(t, err)
	})

	reads := rest / 99
	assert.Greater(t, first, reads, "the first call stamps last_used_at")
	assert.Equal(t, reads*99, rest)
	assert.LessOrEqual(t, rest, int64(99*2), "a call inside the interval only reads")
}

func TestAuthenticateToken_TouchesLastUsedOncePerInterval(t *testing.T) {
	s, st := newCountedStore(t)
	ctx := context.Background()
	_, _, err := s.Users.UpsertUser(ctx, newTestUser("u1", "42", "alice"))
	require.NoError(t, err)
	svc := integrations.NewService(integrations.Config{
		Installs: s.IntegrationInstalls, Tokens: s.IntegrationTokens, Perms: holdsEverywhere{}, Now: time.Now,
	})
	raw, _, _, err := svc.Install(ctx, "u1", "bot", integrations.TrustCommunity, "", []integrations.Scope{integrations.ScopeDocsRead})
	require.NoError(t, err)

	first, rest := statementsPerCall(t, st, func() {
		_, _, err := svc.AuthenticateToken(ctx, raw)
		require.NoError(t, err)
	})

	reads := rest / 99
	assert.Greater(t, first, reads, "the first call stamps last_used_at")
	assert.Equal(t, reads*99, rest)
	assert.LessOrEqual(t, rest, int64(99*2), "a call inside the interval only reads")
}
