package t3rpc

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/harness"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
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

func TestCloneProject_FailsWithT3sOwnSentence(t *testing.T) {
	t.Parallel()
	f := t3rpctest.New(t)
	f.CommandCauses = map[string]any{"projectClone.start": []any{map[string]any{"_tag": "Fail", "error": map[string]any{
		"_tag": "SourceControlRepositoryError", "detail": "Destination path already exists and is not empty."}}}}
	c := connectFake(t, f)

	_, err := c.CloneProject(testCtx(t), "api", "https://github.com/acme/api.git", "/home/me/api")
	require.ErrorIs(t, err, apperrs.ErrInvalid)
	assert.Equal(t, "invalid: Destination path already exists and is not empty.", err.Error(), "the sentence alone, not the raw cause")
}

func TestCloneProject_AsksForANewPairingWhenTheSessionMayNotClone(t *testing.T) {
	t.Parallel()
	f := t3rpctest.New(t)
	f.CommandCauses = map[string]any{"projectClone.start": []any{map[string]any{"_tag": "Fail", "error": map[string]any{
		"_tag": "EnvironmentAuthorizationError", "message": "The authenticated token is missing required scope: source-control:write."}}}}
	c := connectFake(t, f)

	_, err := c.CloneProject(testCtx(t), "api", "https://github.com/acme/api.git", "/home/me/api")
	require.ErrorIs(t, err, apperrs.ErrInvalid)
	assert.Contains(t, err.Error(), "pair it again")
}

func TestCloneProject_StartsTheCloneAndReturnsTheNewProject(t *testing.T) {
	t.Parallel()
	f := t3rpctest.New(t)
	c := connectFake(t, f)

	project, err := c.CloneProject(testCtx(t), "api", "https://github.com/acme/api.git", "/home/me/api")
	require.NoError(t, err)
	sent := <-f.Cloned
	assert.Equal(t, "https://github.com/acme/api.git", sent["remoteUrl"])
	assert.Equal(t, harness.Project{ID: fmt.Sprint(sent["projectId"]), Title: "api", Path: "/home/me/api"}, project)
}
