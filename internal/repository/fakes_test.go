package repository

import (
	"context"

	apperrors "github.com/otal-labs/nexul/internal/platform/errors"
)

// fakeScanner is an in-memory Scanner: tree entries and file contents are set up per test, keyed by path.
type fakeScanner struct {
	tree        []TreeEntry
	files       map[string][]byte
	repos       []Repo
	resolvedRef string

	listErr error
	treeErr error
	fileErr error
}

func (f *fakeScanner) ListInstallationRepos(context.Context) ([]Repo, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	return f.repos, nil
}

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
