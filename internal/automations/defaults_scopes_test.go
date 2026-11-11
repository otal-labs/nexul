package automations_test

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/automations"
	"github.com/otal-labs/nexul/internal/integrations"
)

// The seeder stores a default's scopes as written, so each list must already hold the reads its writes imply.
func TestDefaultDefinitions_ScopesHoldTheReadsTheirWritesImply(t *testing.T) {
	for _, def := range automations.DefaultDefinitions() {
		resolved, err := integrations.ResolveScopes(def.Scopes)
		require.NoError(t, err, def.ID)
		slices.Sort(resolved)
		want := slices.Clone(def.Scopes)
		slices.Sort(want)
		assert.Equal(t, resolved, want, def.ID)
	}
}
