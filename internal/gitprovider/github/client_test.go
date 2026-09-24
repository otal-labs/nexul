package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	githubapi "github.com/google/go-github/v71/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/gitprovider"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

func newTestClient(t *testing.T, h http.Handler) *Client {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	u, err := url.Parse(srv.URL)
	require.NoError(t, err)
	return New("test-token", WithBaseURL(u))
}

func TestNew_EmptyTokenSendsNoAuthorizationHeader(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "" {
			t.Errorf("expected no Authorization header, got %q", r.Header.Get("Authorization"))
		}
		_, _ = fmt.Fprint(w, repoJSON("acme", "app")) // test server: write errors are irrelevant
	}))
	t.Cleanup(srv.Close)
	u, err := url.Parse(srv.URL)
	require.NoError(t, err)

	c := New("", WithBaseURL(u))
	repo, err := c.GetRepo(context.Background(), "acme", "app")
	require.NoError(t, err)
	assert.Equal(t, "acme/app", repo.FullName)
}

func repoJSON(owner, name string) string {
	return fmt.Sprintf(
		`{"id":123,"name":%q,"full_name":%q,"default_branch":"main","html_url":"https://github.com/%s/%s","owner":{"login":%q}}`,
		name, owner+"/"+name, owner, name, owner,
	)
}

func prJSON(number int, title, body, state string, merged bool) string {
	return fmt.Sprintf(
		`{"number":%d,"title":%q,"body":%q,"state":%q,"merged":%t,"head":{"ref":"feature/fix","sha":"abc123"},"base":{"ref":"main"},"user":{"login":"onik97"}}`,
		number, title, body, state, merged,
	)
}

func errorHandler(status int) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(status)
		_, _ = fmt.Fprintln(w, `{"message":"api error","documentation_url":"https://docs.github.com"}`) // test server: write errors are irrelevant
	}
}

func assertErrorIs(t *testing.T, err error, target error) {
	t.Helper()
	require.Error(t, err)
	assert.ErrorIs(t, err, target)
}

func TestGetRepo(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/repos/acme/app", r.URL.Path)
			_, _ = fmt.Fprintln(w, repoJSON("acme", "app")) // test server: write errors are irrelevant
		}))
		repo, err := c.GetRepo(context.Background(), "acme", "app")
		require.NoError(t, err)
		assert.Equal(t, "app", repo.Name)
		assert.Equal(t, "acme", repo.Owner)
		assert.Equal(t, "acme/app", repo.FullName)
		assert.Equal(t, "main", repo.DefaultBranch)
	})
	t.Run("not found maps to ErrNotFound", func(t *testing.T) {
		c := newTestClient(t, errorHandler(http.StatusNotFound))
		_, err := c.GetRepo(context.Background(), "acme", "app")
		assertErrorIs(t, err, apperrs.ErrNotFound)
	})
	t.Run("unauthorized maps to ErrUnauthorized", func(t *testing.T) {
		c := newTestClient(t, errorHandler(http.StatusUnauthorized))
		_, err := c.GetRepo(context.Background(), "acme", "app")
		assertErrorIs(t, err, apperrs.ErrUnauthorized)
	})
	t.Run("server error is retryable", func(t *testing.T) {
		c := newTestClient(t, errorHandler(http.StatusInternalServerError))
		_, err := c.GetRepo(context.Background(), "acme", "app")
		assertErrorIs(t, err, apperrs.ErrRetryable)
	})
	t.Run("unmapped status returns the raw error", func(t *testing.T) {
		c := newTestClient(t, errorHandler(http.StatusGone))
		_, err := c.GetRepo(context.Background(), "acme", "app")
		require.Error(t, err)
		assert.NotErrorIs(t, err, apperrs.ErrNotFound)
		assert.NotErrorIs(t, err, apperrs.ErrRetryable)
	})
}

func TestListPRs(t *testing.T) {
	t.Run("defaults to open state", func(t *testing.T) {
		var gotQuery string
		c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/repos/acme/app/pulls", r.URL.Path)
			gotQuery = r.URL.RawQuery
			_, _ = fmt.Fprintln(w, "["+prJSON(1, "Fix a", "Fixes #42", "open", false)+","+prJSON(2, "Fix b", "", "open", false)+"]") // test server: write errors are irrelevant
		}))
		prs, err := c.ListPRs(context.Background(), "acme", "app", gitprovider.PROpts{})
		require.NoError(t, err)
		require.Len(t, prs, 2)
		assert.Contains(t, gotQuery, "state=open")
		assert.NotContains(t, gotQuery, "per_page")
		assert.Equal(t, "Fix a", prs[0].Title)
		assert.Equal(t, []string{"42"}, prs[0].LinkedTicketIDs)
		assert.Empty(t, prs[1].LinkedTicketIDs)
	})
	t.Run("honors state and limit", func(t *testing.T) {
		var gotQuery string
		c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotQuery = r.URL.RawQuery
			_, _ = fmt.Fprintln(w, "[]") // test server: write errors are irrelevant
		}))
		_, err := c.ListPRs(context.Background(), "acme", "app", gitprovider.PROpts{State: "closed", Limit: 25})
		require.NoError(t, err)
		assert.Contains(t, gotQuery, "state=closed")
		assert.Contains(t, gotQuery, "per_page=25")
	})
	t.Run("not found maps to ErrNotFound", func(t *testing.T) {
		c := newTestClient(t, errorHandler(http.StatusNotFound))
		_, err := c.ListPRs(context.Background(), "acme", "app", gitprovider.PROpts{})
		assertErrorIs(t, err, apperrs.ErrNotFound)
	})
}

