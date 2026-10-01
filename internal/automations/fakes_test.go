package automations

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"slices"
	"strings"
	"sync"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// testLogger discards output so tests don't spam stderr with expected
// warnings from error-path assertions.
func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// fakeRepo is an in-memory Repo (the testing standard: prefer fakes over
// mocks for repos).
type fakeRepo struct {
	mu   sync.Mutex
	rows map[string]*Automation
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{rows: map[string]*Automation{}}
}

func (f *fakeRepo) Create(_ context.Context, a *Automation) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	cp := *a
	f.rows[a.ID] = &cp
	return nil
}

func (f *fakeRepo) Get(_ context.Context, id string) (*Automation, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	a, ok := f.rows[id]
	if !ok {
		return nil, apperrs.ErrNotFound
	}
	cp := *a
	return &cp, nil
}

func (f *fakeRepo) GetByTokenHash(_ context.Context, hash string) (*Automation, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, a := range f.rows {
		if a.TokenHash == hash {
			cp := *a
			return &cp, nil
		}
	}
	return nil, apperrs.ErrNotFound
}

func (f *fakeRepo) List(_ context.Context) ([]Automation, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]Automation, 0, len(f.rows))
	for _, a := range f.rows {
		out = append(out, *a)
	}
	return out, nil
}

func (f *fakeRepo) Update(_ context.Context, a *Automation) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.rows[a.ID]; !ok {
		return apperrs.ErrNotFound
	}
	cp := *a
	f.rows[a.ID] = &cp
	return nil
}

func (f *fakeRepo) Delete(_ context.Context, id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.rows[id]; !ok {
		return apperrs.ErrNotFound
	}
	delete(f.rows, id)
	return nil
}

// fakePerm grants the listed actions per user in every workspace, or only members' ones; outside it is not found.
type fakePerm struct {
	grants  map[string][]permissions.Action
	members map[string][]string
}

func newFakePerm(grants map[string][]permissions.Action) *fakePerm {
	return &fakePerm{grants: grants}
}

func (f *fakePerm) Require(ctx context.Context, workspaceID string, action permissions.Action) error {
	actor, _ := identity.ActorFromCtx(ctx)
	if ws, ok := f.members[actor.ID]; ok && !slices.Contains(ws, workspaceID) {
		return fmt.Errorf("%w: workspace %s", apperrs.ErrNotFound, workspaceID)
	}
	if slices.Contains(f.grants[actor.ID], action) {
		return nil
	}
	return fmt.Errorf("%w: %s required", apperrs.ErrForbidden, action)
}

// allowAll grants a user every automation action, the common "owner" fixture.
func allowAll(userID string) *fakePerm {
	return newFakePerm(map[string][]permissions.Action{
		userID: {
			permissions.AutomationsWrite,
			permissions.AutomationsRead,
			permissions.AutomationsWrite,
			permissions.AutomationsDelete,
		},
	})
}

// fakeVersionsRepo is an in-memory VersionsRepo mirroring the real repo's
// invariants (at most one pending, at most one active per automation) so
// service-level tests catch the same bugs the SQLite repo's tests do.
type fakeVersionsRepo struct {
	mu   sync.Mutex
	rows map[string]*Version
}

func newFakeVersionsRepo() *fakeVersionsRepo {
	return &fakeVersionsRepo{rows: map[string]*Version{}}
}

func (f *fakeVersionsRepo) InsertActive(_ context.Context, v *Version) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	v.Sequence = f.nextSequence(v.AutomationID)
	cp := *v
	f.rows[v.ID] = &cp
	return nil
}

func (f *fakeVersionsRepo) ReplacePending(_ context.Context, v *Version) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	// Sequence first, discard second — mirrors the real repo: a replaced
	// pending version's number is never reused.
	v.Sequence = f.nextSequence(v.AutomationID)
	for id, row := range f.rows {
		if row.AutomationID == v.AutomationID && row.Status == VersionPending {
			delete(f.rows, id)
		}
	}
	cp := *v
	f.rows[v.ID] = &cp
	return nil
}

func (f *fakeVersionsRepo) Activate(_ context.Context, automationID, versionID string) (*Version, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	target, ok := f.rows[versionID]
	if !ok || target.AutomationID != automationID {
		return nil, apperrs.ErrNotFound
	}
	for _, row := range f.rows {
		if row.AutomationID == automationID && row.Status == VersionActive {
			row.Status = VersionInactive
		}
	}
	target.Status = VersionActive
	cp := *target
	return &cp, nil
}

