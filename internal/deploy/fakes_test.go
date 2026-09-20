package deploy

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
)

// fakeRepo is an in-memory deploy.Repo for use-case tests.
type fakeRepo struct {
	mu        sync.Mutex
	stored    map[string]*Deploy
	outbox    []eventbus.OutboxEvent
	createErr error
	getErr    error
	updateErr error
	appendErr error
	cancelErr error
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{stored: map[string]*Deploy{}}
}

func (f *fakeRepo) Create(_ context.Context, d *Deploy, evts ...eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.createErr != nil {
		return f.createErr
	}
	if _, ok := f.stored[d.ID]; ok {
		return apperrs.ErrConflict
	}
	f.stored[d.ID] = d
	f.outbox = append(f.outbox, evts...)
	return nil
}

// CancelRequested enqueues a deploy.cancel_requested outbox row, mirroring
// the storage implementation's no-local-write shape.
func (f *fakeRepo) CancelRequested(_ context.Context, evt eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.cancelErr != nil {
		return f.cancelErr
	}
	f.outbox = append(f.outbox, evt)
	return nil
}

// of returns the last outbox event enqueued for a topic.
func (f *fakeRepo) of(topic string) eventbus.OutboxEvent {
	f.mu.Lock()
	defer f.mu.Unlock()
	var evt eventbus.OutboxEvent
	for _, e := range f.outbox {
		if e.Topic == topic {
			evt = e
		}
	}
	return evt
}

func (f *fakeRepo) GetByID(_ context.Context, id string) (*Deploy, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.getErr != nil {
		return nil, f.getErr
	}
	d, ok := f.stored[id]
	if !ok {
		return nil, apperrs.ErrNotFound
	}
	return d, nil
}

func (f *fakeRepo) List(_ context.Context) ([]*Deploy, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]*Deploy, 0, len(f.stored))
	for _, d := range f.stored {
		out = append(out, d)
	}
	return out, nil
}

func (f *fakeRepo) ListByService(_ context.Context, service string) ([]*Deploy, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []*Deploy
	for _, d := range f.stored {
		if d.Service == service {
			out = append(out, d)
		}
	}
	return out, nil
}

func (f *fakeRepo) ListByStackID(_ context.Context, stackID string) ([]*Deploy, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []*Deploy
	for _, d := range f.stored {
		if d.StackID == stackID {
			out = append(out, d)
		}
	}
	return out, nil
}

func (f *fakeRepo) ListByStatus(_ context.Context, status Status) ([]*Deploy, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []*Deploy
	for _, d := range f.stored {
		if d.Status == status {
			out = append(out, d)
		}
	}
	return out, nil
}

func (f *fakeRepo) HasActive(_ context.Context, stackID string) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, d := range f.stored {
		if d.StackID == stackID && (d.Status == StatusPending || d.Status == StatusRunning) {
			return true, nil
		}
	}
	return false, nil
}

func (f *fakeRepo) LastHealthy(_ context.Context, stackID string) (*Deploy, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var newest *Deploy
	for _, d := range f.stored {
		if d.StackID != stackID || d.Status != StatusHealthy {
			continue
		}
		if newest == nil || d.CreatedAt.After(newest.CreatedAt) {
			newest = d
		}
	}
	if newest == nil {
		return nil, apperrs.ErrNotFound
	}
	return newest, nil
}

func (f *fakeRepo) UpdateStatus(_ context.Context, id string, status Status) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.updateErr != nil {
		return f.updateErr
	}
	d, ok := f.stored[id]
	if !ok {
		return apperrs.ErrNotFound
	}
	d.Status = status
	d.UpdatedAt = time.Now().UTC()
	return nil
}

func (f *fakeRepo) SetAddress(_ context.Context, id, address string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	d, ok := f.stored[id]
	if !ok {
		return apperrs.ErrNotFound
	}
	d.Address = address
	d.UpdatedAt = time.Now().UTC()
	return nil
}

func (f *fakeRepo) AppendLog(_ context.Context, id, entry string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.appendErr != nil {
		return f.appendErr
	}
	d, ok := f.stored[id]
	if !ok {
		return apperrs.ErrNotFound
	}
	d.Log += entry
	return nil
}

// fakeStackRepo is an in-memory deploy.StackRepo for use-case tests.
type fakeStackRepo struct {
	mu        sync.Mutex
	stored    map[string]*Stack
	createErr error
	getErr    error
	updateErr error
	deleteErr error
	outbox    []eventbus.OutboxEvent
}

func newFakeStackRepo() *fakeStackRepo {
	return &fakeStackRepo{stored: map[string]*Stack{}}
}

func (f *fakeStackRepo) Create(_ context.Context, stack *Stack, evts ...eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.createErr != nil {
		return f.createErr
	}
	if _, ok := f.stored[stack.ID]; ok {
		return apperrs.ErrConflict
	}
	cp := *stack
	f.stored[stack.ID] = &cp
	f.outbox = append(f.outbox, evts...)
	return nil
}

func (f *fakeStackRepo) GetByID(_ context.Context, id string) (*Stack, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.getErr != nil {
		return nil, f.getErr
	}
	stack, ok := f.stored[id]
	if !ok {
		return nil, apperrs.ErrNotFound
	}
	cp := *stack
	return &cp, nil
}

func (f *fakeStackRepo) GetBySlugAndMachine(_ context.Context, slug, machine string) (*Stack, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, stack := range f.stored {
		if stack.Slug == slug && stack.Machine == machine {
			cp := *stack
			return &cp, nil
		}
	}
	return nil, apperrs.ErrNotFound
}