func TestGetPR(t *testing.T) {
	t.Run("success parses body tickets", func(t *testing.T) {
		c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/repos/acme/app/pulls/7", r.URL.Path)
			_, _ = fmt.Fprintln(w, prJSON(7, "Fix login", "Refs TICKET-42", "open", false)) // test server: write errors are irrelevant
		}))
		pr, err := c.GetPR(context.Background(), "acme", "app", 7)
		require.NoError(t, err)
		assert.Equal(t, 7, pr.Number)
		assert.Equal(t, "Fix login", pr.Title)
		assert.Equal(t, []string{"42"}, pr.LinkedTicketIDs)
		assert.Equal(t, "abc123", pr.HeadSHA)
		assert.Equal(t, "main", pr.BaseBranch)
		assert.Equal(t, "onik97", pr.Author)
	})
	t.Run("not found maps to ErrNotFound", func(t *testing.T) {
		c := newTestClient(t, errorHandler(http.StatusNotFound))
		_, err := c.GetPR(context.Background(), "acme", "app", 7)
		assertErrorIs(t, err, apperrs.ErrNotFound)
	})
}

func TestPRsForCommit(t *testing.T) {
	t.Run("lists the PRs carrying the commit", func(t *testing.T) {
		c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/repos/acme/app/commits/abc123/pulls", r.URL.Path)
			_, _ = fmt.Fprintln(w, "["+prJSON(7, "Fix login", "", "closed", true)+"]") // test server: write errors are irrelevant
		}))
		prs, err := c.PRsForCommit(context.Background(), "acme", "app", "abc123")
		require.NoError(t, err)
		require.Len(t, prs, 1)
		assert.Equal(t, 7, prs[0].Number)
	})
	t.Run("not found maps to ErrNotFound", func(t *testing.T) {
		c := newTestClient(t, errorHandler(http.StatusNotFound))
		_, err := c.PRsForCommit(context.Background(), "acme", "app", "abc123")
		assertErrorIs(t, err, apperrs.ErrNotFound)
	})
}

func TestCreateWebhook(t *testing.T) {
	t.Run("success returns hook id", func(t *testing.T) {
		c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodPost, r.Method)
			assert.Equal(t, "/repos/acme/app/hooks", r.URL.Path)
			var hook githubapi.Hook
			require.NoError(t, json.NewDecoder(r.Body).Decode(&hook))
			assert.Equal(t, []string{"pull_request"}, hook.Events)
			assert.Equal(t, "https://example.com/hook", hook.Config.GetURL())
			assert.Equal(t, "json", hook.Config.GetContentType())
			assert.Equal(t, "s3cret", hook.Config.GetSecret())
			_, _ = fmt.Fprintln(w, `{"id":99}`) // test server: write errors are irrelevant
		}))
		id, err := c.CreateWebhook(context.Background(), "acme", "app", gitprovider.WebhookConfig{
			URL: "https://example.com/hook", Secret: "s3cret",
		})
		require.NoError(t, err)
		assert.Equal(t, "99", id)
	})
	t.Run("validation failure maps to ErrConflict", func(t *testing.T) {
		c := newTestClient(t, errorHandler(http.StatusUnprocessableEntity))
		_, err := c.CreateWebhook(context.Background(), "acme", "app", gitprovider.WebhookConfig{URL: "https://example.com/hook"})
		assertErrorIs(t, err, apperrs.ErrConflict)
	})
	t.Run("unauthorized maps to ErrUnauthorized", func(t *testing.T) {
		c := newTestClient(t, errorHandler(http.StatusForbidden))
		_, err := c.CreateWebhook(context.Background(), "acme", "app", gitprovider.WebhookConfig{URL: "https://example.com/hook"})
		assertErrorIs(t, err, apperrs.ErrUnauthorized)
	})
}