func (f *fakeVersionsRepo) Get(_ context.Context, automationID, versionID string) (*Version, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	v, ok := f.rows[versionID]
	if !ok || v.AutomationID != automationID {
		return nil, apperrs.ErrNotFound
	}
	cp := *v
	return &cp, nil
}

func (f *fakeVersionsRepo) ListByAutomation(_ context.Context, automationID string) ([]Version, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []Version
	for _, v := range f.rows {
		if v.AutomationID == automationID {
			out = append(out, *v)
		}
	}
	return out, nil
}

func (f *fakeVersionsRepo) Pending(ctx context.Context, automationID string) (*Version, error) {
	return f.byStatus(automationID, VersionPending)
}

func (f *fakeVersionsRepo) Active(ctx context.Context, automationID string) (*Version, error) {
	return f.byStatus(automationID, VersionActive)
}

func (f *fakeVersionsRepo) byStatus(automationID string, status VersionStatus) (*Version, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, v := range f.rows {
		if v.AutomationID == automationID && v.Status == status {
			cp := *v
			return &cp, nil
		}
	}
	return nil, apperrs.ErrNotFound
}

func (f *fakeVersionsRepo) nextSequence(automationID string) int {
	max := 0
	for _, v := range f.rows {
		if v.AutomationID == automationID && v.Sequence > max {
			max = v.Sequence
		}
	}
	return max + 1
}

// fakeSecretsRepo is an in-memory SecretsRepo; it stores plaintext (no
// crypto dependency in unit tests — encryption is exercised by the real
// storage-layer repo's own tests).
type fakeSecretsRepo struct {
	mu   sync.Mutex
	rows map[string]struct {
		value     string
		createdAt time.Time
		updatedAt time.Time
	}
}

func newFakeSecretsRepo() *fakeSecretsRepo {
	return &fakeSecretsRepo{rows: map[string]struct {
		value     string
		createdAt time.Time
		updatedAt time.Time
	}{}}
}

// secretKey keys a secret by workspace and name, the real table's primary key.
func secretKey(workspaceID, name string) string { return workspaceID + "/" + name }

func (f *fakeSecretsRepo) Set(_ context.Context, workspaceID, name, value string, now time.Time) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	name = secretKey(workspaceID, name)
	row, existed := f.rows[name]
	if !existed {
		row.createdAt = now
	}
	row.value = value
	row.updatedAt = now
	f.rows[name] = row
	return nil
}

func (f *fakeSecretsRepo) Delete(_ context.Context, workspaceID, name string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.rows, secretKey(workspaceID, name))
	return nil
}

func (f *fakeSecretsRepo) List(_ context.Context, workspaceID string) ([]SecretMeta, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]SecretMeta, 0, len(f.rows))
	for key, row := range f.rows {
		name, ok := strings.CutPrefix(key, workspaceID+"/")
		if !ok {
			continue
		}
		out = append(out, SecretMeta{Name: name, CreatedAt: row.createdAt, UpdatedAt: row.updatedAt})
	}
	return out, nil
}

func (f *fakeSecretsRepo) All(_ context.Context, workspaceID string) (map[string]string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make(map[string]string, len(f.rows))
	for key, row := range f.rows {
		if name, ok := strings.CutPrefix(key, workspaceID+"/"); ok {
			out[name] = row.value
		}
	}
	return out, nil
}

// erroringRepo wraps a fakeRepo, forcing whichever method's field is set to
// fail with errBoom instead of delegating.
type erroringRepo struct {
	*fakeRepo
	getErr    error
	createErr error
}

func newErroringRepo() *erroringRepo {
	return &erroringRepo{fakeRepo: newFakeRepo()}
}

func (f *erroringRepo) Get(ctx context.Context, id string) (*Automation, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	return f.fakeRepo.Get(ctx, id)
}

func (f *erroringRepo) Create(ctx context.Context, a *Automation) error {
	if f.createErr != nil {
		return f.createErr
	}
	return f.fakeRepo.Create(ctx, a)
}

// errBoom is a generic non-sentinel failure used to exercise the
// "unexpected repo error" branches (as opposed to the expected
// apperrs.ErrNotFound ones) that a happy-path fake can never produce.
var errBoom = errors.New("boom")

// erroringVersionsRepo wraps a fakeVersionsRepo, forcing whichever method's
// field is set to fail with errBoom instead of delegating.
type erroringVersionsRepo struct {
	*fakeVersionsRepo
	activeErr         error
	pendingErr        error
	insertActiveErr   error
	replacePendingErr error
	activateErr       error
	listErr           error
}

func newErroringVersionsRepo() *erroringVersionsRepo {
	return &erroringVersionsRepo{fakeVersionsRepo: newFakeVersionsRepo()}
}

