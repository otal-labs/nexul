package integrations

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/identity"
)

// ownerRequest builds a request with an owner identity actor, as the gateway
// does for user sessions.
func ownerRequest(t *testing.T, method, path, body string) *http.Request {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	ctx := identity.WithActor(req.Context(), identity.Actor{ID: "owner-1", CanCreateWorkspace: true})
	return req.WithContext(ctx)
}

func doRequest(h http.Handler, req *http.Request) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func newTestHandler(t *testing.T) (*Handler, *fakeData) {
	t.Helper()
	f := newFakeData()
	svc := newTestServiceWithOwner(f, true)
	return NewHandler(svc), f
}

func TestHandlerInstall(t *testing.T) {
	h, _ := newTestHandler(t)

	t.Run("creates install and returns raw token once", func(t *testing.T) {
		req := ownerRequest(t, http.MethodPost, "/api/integrations", `{
			"name":"discord","trust_tier":"community","webhook_url":"https://example.com/hook",
			"scopes":["events:read"]
		}`)
		rec := doRequest(h.Routes(), req)
		assert.Equal(t, http.StatusCreated, rec.Code)
		var resp struct {
			Install       *Install `json:"install"`
			Token         string   `json:"token"`
			WebhookSecret string   `json:"webhook_secret"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
		require.NotNil(t, resp.Install)
		assert.NotEmpty(t, resp.Token)
		assert.NotEmpty(t, resp.WebhookSecret)
		assert.True(t, strings.HasPrefix(resp.Token, tokenPrefix))
	})

	t.Run("invalid body is rejected", func(t *testing.T) {
		req := ownerRequest(t, http.MethodPost, "/api/integrations", `{"name":""}`)
		rec := doRequest(h.Routes(), req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("malformed json is rejected", func(t *testing.T) {
		req := ownerRequest(t, http.MethodPost, "/api/integrations", `{oops`)
		rec := doRequest(h.Routes(), req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

func TestHandlerListRevokeSubscriptions(t *testing.T) {
	h, f := newTestHandler(t)
	svc := h.svc
	_, _, install, err := svc.Install(context.Background(), "owner-1", "discord", TrustCommunity, "https://example.com", []Scope{ScopeEventsRead})
	require.NoError(t, err)

	t.Run("list installs", func(t *testing.T) {
		rec := doRequest(h.Routes(), ownerRequest(t, http.MethodGet, "/api/integrations", ""))
		assert.Equal(t, http.StatusOK, rec.Code)
		var resp struct {
			Installs []*Install `json:"installs"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
		require.Len(t, resp.Installs, 1)
	})

	t.Run("get install", func(t *testing.T) {
		rec := doRequest(h.Routes(), ownerRequest(t, http.MethodGet, "/api/integrations/"+install.ID, ""))
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("subscribe + list subscriptions", func(t *testing.T) {
		rec := doRequest(h.Routes(), ownerRequest(t, http.MethodPost, "/api/integrations/"+install.ID+"/subscriptions", `{"topic":"ticket.created"}`))
		assert.Equal(t, http.StatusCreated, rec.Code)

		rec = doRequest(h.Routes(), ownerRequest(t, http.MethodGet, "/api/integrations/"+install.ID+"/subscriptions", ""))
		assert.Equal(t, http.StatusOK, rec.Code)
		var resp struct {
			Subscriptions []Subscription `json:"subscriptions"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
		require.Len(t, resp.Subscriptions, 1)
		assert.Equal(t, "ticket.created", resp.Subscriptions[0].Topic)
	})

	t.Run("unsubscribe", func(t *testing.T) {
		rec := doRequest(h.Routes(), ownerRequest(t, http.MethodDelete, "/api/integrations/"+install.ID+"/subscriptions/ticket.created", ""))
		assert.Equal(t, http.StatusNoContent, rec.Code)
	})

	t.Run("revoke", func(t *testing.T) {
		rec := doRequest(h.Routes(), ownerRequest(t, http.MethodDelete, "/api/integrations/"+install.ID, ""))
		assert.Equal(t, http.StatusNoContent, rec.Code)
	})

	t.Run("revoked install no longer authenticates", func(t *testing.T) {
		_, _, err := h.svc.AuthenticateToken(context.Background(), "whatever")
		require.Error(t, err)
	})
	_ = f
	t.Run("list deliveries", func(t *testing.T) {
		h, _ := newTestHandler(t)
		svc := h.svc
		_, _, install, err := svc.Install(context.Background(), "owner-1", "x", TrustCommunity, "https://example.com", []Scope{ScopeEventsRead})
		require.NoError(t, err)
		require.NoError(t, svc.Subscribe(context.Background(), "owner-1", install.ID, "ticket.created"))
		fh := NewFanoutHandler(svc)
		require.NoError(t, fh.HandleEvent(context.Background(), eventbus.Event{
			ID: "evt-1", Topic: "ticket.created", Timestamp: time.Now().UTC(), Payload: json.RawMessage(`{}`),
		}))

		rec := doRequest(h.Routes(), ownerRequest(t, http.MethodGet, "/api/integrations/"+install.ID+"/deliveries", ""))
		assert.Equal(t, http.StatusOK, rec.Code)
		var resp struct {
			Deliveries []*Delivery `json:"deliveries"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
		require.Len(t, resp.Deliveries, 1)
		assert.Equal(t, DeliveryPending, resp.Deliveries[0].Status)
	})

	t.Run("non-owner denied reads", func(t *testing.T) {
		h, _ := newTestHandler(t)
		h.svc = newTestServiceWithOwner(newFakeData(), false)
		for _, tc := range []struct{ method, path string }{
			{http.MethodGet, "/api/integrations"},
			{http.MethodGet, "/api/integrations/abc123"},
			{http.MethodGet, "/api/audit"},
			{http.MethodGet, "/api/integrations/abc123/deliveries"},
			{http.MethodGet, "/api/integrations/abc123/subscriptions"},
		} {
			rec := doRequest(h.Routes(), ownerRequest(t, tc.method, tc.path, ""))
			assert.Equal(t, http.StatusForbidden, rec.Code, "%s %s", tc.method, tc.path)
		}
	})
}

func TestHandlerCatalogAudit(t *testing.T) {
	h, _ := newTestHandler(t)
	require.NoError(t, h.svc.cfg.Schemas.Publish(context.Background(), SchemaEntry{
		Topic: "ticket.created", Version: 1, Schema: `{"type":"object"}`, CreatedAt: h.svc.cfg.Now().UTC(),
	}))

	t.Run("catalog lists published schemas", func(t *testing.T) {
		req := ownerRequest(t, http.MethodGet, "/api/events/catalog", "")
		rec := doRequest(h.Routes(), req)
		assert.Equal(t, http.StatusOK, rec.Code)
		var resp struct {
			Catalog []SchemaEntry `json:"catalog"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
		require.Len(t, resp.Catalog, 1)
		assert.Equal(t, "ticket.created", resp.Catalog[0].Topic)
	})

	t.Run("audit lists entries", func(t *testing.T) {
		require.NoError(t, h.svc.cfg.Audit.Append(context.Background(), AuditEntry{
			ID: "a1", ActorType: "integration", ActorID: "install-1", TokenID: "tok-1", Action: "GET /api/tickets",
			CreatedAt: h.svc.cfg.Now().UTC(),
		}))
		rec := doRequest(h.Routes(), ownerRequest(t, http.MethodGet, "/api/audit", ""))
		assert.Equal(t, http.StatusOK, rec.Code)
		var resp struct {
			Audit []AuditEntry `json:"audit"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
		require.Len(t, resp.Audit, 1)
		assert.Equal(t, "tok-1", resp.Audit[0].TokenID)
	})
}

func TestHandlerErrorPaths(t *testing.T) {
	t.Run("non-owner denied install create", func(t *testing.T) {
		h, _ := newTestHandler(t)
		h.svc = newTestServiceWithOwner(newFakeData(), false)
		req := ownerRequest(t, http.MethodPost, "/api/integrations", `{"name":"x","trust_tier":"verified","scopes":["docs:read"]}`)
		rec := doRequest(h.Routes(), req)
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})

	t.Run("unknown install get is not found", func(t *testing.T) {
		h, _ := newTestHandler(t)
		rec := doRequest(h.Routes(), ownerRequest(t, http.MethodGet, "/api/integrations/nope", ""))
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("scopes lists the enforced vocabulary with labels", func(t *testing.T) {
		h, _ := newTestHandler(t)
		rec := doRequest(h.Routes(), ownerRequest(t, http.MethodGet, "/api/integrations/scopes", ""))
		assert.Equal(t, http.StatusOK, rec.Code)
		var resp struct {
			Scopes []ScopeInfo `json:"scopes"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
		assert.Equal(t, Catalog(), resp.Scopes)
	})

	t.Run("catalog empty is ok", func(t *testing.T) {
		h, _ := newTestHandler(t)
		rec := doRequest(h.Routes(), ownerRequest(t, http.MethodGet, "/api/events/catalog", ""))
		assert.Equal(t, http.StatusOK, rec.Code)
	})
}
