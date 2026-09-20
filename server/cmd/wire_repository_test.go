package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/connectors"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/workspace"
)

func newTestRepositoryScanner(t *testing.T, h http.Handler, appSlug string) repositoryScanner {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	router := gitProviderRouter{
		workspace:  &fakeGitRepoResolver{repos: map[string]workspace.RepoRef{}},
		connectors: &fakeGitTokenResolver{tokens: map[string]string{"github": "tok"}},
		appConfigs: &fakeAppConfigStore{cfgs: map[string]connectors.AppConfig{
			"github": {ConnectorID: "github", BaseURL: srv.URL, AppSlug: appSlug},
		}},
	}
	return repositoryScanner{git: router, appConfigs: router.appConfigs}
}

func repoJSONFixture(owner, name, defaultBranch string) string {
	return fmt.Sprintf(
		`{"id":1,"name":%q,"full_name":%q,"default_branch":%q,"html_url":"https://github.com/%s/%s","owner":{"login":%q}}`,
		name, owner+"/"+name, defaultBranch, owner, name, owner,
	)
}

func TestRepositoryScanner_ListInstallationRepos(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/user/installations", func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprintln(w, `{"installations":[{"id":10}]}`) // test server: write errors are irrelevant
	})
	mux.HandleFunc("/user/installations/10/repositories", func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprintln(w, `{"repositories":[`+repoJSONFixture("acme", "app", "main")+`]}`) // test server: write errors are irrelevant
	})
	s := newTestRepositoryScanner(t, mux, "my-app")

	repos, err := s.ListInstallationRepos(context.Background())
	require.NoError(t, err)
	require.Len(t, repos, 1)
	assert.Equal(t, "acme/app", repos[0].FullName)
}

func TestRepositoryScanner_GetTree(t *testing.T) {
	t.Run("empty ref resolves and reports the default branch", func(t *testing.T) {
		mux := http.NewServeMux()
		mux.HandleFunc("/repos/acme/app", func(w http.ResponseWriter, r *http.Request) {
			_, _ = fmt.Fprintln(w, repoJSONFixture("acme", "app", "main")) // test server: write errors are irrelevant
		})
		mux.HandleFunc("/repos/acme/app/git/trees/main", func(w http.ResponseWriter, r *http.Request) {
			_, _ = fmt.Fprintln(w, `{"sha":"abc","tree":[{"path":"Dockerfile","type":"blob"}]}`) // test server: write errors are irrelevant
		})
		s := newTestRepositoryScanner(t, mux, "my-app")

		resolved, entries, err := s.GetTree(context.Background(), "acme", "app", "")
		require.NoError(t, err)
		assert.Equal(t, "main", resolved)
		require.Len(t, entries, 1)
		assert.Equal(t, "Dockerfile", entries[0].Path)
	})

	t.Run("an empty repository scans as no entries", func(t *testing.T) {
		mux := http.NewServeMux()
		mux.HandleFunc("/repos/acme/app", func(w http.ResponseWriter, r *http.Request) {
			_, _ = fmt.Fprintln(w, repoJSONFixture("acme", "app", "main")) // test server: write errors are irrelevant
		})
		mux.HandleFunc("/repos/acme/app/git/trees/main", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusConflict)
			_, _ = fmt.Fprintln(w, `{"message":"Git Repository is empty."}`) // test server: write errors are irrelevant
		})
		s := newTestRepositoryScanner(t, mux, "my-app")

		resolved, entries, err := s.GetTree(context.Background(), "acme", "app", "")
		require.NoError(t, err)
		assert.Equal(t, "main", resolved)
		assert.Empty(t, entries)
	})

	t.Run("not found maps to ErrNotFound with the install URL", func(t *testing.T) {
		mux := http.NewServeMux()
		mux.HandleFunc("/repos/acme/app", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			_, _ = fmt.Fprintln(w, `{"message":"not found"}`) // test server: write errors are irrelevant
		})
		s := newTestRepositoryScanner(t, mux, "my-app")

		_, _, err := s.GetTree(context.Background(), "acme", "app", "")
		require.Error(t, err)
		assert.ErrorIs(t, err, apperrs.ErrNotFound)
		assert.Contains(t, err.Error(), "https://github.com/apps/my-app/installations/new")
	})

	t.Run("not found without an app slug omits the install URL", func(t *testing.T) {
		mux := http.NewServeMux()
		mux.HandleFunc("/repos/acme/app", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusForbidden)
			_, _ = fmt.Fprintln(w, `{"message":"forbidden"}`) // test server: write errors are irrelevant
		})
		s := newTestRepositoryScanner(t, mux, "")

		_, _, err := s.GetTree(context.Background(), "acme", "app", "")
		require.Error(t, err)
		assert.ErrorIs(t, err, apperrs.ErrNotFound)
		assert.NotContains(t, err.Error(), "installations/new")
	})

	t.Run("no live token fails without hitting the not-installed mapping", func(t *testing.T) {
		router := gitProviderRouter{
			workspace:  &fakeGitRepoResolver{repos: map[string]workspace.RepoRef{}},
			connectors: &fakeGitTokenResolver{errs: map[string]error{"github": errors.New("credentials revoked")}},
			appConfigs: &fakeAppConfigStore{cfgs: map[string]connectors.AppConfig{}},
		}
		s := repositoryScanner{git: router, appConfigs: router.appConfigs}
		_, _, err := s.GetTree(context.Background(), "acme", "app", "main")
		require.Error(t, err)
		assert.NotErrorIs(t, err, apperrs.ErrNotFound)
	})
}

func TestRepositoryScanner_GetFile(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mux := http.NewServeMux()
		mux.HandleFunc("/repos/acme/app/contents/Dockerfile", func(w http.ResponseWriter, r *http.Request) {
			_, _ = fmt.Fprintln(w, `{"type":"file","encoding":"base64","content":"RlJPTSBnbw==","name":"Dockerfile","path":"Dockerfile"}`) // test server: write errors are irrelevant
		})
		s := newTestRepositoryScanner(t, mux, "my-app")

		b, err := s.GetFile(context.Background(), "acme", "app", "main", "Dockerfile")
		require.NoError(t, err)
		assert.Equal(t, "FROM go", string(b))
	})

	t.Run("not found maps to ErrNotFound", func(t *testing.T) {
		mux := http.NewServeMux()
		mux.HandleFunc("/repos/acme/app/contents/missing.yml", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			_, _ = fmt.Fprintln(w, `{"message":"not found"}`) // test server: write errors are irrelevant
		})
		s := newTestRepositoryScanner(t, mux, "my-app")

		_, err := s.GetFile(context.Background(), "acme", "app", "main", "missing.yml")
		assert.ErrorIs(t, err, apperrs.ErrNotFound)
	})
}
