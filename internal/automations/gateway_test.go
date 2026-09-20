package automations

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/platform/identity"
)

// fakeUserAuth mimics the auth middleware chain, marking the request
// authenticated and calling next.
func fakeUserAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Header.Set("X-Fell-Through", "true")
		next.ServeHTTP(w, r)
	})
}

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
}

// allowScope is a ScopeGate fake standing in for integrations.ScopeAllows in
// unit tests (the real function, including its write-vs-read distinction, is
// exercised by gateway_integration_test.go): it grants GET/HEAD whenever the
// given scope is present and denies every other method, so a scope-mismatch
// test case doesn't need to know the real method/path table.
func allowScope(scope string) ScopeGate {
	return func(method, _ string, scopes []string) bool {
		if method != http.MethodGet && method != http.MethodHead {
			return false
		}
		for _, s := range scopes {
			if s == scope {
				return true
			}
		}
		return false
	}
}

// fakeOwnerGate reports a fixed can_create_workspace bit for every user, or
// fails every lookup when err is set.
type fakeOwnerGate struct {
	can bool
	err error
}

func (f fakeOwnerGate) CanCreateWorkspace(context.Context, string) (bool, error) {
	return f.can, f.err
}

func newGatewaySvc(t *testing.T, scope string, owner OwnerGate) (*Service, string) {
	t.Helper()
	svc := NewService(newFakeRepo(), allowAll("creator-1"))
	if owner != nil {
		svc.SetGateway(allowScope(scope), nil, owner)
	}
	a, raw, err := svc.Create(context.Background(), "creator-1", "webhook-relay", []string{scope})
	require.NoError(t, err)
	require.NotEmpty(t, a.CreatedBy)
	return svc, raw
}

func TestRequireAutomation(t *testing.T) {
	t.Run("automation token with matching scope passes and acts as creator", func(t *testing.T) {
		svc, raw := newGatewaySvc(t, "tickets:read", fakeOwnerGate{can: true})
		var gotActor string
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			aa := AutomationFromCtx(r.Context())
			require.NotNil(t, aa)
			gotActor = aa.Automation.CreatedBy
			w.WriteHeader(http.StatusOK)
		})
		handler := svc.RequireAutomation(fakeUserAuth, next)
		req := httptest.NewRequest(http.MethodGet, "/api/tickets", nil)
		req.Header.Set("Authorization", "Bearer "+raw)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "creator-1", gotActor)
		assert.NotEqual(t, "true", req.Header.Get("X-Fell-Through"))
	})

	t.Run("automation token without matching scope is forbidden", func(t *testing.T) {
		svc, raw := newGatewaySvc(t, "tickets:read", fakeOwnerGate{can: true})
		handler := svc.RequireAutomation(fakeUserAuth, okHandler())
		req := httptest.NewRequest(http.MethodPost, "/api/tickets", nil)
		req.Header.Set("Authorization", "Bearer "+raw)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})

	t.Run("gateway not wired denies rather than panics", func(t *testing.T) {
		svc, raw := newGatewaySvc(t, "tickets:read", nil)
		handler := svc.RequireAutomation(fakeUserAuth, okHandler())
		req := httptest.NewRequest(http.MethodGet, "/api/tickets", nil)
		req.Header.Set("Authorization", "Bearer "+raw)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})

	t.Run("invalid automation token is unauthorized", func(t *testing.T) {
		svc, _ := newGatewaySvc(t, "tickets:read", fakeOwnerGate{can: true})
		handler := svc.RequireAutomation(fakeUserAuth, okHandler())
		req := httptest.NewRequest(http.MethodGet, "/api/tickets", nil)
		req.Header.Set("Authorization", "Bearer "+tokenPrefix+"AAAA")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("non-automation token falls through to user auth", func(t *testing.T) {
		svc, _ := newGatewaySvc(t, "tickets:read", fakeOwnerGate{can: true})
		handler := svc.RequireAutomation(fakeUserAuth, okHandler())
		req := httptest.NewRequest(http.MethodGet, "/api/tickets", nil)
		req.Header.Set("Authorization", "Bearer session-token")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "true", req.Header.Get("X-Fell-Through"))
	})

	t.Run("revoked token rejected", func(t *testing.T) {
		svc, raw := newGatewaySvc(t, "tickets:read", fakeOwnerGate{can: true})
		list, err := svc.List(context.Background(), "creator-1")
		require.NoError(t, err)
		require.Len(t, list, 1)
		_, err = svc.RevokeToken(context.Background(), "creator-1", list[0].ID)
		require.NoError(t, err)
		handler := svc.RequireAutomation(fakeUserAuth, okHandler())
		req := httptest.NewRequest(http.MethodGet, "/api/tickets", nil)
		req.Header.Set("Authorization", "Bearer "+raw)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("owner lookup failure denies workspace authority without failing the request", func(t *testing.T) {
		svc, raw := newGatewaySvc(t, "tickets:read", fakeOwnerGate{err: assert.AnError})
		var actor identity.Actor
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			actor, _ = identity.ActorFromCtx(r.Context())
			w.WriteHeader(http.StatusOK)
		})
		handler := svc.RequireAutomation(fakeUserAuth, next)
		req := httptest.NewRequest(http.MethodGet, "/api/tickets", nil)
		req.Header.Set("Authorization", "Bearer "+raw)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "creator-1", actor.ID)
		assert.False(t, actor.CanCreateWorkspace)
	})
}

