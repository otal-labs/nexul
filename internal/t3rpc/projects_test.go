package t3rpc

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/harness"
	"github.com/otal-labs/nexul/internal/t3rpc/t3rpctest"
)

func TestListProjects_ReturnsLiveRegistryFromShellSnapshot(t *testing.T) {
	t.Parallel()
	c := connectFake(t, t3rpctest.New(t))

	projects, err := c.ListProjects(testCtx(t))
	require.NoError(t, err)
	assert.Equal(t, []harness.Project{
		{ID: "proj-live", Title: "My App", Path: "/home/me/app"},
		{ID: "proj-two", Title: "Second", Path: "/home/me/second"},
	}, projects, "deleted projects are dropped")
}
