package repository

import (
	"context"
	"slices"

	apperrors "github.com/otal-labs/nexul/internal/platform/errors"
)

// fakeScanner is an in-memory Scanner: tree entries and file contents are set up per test, keyed by path.
type fakeScanner struct {
	tree        []TreeEntry
	files       map[string][]byte
	repos       []Repo
	installs    []Installation
	asApp       bool
	installURL  string
	resolvedRef string
	listCalls   int
	lastRefresh bool

	listErr    error
	installErr error
	treeErr    error
	fileErr    error
}

func (f *fakeScanner) ListInstallationRepos(_ context.Context, refresh bool) ([]Repo, error) {
	f.listCalls++
	f.lastRefresh = refresh
	if f.listErr != nil {
		return nil, f.listErr
	}
	return f.repos, nil
}

func (f *fakeScanner) ListInstallations(context.Context) ([]Installation, error) {
	if f.installErr != nil {
		return nil, f.installErr
	}
	return f.installs, nil
}

func (f *fakeScanner) ReadsAsApp(context.Context) (bool, error) { return f.asApp, nil }

func (f *fakeScanner) InstallURL(context.Context) (string, error) { return f.installURL, nil }

func (f *fakeScanner) GetTree(_ context.Context, _, _, ref string) (string, []TreeEntry, error) {
	if f.treeErr != nil {
		return "", nil, f.treeErr
	}
	resolved := f.resolvedRef
	if resolved == "" {
		resolved = ref
	}
	if resolved == "" {
		resolved = "main"
	}
	return resolved, f.tree, nil
}

func (f *fakeScanner) GetFile(_ context.Context, _, _, _, path string) ([]byte, error) {
	if f.fileErr != nil {
		return nil, f.fileErr
	}
	b, ok := f.files[path]
	if !ok {
		return nil, apperrors.ErrNotFound
	}
	return b, nil
}

// fakeStore is an in-memory InstallationStore: account -> workspace ids.
type fakeStore map[string][]string

func (f fakeStore) ListInstallationWorkspaces(context.Context) (map[string][]InstallationWorkspace, error) {
	out := map[string][]InstallationWorkspace{}
	for account, ids := range f {
		for _, id := range ids {
			out[account] = append(out[account], InstallationWorkspace{ID: id, Name: id})
		}
	}
	return out, nil
}

func (f fakeStore) InstallationAccountsIn(_ context.Context, workspaceIDs []string) ([]string, error) {
	var out []string
	for account, ids := range f {
		if slices.ContainsFunc(ids, func(id string) bool { return slices.Contains(workspaceIDs, id) }) {
			out = append(out, account)
		}
	}
	return out, nil
}

func (f fakeStore) AssignInstallation(_ context.Context, account, workspaceID string) error {
	if !slices.Contains(f[account], workspaceID) {
		f[account] = append(f[account], workspaceID)
	}
	return nil
}

func (f fakeStore) UnassignInstallation(_ context.Context, account, workspaceID string) error {
	f[account] = slices.DeleteFunc(f[account], func(id string) bool { return id == workspaceID })
	return nil
}

// fakeInstallers sees the installations in its map, id -> account, for the one code it accepts.
type fakeInstallers map[int64]string

func (f fakeInstallers) InstallerAccount(_ context.Context, code string, id int64) (string, error) {
	account, ok := f[id]
	if code != "good-code" || !ok {
		return "", apperrors.ErrForbidden
	}
	return account, nil
}

func newTestService(s *fakeScanner, store fakeStore) *Service {
	if store == nil {
		store = fakeStore{}
	}
	return NewService(Config{Scanner: s, Installations: s, Store: store, StateKey: []byte("test-key")})
}
