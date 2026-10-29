package plays

import (
	"context"
	"slices"
	"strings"
	"sync"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// fakeRepo is a hand-written in-memory stand-in for Repo (practices/testing.md §6).
type fakeRepo struct {
	mu          sync.Mutex
	byID        map[string]*Play
	published   []eventbus.OutboxEvent
	createErr   error
	getErr      error
	listErr     error
	updateErr   error
	deleteErr   error
	autoPlays   []*AutoPlay
	autoPlayErr error
	// autoPlayLists counts ListAutoPlays calls, so a list proves it reads once per page.
	autoPlayLists int
	limits        map[string]AutoPlayLimits
	// automationNames answers AutomationNamesByLabel by the label asked for; the matching itself is storage's.
	automationNames map[string][]string
	namesErr        error
	// revisions is what each save handed the repo, oldest first; versions and changes are the history they add up to.
	revisions []Revision
	versions  map[string][]*VersionBody
	changes   map[string][]Change
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{byID: map[string]*Play{}, limits: map[string]AutoPlayLimits{}, versions: map[string][]*VersionBody{}, changes: map[string][]Change{}}
}

func (f *fakeRepo) CreateAutoPlay(_ context.Context, a *AutoPlay, evts ...eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.autoPlayErr != nil {
		return f.autoPlayErr
	}
	cp := *a
	f.autoPlays = append(f.autoPlays, &cp)
	f.published = append(f.published, evts...)
	return nil
}

func (f *fakeRepo) GetAutoPlay(_ context.Context, id string) (*AutoPlay, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, a := range f.autoPlays {
		if a.ID == id {
			cp := *a
			return &cp, nil
		}
	}
	return nil, apperrs.ErrNotFound
}

func (f *fakeRepo) ListAutoPlays(_ context.Context, playIDs []string) ([]*AutoPlay, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.autoPlayLists++
	if f.autoPlayErr != nil {
		return nil, f.autoPlayErr
	}
	var out []*AutoPlay
	for _, a := range f.autoPlays {
		if slices.Contains(playIDs, a.PlayID) {
			cp := *a
			out = append(out, &cp)
		}
	}
	return out, nil
}

func (f *fakeRepo) ListEnabledAutoPlays(_ context.Context, workspaceID string, moment Moment) ([]*AutoPlay, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.autoPlayErr != nil {
		return nil, f.autoPlayErr
	}
	var out []*AutoPlay
	for _, a := range f.autoPlays {
		if a.WorkspaceID == workspaceID && a.Moment == moment && a.Enabled {
			cp := *a
			out = append(out, &cp)
		}
	}
	return out, nil
}

func (f *fakeRepo) UpdateAutoPlay(_ context.Context, a *AutoPlay, evts ...eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.autoPlayErr != nil {
		return f.autoPlayErr
	}
	for i, existing := range f.autoPlays {
		if existing.ID == a.ID {
			cp := *a
			f.autoPlays[i] = &cp
			f.published = append(f.published, evts...)
			return nil
		}
	}
	return apperrs.ErrNotFound
}

func (f *fakeRepo) DeleteAutoPlay(_ context.Context, id string, evts ...eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.autoPlayErr != nil {
		return f.autoPlayErr
	}
	for i, a := range f.autoPlays {
		if a.ID == id {
			f.autoPlays = slices.Delete(f.autoPlays, i, i+1)
			f.published = append(f.published, evts...)
			return nil
		}
	}
	return apperrs.ErrNotFound
}

func (f *fakeRepo) AutoPlayLimits(_ context.Context, workspaceID string) (AutoPlayLimits, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.autoPlayErr != nil {
		return AutoPlayLimits{}, f.autoPlayErr
	}
	if limits, ok := f.limits[workspaceID]; ok {
		return limits, nil
	}
	return AutoPlayLimits{DailyCapPerTicket: DefaultAutoPlayDailyCap}, nil
}

func (f *fakeRepo) SetAutoPlayLimits(_ context.Context, workspaceID string, limits AutoPlayLimits, evts ...eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.autoPlayErr != nil {
		return f.autoPlayErr
	}
	f.limits[workspaceID] = limits
	f.published = append(f.published, evts...)
	return nil
}

func (f *fakeRepo) Create(_ context.Context, p *Play, rev Revision, evts ...eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.createErr != nil {
		return f.createErr
	}
	if _, ok := f.byID[p.ID]; ok || f.labelTaken(p) {
		return apperrs.ErrConflict
	}
	cp := *p
	f.byID[p.ID] = &cp
	f.record(p, rev)
	f.published = append(f.published, evts...)
	return nil
}

// record keeps rev and the history it adds to, without the caps storage prunes to.
func (f *fakeRepo) record(p *Play, rev Revision) {
	f.revisions = append(f.revisions, rev)
	if v := rev.Instructions; v != nil {
		n := len(f.versions[p.ID]) + 1
		info := VersionInfo{Version: n, CreatedAt: p.UpdatedAt, EditorIDs: v.EditorIDs, Via: v.Via, BuiltinDefault: v.BuiltinDefault}
		f.versions[p.ID] = append(f.versions[p.ID], &VersionBody{VersionInfo: info, Body: v.Body})
	}
	f.changes[p.ID] = append(slices.Clone(rev.Changes), f.changes[p.ID]...)
}

func (f *fakeRepo) ListHistory(_ context.Context, playID string) (History, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	h := History{Changes: f.changes[playID]}
	for i := len(f.versions[playID]) - 1; i >= 0; i-- {
		h.Versions = append(h.Versions, f.versions[playID][i].VersionInfo)
	}
	return h, nil
}

func (f *fakeRepo) GetVersion(_ context.Context, playID string, version int) (*VersionBody, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if version < 1 || version > len(f.versions[playID]) {
		return nil, apperrs.ErrNotFound
	}
	cp := *f.versions[playID][version-1]
	return &cp, nil
}

// labelTaken mirrors the unique label index: another play of p's workspace has its label, ignoring case.
func (f *fakeRepo) labelTaken(p *Play) bool {
	for _, other := range f.byID {
		if other.ID != p.ID && other.WorkspaceID == p.WorkspaceID && strings.EqualFold(other.Label, p.Label) {
			return true
		}
	}
	return false
}

func (f *fakeRepo) GetByLabel(_ context.Context, workspaceID, label string) (*Play, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, p := range f.byID {
		if p.WorkspaceID == workspaceID && strings.EqualFold(p.Label, label) {
			cp := *p
			return &cp, nil
		}
	}
	return nil, apperrs.ErrNotFound
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

func (f *fakeRepo) Update(_ context.Context, p *Play, rev Revision, evts ...eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.updateErr != nil {
		return f.updateErr
	}
	if _, ok := f.byID[p.ID]; !ok {
		return apperrs.ErrNotFound
	}
	if f.labelTaken(p) {
		return apperrs.ErrConflict
	}
	cp := *p
	f.byID[p.ID] = &cp
	f.record(p, rev)
	f.published = append(f.published, evts...)
	return nil
}

func (f *fakeRepo) AutomationNamesByLabel(_ context.Context, _, label string) ([]string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.namesErr != nil {
		return nil, f.namesErr
	}
	return f.automationNames[label], nil
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
	// projects are the ones CallerProjects knows of; the caller sees each it is not kept out of.
	projects []string
}

func newFakePerm(grants map[string][]permissions.Action) *fakePerm {
	return &fakePerm{grants: grants, denied: map[string]map[string]bool{}}
}

func (f *fakePerm) HasPermission(_ context.Context, userID, _ string, action permissions.Action, resourceType, resourceID string) bool {
	if action == permissions.Member && resourceType == resourceTypeProject {
		return !f.denied[userID][resourceType+":"+resourceID]
	}
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
	key := resourceType + ":" + resourceID
	if resourceType != "" && (f.denied[userID][key] || f.denied[userID][key+"#"+string(action)]) {
		return false
	}
	return true
}

func (f *fakePerm) PlaysWith(ctx context.Context, userID, workspaceID string, action permissions.Action, playIDs []string) map[string]bool {
	out := map[string]bool{}
	for _, id := range playIDs {
		out[id] = f.HasPermission(ctx, userID, workspaceID, action, resourceTypePlay, id)
	}
	return out
}

func (f *fakePerm) CallerProjects(ctx context.Context, _ permissions.Action) ([]string, bool, error) {
	open := []string{}
	for _, p := range f.projects {
		if !f.denied[actorID(ctx)][resourceTypeProject+":"+p] {
			open = append(open, p)
		}
	}
	return open, false, nil
}

// hideProject makes projectID one userID may not open, as a Restricted member holding no access there.
func (f *fakePerm) hideProject(userID, projectID string) {
	f.denyResource(userID, resourceTypeProject, projectID)
}

// deny is a play rule denying userID Run on playID.
func (f *fakePerm) deny(userID, playID string) {
	f.denyResource(userID, resourceTypePlay, playID+"#"+string(permissions.PlaysRun))
}

// hide is a play rule denying userID See on playID.
func (f *fakePerm) hide(userID, playID string) {
	f.denyResource(userID, resourceTypePlay, playID+"#"+string(permissions.PlaysRead))
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
