package plays

import (
	"context"
	"sync"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// fakeRepo is a hand-written in-memory stand-in for Repo (practices/testing.md §6).
type fakeRepo struct {
	mu        sync.Mutex
	byID      map[string]*Play
	published []eventbus.OutboxEvent
	createErr error
	getErr    error
	listErr   error
	updateErr error
	deleteErr error
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{byID: map[string]*Play{}}
}

func (f *fakeRepo) Create(_ context.Context, p *Play, evts ...eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.createErr != nil {
		return f.createErr
	}
	if _, ok := f.byID[p.ID]; ok {
		return apperrs.ErrConflict
	}
	cp := *p
	f.byID[p.ID] = &cp
	f.published = append(f.published, evts...)
	return nil
}

func (f *fakeRepo) Get(_ context.Context, id string) (*Play, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.getErr != nil {
		return nil, f.getErr
	}
	p, ok := f.byID[id]
	if !ok {
		return nil, apperrs.ErrNotFound
	}
	cp := *p
	return &cp, nil
}

func (f *fakeRepo) List(_ context.Context, workspaceID string) ([]*Play, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.listErr != nil {
		return nil, f.listErr
	}
	var out []*Play
	for _, p := range f.byID {
		if p.WorkspaceID == workspaceID {
			cp := *p
			out = append(out, &cp)
		}
	}
	return out, nil
}

func (f *fakeRepo) Update(_ context.Context, p *Play, evts ...eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.updateErr != nil {
		return f.updateErr
	}
	if _, ok := f.byID[p.ID]; !ok {
		return apperrs.ErrNotFound
	}
	cp := *p
	f.byID[p.ID] = &cp
	f.published = append(f.published, evts...)
	return nil
}

func (f *fakeRepo) Delete(_ context.Context, id string, evts ...eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.deleteErr != nil {
		return f.deleteErr
	}
	if _, ok := f.byID[id]; !ok {
		return apperrs.ErrNotFound
	}
	delete(f.byID, id)
	f.published = append(f.published, evts...)
	return nil
}

// fakePerm grants exactly the actions listed per user; denied holds per-user "<resourceType>:<resourceID>" keys
// whose per-resource overwrite refuses the action, mirroring access's resource-instance deny layer.
type fakePerm struct {
	grants map[string][]permissions.Action
	denied map[string]map[string]bool
}

func newFakePerm(grants map[string][]permissions.Action) *fakePerm {
	return &fakePerm{grants: grants, denied: map[string]map[string]bool{}}
}

func (f *fakePerm) HasPermission(_ context.Context, userID, _ string, action permissions.Action, resourceType, resourceID string) bool {
	granted := false
	for _, a := range f.grants[userID] {
		if a == action {
			granted = true
			break
		}
	}
	if !granted {
		return false
	}
	if resourceType != "" && f.denied[userID][resourceType+":"+resourceID] {
		return false
	}
	return true
}

func (f *fakePerm) deny(userID, playID string) {
	f.denyResource(userID, resourceTypePlay, playID)
}

func (f *fakePerm) denyResource(userID, resourceType, resourceID string) {
	if f.denied[userID] == nil {
		f.denied[userID] = map[string]bool{}
	}
	f.denied[userID][resourceType+":"+resourceID] = true
}

func allowAll(userID string) *fakePerm {
	return newFakePerm(map[string][]permissions.Action{
		userID: {permissions.PlaysRead, permissions.PlaysWrite, permissions.PlaysDelete, permissions.PlaysRun},
	})
}
