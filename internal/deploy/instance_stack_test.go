package deploy

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

func TestCreateStackWithOptions_RepositoryWithoutProject_IsInvalidAndLinksNothing(t *testing.T) {
	projects := newFakeProjects()
	s := newTestServiceWith(newFakeRepo(), newFakeStackRepo(), newFakeContainerRepo(), projects, newFakeBus())
	stack := validStack()
	stack.ID, stack.ProjectID = "", ""
	stack.BuildSource = &BuildSource{RepoOwner: "acme", RepoName: "api", Branch: "main"}

	_, _, err := s.CreateStackWithOptions(t.Context(), stack, nil, CreateStackOptions{LinkRepository: true})

	require.ErrorIs(t, err, apperrs.ErrInvalid)
	assert.Empty(t, projects.repos, "a repository is never linked to no project")
}

func TestCreateStack_InstanceStack_NeedsNoProject(t *testing.T) {
	stacks := newFakeStackRepo()
	s := newTestServiceWith(newFakeRepo(), stacks, newFakeContainerRepo(), newFakeProjects(), newFakeBus())
	stack := validStack()
	stack.ID, stack.ProjectID = "", ""

	created, err := s.CreateStack(t.Context(), stack, nil)

	require.NoError(t, err)
	assert.Empty(t, created.ProjectID)
	listed, err := s.ListStacks(t.Context(), "")
	require.NoError(t, err)
	require.Len(t, listed, 1, "every-stack listing includes the instance's own")
}
