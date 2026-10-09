package main

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/auth"
	"github.com/otal-labs/nexul/internal/integrations"
	"github.com/otal-labs/nexul/internal/platform/config"
	"github.com/otal-labs/nexul/internal/platform/eventbus/inprocess"
	"github.com/otal-labs/nexul/internal/platform/storage"
	"github.com/otal-labs/nexul/internal/roles"
	"github.com/otal-labs/nexul/internal/tenancy"
)

// readOnlyRoutes are the non-GET routes that only read, so they write no audit row; every other non-GET route does.
var readOnlyRoutes = []string{
	"POST /api/auth/bootstrap/verify",
	"POST /api/connectors/{id}/manual/verify",
	"POST /api/dns/tunnels/{tunnelID}/verify",
	"POST /api/dns/verify",
	"POST /api/invitations/preview",
	"POST /api/logs",
	"POST /api/machines/{id}/discover",
	"POST /api/mentions/resolve",
	"POST /api/repositories/scan",
	"POST /api/tickets/dev-status",
	"POST /api/tickets/labels/colors",
}

// newRouter wires the server the way serve does, over a fresh database, and returns its real handler.
func newRouter(t *testing.T) (http.Handler, *storage.Store, *coreServices) {
	t.Helper()
	ctx := t.Context()
	db := mentionsTestDB(t)
	key := []byte("0123456789abcdef0123456789abcdef")
	store := storage.New(db, key)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	bus := inprocess.New(inprocess.Options{Logger: logger, DedupeStore: store.ProcessedEvents, DeadLetterStore: store.DeadLetters})
	t.Cleanup(func() { _ = bus.Close() })
	cfg := &config.Config{DBPath: filepath.Join(t.TempDir(), "nexul.db"), HTTPAddr: "127.0.0.1:0", AuthSecret: string(key)}
	svc := wireCoreServices(cfg, store, key, bus, logger)
	wireDomainEventSubscriptions(ctx, bus, svc, logger)
	wireIntegrationFanout(ctx, bus, store, svc, logger)
	wsHandler, runnerSvc, runnerHTTP, automationsDialin := startBackgroundWorkers(ctx, cfg, store, bus, svc, logger)
	svc.deploySvc.SetMachineLookup(deployMachineLookupAdapter{runner: runnerSvc})
	svc.deploySvc.SetMachineDiscoverer(deployMachineDiscovererAdapter{runner: runnerSvc})
	liveHub, agentHandler := wireLiveHubAndAgent(ctx, bus, store, svc, logger)
	return buildRoutes(cfg, bus, store, svc, wsHandler, runnerSvc, runnerHTTP, automationsDialin, liveHub, agentHandler, logger).Handler, store, svc
}

// signedInOwner makes alice the default workspace's Owner and returns her personal access token.
func signedInOwner(t *testing.T, store *storage.Store, svc *coreServices) (userID, token string) {
	t.Helper()
	ctx := t.Context()
	_, _, err := store.Users.UpsertUser(ctx, &auth.Identity{UserID: "u-alice", Provider: auth.ProviderGitHub, ProviderUserID: "alice", Login: "alice"})
	require.NoError(t, err)
	now := time.Now()
	require.NoError(t, store.Roles.Create(ctx, &roles.Role{ID: "role-owner", WorkspaceID: "workspace-default", Name: "Owner", IsOwnerRole: true, CreatedAt: now, UpdatedAt: now}))
	require.NoError(t, store.WorkspaceMembers.AddMember(ctx, &tenancy.Member{UserID: "u-alice", WorkspaceID: "workspace-default", RoleID: "role-owner", CreatedAt: now}))
	raw, _, err := svc.authSvc.MintPAT(ctx, "u-alice", "audit test")
	require.NoError(t, err)
	return "u-alice", raw
}

func auditRows(t *testing.T, store *storage.Store) []integrations.AuditEntry {
	t.Helper()
	rows, err := store.Audit.List(t.Context(), 100)
	require.NoError(t, err)
	return rows
}

func callAPI(t *testing.T, h http.Handler, method, path, token, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

// TestAudited_EveryRouteOfTheRouter_IsClassifiedByMethod walks the real route table: a GET is never audited, and a
// non-GET route is audited unless it is one of the read-only routes, each of which must still exist.
func TestAudited_EveryRouteOfTheRouter_IsClassifiedByMethod(t *testing.T) {
	h, _, _ := newRouter(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/openapi.json", nil))
	require.Equal(t, http.StatusOK, rec.Code)
	var doc struct {
		Paths map[string]map[string]json.RawMessage `json:"paths"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &doc))

	readOnly := map[string]bool{}
	for _, route := range readOnlyRoutes {
		readOnly[route] = true
	}
	wildcard := regexp.MustCompile(`\{[^}]+\}`)
	seen := map[string]bool{}
	for path, ops := range doc.Paths {
		for method := range ops {
			route := strings.ToUpper(method) + " " + path
			seen[route] = true
			req := httptest.NewRequest(strings.ToUpper(method), wildcard.ReplaceAllString(path, "x1"), nil)
			want := req.Method != http.MethodGet && !readOnly[route]
			assert.Equal(t, want, integrations.Audited(req), route)
		}
	}
	require.Greater(t, len(seen), 200, "the route table came from the real router")
	for _, route := range readOnlyRoutes {
		assert.True(t, seen[route], "%s is listed as read-only but the router has no such route", route)
	}
}

func TestAuditLog_ThroughTheRouter_AGetWritesNothingAndAPatchWritesOneRow(t *testing.T) {
	h, store, svc := newRouter(t)
	userID, token := signedInOwner(t, store, svc)

	require.Equal(t, http.StatusOK, callAPI(t, h, http.MethodGet, "/api/auth/me", token, "").Code)
	assert.Empty(t, auditRows(t, store), "a read writes no audit row")

	rec := callAPI(t, h, http.MethodPatch, "/api/workspaces/workspace-default", token, `{"name":"Acme"}`)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	rows := auditRows(t, store)
	require.Len(t, rows, 1)
	assert.Equal(t, "user", rows[0].ActorType)
	assert.Equal(t, userID, rows[0].ActorID)
	assert.Equal(t, "PATCH /api/workspaces/workspace-default", rows[0].Action)
}
