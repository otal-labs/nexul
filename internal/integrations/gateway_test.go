package integrations

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestScopeAllows_StringWrapper(t *testing.T) {
	assert.True(t, ScopeAllows("GET", "/api/docs", []string{"docs:read"}))
	assert.False(t, ScopeAllows("POST", "/api/docs", []string{"docs:read"}))
}

func TestRequireIntegration(t *testing.T) {
	newSvc := func() *Service {
		f := newFakeData()
		return newTestServiceWithOwner(f, true)
	}
	mint := func(t *testing.T, svc *Service, scopes []Scope) string {
		t.Helper()
		raw, _, _, err := svc.Install(context.Background(), "owner-1", "discord", TrustCommunity, "https://example.com", scopes)
		require.NoError(t, err)
		return raw
	}

	t.Run("integration token with matching scope passes", func(t *testing.T) {
		svc := newSvc()
		raw := mint(t, svc, []Scope{ScopeDocsRead})
		handler := svc.RequireIntegration(fakeUserAuth, okHandler())
		req := httptest.NewRequest(http.MethodGet, "/api/docs", nil)
		req.Header.Set("Authorization", "Bearer "+raw)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.NotEqual(t, "true", req.Header.Get("X-Fell-Through"))
	})

	t.Run("integration token without matching scope is forbidden", func(t *testing.T) {
		svc := newSvc()
		raw := mint(t, svc, []Scope{ScopeDocsRead})
		handler := svc.RequireIntegration(fakeUserAuth, okHandler())
		req := httptest.NewRequest(http.MethodPost, "/api/tickets", nil)
		req.Header.Set("Authorization", "Bearer "+raw)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})

	t.Run("integration token on out-of-scope surface is forbidden", func(t *testing.T) {
		svc := newSvc()
		raw := mint(t, svc, []Scope{ScopeEventsRead})
		handler := svc.RequireIntegration(fakeUserAuth, okHandler())
		req := httptest.NewRequest(http.MethodGet, "/api/notifications", nil)
		req.Header.Set("Authorization", "Bearer "+raw)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})

	t.Run("invalid integration token is unauthorized", func(t *testing.T) {
		svc := newSvc()
		handler := svc.RequireIntegration(fakeUserAuth, okHandler())
		req := httptest.NewRequest(http.MethodGet, "/api/docs", nil)
		req.Header.Set("Authorization", "Bearer "+tokenPrefix+"AAAA")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("non-integration token falls through to user auth", func(t *testing.T) {
		svc := newSvc()
		handler := svc.RequireIntegration(fakeUserAuth, okHandler())
		req := httptest.NewRequest(http.MethodGet, "/api/docs", nil)
		req.Header.Set("Authorization", "Bearer session-token")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "true", req.Header.Get("X-Fell-Through"))
	})

	t.Run("revoked token rejected", func(t *testing.T) {
		f := newFakeData()
		svc := newTestServiceWithOwner(f, true)
		raw, _, install, err := svc.Install(context.Background(), "owner-1", "discord", TrustCommunity, "", []Scope{ScopeDocsRead})
		require.NoError(t, err)
		require.NoError(t, svc.RevokeInstall(context.Background(), "owner-1", install.ID))
		handler := svc.RequireIntegration(fakeUserAuth, okHandler())
		req := httptest.NewRequest(http.MethodGet, "/api/docs", nil)
		req.Header.Set("Authorization", "Bearer "+raw)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})
}

func TestAuditLog(t *testing.T) {
	t.Run("records integration request with token id", func(t *testing.T) {
		f := newFakeData()
		svc := newTestServiceWithOwner(f, true)
		raw, _, install, err := svc.Install(context.Background(), "owner-1", "discord", TrustCommunity, "", []Scope{ScopeDocsRead})
		require.NoError(t, err)

		tokens, err := f.tokensList(install.ID)
		require.NoError(t, err)
		require.Len(t, tokens, 1)

		// Composition mirrors the gateway: audit innermost, integration auth
		// outermost so the audited request context carries the install.
		audited := svc.AuditLog(func(ctx context.Context) (string, string, string) {
			if ia := IntegrationFromCtx(ctx); ia != nil {
				return "integration", ia.Install.ID, ia.Token.ID
			}
			return "user", "owner-1", ""
		}, okHandler())
		handler := svc.RequireIntegration(fakeUserAuth, audited)

		req := httptest.NewRequest(http.MethodGet, "/api/docs", nil)
		req.Header.Set("Authorization", "Bearer "+raw)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)

		entries := f.auditEntries()
		require.Len(t, entries, 1)
		assert.Equal(t, "integration", entries[0].ActorType)
		assert.Equal(t, install.ID, entries[0].ActorID)
		assert.Equal(t, tokens[0].ID, entries[0].TokenID)
		assert.Equal(t, "GET /api/docs", entries[0].Action)
	})

	t.Run("records user request", func(t *testing.T) {
		f := newFakeData()
		svc := newTestServiceWithOwner(f, true)
		handler := svc.AuditLog(func(ctx context.Context) (string, string, string) {
			return "user", "owner-1", ""
		}, okHandler())
		req := httptest.NewRequest(http.MethodGet, "/api/tickets", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
		entries := f.auditEntries()
		require.Len(t, entries, 1)
		assert.Equal(t, "user", entries[0].ActorType)
	})
}

func (f *fakeData) tokensList(installID string) ([]*IntegrationToken, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []*IntegrationToken
	for _, t := range f.tokens {
		if t.InstallID == installID {
			cp := *t
			out = append(out, &cp)
		}
	}
	return out, nil
}

func (f *fakeData) auditEntries() []AuditEntry {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]AuditEntry, len(f.audit))
	copy(out, f.audit)
	return out
}
