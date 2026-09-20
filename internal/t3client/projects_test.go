package t3client

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/harness"
)

func TestListProjects_ReturnsLiveRegistryFromShellSnapshot(t *testing.T) {
	t.Parallel()
	f := newFakeT3(t)
	c := f.connect(t, testCtx(t))

	projects, err := c.ListProjects(testCtx(t))
	require.NoError(t, err)
	assert.Equal(t, []harness.Project{
		{ID: "proj-live", Title: "My App"},
		{ID: "proj-two", Title: "Second"},
	}, projects, "deleted projects are dropped")
}
