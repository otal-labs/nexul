package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/auth"
	"github.com/otal-labs/nexul/internal/connectors"
	"github.com/otal-labs/nexul/internal/gitprovider/github"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/permissions"
	"github.com/otal-labs/nexul/internal/platform/storage"
	"github.com/otal-labs/nexul/internal/repository"
	"github.com/otal-labs/nexul/internal/workspace"
)

// projectWriter holds every action in the default workspace, as its Owner does.
type projectWriter struct{}

func (projectWriter) RequireAnywhere(context.Context, permissions.Action) error { return nil }

func (projectWriter) Require(context.Context, string, permissions.Action) error { return nil }

func (projectWriter) WorkspacesWith(context.Context, permissions.Action) ([]string, error) {
	return []string{"workspace-default"}, nil
}

// twoPeopleOnGitHub is GitHub with the App installed on Alice's acme and on Bob's own account: each person's token
// lists only their own installation, while the App, its installation tokens and the connector could read both.
type twoPeopleOnGitHub struct {
	mu       sync.Mutex
	requests []string
	mints    []string
}

func (g *twoPeopleOnGitHub) seen() []string {
	g.mu.Lock()
	defer g.mu.Unlock()
	return append([]string(nil), g.requests...)
}

func (g *twoPeopleOnGitHub) handler() http.Handler {
	people := map[string]struct{ install, repos string }{
		"Bearer ghu_alice": {`{"id":1,"account":{"id":11,"login":"acme"},"repository_selection":"all"}`, repoJSONFixture("acme", "api", "main")},
		"Bearer ghu_bob":   {`{"id":2,"account":{"id":22,"login":"bob"},"repository_selection":"all"}`, repoJSONFixture("bob", "secret", "main")},
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /user/installations", func(w http.ResponseWriter, r *http.Request) {
		p, ok := people[r.Header.Get("Authorization")]
		if !ok {
			http.Error(w, `{"message":"Bad credentials"}`, http.StatusUnauthorized)
			return
		}
		_, _ = fmt.Fprintf(w, `{"total_count":1,"installations":[%s]}`, p.install) // test server: write errors are irrelevant
	})
	mux.HandleFunc("GET /user/installations/{id}/repositories", func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprintf(w, `{"total_count":1,"repositories":[%s]}`, people[r.Header.Get("Authorization")].repos) // test server: write errors are irrelevant
	})
	mux.HandleFunc("GET /repos/{owner}/{repo}/installation", func(w http.ResponseWriter, r *http.Request) {
		id := map[string]int{"acme": 1, "bob": 2}[r.PathValue("owner")]
		_, _ = fmt.Fprintf(w, `{"id":%d,"account":{"id":%d,"login":%q}}`, id, id*11, r.PathValue("owner")) // test server: write errors are irrelevant
	})
	mux.HandleFunc("GET /app/installations", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprint(w, `[{"id":1,"account":{"id":11,"login":"acme"}},{"id":2,"account":{"id":22,"login":"bob"}}]`) // test server: write errors are irrelevant
	})
	mux.HandleFunc("POST /app/installations/{id}/access_tokens", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body) // test server: a short read fails the body assertion
		g.mu.Lock()
		g.mints = append(g.mints, string(body))
		g.mu.Unlock()
		_, _ = fmt.Fprint(w, `{"token":"ghs_clone","expires_at":"2099-01-01T00:00:00Z"}`) // test server: write errors are irrelevant
	})
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		g.mu.Lock()
		g.requests = append(g.requests, r.Method+" "+r.URL.Path+" "+r.Header.Get("Authorization"))
		g.mu.Unlock()
		mux.ServeHTTP(w, r)
	})
}

type personViewFixture struct {
	github    *twoPeopleOnGitHub
	store     *storage.Store
	repos     *repository.Service
	router    gitProviderRouter
	projects  hookedProjects
	connector *fakeGitTokenResolver
}

// newPersonViewFixture reads GitHub as the App, with Alice and Bob each holding their own token and Carol none.
func newPersonViewFixture(t *testing.T) personViewFixture {
	t.Helper()
	gh := &twoPeopleOnGitHub{}
	srv := httptest.NewServer(gh.handler())
	t.Cleanup(srv.Close)
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	pemKey := string(pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)}))
	store := storage.New(mentionsTestDB(t), []byte("0123456789abcdef0123456789abcdef"))
	cfg := &fakeAppConfigStore{cfgs: map[string]connectors.AppConfig{"github": {ConnectorID: "github", ClientID: "Iv1.acme", PrivateKey: pemKey, BaseURL: srv.URL, AppSlug: "nexul-acme"}}}
	connector := &fakeGitTokenResolver{tokens: map[string]string{"github": "tok-connector"}}
	scope := &githubInstallationScope{appConfigs: cfg, projects: store.Projects}
	router := gitProviderRouter{workspace: store.Projects, connectors: connector, appConfigs: cfg, apps: &github.AppCache{}, scope: scope}
	scanner := newRepositoryScanner(router, cfg)
	svc := repository.NewService(repository.Config{
		Gate: projectWriter{}, People: newGitHubPeople(fakeGitHubTokens{"alice": "ghu_alice", "bob": "ghu_bob"}, cfg),
		Installations: scanner, Accounts: scanner, Store: store.GitHubInstallations,
	})
	scope.repos = svc
	return personViewFixture{
		github: gh, store: store, repos: svc, router: router, connector: connector,
		projects: hookedProjects{Repo: store.Projects, scope: scope, hooks: newTestRepoWebhooks(&fakeHookHost{}, "https://nexul.example.com")},
	}
}

