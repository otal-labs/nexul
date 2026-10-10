package repository

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrors "github.com/otal-labs/nexul/internal/platform/errors"
)

const rootCompose = `
services:
  web:
    image: nginx
    ports:
      - "8080:80"
`

const rootDockerfile = "FROM go\nEXPOSE 3000\n"

func TestScan_RequiresOwnerAndName(t *testing.T) {
	_, err := Scan(context.Background(), nil, &fakeScanner{}, "", "app", "")
	assert.ErrorIs(t, err, apperrors.ErrInvalid)

	_, err = Scan(context.Background(), nil, &fakeScanner{}, "acme", "", "")
	assert.ErrorIs(t, err, apperrors.ErrInvalid)
}

func TestScan_NoCandidates(t *testing.T) {
	s := &fakeScanner{
		resolvedRef: "main",
		tree: []TreeEntry{
			{Path: "README.md", Type: "blob"},
			{Path: "src", Type: "tree"},
			{Path: "src/main.go", Type: "blob"},
		},
	}
	result, err := Scan(context.Background(), nil, s, "acme", "app", "")
	require.NoError(t, err)
	assert.Equal(t, "main", result.DefaultBranch)
	assert.Empty(t, result.Candidates)
	assert.Empty(t, result.EnvKeys)
}

func TestScan_ComposeSortsBeforeDockerfile(t *testing.T) {
	s := &fakeScanner{
		resolvedRef: "main",
		tree: []TreeEntry{
			{Path: "Dockerfile", Type: "blob"},
			{Path: "docker-compose.yml", Type: "blob"},
		},
		files: map[string][]byte{
			"docker-compose.yml": []byte(rootCompose),
			"Dockerfile":         []byte(rootDockerfile),
		},
	}
	result, err := Scan(context.Background(), nil, s, "acme", "app", "")
	require.NoError(t, err)
	require.Len(t, result.Candidates, 2)
	assert.Equal(t, KindCompose, result.Candidates[0].Kind)
	assert.Equal(t, "docker-compose.yml", result.Candidates[0].Path)
	assert.Equal(t, "app", result.Candidates[0].Name)
	require.NotNil(t, result.Candidates[0].Reachable)
	assert.Equal(t, "web", result.Candidates[0].Reachable.Service)
	assert.Equal(t, 80, result.Candidates[0].Reachable.Port)

	assert.Equal(t, KindDockerfile, result.Candidates[1].Kind)
	assert.Equal(t, "Dockerfile", result.Candidates[1].Path)
	require.Len(t, result.Candidates[1].Services, 1)
	assert.Equal(t, []int{3000}, result.Candidates[1].Services[0].Expose)
}

func TestScan_SkipsVendorNodeModulesAndGit(t *testing.T) {
	s := &fakeScanner{
		resolvedRef: "main",
		tree: []TreeEntry{
			{Path: "vendor/pkg/Dockerfile", Type: "blob"},
			{Path: "node_modules/dep/docker-compose.yml", Type: "blob"},
			{Path: ".git/hooks/Dockerfile", Type: "blob"},
		},
	}
	result, err := Scan(context.Background(), nil, s, "acme", "app", "")
	require.NoError(t, err)
	assert.Empty(t, result.Candidates)
}

func TestScan_NestedComposeNameIncludesDirectory(t *testing.T) {
	s := &fakeScanner{
		resolvedRef: "main",
		tree: []TreeEntry{
			{Path: "services/web/docker-compose.yml", Type: "blob"},
		},
		files: map[string][]byte{
			"services/web/docker-compose.yml": []byte(rootCompose),
		},
	}
	result, err := Scan(context.Background(), nil, s, "acme", "app", "")
	require.NoError(t, err)
	require.Len(t, result.Candidates, 1)
	assert.Equal(t, "app/services/web", result.Candidates[0].Name)
}

func TestScan_EnvKeysMergedAndDedupedAcrossFiles(t *testing.T) {
	s := &fakeScanner{
		resolvedRef: "main",
		tree: []TreeEntry{
			{Path: ".env.example", Type: "blob"},
			{Path: "api/.env.sample", Type: "blob"},
		},
		files: map[string][]byte{
			".env.example":    []byte("FOO=bar\nSHARED=1\n"),
			"api/.env.sample": []byte("SHARED=2\nBAZ=qux\n"),
		},
	}
	result, err := Scan(context.Background(), nil, s, "acme", "app", "")
	require.NoError(t, err)
	assert.Equal(t, []string{"FOO", "SHARED", "BAZ"}, result.EnvKeys)
}