func (f *erroringVersionsRepo) Active(ctx context.Context, automationID string) (*Version, error) {
	if f.activeErr != nil {
		return nil, f.activeErr
	}
	return f.fakeVersionsRepo.Active(ctx, automationID)
}

func (f *erroringVersionsRepo) Pending(ctx context.Context, automationID string) (*Version, error) {
	if f.pendingErr != nil {
		return nil, f.pendingErr
	}
	return f.fakeVersionsRepo.Pending(ctx, automationID)
}

func (f *erroringVersionsRepo) InsertActive(ctx context.Context, v *Version) error {
	if f.insertActiveErr != nil {
		return f.insertActiveErr
	}
	return f.fakeVersionsRepo.InsertActive(ctx, v)
}

func (f *erroringVersionsRepo) ReplacePending(ctx context.Context, v *Version) error {
	if f.replacePendingErr != nil {
		return f.replacePendingErr
	}
	return f.fakeVersionsRepo.ReplacePending(ctx, v)
}

func (f *erroringVersionsRepo) Activate(ctx context.Context, automationID, versionID string) (*Version, error) {
	if f.activateErr != nil {
		return nil, f.activateErr
	}
	return f.fakeVersionsRepo.Activate(ctx, automationID, versionID)
}

func (f *erroringVersionsRepo) ListByAutomation(ctx context.Context, automationID string) ([]Version, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	return f.fakeVersionsRepo.ListByAutomation(ctx, automationID)
}

// erroringSecretsRepo wraps a fakeSecretsRepo, forcing whichever method's
// field is set to fail with errBoom instead of delegating.
type erroringSecretsRepo struct {
	*fakeSecretsRepo
	setErr    error
	deleteErr error
	listErr   error
	allErr    error
}

func newErroringSecretsRepo() *erroringSecretsRepo {
	return &erroringSecretsRepo{fakeSecretsRepo: newFakeSecretsRepo()}
}

func (f *erroringSecretsRepo) Set(ctx context.Context, workspaceID, name, value string, now time.Time) error {
	if f.setErr != nil {
		return f.setErr
	}
	return f.fakeSecretsRepo.Set(ctx, workspaceID, name, value, now)
}

func (f *erroringSecretsRepo) Delete(ctx context.Context, workspaceID, name string) error {
	if f.deleteErr != nil {
		return f.deleteErr
	}
	return f.fakeSecretsRepo.Delete(ctx, workspaceID, name)
}

func (f *erroringSecretsRepo) List(ctx context.Context, workspaceID string) ([]SecretMeta, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	return f.fakeSecretsRepo.List(ctx, workspaceID)
}

func (f *erroringSecretsRepo) All(ctx context.Context, workspaceID string) (map[string]string, error) {
	if f.allErr != nil {
		return nil, f.allErr
	}
	return f.fakeSecretsRepo.All(ctx, workspaceID)
}

// fakeConnRegistry records Disconnect calls (the testing standard: a fake
// over a mock, since the assertion is just "was it called with what").
type fakeConnRegistry struct {
	mu      sync.Mutex
	calls   []string
	reasons []string
}

func (f *fakeConnRegistry) Disconnect(automationID, reason string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, automationID)
	f.reasons = append(f.reasons, reason)
}

func (f *fakeConnRegistry) SkipBacklog(context.Context, string) error {
	return nil
}

func (f *fakeConnRegistry) calledWith() ([]string, []string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.calls...), append([]string(nil), f.reasons...)
}

// fakeRunsRepo is an in-memory RunsRepo.
type fakeRunsRepo struct {
	mu   sync.Mutex
	rows map[string]*Run
}

func newFakeRunsRepo() *fakeRunsRepo {
	return &fakeRunsRepo{rows: map[string]*Run{}}
}

func (f *fakeRunsRepo) Create(_ context.Context, r *Run) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	cp := *r
	f.rows[r.ID] = &cp
	return nil
}

func (f *fakeRunsRepo) Get(_ context.Context, id string) (*Run, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	r, ok := f.rows[id]
	if !ok {
		return nil, apperrs.ErrNotFound
	}
	cp := *r
	return &cp, nil
}

func (f *fakeRunsRepo) ListByAutomation(_ context.Context, automationID string, limit int) ([]Run, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []Run
	for _, r := range f.rows {
		if r.AutomationID == automationID {
			out = append(out, *r)
		}
	}
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (f *fakeRunsRepo) DeleteOlderThan(_ context.Context, before time.Time) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var n int64
	for id, r := range f.rows {
		if r.CreatedAt.Before(before) {
			delete(f.rows, id)
			n++
		}
	}
	return n, nil
}
