package connectors_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/connectors"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/permissions"
	"github.com/otal-labs/nexul/internal/platform/secretref"
)

// readsOnly lets a caller read the connectors named in areas and nothing else.
type readsOnly map[string]bool

func (g readsOnly) RequireArea(_ context.Context, area string, action permissions.Action) error {
	if action == permissions.ConnectorsRead && g[area] {
		return nil
	}
	return apperrs.ErrForbidden
}

func (readsOnly) HoldsArea(context.Context, string, string, permissions.Action) (bool, error) {
	return false, nil
}

type managersAre []secretref.ManagerState

func (m managersAre) States() []secretref.ManagerState { return m }

func TestManagersRead_AnswersAnyoneWhoCanReadAConnector_NamingNoManagerButByScheme(t *testing.T) {
	registry := []connectors.Connector{{ID: "github", Name: "GitHub"}, {ID: "cloudflare", Name: "Cloudflare"}}
	read := func(gate connectors.Gate) *httptest.ResponseRecorder {
		svc := connectors.NewService(connectors.Config{
			Gate:     gate,
			Registry: registry,
			Managers: managersAre{{Manager: "op", State: secretref.ManagerTokenRejected}},
		})
		rec := httptest.NewRecorder()
		withUserID("user-1", connectors.NewHandler(svc).Routes()).
			ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/connectors/managers", nil))
		return rec
	}

	one := read(readsOnly{permissions.ConnectorAreaKey("cloudflare"): true})
	require.Equal(t, http.StatusOK, one.Code)
	assert.JSONEq(t, `{"managers":[{"manager":"op","state":"token rejected"}]}`, one.Body.String())

	none := read(readsOnly{})
	assert.Equal(t, http.StatusForbidden, none.Code, "a caller who sees no connector does not learn the server's manager")
	assert.NotContains(t, none.Body.String(), "token rejected")
}

func TestManagersRead_WithNoManagerConfigured_IsAnEmptyList(t *testing.T) {
	svc := connectors.NewService(connectors.Config{Registry: []connectors.Connector{{ID: "github"}}})
	rec := httptest.NewRecorder()
	connectors.NewHandler(svc).Routes().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/connectors/managers", nil))
	assert.JSONEq(t, `{"managers":[]}`, rec.Body.String())
}
