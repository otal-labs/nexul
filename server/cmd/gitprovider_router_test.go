package main

import (
	"context"
	"errors"
	"testing"

	"github.com/otal-labs/nexul/internal/connectors"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
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
