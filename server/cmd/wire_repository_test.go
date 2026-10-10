package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/auth"
	"github.com/otal-labs/nexul/internal/connectors"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/repository"
)

// fakeGitHubTokens is auth's person-token read: each person's own token, or the not-connected error.
type fakeGitHubTokens map[string]string

func (f fakeGitHubTokens) GitHubToken(_ context.Context, userID string) (string, error) {
	tok, ok := f[userID]
	if !ok {
		return "", auth.ErrGitHubNotConnected
	}
	return tok, nil
}

// newTestPersonView is Alice's GitHub view, read with her own token against h.
func newTestPersonView(t *testing.T, h http.Handler, appSlug string) repository.GitHubView {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	people := newGitHubPeople(fakeGitHubTokens{"alice": "ghu_alice"}, &fakeAppConfigStore{cfgs: map[string]connectors.AppConfig{
		"github": {ConnectorID: "github", BaseURL: srv.URL, AppSlug: appSlug},
	}})
	v, err := people.GitHubView(t.Context(), "alice")
	require.NoError(t, err)
	return v
}

func repoJSONFixture(owner, name, defaultBranch string) string {
	return fmt.Sprintf(
		`{"id":1,"name":%q,"full_name":%q,"default_branch":%q,"html_url":"https://github.com/%s/%s","owner":{"login":%q}}`,
		name, owner+"/"+name, defaultBranch, owner, name, owner,
	)
}

func TestPersonView_Repos(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/user/installations", func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprintln(w, `{"installations":[{"id":10}]}`) // test server: write errors are irrelevant
	})
	mux.HandleFunc("/user/installations/10/repositories", func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprintln(w, `{"repositories":[`+repoJSONFixture("acme", "app", "main")+`]}`) // test server: write errors are irrelevant
	})
	s := newTestPersonView(t, mux, "my-app")

	repos, err := s.Repos(context.Background(), false)
	require.NoError(t, err)
	require.Len(t, repos, 1)
	assert.Equal(t, "acme/app", repos[0].FullName)
	assert.Equal(t, "github", repos[0].Provider)
}

func TestPersonView_Repos_Cache(t *testing.T) {
	newScanner := func(t *testing.T, walks *atomic.Int32, failFirst bool) repository.GitHubView {
		mux := http.NewServeMux()
		mux.HandleFunc("/user/installations", func(w http.ResponseWriter, r *http.Request) {
			if walks.Add(1) == 1 && failFirst {
				http.Error(w, `{"message":"boom"}`, http.StatusInternalServerError)
				return
			}
			_, _ = fmt.Fprintln(w, `{"installations":[]}`) // test server: write errors are irrelevant
		})
		return newTestPersonView(t, mux, "my-app")
	}

	t.Run("a second search is served without walking GitHub again, until a refresh", func(t *testing.T) {
		var walks atomic.Int32
		s := newScanner(t, &walks, false)

		for range 2 {
			_, err := s.Repos(t.Context(), false)
			require.NoError(t, err)
		}
		assert.EqualValues(t, 1, walks.Load())

		_, err := s.Repos(t.Context(), true)
		require.NoError(t, err)
		assert.EqualValues(t, 2, walks.Load())
	})

	t.Run("a failed walk is not remembered", func(t *testing.T) {
		var walks atomic.Int32
		s := newScanner(t, &walks, true)

		_, err := s.Repos(t.Context(), false)
		require.Error(t, err)
		_, err = s.Repos(t.Context(), false)
		require.NoError(t, err)
		assert.EqualValues(t, 2, walks.Load())
	})
}

func TestPersonView_GitHubRefusingTheTokenIsNotASignedOutSession(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/user/installations", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"message":"Bad credentials"}`, http.StatusUnauthorized)
	})
	s := newTestPersonView(t, mux, "my-app")

	_, reposErr := s.Repos(context.Background(), false)
	_, installsErr := s.Installations(context.Background())
	for _, err := range []error{reposErr, installsErr} {
		require.ErrorIs(t, err, apperrs.ErrForbidden)
		assert.NotErrorIs(t, err, apperrs.ErrUnauthorized)
		assert.Contains(t, err.Error(), "reconnect GitHub")
	}
}

func TestPersonView_GetTree(t *testing.T) {
	t.Run("empty ref resolves and reports the default branch", func(t *testing.T) {
		mux := http.NewServeMux()
		mux.HandleFunc("/repos/acme/app", func(w http.ResponseWriter, r *http.Request) {
			_, _ = fmt.Fprintln(w, repoJSONFixture("acme", "app", "main")) // test server: write errors are irrelevant
		})
		mux.HandleFunc("/repos/acme/app/git/trees/main", func(w http.ResponseWriter, r *http.Request) {
			_, _ = fmt.Fprintln(w, `{"sha":"abc","tree":[{"path":"Dockerfile","type":"blob"}]}`) // test server: write errors are irrelevant
		})
		s := newTestPersonView(t, mux, "my-app")

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
		s := newTestPersonView(t, mux, "my-app")

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
		s := newTestPersonView(t, mux, "my-app")

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
		s := newTestPersonView(t, mux, "")

		_, _, err := s.GetTree(context.Background(), "acme", "app", "")
		require.Error(t, err)
		assert.ErrorIs(t, err, apperrs.ErrNotFound)
		assert.NotContains(t, err.Error(), "installations/new")
	})

	t.Run("a person with no GitHub link gets no view and GitHub is never asked", func(t *testing.T) {
		var asked atomic.Int32
		srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { asked.Add(1) }))
		t.Cleanup(srv.Close)
		people := newGitHubPeople(fakeGitHubTokens{}, &fakeAppConfigStore{cfgs: map[string]connectors.AppConfig{"github": {BaseURL: srv.URL}}})
		_, err := people.GitHubView(t.Context(), "carol")
		require.ErrorIs(t, err, auth.ErrGitHubNotConnected)
		assert.Zero(t, asked.Load(), "no other credential is tried in its place")
	})
}

func TestPersonView_GetFile(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mux := http.NewServeMux()
		mux.HandleFunc("/repos/acme/app/contents/Dockerfile", func(w http.ResponseWriter, r *http.Request) {
			_, _ = fmt.Fprintln(w, `{"type":"file","encoding":"base64","content":"RlJPTSBnbw==","name":"Dockerfile","path":"Dockerfile"}`) // test server: write errors are irrelevant
		})
		s := newTestPersonView(t, mux, "my-app")

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
		s := newTestPersonView(t, mux, "my-app")

		_, err := s.GetFile(context.Background(), "acme", "app", "main", "missing.yml")
		assert.ErrorIs(t, err, apperrs.ErrNotFound)
	})
}
