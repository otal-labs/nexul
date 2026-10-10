package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/connectors"
	"github.com/otal-labs/nexul/internal/gitprovider"
	"github.com/otal-labs/nexul/internal/gitprovider/github"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/storage"
	"github.com/otal-labs/nexul/internal/repository"
	"github.com/otal-labs/nexul/internal/workspace"
)

// fakeGitRepoResolver is a fake workspace lookup: owner/name -> RepoRef, keyed by "owner/name".
type fakeGitRepoResolver struct {
	repos map[string]workspace.RepoRef
}

func (f *fakeGitRepoResolver) GetRepoByFullName(_ context.Context, owner, name string) (workspace.RepoRef, error) {
	ref, ok := f.repos[owner+"/"+name]
	if !ok {
		return workspace.RepoRef{}, apperrs.ErrNotFound
	}
	return ref, nil
}

// fakeGitTokenResolver is a fake connectors.Service; it records every connectorID it was asked for.
type fakeGitTokenResolver struct {
	tokens map[string]string
	errs   map[string]error
	calls  []string
}

func (f *fakeGitTokenResolver) AccessToken(_ context.Context, connectorID string) (string, error) {
	f.calls = append(f.calls, connectorID)
	if err, ok := f.errs[connectorID]; ok {
		return "", err
	}
	tok, ok := f.tokens[connectorID]
	if !ok {
		return "", apperrs.ErrNotFound
	}
	return tok, nil
}

// fakeAppConfigStore is a fake connectors.AppConfigStore, mirroring fakeGitTokenResolver's spy behavior.
type fakeAppConfigStore struct {
	cfgs  map[string]connectors.AppConfig
	calls []string
}

func (f *fakeAppConfigStore) GetAppConfig(_ context.Context, connectorID string) (connectors.AppConfig, error) {
	f.calls = append(f.calls, connectorID)
	cfg, ok := f.cfgs[connectorID]
	if !ok {
		return connectors.AppConfig{ConnectorID: connectorID}, nil
	}
	return cfg, nil
}

func (f *fakeAppConfigStore) SetAppConfig(_ context.Context, c connectors.AppConfig) error {
	f.cfgs[c.ConnectorID] = c
	return nil
}

func (f *fakeAppConfigStore) SetPrivateKey(_ context.Context, connectorID, key string) error {
	c := f.cfgs[connectorID]
	c.PrivateKey = key
	f.cfgs[connectorID] = c
	return nil
}

