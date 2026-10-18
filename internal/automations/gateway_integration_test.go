package automations_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/automations"
	"github.com/otal-labs/nexul/internal/integrations"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/permissions"
	"github.com/otal-labs/nexul/internal/platform/storage/testutil/migrated"
)

// allowAutomationPerm grants every automation.* action to userID, standing in for the access domain's Require.
type allowAutomationPerm struct{ userID string }

func (g allowAutomationPerm) Require(ctx context.Context, _ string, action permissions.Action) error {
	if actor, _ := identity.ActorFromCtx(ctx); actor.ID == g.userID {
		return nil
	}
	return apperrs.ErrForbidden
}

func fellThroughUserAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "no user session", http.StatusUnauthorized)
	})
}

// TestRequireAutomation_EndToEndScopedTicketsRead proves ticket 16's actual
// deliverable: a dat_ automation token, wired the same way the composition
// root wires it (automations.Service.RequireAutomation + the exported
// integrations.ScopeAllows table, not a duplicated one), can reach a
// tickets:read-scoped endpoint and is refused a tickets:write one it wasn't
// granted.
func TestRequireAutomation_EndToEndScopedTicketsRead(t *testing.T) {
	store := migrated.Store(t)
	svc := automations.NewService(store.Automations, allowAutomationPerm{userID: "creator-1"})
	svc.SetGateway(integrations.ScopeAllows, integrations.ResolveScopes)

	a, raw, err := svc.Create(context.Background(), "creator-1", "ws-1", "ticket-reader", []string{"tickets:read"})
	require.NoError(t, err)
	_, err = svc.SetEnabled(context.Background(), "creator-1", a.ID, true)
	require.NoError(t, err)

	ticketsHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("tickets"))
	})
	handler := svc.RequireAutomation(fellThroughUserAuth, ticketsHandler)

	t.Run("granted scope reaches the endpoint", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/tickets", nil)
		req.Header.Set("Authorization", "Bearer "+raw)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("missing scope is refused", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/tickets", nil)
		req.Header.Set("Authorization", "Bearer "+raw)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})

	t.Run("out-of-catalog surface is refused even with a real scope", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/notifications", nil)
		req.Header.Set("Authorization", "Bearer "+raw)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})
}

func TestRequireAutomation_SwitchedOff_CannotQueuePlays(t *testing.T) {
	store := migrated.Store(t)
	svc := automations.NewService(store.Automations, allowAutomationPerm{userID: "creator-1"})
	svc.SetGateway(integrations.ScopeAllows, integrations.ResolveScopes)
	a, raw, err := svc.Create(t.Context(), "creator-1", "ws-1", "fixer", []string{"plays:run"})
	require.NoError(t, err)
	queued := 0
	handler := svc.RequireAutomation(fellThroughUserAuth, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		queued++
		w.WriteHeader(http.StatusCreated)
	}))
	queue := func() int {
		req := httptest.NewRequest(http.MethodPost, "/api/plays/queue", nil)
		req.Header.Set("Authorization", "Bearer "+raw)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		return rec.Code
	}

	assert.Equal(t, http.StatusUnauthorized, queue(), "a new automation is off until switched on")
	_, err = svc.SetEnabled(t.Context(), "creator-1", a.ID, true)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, queue())
	_, err = svc.SetEnabled(t.Context(), "creator-1", a.ID, false)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, queue(), "switching off refuses the token")
	assert.Equal(t, 1, queued, "only the switched-on call reached the route")
}

func TestCreate_ScopesUseTheIntegrationsVocabulary(t *testing.T) {
	store := migrated.Store(t)
	svc := automations.NewService(store.Automations, allowAutomationPerm{userID: "creator-1"})
	svc.SetGateway(integrations.ScopeAllows, integrations.ResolveScopes)

	t.Run("a typo'd scope is rejected instead of minting a token the gate refuses everywhere", func(t *testing.T) {
		_, _, err := svc.Create(context.Background(), "creator-1", "ws-1", "typo", []string{"ticket:write"})
		require.Error(t, err)
	})

	t.Run("a write scope stores the reads it implies", func(t *testing.T) {
		a, _, err := svc.Create(context.Background(), "creator-1", "ws-1", "writer", []string{"tickets:write"})
		require.NoError(t, err)
		assert.ElementsMatch(t, []string{"tickets:write", "tickets:read"}, a.Scopes)
	})

	t.Run("every shipped default's scopes resolve", func(t *testing.T) {
		for _, def := range automations.DefaultDefinitions() {
			_, err := integrations.ResolveScopes(def.Scopes)
			assert.NoError(t, err, def.ID)
		}
	})
}
