package templates

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

var fixedNow = time.Date(2026, 10, 1, 9, 0, 0, 0, time.UTC)

type fakeRepo struct {
	mu      sync.Mutex
	rows    map[string]*Record
	events  []eventbus.OutboxEvent
	failing error
}

func newFakeRepo() *fakeRepo { return &fakeRepo{rows: map[string]*Record{}} }

func (f *fakeRepo) Get(_ context.Context, kind, key string) (*Record, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.failing != nil {
		return nil, f.failing
	}
	r, ok := f.rows[kind+"/"+key]
	if !ok {
		return nil, apperrs.ErrNotFound
	}
	cp := *r
	return &cp, nil
}

func (f *fakeRepo) List(context.Context) ([]*Record, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.failing != nil {
		return nil, f.failing
	}
	var out []*Record
	for _, r := range f.rows {
		cp := *r
		out = append(out, &cp)
	}
	return out, nil
}

func (f *fakeRepo) Save(_ context.Context, r *Record, evts ...eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.failing != nil {
		return f.failing
	}
	cp := *r
	f.rows[r.Kind+"/"+r.Key] = &cp
	f.events = append(f.events, evts...)
	return nil
}

func (f *fakeRepo) Delete(_ context.Context, kind, key string, evts ...eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.failing != nil {
		return f.failing
	}
	delete(f.rows, kind+"/"+key)
	f.events = append(f.events, evts...)
	return nil
}

// fakeGate holds templates:write for the users named in it.
type fakeGate map[string]bool

func (g fakeGate) RequireAnywhere(ctx context.Context, action permissions.Action) error {
	actor, _ := identity.ActorFromCtx(ctx)
	if g[actor.ID] && action == permissions.TemplatesWrite {
		return nil
	}
	return fmt.Errorf("%w: %s required", apperrs.ErrForbidden, action)
}

// fakeLayer is one lower layer: a text per location and key, absent until written unless it follows.
type fakeLayer struct {
	follows bool
	texts   map[string]string
}

func (l *fakeLayer) slot(at Location, key string) string {
	return at.WorkspaceID + at.ProjectID + "/" + strings.ToLower(key)
}

func (l *fakeLayer) Read(_ context.Context, at Location, key string) (string, bool, error) {
	body, ok := l.texts[l.slot(at, key)]
	if !ok && !l.follows {
		return "", false, fmt.Errorf("%w: no %s here", apperrs.ErrNotFound, key)
	}
	return body, ok, nil
}

func (l *fakeLayer) Write(_ context.Context, at Location, key, body string) error {
	if _, ok := l.texts[l.slot(at, key)]; !ok && !l.follows {
		return fmt.Errorf("%w: no %s here", apperrs.ErrNotFound, key)
	}
	l.texts[l.slot(at, key)] = body
	return nil
}

func (l *fakeLayer) Reset(ctx context.Context, at Location, key, instanceBody string) error {
	if l.follows {
		delete(l.texts, l.slot(at, key))
		return nil
	}
	return l.Write(ctx, at, key, instanceBody)
}

type fixture struct {
	svc       *Service
	repo      *fakeRepo
	followed  *fakeLayer
	copied    *fakeLayer
	workspace Location
	project   Location
}

// newFixture registers a following singleton kind "note" and a copied keyed kind "body" with keys bug and task;
// project p-1 holds a Bug and a Chore, and "admin" holds templates:write.
func newFixture() fixture {
	repo := newFakeRepo()
	followed := &fakeLayer{follows: true, texts: map[string]string{}}
	copied := &fakeLayer{texts: map[string]string{"p-1/bug": "bug in p-1", "p-1/chore": "chore in p-1", "p-2/bug": "bug in p-2"}}
	svc := NewService(repo, fakeGate{"admin": true},
		Kind{Name: "note", Below: ScopeWorkspace, Follows: true, Defaults: []Default{{Name: "Note", Body: "code note"}}, Layer: followed,
			Check: func(body string) error {
				if body == "" {
					return fmt.Errorf("%w: empty", apperrs.ErrInvalid)
				}
				return nil
			}},
		Kind{Name: "body", Below: ScopeProject, Defaults: []Default{{Key: "bug", Name: "bug", Body: "code bug"}, {Key: "task", Name: "task", Body: "code task"}}, Layer: copied},
	)
	svc.now = func() time.Time { return fixedNow }
	return fixture{svc: svc, repo: repo, followed: followed, copied: copied,
		workspace: Location{Scope: ScopeWorkspace, WorkspaceID: "ws-1"}, project: Location{Scope: ScopeProject, ProjectID: "p-1"}}
}

