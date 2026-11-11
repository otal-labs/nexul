package connectors

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/platform/secretref"
	"github.com/otal-labs/nexul/internal/platform/secretref/secretreftest"
)

func TestStatus_ASecretThatFailsCarriesAFixedReasonNeverTheManagersText(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{"missing item", secretref.ErrNotFound, "not found"},
		{"missing field", secretref.ErrFieldMissing, "field not found"},
		{"no access", secretref.ErrNoAccess, "no access"},
		{"the manager's own text", errors.New("sdk: item 7f3a in vault Acme returned status 500"), "could not be resolved"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &secretreftest.Fake{Errs: map[string]error{secretRef: tt.err}}
			svc, store := serviceWithSecrets(t, &fakeVerifier{}, secretsOver(t, serving(fake)))
			store.rows["livekit"] = Credentials{ConnectorID: "livekit", ManualFields: livekitFields(secretRef)}

			list, err := svc.List(t.Context())

			require.NoError(t, err)
			for _, c := range list {
				if c.Connector.ID == "livekit" {
					assert.Equal(t, &SecretProblem{Manager: "op", Error: tt.want}, c.Status.Secret)
				}
			}
		})
	}
}

func TestSaveManualCredentials_RefusalNamesTheManagerByItsNameAndOnlyAFixedReason(t *testing.T) {
	fake := &secretreftest.Fake{Errs: map[string]error{secretRef: errors.New("sdk: vault Acme item 7f3a denied")}}
	svc, _ := serviceWithSecrets(t, &fakeVerifier{}, secretsOver(t, serving(fake)))

	_, err := svc.SaveManualCredentials(t.Context(), "livekit", "u1", livekitFields(secretRef))

	require.Error(t, err)
	assert.ErrorContains(t, err, "could not be resolved")
	assert.NotContains(t, err.Error(), "7f3a")
}
