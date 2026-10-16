package deploy

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// fakeWatchRepo keeps the opt-outs in a map, as the stack_unwatchers table does.
type fakeWatchRepo struct {
	mu  sync.Mutex
	out map[string]bool // "stack:user"
	err error
}

func (f *fakeWatchRepo) Unwatching(_ context.Context, stackID, userID string) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.out[stackID+":"+userID], f.err
}

func (f *fakeWatchRepo) ListUnwatcherIDs(context.Context, string) ([]string, error) {
	return nil, f.err
}

func (f *fakeWatchRepo) SetWatching(_ context.Context, stackID, userID string, watching bool, _ time.Time) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return f.err
	}
	f.out[stackID+":"+userID] = !watching
	return nil
}

func watchFixture(t *testing.T) (*Service, *Stack, *fakeWatchRepo) {
	t.Helper()
	s, stack, _ := stackFixture(t)
	repo := &fakeWatchRepo{out: map[string]bool{}}
	s.SetStackWatchRepo(repo)
	return s, stack, repo
}

func asUserID(t *testing.T, id string) context.Context {
	t.Helper()
	return identity.WithActor(t.Context(), identity.Actor{ID: id})
}

func TestWatching_IsOnUntilTheViewerOptsOutAndBackIn(t *testing.T) {
	s, stack, _ := watchFixture(t)
	alice, bob := asUserID(t, "alice"), asUserID(t, "bob")

	got, err := s.Watching(alice, stack.ID)
	require.NoError(t, err)
	assert.True(t, got, "everyone who reads a stack watches it by default")

	got, err = s.SetWatching(alice, stack.ID, false)
	require.NoError(t, err)
	assert.False(t, got)
	got, err = s.Watching(alice, stack.ID)
	require.NoError(t, err)
	assert.False(t, got)
	got, err = s.Watching(bob, stack.ID)
	require.NoError(t, err)
	assert.True(t, got, "an opt-out is the caller's own")

	got, err = s.SetWatching(alice, stack.ID, true)
	require.NoError(t, err)
	assert.True(t, got, "turning it back on is the way out")
	got, err = s.Watching(alice, stack.ID)
	require.NoError(t, err)
	assert.True(t, got)
}

func TestWatching_Errors(t *testing.T) {
	t.Run("an unknown stack is not found", func(t *testing.T) {
		s, _, _ := watchFixture(t)

		_, err := s.SetWatching(asUserID(t, "alice"), "nope", false)
		assert.ErrorIs(t, err, apperrs.ErrNotFound)
		_, err = s.Watching(asUserID(t, "alice"), "nope")
		assert.ErrorIs(t, err, apperrs.ErrNotFound)
	})
	t.Run("a blank stack id is invalid", func(t *testing.T) {
		s, _, _ := watchFixture(t)

		_, err := s.Watching(asUserID(t, "alice"), " ")
		assert.ErrorIs(t, err, apperrs.ErrInvalid)
	})
	t.Run("a caller who cannot read the stack cannot change it", func(t *testing.T) {
		s, stack, _ := watchFixture(t)
		s.SetGate(refusingGate{refused: []permissions.Action{permissions.StacksRead}})

		_, err := s.SetWatching(asUserID(t, "alice"), stack.ID, false)
		assert.ErrorIs(t, err, apperrs.ErrForbidden)
	})
	t.Run("a read-only caller may still turn it off", func(t *testing.T) {
		s, stack, _ := watchFixture(t)
		s.SetGate(refusingGate{refused: []permissions.Action{permissions.StacksWrite}})

		got, err := s.SetWatching(asUserID(t, "alice"), stack.ID, false)
		require.NoError(t, err)
		assert.False(t, got)
	})
	t.Run("the server's own call has no one to watch for", func(t *testing.T) {
		s, stack, _ := watchFixture(t)

		_, err := s.SetWatching(t.Context(), stack.ID, false)
		assert.ErrorIs(t, err, apperrs.ErrUnauthorized)
		got, err := s.Watching(t.Context(), stack.ID)
		require.NoError(t, err)
		assert.False(t, got)
	})
	t.Run("a failing store surfaces", func(t *testing.T) {
		s, stack, repo := watchFixture(t)
		repo.err = errors.New("db down")

		_, err := s.SetWatching(asUserID(t, "alice"), stack.ID, false)
		assert.ErrorIs(t, err, repo.err)
		_, err = s.Watching(asUserID(t, "alice"), stack.ID)
		assert.ErrorIs(t, err, repo.err)
	})
	t.Run("without storage every viewer reads as watching and cannot change it", func(t *testing.T) {
		s, stack, _ := stackFixture(t)

		got, err := s.Watching(asUserID(t, "alice"), stack.ID)
		require.NoError(t, err)
		assert.True(t, got)
		_, err = s.SetWatching(asUserID(t, "alice"), stack.ID, false)
		assert.ErrorIs(t, err, apperrs.ErrInvalid)
	})
}