func TestScan_MultipleComposeFilesAreAllCandidates(t *testing.T) {
	s := &fakeScanner{
		resolvedRef: "main",
		tree: []TreeEntry{
			{Path: "docker-compose.yml", Type: "blob"},
			{Path: "backend/compose.prod.yaml", Type: "blob"},
		},
		files: map[string][]byte{
			"docker-compose.yml":        []byte(rootCompose),
			"backend/compose.prod.yaml": []byte(rootCompose),
		},
	}
	result, err := Scan(context.Background(), nil, s, "acme", "app", "")
	require.NoError(t, err)
	require.Len(t, result.Candidates, 2)
	assert.Equal(t, "backend/compose.prod.yaml", result.Candidates[0].Path)
	assert.Equal(t, "docker-compose.yml", result.Candidates[1].Path)
}

func TestScan_GetTreeErrorPropagates(t *testing.T) {
	s := &fakeScanner{treeErr: apperrors.ErrNotFound}
	_, err := Scan(context.Background(), nil, s, "acme", "app", "")
	assert.ErrorIs(t, err, apperrors.ErrNotFound)
}

func TestScan_GetFileErrorPropagates(t *testing.T) {
	s := &fakeScanner{
		resolvedRef: "main",
		tree:        []TreeEntry{{Path: "docker-compose.yml", Type: "blob"}},
		fileErr:     errors.New("boom"),
	}
	_, err := Scan(context.Background(), nil, s, "acme", "app", "")
	require.Error(t, err)
}

func TestScan_InvalidComposeYAMLIsNotACandidate(t *testing.T) {
	s := &fakeScanner{
		resolvedRef: "main",
		tree:        []TreeEntry{{Path: "docker-compose.yml", Type: "blob"}},
		files:       map[string][]byte{"docker-compose.yml": []byte("services: [this is not")},
	}
	result, err := Scan(context.Background(), nil, s, "acme", "app", "")
	require.NoError(t, err)
	assert.Empty(t, result.Candidates)
}

func TestListRepos(t *testing.T) {
	t.Run("returns the scanner's repos", func(t *testing.T) {
		s := &fakeScanner{repos: []Repo{{Owner: "acme", Name: "app"}}}
		repos, err := newTestService(s, nil).ListRepos(context.Background(), "", "", false)
		require.NoError(t, err)
		assert.Equal(t, []Repo{{Owner: "acme", Name: "app"}}, repos)
	})
	t.Run("q keeps repositories whose full name contains it, ignoring case", func(t *testing.T) {
		s := &fakeScanner{repos: []Repo{{FullName: "acme/api"}, {FullName: "Acme/Web-App"}, {FullName: "other/tool"}}}
		repos, err := newTestService(s, nil).ListRepos(context.Background(), "", " ACME/w ", false)
		require.NoError(t, err)
		assert.Equal(t, []Repo{{FullName: "Acme/Web-App"}}, repos)

		repos, err = newTestService(s, nil).ListRepos(context.Background(), "", "acme", false)
		require.NoError(t, err)
		assert.Len(t, repos, 2)
	})
	t.Run("q with no match is an empty list", func(t *testing.T) {
		s := &fakeScanner{repos: []Repo{{FullName: "acme/api"}}}
		repos, err := newTestService(s, nil).ListRepos(context.Background(), "", "zzz", false)
		require.NoError(t, err)
		assert.NotNil(t, repos)
		assert.Empty(t, repos)
	})
	t.Run("q shorter than three characters is invalid and never reaches the provider", func(t *testing.T) {
		s := &fakeScanner{repos: []Repo{{FullName: "acme/api"}}}
		_, err := newTestService(s, nil).ListRepos(context.Background(), "", " ap ", false)
		require.ErrorIs(t, err, apperrors.ErrInvalid)
		assert.Zero(t, s.listCalls)
	})
	t.Run("propagates the scanner's error", func(t *testing.T) {
		s := &fakeScanner{listErr: apperrors.ErrUnauthorized}
		_, err := newTestService(s, nil).ListRepos(context.Background(), "", "", false)
		assert.ErrorIs(t, err, apperrors.ErrUnauthorized)
	})
}

func TestScan_SkipsUnparsableComposeFile(t *testing.T) {
	s := &fakeScanner{
		resolvedRef: "main",
		tree: []TreeEntry{
			{Path: "broken/docker-compose.yml", Type: "blob"},
			{Path: "docker-compose.yml", Type: "blob"},
		},
		files: map[string][]byte{
			"broken/docker-compose.yml": []byte("services: [\n  not: yaml"),
			"docker-compose.yml":        []byte(rootCompose),
		},
	}
	result, err := Scan(context.Background(), nil, s, "acme", "app", "")
	require.NoError(t, err)
	require.Len(t, result.Candidates, 1)
	assert.Equal(t, "docker-compose.yml", result.Candidates[0].Path)
}
