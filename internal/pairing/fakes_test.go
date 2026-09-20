package pairing

import (
	"context"
	"errors"
	"sync"

	"github.com/otal-labs/nexul/internal/harness"
	"github.com/otal-labs/nexul/internal/harness/harnesstest"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// fakeRepo is an in-memory pairing.Repo for use-case tests.
type fakeRepo struct {
	mu           sync.Mutex
	computers    map[string]Computer
	defaults     map[string]Defaults
	projectLinks map[string]ProjectLink
	saveErr      error
	getErr       error
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{computers: map[string]Computer{}, defaults: map[string]Defaults{}, projectLinks: map[string]ProjectLink{}}
}

func (f *fakeRepo) SaveComputer(_ context.Context, c Computer) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.saveErr != nil {
		return f.saveErr
	}
	f.computers[c.ID] = c
	return nil
}

func (f *fakeRepo) GetComputer(_ context.Context, userID, id string) (*Computer, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.getErr != nil {
		return nil, f.getErr
	}
	c, ok := f.computers[id]
	if !ok || c.UserID != userID {
		return nil, apperrs.ErrNotFound
	}
	copied := c
	return &copied, nil
}

func (f *fakeRepo) ListComputers(_ context.Context, userID string) ([]Computer, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]Computer, 0)
	for _, c := range f.computers {
		if c.UserID == userID {
			out = append(out, c)
		}
	}
	return out, nil
}

func (f *fakeRepo) DeleteComputer(_ context.Context, userID, id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	c, ok := f.computers[id]
	if !ok || c.UserID != userID {
		return apperrs.ErrNotFound
	}
	delete(f.computers, id)
	return nil
}

func (f *fakeRepo) GetDefaults(_ context.Context, userID string) (Defaults, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.defaults[userID], nil
}

func (f *fakeRepo) SaveDefaults(_ context.Context, d Defaults) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.defaults[d.UserID] = d
	return nil
}

func (f *fakeRepo) GetComputerByID(_ context.Context, id string) (*Computer, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.getErr != nil {
		return nil, f.getErr
	}
	c, ok := f.computers[id]
	if !ok {
		return nil, apperrs.ErrNotFound
	}
	copied := c
	return &copied, nil
}

func (f *fakeRepo) GetProjectLink(_ context.Context, projectID string) (ProjectLink, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.projectLinks[projectID], nil
}

func (f *fakeRepo) SaveProjectLink(_ context.Context, link ProjectLink) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.saveErr != nil {
		return f.saveErr
	}
	f.projectLinks[link.ProjectID] = link
	return nil
}

func (f *fakeRepo) DeleteProjectLink(_ context.Context, projectID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.projectLinks, projectID)
	return nil
}

// fakeExchanger scripts the pairing half of a harness.Client; the listing half comes from the embedded harnesstest.Client.
type fakeExchanger struct {
	harnesstest.Client
	result      harness.PairResult
	exchangeErr error
	version     string
	versionErr  error
}

func (f *fakeExchanger) Pair(_ context.Context, _, _ string) (harness.PairResult, error) {
	if f.exchangeErr != nil {
		return harness.PairResult{}, f.exchangeErr
	}
	if f.versionErr != nil {
		return harness.PairResult{}, f.versionErr
	}
	r := f.result
	r.Version = f.version
	return r, nil
}

func (f *fakeExchanger) Version(_ context.Context, _ string) (string, error) {
	if f.versionErr != nil {
		return "", f.versionErr
	}
	return f.version, nil
}

// registry wraps exch as the only T3 client.
func registry(exch *fakeExchanger) harness.Registry {
	return harness.Registry{harness.KindT3Code: exch}
}

var errBoom = errors.New("boom")