func TestStackMCP_Watching(t *testing.T) {
	t.Run("stack_get shows the caller's state and stack_update changes only that", func(t *testing.T) {
		s, stack, _ := watchFixture(t)
		alice := asUserID(t, "alice")

		got, err := callAs(t, alice, s, "stack_get", `{"id":"`+stack.ID+`"}`)
		require.NoError(t, err)
		assert.True(t, got.(stackDetail).Watching)

		updated, err := callAs(t, alice, s, "stack_update", `{"id":"`+stack.ID+`","watching":false}`)
		require.NoError(t, err)
		assert.False(t, updated.(stackUpdated).Watching)
		assert.Equal(t, stack.Name, updated.(stackUpdated).Name, "nothing else changed")

		got, err = callAs(t, alice, s, "stack_get", `{"id":"`+stack.ID+`"}`)
		require.NoError(t, err)
		assert.False(t, got.(stackDetail).Watching)
	})
	t.Run("watching alone needs read and not write", func(t *testing.T) {
		s, stack, _ := watchFixture(t)
		s.SetGate(refusingGate{refused: []permissions.Action{permissions.StacksWrite}})
		alice := asUserID(t, "alice")

		_, err := callAs(t, alice, s, "stack_update", `{"id":"`+stack.ID+`","watching":false}`)
		require.NoError(t, err)
		_, err = callAs(t, alice, s, "stack_update", `{"id":"`+stack.ID+`","watching":false,"name":"renamed"}`)
		assert.ErrorIs(t, err, apperrs.ErrForbidden, "a field beside it still needs write")
	})
	t.Run("an edit with no watching choice reports the current state", func(t *testing.T) {
		s, stack, repo := watchFixture(t)
		repo.out[stack.ID+":alice"] = true

		updated, err := callAs(t, asUserID(t, "alice"), s, "stack_update", `{"id":"`+stack.ID+`","name":"renamed"}`)
		require.NoError(t, err)
		assert.False(t, updated.(stackUpdated).Watching)
		assert.Equal(t, "renamed", updated.(stackUpdated).Name)
	})
}

func TestWatchRoutes(t *testing.T) {
	routes := func(t *testing.T) (http.Handler, *fakeWatchRepo) {
		t.Helper()
		stacks := newFakeStackRepo()
		projects := newFakeProjects()
		projects.exists["proj-1"] = true
		requireStack(t, stacks, validStack())
		s := newTestServiceWith(newFakeRepo(), stacks, newFakeContainerRepo(), projects, newFakeBus())
		repo := &fakeWatchRepo{out: map[string]bool{}}
		s.SetStackWatchRepo(repo)
		return NewHandler(s).Routes(), repo
	}
	send := func(h http.Handler, method, path string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, withActor(httptest.NewRequest(method, path, nil)))
		return rec
	}

	t.Run("DELETE opts out, PUT opts back in, and the stack read follows", func(t *testing.T) {
		h, _ := routes(t)

		rec := send(h, http.MethodDelete, "/api/stacks/svc-1/watchers/me")
		require.Equal(t, http.StatusOK, rec.Code)
		assert.JSONEq(t, `{"watching":false}`, rec.Body.String())

		var got stackResponse
		decodeDeploy(t, send(h, http.MethodGet, "/api/stacks/svc-1"), &got)
		assert.False(t, got.Watching)

		rec = send(h, http.MethodPut, "/api/stacks/svc-1/watchers/me")
		require.Equal(t, http.StatusOK, rec.Code)
		assert.JSONEq(t, `{"watching":true}`, rec.Body.String())
		decodeDeploy(t, send(h, http.MethodGet, "/api/stacks/svc-1"), &got)
		assert.True(t, got.Watching)
	})
	t.Run("an unknown stack is 404", func(t *testing.T) {
		h, _ := routes(t)

		assert.Equal(t, http.StatusNotFound, send(h, http.MethodPut, "/api/stacks/nope/watchers/me").Code)
	})
	t.Run("a stack read fails when the store does", func(t *testing.T) {
		h, repo := routes(t)
		repo.err = errors.New("db down")

		assert.Equal(t, http.StatusInternalServerError, send(h, http.MethodGet, "/api/stacks/svc-1").Code)
	})
	t.Run("the deprecated services alias has no watching routes", func(t *testing.T) {
		h, _ := routes(t)

		assert.Equal(t, http.StatusNotFound, send(h, http.MethodPut, "/api/services/svc-1/watchers/me").Code)
	})
}