func TestDeleteWebhook(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodDelete, r.Method)
			assert.Equal(t, "/repos/acme/app/hooks/42", r.URL.Path)
			w.WriteHeader(http.StatusNoContent)
		}))
		require.NoError(t, c.DeleteWebhook(context.Background(), "acme", "app", "42"))
	})
	t.Run("non-numeric id is invalid", func(t *testing.T) {
		c := newTestClient(t, errorHandler(http.StatusOK))
		err := c.DeleteWebhook(context.Background(), "acme", "app", "abc")
		assertErrorIs(t, err, apperrs.ErrInvalid)
	})
	t.Run("not found maps to ErrNotFound", func(t *testing.T) {
		c := newTestClient(t, errorHandler(http.StatusNotFound))
		err := c.DeleteWebhook(context.Background(), "acme", "app", "42")
		assertErrorIs(t, err, apperrs.ErrNotFound)
	})
}

func TestListInstallationRepos(t *testing.T) {
	t.Run("lists repos across every installation, paginating both levels", func(t *testing.T) {
		var installPages, repoPages []string
		mux := http.NewServeMux()
		mux.HandleFunc("/user/installations", func(w http.ResponseWriter, r *http.Request) {
			installPages = append(installPages, r.URL.Query().Get("page"))
			if r.URL.Query().Get("page") == "2" {
				_, _ = fmt.Fprintln(w, `{"installations":[{"id":20}]}`) // test server: write errors are irrelevant
				return
			}
			w.Header().Set("Link", `<http://x/user/installations?page=2>; rel="next"`)
			_, _ = fmt.Fprintln(w, `{"installations":[{"id":10}]}`) // test server: write errors are irrelevant
		})
		mux.HandleFunc("/user/installations/10/repositories", func(w http.ResponseWriter, r *http.Request) {
			repoPages = append(repoPages, "10:"+r.URL.Query().Get("page"))
			if r.URL.Query().Get("page") == "2" {
				_, _ = fmt.Fprintln(w, `{"repositories":[`+repoJSON("acme", "beta")+`]}`) // test server: write errors are irrelevant
				return
			}
			w.Header().Set("Link", `<http://x/user/installations/10/repositories?page=2>; rel="next"`)
			_, _ = fmt.Fprintln(w, `{"repositories":[`+repoJSON("acme", "alpha")+`]}`) // test server: write errors are irrelevant
		})
		mux.HandleFunc("/user/installations/20/repositories", func(w http.ResponseWriter, r *http.Request) {
			repoPages = append(repoPages, "20:"+r.URL.Query().Get("page"))
			_, _ = fmt.Fprintln(w, `{"repositories":[`+repoJSON("acme", "gamma")+`]}`) // test server: write errors are irrelevant
		})
		c := newTestClient(t, mux)

		repos, err := c.ListInstallationRepos(context.Background())
		require.NoError(t, err)
		require.Len(t, repos, 3)
		var names []string
		for _, r := range repos {
			names = append(names, r.Name)
		}
		assert.ElementsMatch(t, []string{"alpha", "beta", "gamma"}, names)
		assert.Len(t, installPages, 2)
		assert.ElementsMatch(t, []string{"10:1", "10:2", "20:1"}, repoPages)
	})
	t.Run("unauthorized maps to ErrUnauthorized", func(t *testing.T) {
		mux := http.NewServeMux()
		mux.HandleFunc("/user/installations", errorHandler(http.StatusUnauthorized))
		c := newTestClient(t, mux)
		_, err := c.ListInstallationRepos(context.Background())
		assertErrorIs(t, err, apperrs.ErrUnauthorized)
	})
	t.Run("installation repos error propagates", func(t *testing.T) {
		mux := http.NewServeMux()
		mux.HandleFunc("/user/installations", func(w http.ResponseWriter, r *http.Request) {
			_, _ = fmt.Fprintln(w, `{"installations":[{"id":10}]}`) // test server: write errors are irrelevant
		})
		mux.HandleFunc("/user/installations/10/repositories", errorHandler(http.StatusForbidden))
		c := newTestClient(t, mux)
		_, err := c.ListInstallationRepos(context.Background())
		assertErrorIs(t, err, apperrs.ErrUnauthorized)
	})
}