func TestResolveOwner(t *testing.T) {
	svc := NewService(newFakeRepo(), allowAll("creator-1"))

	t.Run("no gate wired denies", func(t *testing.T) {
		assert.False(t, svc.resolveOwner(context.Background(), "creator-1"))
	})

	t.Run("no creator denies without a lookup", func(t *testing.T) {
		svc.SetGateway(nil, nil, fakeOwnerGate{err: assert.AnError})
		assert.False(t, svc.resolveOwner(context.Background(), ""))
	})

	t.Run("gate error denies", func(t *testing.T) {
		svc.SetGateway(nil, nil, fakeOwnerGate{err: assert.AnError})
		assert.False(t, svc.resolveOwner(context.Background(), "creator-1"))
	})

	t.Run("gate grants", func(t *testing.T) {
		svc.SetGateway(nil, nil, fakeOwnerGate{can: true})
		assert.True(t, svc.resolveOwner(context.Background(), "creator-1"))
	})
}

func TestRequireAutomation_SelfRead(t *testing.T) {
	svc := NewService(newFakeRepo(), allowAll("creator-1"))
	svc.SetGateway(allowScope("tickets:read"), nil, fakeOwnerGate{can: true})
	a, raw, err := svc.Create(context.Background(), "creator-1", "self-reader", []string{"docs:read"})
	require.NoError(t, err)
	other, _, err := svc.Create(context.Background(), "creator-1", "someone-else", []string{"docs:read"})
	require.NoError(t, err)
	handler := svc.RequireAutomation(fakeUserAuth, okHandler())

	tests := []struct {
		name   string
		method string
		path   string
		want   int
	}{
		{"own record", http.MethodGet, "/api/automations/" + a.ID, http.StatusOK},
		{"own versions diff", http.MethodGet, "/api/automations/" + a.ID + "/versions/diff", http.StatusOK},
		{"own record, HEAD", http.MethodHead, "/api/automations/" + a.ID, http.StatusOK},
		{"own record, write", http.MethodPatch, "/api/automations/" + a.ID + "/enabled", http.StatusForbidden},
		{"another automation's record", http.MethodGet, "/api/automations/" + other.ID, http.StatusForbidden},
		{"id prefix collision", http.MethodGet, "/api/automations/" + a.ID + "x", http.StatusForbidden},
		{"the list", http.MethodGet, "/api/automations", http.StatusForbidden},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			req.Header.Set("Authorization", "Bearer "+raw)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			assert.Equal(t, tt.want, rec.Code)
		})
	}
}
