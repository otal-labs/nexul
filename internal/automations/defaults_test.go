package automations

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultDefinitions(t *testing.T) {
	defs := DefaultDefinitions()
	require.Len(t, defs, 2, "the board pair")

	seen := map[string]bool{}
	for _, def := range defs {
		assert.NotEmpty(t, def.ID)
		assert.False(t, seen[def.ID], "duplicate default id %s", def.ID)
		seen[def.ID] = true
		assert.NotEmpty(t, def.Name)
		assert.NotEmpty(t, def.Description)
		assert.NotEmpty(t, def.Code, "bundle must be embedded, run automations/defaults/build.ts if empty")
		assert.Contains(t, def.Code, "export {")
		assert.NotEmpty(t, def.Scopes)
		assert.True(t, def.Enabled, "the board pair ships enabled out of the box")
	}
}