func TestGetTree(t *testing.T) {
	t.Run("explicit ref skips resolving the default branch", func(t *testing.T) {
		var repoCalls int
		mux := http.NewServeMux()
		mux.HandleFunc("/repos/acme/app", func(w http.ResponseWriter, r *http.Request) {
			repoCalls++
			_, _ = fmt.Fprintln(w, repoJSON("acme", "app")) // test server: write errors are irrelevant
		})
		mux.HandleFunc("/repos/acme/app/git/trees/feature", func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "1", r.URL.Query().Get("recursive"))
			_, _ = fmt.Fprintln(w, `{"sha":"abc","tree":[{"path":"Dockerfile","type":"blob"},{"path":"src","type":"tree"}]}`) // test server: write errors are irrelevant
		})
		c := newTestClient(t, mux)

		entries, err := c.GetTree(context.Background(), "acme", "app", "feature")
		require.NoError(t, err)
		require.Len(t, entries, 2)
		assert.Equal(t, gitprovider.TreeEntry{Path: "Dockerfile", Type: "blob"}, entries[0])
		assert.Equal(t, 0, repoCalls)
	})
	t.Run("empty ref resolves the default branch via GetRepo", func(t *testing.T) {
		mux := http.NewServeMux()
		mux.HandleFunc("/repos/acme/app", func(w http.ResponseWriter, r *http.Request) {
			_, _ = fmt.Fprintln(w, repoJSON("acme", "app")) // test server: write errors are irrelevant
		})
		mux.HandleFunc("/repos/acme/app/git/trees/main", func(w http.ResponseWriter, r *http.Request) {
			_, _ = fmt.Fprintln(w, `{"sha":"abc","tree":[{"path":"compose.yml","type":"blob"}]}`) // test server: write errors are irrelevant
		})
		c := newTestClient(t, mux)

		entries, err := c.GetTree(context.Background(), "acme", "app", "")
		require.NoError(t, err)
		require.Len(t, entries, 1)
		assert.Equal(t, "compose.yml", entries[0].Path)
	})
	t.Run("repo not found while resolving default branch maps to ErrNotFound", func(t *testing.T) {
		c := newTestClient(t, errorHandler(http.StatusNotFound))
		_, err := c.GetTree(context.Background(), "acme", "app", "")
		assertErrorIs(t, err, apperrs.ErrNotFound)
	})
	t.Run("not found maps to ErrNotFound", func(t *testing.T) {
		mux := http.NewServeMux()
		mux.HandleFunc("/repos/acme/app/git/trees/feature", errorHandler(http.StatusNotFound))
		c := newTestClient(t, mux)
		_, err := c.GetTree(context.Background(), "acme", "app", "feature")
		assertErrorIs(t, err, apperrs.ErrNotFound)
	})
}

func TestGetFile(t *testing.T) {
	t.Run("returns decoded content at an explicit ref", func(t *testing.T) {
		mux := http.NewServeMux()
		mux.HandleFunc("/repos/acme/app/contents/Dockerfile", func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "feature", r.URL.Query().Get("ref"))
			_, _ = fmt.Fprintln(w, `{"type":"file","encoding":"base64","content":"RlJPTSBnbw==","name":"Dockerfile","path":"Dockerfile"}`) // test server: write errors are irrelevant
		})
		c := newTestClient(t, mux)

		b, err := c.GetFile(context.Background(), "acme", "app", "feature", "Dockerfile")
		require.NoError(t, err)
		assert.Equal(t, "FROM go", string(b))
	})
	t.Run("empty ref resolves the default branch via GetRepo", func(t *testing.T) {
		mux := http.NewServeMux()
		mux.HandleFunc("/repos/acme/app", func(w http.ResponseWriter, r *http.Request) {
			_, _ = fmt.Fprintln(w, repoJSON("acme", "app")) // test server: write errors are irrelevant
		})
		mux.HandleFunc("/repos/acme/app/contents/.env.example", func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "main", r.URL.Query().Get("ref"))
			_, _ = fmt.Fprintln(w, `{"type":"file","encoding":"base64","content":"S0VZPXZhbHVl","name":".env.example","path":".env.example"}`) // test server: write errors are irrelevant
		})
		c := newTestClient(t, mux)

		b, err := c.GetFile(context.Background(), "acme", "app", "", ".env.example")
		require.NoError(t, err)
		assert.Equal(t, "KEY=value", string(b))
	})
	t.Run("directory path is invalid", func(t *testing.T) {
		mux := http.NewServeMux()
		mux.HandleFunc("/repos/acme/app/contents/src", func(w http.ResponseWriter, r *http.Request) {
			_, _ = fmt.Fprintln(w, `[{"type":"file","name":"main.go","path":"src/main.go"}]`) // test server: write errors are irrelevant
		})
		c := newTestClient(t, mux)
		_, err := c.GetFile(context.Background(), "acme", "app", "main", "src")
		assertErrorIs(t, err, apperrs.ErrInvalid)
	})
	t.Run("not found maps to ErrNotFound", func(t *testing.T) {
		mux := http.NewServeMux()
		mux.HandleFunc("/repos/acme/app/contents/missing.yml", errorHandler(http.StatusNotFound))
		c := newTestClient(t, mux)
		_, err := c.GetFile(context.Background(), "acme", "app", "main", "missing.yml")
		assertErrorIs(t, err, apperrs.ErrNotFound)
	})
}
