package deploy

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

func testsRepoStack() Stack {
	stack := validStack()
	stack.BuildSource = &BuildSource{RepoOwner: "acme", RepoName: "e2e", Branch: "main", ComposePath: "docker-compose.yml"}
	return stack
}

func testsRepoProjects() *fakeProjects {
	projects := newFakeProjects()
	projects.exists["proj-1"] = true
	projects.repos["proj-1"] = []string{"acme/e2e"}
	projects.tests["acme/e2e"] = true
	return projects
}

func TestCreateStack_FromTestsRepository_IsInvalid(t *testing.T) {
	stacks := newFakeStackRepo()
	s := newTestServiceWith(newFakeRepo(), stacks, newFakeContainerRepo(), testsRepoProjects(), newFakeBus())
	stack := testsRepoStack()
	stack.ID = ""

	_, err := s.CreateStack(t.Context(), stack, nil)

	require.ErrorIs(t, err, apperrs.ErrInvalid)
	assert.Contains(t, err.Error(), "tests repository")
}

func TestCreateStack_TestsRepositoryLookupFails_ReturnsError(t *testing.T) {
	projects := testsRepoProjects()
	projects.testsErr = errors.New("db down")
	s := newTestServiceWith(newFakeRepo(), newFakeStackRepo(), newFakeContainerRepo(), projects, newFakeBus())
	stack := testsRepoStack()
	stack.ID = ""

	_, err := s.CreateStack(t.Context(), stack, nil)

	require.ErrorIs(t, err, projects.testsErr)
}

func TestDeploy_StackBuildingFromTestsRepository_IsRefused(t *testing.T) {
	repo := newFakeRepo()
	stacks := newFakeStackRepo()
	requireStack(t, stacks, testsRepoStack())
	s := newTestServiceWith(repo, stacks, newFakeContainerRepo(), testsRepoProjects(), newFakeBus())

	_, err := s.Deploy(t.Context(), DeployRequest{StackID: "svc-1", Ref: "main"})

	require.ErrorIs(t, err, apperrs.ErrInvalid)
	assert.Empty(t, repo.stored)
}

func TestDeploy_AppRepository_StillDeploys(t *testing.T) {
	projects := testsRepoProjects()
	projects.tests["acme/e2e"] = false
	stacks := newFakeStackRepo()
	requireStack(t, stacks, testsRepoStack())
	s := newTestServiceWith(newFakeRepo(), stacks, newFakeContainerRepo(), projects, newFakeBus())

	d, err := s.Deploy(t.Context(), DeployRequest{StackID: "svc-1", Ref: "main"})

	require.NoError(t, err)
	assert.Equal(t, KindBuild, d.Kind)
}
