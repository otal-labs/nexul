package main

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/platform/httpx"
	"github.com/otal-labs/nexul/internal/platform/openapi"
)

func TestRegisterOpenAPIRoutes_IncludesMountedRoutes(t *testing.T) {
	child := httpx.NewServeMux()
	child.HandleFunc("GET /api/new-route", func(http.ResponseWriter, *http.Request) {})
	apiMux := httpx.NewServeMux()
	mountGateway(apiMux, "/api/new-route", child)

	spec := openapi.New(openapi.Info{Title: "x", Version: "1"})
	registerOpenAPIRoutes(spec, apiMux.Routes())

	doc, err := spec.JSON()
	require.NoError(t, err)
	var parsed struct {
		Paths map[string]json.RawMessage `json:"paths"`
	}
	require.NoError(t, json.Unmarshal(doc, &parsed))
	require.Contains(t, parsed.Paths, "/api/new-route")
	require.NotContains(t, parsed.Paths, "/api/docs")
}

func TestMountGateway_RequiresTrackedAPIRoutes(t *testing.T) {
	require.Panics(t, func() {
		mountGateway(httpx.NewServeMux(), "/api/untracked", http.NewServeMux())
	})
}