func as(userID string) context.Context {
	return identity.WithActor(context.Background(), identity.Actor{ID: userID})
}

func TestService_RefusesWhatItCannotPlace(t *testing.T) {
	f := newFixture()
	tests := []struct {
		name string
		call func() error
		want error
	}{
		{"an unknown kind", func() error { _, err := f.svc.Get(as("admin"), "stationery", "", Instance); return err }, apperrs.ErrInvalid},
		{"a layer the kind does not live at", func() error { _, err := f.svc.Get(as("admin"), "note", "", f.project); return err }, apperrs.ErrInvalid},
		{"a workspace without its id", func() error {
			_, err := f.svc.Get(as("admin"), "note", "", Location{Scope: ScopeWorkspace})
			return err
		}, apperrs.ErrInvalid},
		{"a project without its id", func() error {
			_, err := f.svc.Get(as("admin"), "body", "bug", Location{Scope: ScopeProject})
			return err
		}, apperrs.ErrInvalid},
		{"no scope at all", func() error { _, err := f.svc.Get(as("admin"), "note", "", Location{}); return err }, apperrs.ErrInvalid},
		{"a key on a kind that has none", func() error { _, err := f.svc.Get(as("admin"), "note", "x", Instance); return err }, apperrs.ErrInvalid},
		{"an instance key that does not exist", func() error { _, err := f.svc.Get(as("admin"), "body", "chore", Instance); return err }, apperrs.ErrNotFound},
		{"an instance write without templates:write", func() error {
			_, err := f.svc.Update(as("someone"), "body", "bug", Instance, "x")
			return err
		}, apperrs.ErrForbidden},
		{"an instance reset without templates:write", func() error { _, err := f.svc.Reset(as("someone"), "note", "", Instance); return err }, apperrs.ErrForbidden},
		{"a body the kind's check refuses", func() error { _, err := f.svc.Update(as("admin"), "note", "", Instance, ""); return err }, apperrs.ErrInvalid},
		{"a clone onto itself", func() error { _, err := f.svc.Clone(as("admin"), "note", "", Instance, Instance); return err }, apperrs.ErrInvalid},
		{"a clone from a bad source", func() error {
			_, err := f.svc.Clone(as("admin"), "note", "", f.project, Instance)
			return err
		}, apperrs.ErrInvalid},
		{"a clone to a bad target", func() error {
			_, err := f.svc.Clone(as("admin"), "note", "", Instance, f.project)
			return err
		}, apperrs.ErrInvalid},
		{"a clone with no match below", func() error {
			_, err := f.svc.Clone(as("admin"), "body", "chore", f.project, Location{Scope: ScopeProject, ProjectID: "p-2"})
			return err
		}, apperrs.ErrNotFound},
		{"a clone with no match at the instance", func() error {
			_, err := f.svc.Clone(as("admin"), "body", "chore", f.project, Instance)
			return err
		}, apperrs.ErrNotFound},
		{"a reset below for a key the instance lacks", func() error {
			_, err := f.svc.Reset(as("admin"), "body", "chore", f.project)
			return err
		}, apperrs.ErrNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.ErrorIs(t, tt.call(), tt.want)
		})
	}
	assert.Empty(t, f.repo.events, "nothing refused was written")
}

func TestService_InstanceOnlyKind(t *testing.T) {
	svc := NewService(newFakeRepo(), fakeGate{"admin": true}, Kind{Name: "frame", Defaults: []Default{{Key: "intro", Name: "Intro", Body: "hello"}}})

	_, err := svc.Update(as("admin"), "frame", "intro", Instance, "")
	require.NoError(t, err)
	body, err := svc.Effective(context.Background(), "frame", "intro")
	require.NoError(t, err)
	assert.Empty(t, body, "an emptied template reads as empty, not as the code default")

	_, err = svc.Clone(as("admin"), "frame", "intro", Instance, Location{Scope: ScopeWorkspace, WorkspaceID: "ws-1"})
	require.ErrorIs(t, err, apperrs.ErrInvalid)
	assert.ErrorContains(t, err, "frame templates live only at the instance")
	_, err = svc.Reset(as("admin"), "frame", "intro", Location{Scope: ScopeProject, ProjectID: "p-1"})
	assert.ErrorContains(t, err, "frame templates live only at the instance")
}