func TestDiscovery_ListsOnlyThePersonsOwnRepositoriesEvenWithAnotherAccountLinkedToTheirWorkspace(t *testing.T) {
	f := newPersonViewFixture(t)
	_, err := f.store.GitHubInstallations.AssignInstallation(t.Context(), repository.Assignment{AccountID: 22, AccountLogin: "bob", WorkspaceID: "workspace-default"})
	require.NoError(t, err)

	repos, err := f.repos.ListRepos(as("alice"), "workspace-default", "", true)
	require.NoError(t, err)
	assert.Equal(t, []string{"acme/api"}, repoNames(repos), "Bob's private repository never lists for Alice")
	installs, err := f.repos.ListInstallations(as("alice"))
	require.NoError(t, err)
	require.Len(t, installs, 1, "Bob's account never appears on Alice's installations card")
	assert.Equal(t, "acme", installs[0].AccountLogin)
	for _, req := range f.github.seen() {
		if strings.HasPrefix(req, "GET /app/installations ") {
			continue // the card marks uninstalled accounts gone from the App's own list, which names no repository
		}
		assert.Contains(t, req, "Bearer ghu_alice", "every discovery read uses Alice's own token")
	}
	assert.Empty(t, f.connector.calls)
}

func TestDiscovery_WithoutAPersonTokenReadsNothingAndNeverTheConnector(t *testing.T) {
	f := newPersonViewFixture(t)
	_, err := f.repos.ListRepos(as("carol"), "workspace-default", "", false)
	require.ErrorIs(t, err, auth.ErrGitHubNotConnected)
	_, err = f.repos.Scan(as("carol"), "workspace-default", "acme", "api", "")
	require.ErrorIs(t, err, auth.ErrGitHubNotConnected)
	_, err = f.repos.ListInstallations(as("carol"))
	require.ErrorIs(t, err, auth.ErrGitHubNotConnected)
	assert.Empty(t, f.github.seen(), "GitHub is never asked as anyone else")
	assert.Empty(t, f.connector.calls, "the connector's token is never borrowed for discovery")
}

func TestAttach_LinksOnlyWhatTheAttacherCanOpenAndBackgroundWorkStaysOnAttachedRepositories(t *testing.T) {
	f := newPersonViewFixture(t)
	bobsRepo := workspace.RepoRef{Owner: "bob", Name: "secret", FullName: "bob/secret", ConnectorID: "github"}

	err := f.projects.AddRepo(as("alice"), "project-general", bobsRepo)
	require.ErrorIs(t, err, apperrs.ErrNotFound, "an Owner cannot attach, so cannot link, an account her own GitHub cannot open")
	_, err = f.store.Projects.GetRepoByFullName(t.Context(), "bob", "secret")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
	rows, err := f.store.GitHubInstallations.ListAssignments(t.Context())
	require.NoError(t, err)
	assert.Empty(t, rows)
	_, err = f.router.RepoToken(t.Context(), "bob/secret")
	require.Error(t, err, "background work never clones an unattached repository")

	require.NoError(t, f.projects.AddRepo(as("alice"), "project-general", workspace.RepoRef{Owner: "acme", Name: "api", FullName: "acme/api", ConnectorID: "github"}))
	rows, err = f.store.GitHubInstallations.ListAssignments(t.Context())
	require.NoError(t, err)
	assert.Equal(t, []repository.Assignment{{AccountID: 11, AccountLogin: "acme", WorkspaceID: "workspace-default", WorkspaceName: "Default"}}, rows, "attaching links its account")

	clone, err := f.router.RepoToken(t.Context(), "acme/api")
	require.NoError(t, err)
	assert.Equal(t, "ghs_clone", clone.Token)
	assert.False(t, clone.AllowFallback)
	require.Len(t, f.github.mints, 1)
	assert.JSONEq(t, `{"repositories":["api"],"permissions":{"contents":"read"}}`, f.github.mints[0], "the runner's token reads that one repository")
	_, err = f.router.RepoToken(t.Context(), "acme/other")
	require.Error(t, err, "another repository of the same account stays out of reach until it is attached")
	assert.Len(t, f.github.mints, 1)
}

func repoNames(repos []repository.Repo) []string {
	out := []string{}
	for _, r := range repos {
		out = append(out, r.FullName)
	}
	return out
}
