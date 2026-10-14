package main

import (
	"encoding/json"
	"flag"
	"maps"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/platform/permissions"
)

var updateRoutePermissions = flag.Bool("update-permission-catalog", false, "rewrite the clients' copy of the route permissions")

// routePermissionsFile is the clients' typed copy of routePermissions, read by client-core's routePermissions.test.ts.
const routePermissionsFile = "../../client-core/routePermissions.json"

// writingRoutes is every non-GET route of the real router, as "METHOD /api/path".
func writingRoutes(t *testing.T) []string {
	t.Helper()
	h, _, _ := newRouter(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/openapi.json", nil))
	require.Equal(t, http.StatusOK, rec.Code)
	var doc struct {
		Paths map[string]map[string]json.RawMessage `json:"paths"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &doc))
	var routes []string
	for path, ops := range doc.Paths {
		for method := range ops {
			if strings.ToUpper(method) != http.MethodGet {
				routes = append(routes, strings.ToUpper(method)+" "+path)
			}
		}
	}
	slices.Sort(routes)
	return routes
}

func TestRoutePermissions_NameEveryWritingRouteAndOnlyCatalogActions(t *testing.T) {
	routes := writingRoutes(t)
	require.Greater(t, len(routes), 200, "the route table came from the real router")
	catalog := map[string]bool{"none": true}
	for _, info := range permissions.Catalog() {
		catalog[string(info.Value)] = true
	}
	for _, route := range routes {
		gate, ok := routePermissions[route]
		assert.True(t, ok, "%s names no permission; add it to routePermissions", route)
		assert.NotEmpty(t, gate, route)
		for _, action := range gate {
			assert.True(t, catalog[action], "%s names %q, which is not a catalog action", route, action)
		}
	}
	for route := range routePermissions {
		assert.True(t, slices.Contains(routes, route), "%s is in routePermissions but the router has no such route", route)
	}
}

// renderRoutePermissions is the table as JSON with one route per line, so a diff names each route that changed.
func renderRoutePermissions(t *testing.T) []byte {
	t.Helper()
	keys := slices.Sorted(maps.Keys(routePermissions))
	lines := make([]string, 0, len(keys))
	for _, key := range keys {
		k, err := json.Marshal(key)
		require.NoError(t, err)
		v, err := json.Marshal(routePermissions[key])
		require.NoError(t, err)
		lines = append(lines, "  "+string(k)+": "+string(v))
	}
	return []byte("{\n" + strings.Join(lines, ",\n") + "\n}\n")
}

// TestRoutePermissionsFile_MatchesTheRoutes fails when the clients' copy is stale, the way sqlc diff does.
func TestRoutePermissionsFile_MatchesTheRoutes(t *testing.T) {
	want := renderRoutePermissions(t)
	if *updateRoutePermissions {
		require.NoError(t, os.WriteFile(routePermissionsFile, want, 0o644))
	}
	got, err := os.ReadFile(routePermissionsFile)
	require.NoError(t, err)
	assert.Equal(t, string(want), string(got), "stale; regenerate with make permission-catalog")
}