// TestGitProviderRouter_ResolvesCorrectConnectorPerRepo proves the router never shares one global client across repos.
func TestGitProviderRouter_ResolvesCorrectConnectorPerRepo(t *testing.T) {
	tokens := &fakeGitTokenResolver{tokens: map[string]string{
		"github": "token-public",
		"gitlab": "token-acme-gitlab",
	}}
	appConfigs := &fakeAppConfigStore{cfgs: map[string]connectors.AppConfig{
		"gitlab": {ConnectorID: "gitlab", BaseURL: "https://gitlab.acme.example.com/api/v4/"},
	}}
	router := gitProviderRouter{
		workspace: &fakeGitRepoResolver{repos: map[string]workspace.RepoRef{
			"acme/widgets": {Owner: "acme", Name: "widgets", FullName: "acme/widgets", ConnectorID: "github"},
			"acme/other":   {Owner: "acme", Name: "other", FullName: "acme/other", ConnectorID: "gitlab"},
		}},
		connectors: tokens,
		appConfigs: appConfigs,
	}

	// acme/widgets: connector "github", the one real dispatchable client.
	p1, err := router.resolve(context.Background(), "acme", "widgets")
	if err != nil {
		t.Fatalf("resolve acme/widgets: %v", err)
	}
	if p1 == nil {
		t.Fatal("resolve acme/widgets: nil provider")
	}

	// acme/other's "gitlab" has no concrete client, so this fails at dispatch, after resolving its own token/URL.
	if _, err := router.resolve(context.Background(), "acme", "other"); err == nil {
		t.Fatal("resolve acme/other: want error (no gitlab client wired), got nil")
	}

	wantTokenCalls := []string{"github", "gitlab"}
	if got := tokens.calls; !equalStrings(got, wantTokenCalls) {
		t.Fatalf("AccessToken calls = %v, want %v (each repo must resolve its own connector)", got, wantTokenCalls)
	}
	wantConfigCalls := []string{"github", "gitlab"}
	if got := appConfigs.calls; !equalStrings(got, wantConfigCalls) {
		t.Fatalf("GetAppConfig calls = %v, want %v (each repo must resolve its own connector)", got, wantConfigCalls)
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestGitProviderRouter_BadBaseURLFailsClearly(t *testing.T) {
	router := gitProviderRouter{
		workspace: &fakeGitRepoResolver{repos: map[string]workspace.RepoRef{
			"acme/widgets": {Owner: "acme", Name: "widgets", FullName: "acme/widgets", ConnectorID: "github"},
		}},
		connectors: &fakeGitTokenResolver{tokens: map[string]string{"github": "token"}},
		appConfigs: &fakeAppConfigStore{cfgs: map[string]connectors.AppConfig{
			"github": {ConnectorID: "github", BaseURL: "://not a url"},
		}},
	}
	if _, err := router.resolve(context.Background(), "acme", "widgets"); err == nil {
		t.Fatal("resolve: want error for unparseable base url, got nil")
	}
}

func TestGitProviderRouter_NoTokenFailsOnlyThatRepo(t *testing.T) {
	router := gitProviderRouter{
		workspace: &fakeGitRepoResolver{repos: map[string]workspace.RepoRef{
			"acme/broken":  {Owner: "acme", Name: "broken", FullName: "acme/broken", ConnectorID: "github"},
			"acme/working": {Owner: "acme", Name: "working", FullName: "acme/working", ConnectorID: "gitlab"},
		}},
		connectors: &fakeGitTokenResolver{
			tokens: map[string]string{"gitlab": "token-gitlab"},
			errs:   map[string]error{"github": errors.New("credentials revoked")},
		},
		appConfigs: &fakeAppConfigStore{cfgs: map[string]connectors.AppConfig{}},
	}

	// The repo whose connector has no live token fails clearly for that call.
	if _, err := router.resolve(context.Background(), "acme", "broken"); err == nil {
		t.Fatal("resolve acme/broken: want error, got nil")
	}

	// A different, still-connected repo's call is unaffected, proving the earlier failure didn't leak state.
	_, err := router.resolve(context.Background(), "acme", "working")
	if err == nil {
		t.Fatal("resolve acme/working: want error (no gitlab client wired), got nil")
	}
	if errors.Is(err, apperrs.ErrNotFound) {
		t.Fatalf("resolve acme/working: got the wrong failure (repo lookup), want the unrecognized-connector failure: %v", err)
	}
}

func TestGitProviderRouter_RepoNotLinkedFailsClearly(t *testing.T) {
	router := gitProviderRouter{
		workspace:  &fakeGitRepoResolver{repos: map[string]workspace.RepoRef{}},
		connectors: &fakeGitTokenResolver{},
		appConfigs: &fakeAppConfigStore{cfgs: map[string]connectors.AppConfig{}},
	}
	_, err := router.resolve(context.Background(), "nobody", "nothing")
	if err == nil {
		t.Fatal("resolve: want error for an unlinked repo, got nil")
	}
	if !errors.Is(err, apperrs.ErrNotFound) {
		t.Fatalf("resolve: want ErrNotFound, got %v", err)
	}
}

func TestGitProviderRouter_UnrecognizedConnectorFailsClearly(t *testing.T) {
	router := gitProviderRouter{
		workspace: &fakeGitRepoResolver{repos: map[string]workspace.RepoRef{
			"acme/widgets": {Owner: "acme", Name: "widgets", FullName: "acme/widgets", ConnectorID: "bitbucket"},
		}},
		connectors: &fakeGitTokenResolver{tokens: map[string]string{"bitbucket": "token"}},
		appConfigs: &fakeAppConfigStore{cfgs: map[string]connectors.AppConfig{}},
	}
	_, err := router.resolve(context.Background(), "acme", "widgets")
	if err == nil {
		t.Fatal("resolve: want error for unrecognized connector, got nil")
	}
	if !errors.Is(err, apperrs.ErrInvalid) {
		t.Fatalf("resolve: want ErrInvalid, got %v", err)
	}
}

// TestGitProviderRouter_ReadsAsTheAppOnlyOnceAKeyIsSet: reads and clones use the installation token only with a key.
func TestGitProviderRouter_ReadsAsTheAppOnlyOnceAKeyIsSet(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	pemKey := string(pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)}))
	var seen []string
	mux := http.NewServeMux()
	mux.HandleFunc("GET /repos/acme/widgets/installation", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"id":9}`)) // test server: write errors are irrelevant
	})
	mux.HandleFunc("POST /app/installations/9/access_tokens", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"token":"ghs_installation","expires_at":"2099-01-01T00:00:00Z"}`)) // test server: write errors are irrelevant
	})
	mux.HandleFunc("GET /repos/acme/widgets", func(w http.ResponseWriter, r *http.Request) {
		seen = append(seen, r.Header.Get("Authorization"))
		_, _ = w.Write([]byte(`{"full_name":"acme/widgets","owner":{"login":"acme"}}`)) // test server: write errors are irrelevant
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	appConfigs := &fakeAppConfigStore{cfgs: map[string]connectors.AppConfig{
		"github": {ConnectorID: "github", ClientID: "Iv1.acme", BaseURL: srv.URL},
	}}
	router := gitProviderRouter{
		workspace: &fakeGitRepoResolver{repos: map[string]workspace.RepoRef{
			"acme/widgets": {Owner: "acme", Name: "widgets", FullName: "acme/widgets", ConnectorID: "github"},
		}},
		connectors: &fakeGitTokenResolver{tokens: map[string]string{"github": "token-connected-account"}},
		appConfigs: appConfigs,
		apps:       &github.AppCache{},
	}

	_, err = router.GetRepo(t.Context(), "acme", "widgets")
	require.NoError(t, err)
	clone, err := router.RepoToken(t.Context(), "acme/widgets")
	require.NoError(t, err)
	assert.Equal(t, "token-connected-account", clone.Token, "with no key a runner clones as the connected account")

	legacyClone, err := router.RepoToken(t.Context(), "acme/unlinked")
	require.NoError(t, err)
	assert.Equal(t, "token-connected-account", legacyClone.Token, "pre-key instance stacks keep cloning repositories without a project link")
	assert.True(t, legacyClone.AllowFallback)

	require.NoError(t, appConfigs.SetPrivateKey(t.Context(), "github", pemKey))
	_, err = router.GetRepo(t.Context(), "acme", "widgets")
	require.NoError(t, err)
	clone, err = router.RepoToken(t.Context(), "acme/widgets")
	require.NoError(t, err)
	assert.Equal(t, "ghs_installation", clone.Token, "with a key a runner clones as the App's installation")
	assert.False(t, clone.AllowFallback)

	assert.Equal(t, []string{"Bearer token-connected-account", "Bearer ghs_installation"}, seen)
}

func TestGitProviderRouter_InstallationAssignmentConfinesEveryLinkedOperation(t *testing.T) {
	db := mentionsTestDB(t)
	store := storage.New(db, []byte("0123456789abcdef0123456789abcdef"))
	_, err := db.Exec(`INSERT INTO workspaces (id, name, slug, created_at, updated_at) VALUES ('ws-globex', 'Globex', 'globex', 1, 1)`)
	require.NoError(t, err)
	_, err = store.GitHubInstallations.AssignInstallation(t.Context(), repository.Assignment{AccountID: 77, AccountLogin: "globex", WorkspaceID: "ws-globex"})
	require.NoError(t, err)
	require.NoError(t, store.Projects.AddRepo(t.Context(), "project-general", workspace.RepoRef{Owner: "globex", Name: "api", FullName: "globex/api", ConnectorID: "github"}))
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	pemKey := string(pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)}))
	var requests atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if strings.HasSuffix(r.URL.Path, "/installation") {
			_, _ = w.Write([]byte(`{"id":9,"account":{"id":77,"login":"globex"}}`))
			return
		}
		if strings.HasSuffix(r.URL.Path, "/access_tokens") {
			_, _ = w.Write([]byte(`{"token":"ghs_globex","expires_at":"2099-01-01T00:00:00Z"}`))
			return
		}
		if strings.HasSuffix(r.URL.Path, "/hooks") {
			_, _ = w.Write([]byte(`[]`))
			return
		}
		_, _ = w.Write([]byte(`{"full_name":"globex/api","number":1}`))
	}))
	t.Cleanup(srv.Close)
	cfg := &fakeAppConfigStore{cfgs: map[string]connectors.AppConfig{"github": {ConnectorID: "github", ClientID: "Iv1.acme", PrivateKey: pemKey, BaseURL: srv.URL}}}
	scope := &githubInstallationScope{appConfigs: cfg, projects: store.Projects}
	router := gitProviderRouter{workspace: store.Projects, appConfigs: cfg, apps: &github.AppCache{}, scope: scope}
	scanner := newRepositoryScanner(router, cfg)
	scope.repos = repository.NewService(repository.Config{Installations: scanner, Accounts: scanner, Store: store.GitHubInstallations})
	for _, tt := range []struct {
		name string
		call func() error
	}{
		{"repository", func() error { _, err := router.GetRepo(t.Context(), "globex", "api"); return err }},
		{"pull request", func() error { _, err := router.GetPR(t.Context(), "globex", "api", 1); return err }},
		{"webhooks", func() error { _, err := router.ListWebhooks(t.Context(), "globex", "api"); return err }},
		{"clone credential", func() error { _, err := router.RepoToken(t.Context(), "globex/api"); return err }},
		{"attachment", func() error {
			p := hookedProjects{Repo: store.Projects, scope: scope, hooks: newTestRepoWebhooks(&fakeHookHost{}, "https://nexul.example.com")}
			return p.AddRepo(t.Context(), "project-general", workspace.RepoRef{Owner: "globex", Name: "other", FullName: "globex/other", ConnectorID: "github"})
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			require.ErrorIs(t, tt.call(), apperrs.ErrNotFound)
			assert.Zero(t, requests.Load(), "another workspace's installation is never contacted")
		})
	}
	_, err = store.Projects.GetRepoByFullName(t.Context(), "globex", "other")
	require.ErrorIs(t, err, apperrs.ErrNotFound, "a refused attachment is not stored")
	_, err = store.GitHubInstallations.AssignInstallation(t.Context(), repository.Assignment{AccountID: 77, AccountLogin: "globex", WorkspaceID: "workspace-default"})
	require.NoError(t, err)
	token, err := router.RepoToken(t.Context(), "globex/api")
	require.NoError(t, err)
	assert.Equal(t, "ghs_globex", token.Token, "assignment grants this project's runner the installation credential")
	_, err = store.GitHubInstallations.UnassignInstallation(t.Context(), repository.Assignment{AccountID: 77, AccountLogin: "globex", WorkspaceID: "workspace-default"})
	require.NoError(t, err)
	denied, err := router.RepoToken(t.Context(), "globex/api")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
	assert.False(t, denied.AllowFallback, "unassignment cannot permit the runner's own token")
}

