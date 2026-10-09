package access

import (
	"context"
	"errors"
	"sync"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/wake"
)

// Memo holds the access reads one request has already made: memberships, overwrites, and where projects and docs
// live, keyed by who and what they are about. Every answer is dropped at the next commit anywhere, so a memo never
// answers past a write (ADR 0135).
type Memo struct {
	commits *wake.Broadcast
	mu      sync.Mutex
	fence   <-chan struct{}
	reads   map[any]memoRead
}

type memoRead struct {
	value any
	err   error
}

type memoKey struct{}

// SetCommits wires the storage commit broadcast that clears every memo; unset, a memo lasts as long as its context.
func (s *Service) SetCommits(commits *wake.Broadcast) {
	s.commits = commits
}

// NewMemo starts an empty memo; install it with WithMemo where a request attaches its actor.
func (s *Service) NewMemo() *Memo {
	m := &Memo{commits: s.commits, reads: map[any]memoRead{}}
	if m.commits != nil {
		m.fence = m.commits.Next()
	}
	return m
}

// WithMemo carries m on ctx, so every access check made with ctx reads each layer at most once.
func WithMemo(ctx context.Context, m *Memo) context.Context {
	return context.WithValue(ctx, memoKey{}, m)
}

// memoized is ctx with a memo, a fresh one when ctx carries none, for a batch that would otherwise read per item.
func (s *Service) memoized(ctx context.Context) context.Context {
	if _, ok := ctx.Value(memoKey{}).(*Memo); ok {
		return ctx
	}
	return WithMemo(ctx, s.NewMemo())
}

// get returns key's answer and the fence it is valid under, first dropping everything if a commit has landed.
func (m *Memo) get(key any) (<-chan struct{}, memoRead, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	select {
	case <-m.fence:
		m.fence = m.commits.Next()
		clear(m.reads)
	default:
	}
	r, ok := m.reads[key]
	return m.fence, r, ok
}

// put keeps an answer read under fence; one read before a commit the memo has since seen is thrown away.
func (m *Memo) put(fence <-chan struct{}, key any, r memoRead) {
	if r.err != nil && !errors.Is(r.err, apperrs.ErrNotFound) {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if fence == m.fence {
		m.reads[key] = r
	}
}

// remember answers key from ctx's memo, reading it once when absent; a failure other than not found is never kept.
func remember[V any](ctx context.Context, key any, read func() (V, error)) (V, error) {
	m, ok := ctx.Value(memoKey{}).(*Memo)
	if !ok {
		return read()
	}
	fence, r, found := m.get(key)
	if found {
		v, _ := r.value.(V)
		return v, r.err
	}
	v, err := read()
	m.put(fence, key, memoRead{value: v, err: err})
	return v, err
}

// rememberEach is remember for many keys of one kind, reading every missing one in a single call.
func rememberEach[V any](ctx context.Context, ids []string, key func(id string) any, read func(missing []string) (map[string]V, error)) (map[string]V, error) {
	m, ok := ctx.Value(memoKey{}).(*Memo)
	if !ok {
		return read(ids)
	}
	out := make(map[string]V, len(ids))
	var missing []string
	var fence <-chan struct{}
	for _, id := range ids {
		f, r, found := m.get(key(id))
		fence = f
		if !found {
			missing = append(missing, id)
			continue
		}
		if v, ok := r.value.(V); ok && r.err == nil {
			out[id] = v
		}
	}
	if len(missing) == 0 {
		return out, nil
	}
	got, err := read(missing)
	if err != nil {
		return nil, err
	}
	for _, id := range missing {
		v, ok := got[id]
		if !ok {
			m.put(fence, key(id), memoRead{err: apperrs.ErrNotFound})
			continue
		}
		out[id] = v
		m.put(fence, key(id), memoRead{value: v})
	}
	return out, nil
}

// The memo's keys, one type per kind of read, so two kinds never share an entry.
type (
	memberKey     struct{ workspaceID, userID string }
	overwriteKey  struct{ resourceType, resourceID, userID string }
	projectKey    struct{ projectID string }
	docKey        struct{ docID string }
	workspacesKey struct{ userID string }
	membershipKey struct{ userID string }
)

func (s *Service) memberRole(ctx context.Context, workspaceID, userID string) (RoleInfo, error) {
	return remember(ctx, memberKey{workspaceID, userID}, func() (RoleInfo, error) {
		return s.roles.MemberRole(ctx, workspaceID, userID)
	})
}

func (s *Service) overwrite(ctx context.Context, resourceType, resourceID, userID string) (*Overwrite, error) {
	return remember(ctx, overwriteKey{resourceType, resourceID, userID}, func() (*Overwrite, error) {
		return s.repo.Get(ctx, resourceType, resourceID, userID)
	})
}

func (s *Service) overwrites(ctx context.Context, resourceType string, resourceIDs []string, userID string) (map[string]*Overwrite, error) {
	key := func(id string) any { return overwriteKey{resourceType, id, userID} }
	return rememberEach(ctx, resourceIDs, key, func(missing []string) (map[string]*Overwrite, error) {
		return s.repo.GetMany(ctx, resourceType, missing, userID)
	})
}
