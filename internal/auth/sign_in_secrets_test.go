package auth

import (
	"context"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/secretref"
	"github.com/otal-labs/nexul/internal/platform/secretref/secretreftest"
)

const googleSecretRef = "op://Acme/google/secret"

// secretsOver opens one manager over fake and returns the cache the service resolves through.
func secretsOver(t *testing.T, fake *secretreftest.Fake) *secretref.Cache {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	managers := secretref.NewManagers(slog.New(slog.NewTextHandler(io.Discard, nil)), secretref.Setup{
		Scheme: "op",
		Open:   func(context.Context) (secretref.Resolver, error) { return fake, nil },
	})
	managers.SetRetry(time.Hour, time.Hour)
	var wg sync.WaitGroup
	managers.Start(ctx, &wg)
	t.Cleanup(func() { cancel(); wg.Wait() })
	return secretref.NewCache(managers, nil)
}

func TestSetProviderOAuth_AReferenceThatDoesNotResolveIsRefusedWithAFixedReason(t *testing.T) {
	s, users, _, settings := newTestHarness(&fakeGitHub{user: ghUser("1", "owner")})
	seedOwner(t, users, "u1", "1", "owner")
	fake := &secretreftest.Fake{Errs: map[string]error{googleSecretRef: secretref.ErrNoAccess}}
	s.cfg.Secrets = secretsOver(t, fake)

	_, err := s.SetProviderOAuth(t.Context(), "u1", ProviderGoogle, "g-id", googleSecretRef)

	require.ErrorIs(t, err, apperrs.ErrInvalid)
	assert.ErrorContains(t, err, "no access")
	assert.False(t, settings.st.ProviderConfigured(ProviderGoogle), "nothing is stored")
}