func TestRepositoryScan_ReadsWithThePersonsOwnTokenAfterValidatingTheName(t *testing.T) {
	for _, name := range []string{"../../repos/globex/private-api", "../api", ".", "..", "api/other", `api\other`, "%2e%2e%2frepos%2fglobex%2fprivate-api", "api%2fother", "api%5cother", "api%252fother"} {
		t.Run(name, func(t *testing.T) {
			var paths, treeTokens []string
			mux := http.NewServeMux()
			mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				paths = append(paths, r.URL.Path)
				treeTokens = append(treeTokens, r.Header.Get("Authorization"))
				_, _ = w.Write([]byte(`{"tree":[]}`)) // test server: write errors are irrelevant
			})
			srv := httptest.NewServer(mux)
			t.Cleanup(srv.Close)
			cfg := &fakeAppConfigStore{cfgs: map[string]connectors.AppConfig{"github": {ClientID: "Iv1.acme", BaseURL: srv.URL}}}
			svc := repository.NewService(repository.Config{Gate: projectWriter{}, People: newGitHubPeople(fakeGitHubTokens{"alice": "ghu_alice"}, cfg)})
			alice := identity.WithActor(t.Context(), identity.Actor{ID: "alice"})
			got, err := svc.Scan(alice, "", "acme", name, "main")
			assert.ErrorIs(t, err, apperrs.ErrInvalid)
			assert.Nil(t, got)
			assert.Empty(t, paths, "validation precedes every request")
			_, err = svc.Scan(alice, "", "acme", "api.v2-web_tools", "main")
			require.NoError(t, err)
			assert.Equal(t, []string{"/repos/acme/api.v2-web_tools/git/trees/main"}, paths)
			assert.Equal(t, []string{"Bearer ghu_alice"}, treeTokens, "a scan reads as the person, never as an installation")
		})
	}
}