func (f *fakeStackRepo) GetByName(_ context.Context, name string) (*Stack, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, stack := range f.stored {
		if stack.Name == name {
			cp := *stack
			return &cp, nil
		}
	}
	return nil, apperrs.ErrNotFound
}

func (f *fakeStackRepo) ListByProject(_ context.Context, projectID string) ([]*Stack, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []*Stack
	for _, stack := range f.stored {
		if projectID == "" || stack.ProjectID == projectID {
			cp := *stack
			out = append(out, &cp)
		}
	}
	return out, nil
}

func (f *fakeStackRepo) ListByBuildRepo(_ context.Context, owner, name string) ([]*Stack, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []*Stack
	for _, stack := range f.stored {
		if stack.DerivedFrom != "" || stack.BuildSource == nil {
			continue
		}
		if stack.BuildSource.RepoOwner == owner && stack.BuildSource.RepoName == name {
			cp := *stack
			out = append(out, &cp)
		}
	}
	return out, nil
}

func (f *fakeStackRepo) ListByDerivedFrom(_ context.Context, baseStackID string) ([]*Stack, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []*Stack
	for _, stack := range f.stored {
		if stack.DerivedFrom == baseStackID {
			cp := *stack
			out = append(out, &cp)
		}
	}
	return out, nil
}

func (f *fakeStackRepo) Update(_ context.Context, stack *Stack, evts ...eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.updateErr != nil {
		return f.updateErr
	}
	if _, ok := f.stored[stack.ID]; !ok {
		return apperrs.ErrNotFound
	}
	cp := *stack
	f.stored[stack.ID] = &cp
	f.outbox = append(f.outbox, evts...)
	return nil
}

func (f *fakeStackRepo) Delete(_ context.Context, id string, evts ...eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.deleteErr != nil {
		return f.deleteErr
	}
	if _, ok := f.stored[id]; !ok {
		return apperrs.ErrNotFound
	}
	delete(f.stored, id)
	f.outbox = append(f.outbox, evts...)
	return nil
}

func (f *fakeStackRepo) topics() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]string, 0, len(f.outbox))
	for _, evt := range f.outbox {
		out = append(out, evt.Topic)
	}
	return out
}

// fakeContainerRepo is an in-memory deploy.ServiceRepo (containers) for use-case tests.
type fakeContainerRepo struct {
	mu     sync.Mutex
	stored map[string]*Container
}

func newFakeContainerRepo() *fakeContainerRepo {
	return &fakeContainerRepo{stored: map[string]*Container{}}
}

func (f *fakeContainerRepo) Create(_ context.Context, svc *Container) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	cp := *svc
	f.stored[svc.ID] = &cp
	return nil
}

func (f *fakeContainerRepo) Upsert(_ context.Context, svc *Container) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, existing := range f.stored {
		if existing.StackID == svc.StackID && existing.Name == svc.Name {
			cp := *svc
			cp.ID = existing.ID
			f.stored[existing.ID] = &cp
			return nil
		}
	}
	cp := *svc
	f.stored[svc.ID] = &cp
	return nil
}

func (f *fakeContainerRepo) ListByStack(_ context.Context, stackID string) ([]*Container, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []*Container
	for _, svc := range f.stored {
		if svc.StackID == stackID {
			cp := *svc
			out = append(out, &cp)
		}
	}
	return out, nil
}

func (f *fakeContainerRepo) Get(_ context.Context, id string) (*Container, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	svc, ok := f.stored[id]
	if !ok {
		return nil, apperrs.ErrNotFound
	}
	cp := *svc
	return &cp, nil
}

func (f *fakeContainerRepo) DeleteByStack(_ context.Context, stackID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for id, svc := range f.stored {
		if svc.StackID == stackID {
			delete(f.stored, id)
		}
	}
	return nil
}

// fakeProjects is an in-memory deploy.ProjectStore for use-case tests.
type fakeProjects struct {
	exists  map[string]bool
	repos   map[string][]string // projectID -> "owner/name"
	repoErr error
}

func newFakeProjects() *fakeProjects {
	return &fakeProjects{exists: map[string]bool{}, repos: map[string][]string{}}
}

func (f *fakeProjects) ProjectExists(_ context.Context, projectID string) (bool, error) {
	return f.exists[projectID], nil
}

func (f *fakeProjects) LinkRepo(_ context.Context, projectID, owner, name string) error {
	if f.repoErr != nil {
		return f.repoErr
	}
	full := owner + "/" + name
	for _, r := range f.repos[projectID] {
		if r == full {
			return nil
		}
	}
	f.repos[projectID] = append(f.repos[projectID], full)
	return nil
}

func (f *fakeProjects) RepoInProject(_ context.Context, projectID, owner, name string) (bool, error) {
	if f.repoErr != nil {
		return false, f.repoErr
	}
	full := owner + "/" + name
	for _, r := range f.repos[projectID] {
		if r == full {
			return true, nil
		}
	}
	return false, nil
}

// fakeBus records published events, mirroring the topology/gitprovider fakes.
type fakeBus struct {
	mu         sync.Mutex
	published  []eventbus.Event
	publishErr error
}

func newFakeBus() *fakeBus {
	return &fakeBus{}
}

func (f *fakeBus) Publish(_ context.Context, topic string, payload any) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.publishErr != nil {
		return f.publishErr
	}
	b, _ := json.Marshal(payload)
	f.published = append(f.published, eventbus.Event{Topic: topic, Payload: b})
	return nil
}