func TestService_StorageFailuresSurface(t *testing.T) {
	f := newFixture()
	boom := errors.New("disk")
	f.repo.failing = boom
	_, err := f.svc.List(as("admin"))
	require.ErrorIs(t, err, boom)
	_, err = f.svc.Effective(context.Background(), "note", "")
	require.ErrorIs(t, err, boom)
	_, err = f.svc.Update(as("admin"), "note", "", Instance, "x")
	require.ErrorIs(t, err, boom)
	_, err = f.svc.Reset(as("admin"), "note", "", Instance)
	require.ErrorIs(t, err, boom)
	_, err = f.svc.Get(as("admin"), "note", "", f.workspace)
	require.ErrorIs(t, err, boom, "a lower layer's default comes from the instance")
}

func TestService_InstanceLayer(t *testing.T) {
	f := newFixture()

	got, err := f.svc.Get(as("anyone"), "body", "BUG", Instance)
	require.NoError(t, err)
	assert.Equal(t, &Template{Kind: "body", Key: "bug", Name: "bug", Scope: ScopeInstance, Body: "code bug", DefaultBody: "code bug"}, got, "unedited: the code default, matched ignoring case")

	got, err = f.svc.Update(as("admin"), "body", "bug", Instance, "ours")
	require.NoError(t, err)
	assert.Equal(t, "ours", got.Body)
	assert.Equal(t, "code bug", got.DefaultBody)
	assert.True(t, got.Edited)
	assert.Equal(t, "admin", got.UpdatedBy)
	require.Len(t, f.repo.events, 1)
	assert.Equal(t, UpdatedEvent{Kind: "body", Key: "bug", AuthorID: "admin", UpdatedAt: fixedNow}, f.repo.events[0].Payload)

	list, err := f.svc.List(as("anyone"))
	require.NoError(t, err)
	var keys []string
	for _, t := range list {
		keys = append(keys, t.Kind+"/"+t.Key+"="+t.Body)
	}
	assert.Equal(t, []string{"note/=code note", "body/bug=ours", "body/task=code task"}, keys, "registration order, stored text where edited")

	got, err = f.svc.Reset(as("admin"), "body", "bug", Instance)
	require.NoError(t, err)
	assert.Equal(t, "code bug", got.Body)
	assert.False(t, got.Edited)
	assert.True(t, f.repo.events[1].Payload.(UpdatedEvent).Reset)
}

func TestService_LowerLayers(t *testing.T) {
	f := newFixture()
	_, err := f.svc.Update(as("admin"), "note", "", Instance, "instance note")
	require.NoError(t, err)

	got, err := f.svc.Get(as("member"), "note", "", f.workspace)
	require.NoError(t, err)
	assert.Equal(t, "", got.Body, "the fake layer reads nothing until written; the domain resolves the follow itself")
	assert.Equal(t, "instance note", got.DefaultBody)
	assert.False(t, got.Edited)
	assert.True(t, got.Follows)

	got, err = f.svc.Update(as("member"), "note", "", f.workspace, "workspace note")
	require.NoError(t, err)
	assert.True(t, got.Edited, "a following kind is edited once it holds its own")

	got, err = f.svc.Get(as("member"), "body", "Chore", f.project)
	require.NoError(t, err)
	assert.Equal(t, "Chore", got.Key, "a project-only type keeps its own name and has no default")
	assert.Equal(t, "", got.DefaultBody)

	got, err = f.svc.Reset(as("member"), "body", "bug", f.project)
	require.NoError(t, err)
	assert.Equal(t, "code bug", got.Body)
	assert.False(t, got.Edited, "a copy equal to the instance is not edited")

	got, err = f.svc.Clone(as("member"), "body", "bug", Location{Scope: ScopeProject, ProjectID: "p-2"}, f.project)
	require.NoError(t, err)
	assert.Equal(t, "bug in p-2", got.Body)
	assert.True(t, got.Edited)
}