// TestGitProviderRouter_RepoToken_MintsAFreshTokenForTheOneRepositoryWithContentsRead: a runner never gets the
// installation-wide token the server reads with, nor one another clone was handed.
func TestGitProviderRouter_RepoToken_MintsAFreshTokenForTheOneRepositoryWithContentsRead(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	pemKey := string(pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)}))
	var bodies []string
	mux := http.NewServeMux()
	mux.HandleFunc("GET /repos/acme/widgets/installation", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"id":9,"account":{"id":101,"login":"acme"}}`)) // test server: write errors are irrelevant
	})
	mux.HandleFunc("POST /app/installations/9/access_tokens", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body) // test server: a short read fails the body assertion below
		bodies = append(bodies, string(body))
		_, _ = fmt.Fprintf(w, `{"token":"ghs_%d","expires_at":"2099-01-01T00:00:00Z"}`, len(bodies)) // test server: write errors are irrelevant
	})
	mux.HandleFunc("GET /repos/acme/widgets", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"full_name":"acme/widgets","owner":{"login":"acme"}}`)) // test server: write errors are irrelevant
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	router := gitProviderRouter{
		workspace: &fakeGitRepoResolver{repos: map[string]workspace.RepoRef{
			"acme/widgets": {Owner: "acme", Name: "widgets", FullName: "acme/widgets", ConnectorID: "github"},
		}},
		appConfigs: &fakeAppConfigStore{cfgs: map[string]connectors.AppConfig{
			"github": {ConnectorID: "github", ClientID: "Iv1.acme", PrivateKey: pemKey, BaseURL: srv.URL},
		}},
		apps: &github.AppCache{},
	}

	_, err = router.GetRepo(t.Context(), "acme", "widgets")
	require.NoError(t, err)
	first, err := router.RepoToken(t.Context(), "acme/widgets")
	require.NoError(t, err)
	second, err := router.RepoToken(t.Context(), "acme/widgets")
	require.NoError(t, err)

	assert.NotEqual(t, "ghs_1", first.Token, "the server's installation-wide token never reaches a runner")
	assert.NotEqual(t, first.Token, second.Token, "each clone mints its own token")
	require.Len(t, bodies, 3)
	for _, body := range bodies[1:] {
		assert.JSONEq(t, `{"repositories":["widgets"],"permissions":{"contents":"read"}}`, body)
	}
}

