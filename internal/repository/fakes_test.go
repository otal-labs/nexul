package repository

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	apperrors "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
)

// fakeScanner is an in-memory GitHubView and InstallationLister: tree entries and file contents are set up per test,
// keyed by path.
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

func (f *fakeScanner) Repos(_ context.Context, refresh bool) ([]Repo, error) {
	f.listCalls++
	f.lastRefresh = refresh
	if f.listErr != nil {
		return nil, f.listErr
	}
	return f.repos, nil
}

func (f *fakeScanner) Installations(context.Context) ([]Installation, error) {
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

// fakeStore is an in-memory InstallationStore; events records what each write published, and attached names the
// workspaces whose projects attach a repository of each account login.
type fakeStore struct {
	rows     []Assignment
	events   []eventbus.OutboxEvent
	attached map[string][]string
}

func (f *fakeStore) ListAssignments(context.Context) ([]Assignment, error) {
	return slices.Clone(f.rows), nil
}

func (f *fakeStore) AssignmentsIn(_ context.Context, workspaceIDs []string) ([]Assignment, error) {
	var out []Assignment
	for _, r := range f.rows {
		if slices.Contains(workspaceIDs, r.WorkspaceID) {
			r.Attached = slices.Contains(f.attached[r.AccountLogin], r.WorkspaceID)
			out = append(out, r)
		}
	}
	return out, nil
}

func (f *fakeStore) AssignedAccountsIn(ctx context.Context, workspaceIDs []string) ([]int64, error) {
	rows, _ := f.AssignmentsIn(ctx, workspaceIDs) // the fake's read cannot fail
	return liveIDs(rows), nil
}

func (f *fakeStore) AssignedAccounts(context.Context) ([]int64, error) {
	return liveIDs(f.rows), nil
}

func liveIDs(rows []Assignment) []int64 {
	var out []int64
	for _, r := range rows {
		if r.AccountID != 0 && !r.Gone && !slices.Contains(out, r.AccountID) {
			out = append(out, r.AccountID)
		}
	}
	return out
}

func (f *fakeStore) HasUnresolvedAssignments(context.Context) (bool, error) {
	return slices.ContainsFunc(f.rows, func(r Assignment) bool { return r.AccountID == 0 && !r.Gone }), nil
}

func (f *fakeStore) AssignInstallation(_ context.Context, a Assignment, events ...eventbus.OutboxEvent) (bool, error) {
	i := slices.IndexFunc(f.rows, func(r Assignment) bool { return r.AccountID == a.AccountID && r.WorkspaceID == a.WorkspaceID })
	if i >= 0 && !f.rows[i].Gone {
		return false, nil
	}
	a.WorkspaceName = a.WorkspaceID
	if i >= 0 {
		f.rows[i] = a
	}
	if i < 0 {
		f.rows = append(f.rows, a)
	}
	f.events = append(f.events, events...)
	return true, nil
}

func (f *fakeStore) UnassignInstallation(_ context.Context, a Assignment, events ...eventbus.OutboxEvent) (bool, error) {
	n := len(f.rows)
	f.rows = slices.DeleteFunc(f.rows, func(r Assignment) bool {
		return r.WorkspaceID == a.WorkspaceID && r.sameAccount(a.AccountID, a.AccountLogin)
	})
	if len(f.rows) == n {
		return false, nil
	}
	f.events = append(f.events, events...)
	return true, nil
}

func (f *fakeStore) SyncAccounts(_ context.Context, s AccountSync, events ...eventbus.OutboxEvent) error {
	for i, r := range f.rows {
		if id, ok := s.Resolved[r.AccountLogin]; ok && r.AccountID == 0 {
			f.rows[i].AccountID = id
		}
		if login, ok := s.Renamed[r.AccountID]; ok {
			f.rows[i].AccountLogin = login
		}
		if slices.ContainsFunc(s.Gone, func(g Assignment) bool {
			return g.WorkspaceID == r.WorkspaceID && r.sameAccount(g.AccountID, g.AccountLogin)
		}) {
			f.rows[i].Gone = true
		}
	}
	f.rows = slices.DeleteFunc(f.rows, func(r Assignment) bool { return r.Gone && slices.Contains(s.Reinstalled, r.AccountID) })
	f.events = append(f.events, events...)
	return nil
}

// fakeAccounts is the App's installations as AccountResolver: InstallationAccounts lists them, AccountOf finds the
// one on a repository's owner by login, and lists counts how often GitHub was asked.
type fakeAccounts struct {
	installs []Installation
	lists    int
}

func (f *fakeAccounts) InstallationAccounts(context.Context) ([]Installation, error) {
	f.lists++
	return f.installs, nil
}

func (f *fakeAccounts) AccountOf(_ context.Context, owner, _ string) (int64, error) {
	for _, inst := range f.installs {
		if strings.EqualFold(inst.AccountLogin, owner) {
			return inst.AccountID, nil
		}
	}
	return 0, apperrors.ErrNotFound
}

// fakePeople opens each person's GitHub view; a person missing from it has not connected GitHub, and asked records
// whose view was opened.
type fakePeople struct {
	views map[string]*fakeScanner
	asked []string
}

var errNotConnected = errors.New("not connected")

func (f *fakePeople) GitHubView(_ context.Context, userID string) (GitHubView, error) {
	f.asked = append(f.asked, userID)
	v, ok := f.views[userID]
	if !ok {
		return nil, fmt.Errorf("%w: %w", apperrors.ErrForbidden, errNotConnected)
	}
	return v, nil
}

// newTestService serves s as the view of every caller, a signed-in one or the server itself.
func newTestService(s *fakeScanner, store *fakeStore) *Service {
	if store == nil {
		store = &fakeStore{}
	}
	return NewService(Config{People: &fakePeople{views: map[string]*fakeScanner{"": s}}, Installations: s, Accounts: &fakeAccounts{}, Store: store})
}
