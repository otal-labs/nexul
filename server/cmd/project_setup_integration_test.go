package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/workspace"
)

// A wizard-made project names its service and the environment keys it detected; a member who may open the project but
// not read its stacks gets the step marks and nothing that names the service, on every read the gateway offers.
func TestIntegration_ProjectSetup_ServiceIsShownOnlyToReadersOfStacks(t *testing.T) {
	f := newPermFixture(t)
	keys := []string{"STRIPE_SECRET_KEY"}
	_, err := f.svc.workspaceSvc.ChangeSetup(as(uOwner), pGeneral, workspace.SetupChange{
		StackID: &f.stack, EnvKeys: &keys, Steps: map[workspace.SetupStep]workspace.SetupMark{"service": workspace.SetupDone},
	})
	require.NoError(t, err)
	routes := workspace.NewHandler(f.svc.workspaceSvc).Routes()
	get := func(person, path string) string {
		rec := httptest.NewRecorder()
		routes.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil).WithContext(as(person)))
		require.Equal(t, http.StatusOK, rec.Code, path)
		return rec.Body.String()
	}

	for _, path := range []string{"/api/projects/" + pGeneral, "/api/projects?workspace_id=" + wsDefault} {
		t.Run(path, func(t *testing.T) {
			blind := get(uPlain, path)
			assert.NotContains(t, blind, f.stack)
			assert.NotContains(t, blind, "STRIPE_SECRET_KEY")
			assert.Contains(t, blind, `"service":"done"`)

			sighted := get(uReader, path)
			assert.Contains(t, sighted, f.stack)
			assert.Contains(t, sighted, "STRIPE_SECRET_KEY")
		})
	}
}