// TestPushesRunnersMayClone_AnUnassignedInstallationQueuesNoBuild: a push GitHub still delivers for a repository whose
// installation left the project's workspace, or that no project links, never reaches the branch deploy rules.
func TestPushesRunnersMayClone_AnUnassignedInstallationQueuesNoBuild(t *testing.T) {
	store := storage.New(mentionsTestDB(t), []byte("0123456789abcdef0123456789abcdef"))
	require.NoError(t, store.Projects.AddRepo(t.Context(), "project-general", workspace.RepoRef{Owner: "globex", Name: "api", FullName: "globex/api", ConnectorID: "github"}))
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	pemKey := string(pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)}))
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"id":9,"account":{"id":77,"login":"globex"}}`)) // test server: write errors are irrelevant
	}))
	t.Cleanup(srv.Close)
	cfg := &fakeAppConfigStore{cfgs: map[string]connectors.AppConfig{"github": {ConnectorID: "github", ClientID: "Iv1.acme", PrivateKey: pemKey, BaseURL: srv.URL}}}
	scope := &githubInstallationScope{appConfigs: cfg, projects: store.Projects}
	router := gitProviderRouter{workspace: store.Projects, appConfigs: cfg, apps: &github.AppCache{}, scope: scope}
	scanner := newRepositoryScanner(router, cfg)
	scope.repos = repository.NewService(repository.Config{Installations: scanner, Accounts: scanner, Store: store.GitHubInstallations})
	var built []string
	handler := pushesRunnersMayClone(router, func(_ context.Context, ev eventbus.Event) error {
		built = append(built, string(ev.Payload))
		return nil
	})
	push := func(repo string) eventbus.Event {
		raw, err := json.Marshal(gitprovider.PushEvent{Owner: "globex", Repo: repo, Branch: "main", SHA: "abc123"})
		require.NoError(t, err)
		return eventbus.Event{Topic: gitprovider.TopicPush, Payload: raw}
	}

	require.NoError(t, handler(t.Context(), push("api")), "an ignored push is acknowledged, never retried")
	require.NoError(t, handler(t.Context(), push("unlinked")))
	assert.Empty(t, built)

	_, err = store.GitHubInstallations.AssignInstallation(t.Context(), repository.Assignment{AccountID: 77, AccountLogin: "globex", WorkspaceID: "workspace-default"})
	require.NoError(t, err)
	require.NoError(t, handler(t.Context(), push("api")))
	assert.Len(t, built, 1, "once assigned, the push reaches the deploy rules")
}
