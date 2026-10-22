package t3rpc

import (
	"context"
	"fmt"
	"net/url"
	"testing"
	"time"

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
		{ID: "proj-live", Title: "My App", Path: "/home/me/app", Repository: "github.com/acme/app"},
		{ID: "proj-two", Title: "Second", Path: "/home/me/second"},
	}, projects, "deleted projects are dropped")
}

func TestListProjects_WaitsForRemotesT3ResolvesAfterTheRegistry(t *testing.T) {
	t.Parallel()
	f := t3rpctest.New(t)
	f.Protocol = 2
	f.ColdRepositories = true
	c, err := Connect(testCtx(t), f.Session(), Options{HTTPClient: f.Client(), Query: url.Values{"orchestrationProtocol": {"2"}}})
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.Close() })

	projects, err := c.ListProjects(testCtx(t))
	require.NoError(t, err)
	assert.Equal(t, []harness.Project{
		{ID: "proj-live", Title: "My App", Path: "/home/me/app", Repository: "github.com/acme/app"},
		{ID: "proj-two", Title: "Second", Path: "/home/me/second"},
	}, projects)
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

func TestCreateProject_RegistersTheFolderAsAProjectAndCreatesItWhenMissing(t *testing.T) {
	t.Parallel()
	f := t3rpctest.New(t)
	c := connectFake(t, f)

	project, err := c.CreateProject(testCtx(t), "notes", "~/Code/notes")
	require.NoError(t, err)
	sent := <-f.Mutated
	assert.Equal(t, "project.create", sent["type"])
	assert.Equal(t, "notes", sent["title"])
	assert.Equal(t, "~/Code/notes", sent["workspaceRoot"])
	assert.Equal(t, true, sent["createWorkspaceRootIfMissing"], "T3 makes the folder, Nexul has no way into the computer's disk")
	assert.NotEmpty(t, sent["commandId"])
	assert.Equal(t, harness.Project{ID: fmt.Sprint(sent["projectId"]), Title: "notes", Path: "~/Code/notes"}, project)
}

func TestCreateProject_FailsWithT3sOwnSentence(t *testing.T) {
	t.Parallel()
	f := t3rpctest.New(t)
	f.CommandCauses = map[string]any{"projects.mutate": []any{map[string]any{"_tag": "Fail", "error": map[string]any{
		"_tag": "ProjectMutationError", "message": "Project directory is not empty."}}}}
	c := connectFake(t, f)

	_, err := c.CreateProject(testCtx(t), "notes", "/home/me/notes")
	require.ErrorIs(t, err, apperrs.ErrInvalid)
	assert.Equal(t, "invalid: Project directory is not empty.", err.Error())
}

func TestCloneProject_AFailedCloneRemovesItsProjectAndReturnsGitsReason(t *testing.T) {
	t.Parallel()
	f := t3rpctest.New(t)
	f.CloneError = "fatal: could not read Username for 'https://github.com': terminal prompts disabled"
	c := connectFake(t, f)

	_, err := c.CloneProject(testCtx(t), "api", "https://github.com/acme/api.git", "/home/me/api")
	var failed *harness.CloneFailedError
	require.ErrorAs(t, err, &failed)
	assert.Equal(t, f.CloneError, failed.Detail)
	sent := <-f.Cloned
	deleted := t3rpctest.WaitFor(t, f.Mutated, "the project delete")
	assert.Equal(t, map[string]any{"type": "project.delete", "projectId": sent["projectId"], "commandId": deleted["commandId"]}, deleted)
	assert.NotEmpty(t, deleted["commandId"])
}

func TestCloneProject_AFailedCloneStillFailsWhenItsProjectCannotBeRemoved(t *testing.T) {
	t.Parallel()
	f := t3rpctest.New(t)
	f.CloneError = "Repository not found."
	f.CommandCauses = map[string]any{"projects.mutate": []any{map[string]any{"_tag": "Fail", "error": map[string]any{
		"_tag": "ProjectMutationError", "message": "Project is busy."}}}}
	c := connectFake(t, f)

	_, err := c.CloneProject(testCtx(t), "api", "https://github.com/acme/api.git", "/home/me/api")
	var failed *harness.CloneFailedError
	require.ErrorAs(t, err, &failed)
	assert.Equal(t, "Repository not found.", failed.Detail)
}

func TestCloneProject_ACloneT3NoLongerListsHasFinished(t *testing.T) {
	t.Parallel()
	f := t3rpctest.New(t)
	f.CloneRetired = true
	c := connectFake(t, f)
	ctx, cancel := context.WithTimeout(testCtx(t), time.Second)
	defer cancel()

	project, err := c.CloneProject(ctx, "api", "https://github.com/acme/api.git", "/home/me/api")
	require.NoError(t, err)
	assert.Equal(t, "api", project.Title)
}