func TestTemplateUpdateTool(t *testing.T) {
	f := newFixture()
	tool := MCPTools(f.svc)[1]
	call := func(args string) (*Template, error) {
		out, err := tool.Call(as("admin"), json.RawMessage(args))
		if err != nil {
			return nil, err
		}
		return out.(*Template), nil
	}

	_, err := call(`{"kind":"body","key":"bug","scope":"instance","body":"x","reset":true}`)
	require.ErrorIs(t, err, apperrs.ErrInvalid, "one change at a time")
	_, err = call(`{"kind":"body","key":"bug"}`)
	require.ErrorIs(t, err, apperrs.ErrInvalid, "the scope is required, so the instance is never written by omission")

	got, err := call(`{"kind":"body","key":"bug","scope":"instance","body":"via mcp"}`)
	require.NoError(t, err)
	assert.Equal(t, "via mcp", got.Body)
	got, err = call(`{"kind":"body","key":"bug","scope":"instance"}`)
	require.NoError(t, err)
	assert.Equal(t, "via mcp", got.Body, "no change requested leaves it as it stands")

	got, err = call(`{"kind":"body","key":"bug","scope":"project","project_id":"p-1","clone_from":{"scope":"instance"}}`)
	require.NoError(t, err)
	assert.Equal(t, "via mcp", got.Body)
	got, err = call(`{"kind":"body","key":"bug","scope":"instance","reset":true}`)
	require.NoError(t, err)
	assert.Equal(t, "code bug", got.Body)

	read, err := MCPTools(f.svc)[0].Call(as("member"), json.RawMessage(`{"kind":"body","key":"bug","scope":"project","project_id":"p-1"}`))
	require.NoError(t, err)
	assert.Equal(t, "via mcp", read.(*Template).Body)
}

func TestHandler(t *testing.T) {
	f := newFixture()
	routes := NewHandler(f.svc).Routes()
	tests := []struct {
		user, method, path, body string
		want                     int
	}{
		{"member", http.MethodGet, "/api/templates", "", http.StatusOK},
		{"member", http.MethodGet, "/api/templates/body?key=task", "", http.StatusOK},
		{"member", http.MethodGet, "/api/templates/stationery", "", http.StatusBadRequest},
		{"member", http.MethodPut, "/api/templates/note", `{"body":"x"}`, http.StatusForbidden},
		{"admin", http.MethodPut, "/api/templates/note", `{`, http.StatusBadRequest},
		{"admin", http.MethodPut, "/api/templates/body", `{"key":"bug","body":"x"}`, http.StatusOK},
		{"admin", http.MethodDelete, "/api/templates/body?key=bug", "", http.StatusOK},
		{"admin", http.MethodPost, "/api/templates/clone", `{`, http.StatusBadRequest},
		{"admin", http.MethodPost, "/api/templates/clone", `{"kind":"body","key":"chore","from":{"scope":"project","project_id":"p-1"},"to":{"scope":"instance"}}`, http.StatusNotFound},
		{"admin", http.MethodPost, "/api/templates/clone", `{"kind":"body","key":"bug","from":{"scope":"project","project_id":"p-2"},"to":{"scope":"instance"}}`, http.StatusOK},
		{"admin", http.MethodPost, "/api/templates/reset", `{`, http.StatusBadRequest},
		{"admin", http.MethodPost, "/api/templates/reset", `{"kind":"note","at":{"scope":"project","project_id":"p-1"}}`, http.StatusBadRequest},
		{"admin", http.MethodPost, "/api/templates/reset", `{"kind":"note","at":{"scope":"workspace","workspace_id":"ws-1"}}`, http.StatusOK},
	}
	for _, tt := range tests {
		rec := httptest.NewRecorder()
		routes.ServeHTTP(rec, httptest.NewRequest(tt.method, tt.path, strings.NewReader(tt.body)).WithContext(as(tt.user)))
		assert.Equal(t, tt.want, rec.Code, "%s %s as %s: %s", tt.method, tt.path, tt.user, rec.Body.String())
	}
}

func TestHandler_GetBelowTheInstance_ReadsThatLocation(t *testing.T) {
	routes := NewHandler(newFixture().svc).Routes()
	get := func(path string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		routes.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil).WithContext(as("member")))
		return rec
	}

	rec := get("/api/templates/body?key=bug&scope=project&project_id=p-1")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var got Template
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	assert.Equal(t, "bug in p-1", got.Body)
	assert.Equal(t, "code bug", got.DefaultBody)
	assert.True(t, got.Edited)

	assert.Equal(t, http.StatusBadRequest, get("/api/templates/body?key=bug&scope=project").Code)
	assert.Equal(t, http.StatusNotFound, get("/api/templates/body?key=chore&scope=project&project_id=p-2").Code)
}

func TestTopics(t *testing.T) {
	assert.Equal(t, []string{"instance_template.updated"}, Topics())
}
