package collab

import (
	"context"
	"slices"
	"sync"
	"time"
)

var _ Store = (*MemoryStore)(nil)

// MemoryStore keeps each room's newest snapshot and later increments in memory, for a file-canonical body (ADR 0110).
// ponytail: a room stays until it is reset or the process restarts; evict on the last leave if memory ever shows.
type MemoryStore struct {
	mu    sync.Mutex
	rooms map[string]*memoryRoom
}

type memoryRoom struct {
	seq        int64
	snapshot   *StoredUpdate
	increments []StoredUpdate
}

// NewMemoryStore returns an empty in-memory store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{rooms: make(map[string]*memoryRoom)}
}

// room returns the room's state, creating it; callers hold m.mu.
func (m *MemoryStore) room(id string) *memoryRoom {
	r, ok := m.rooms[id]
	if !ok {
		r = &memoryRoom{}
		m.rooms[id] = r
	}
	return r
}

// AppendUpdate stores one payload; a snapshot replaces the older state, since a replay starts from the newest one.
func (m *MemoryStore) AppendUpdate(_ context.Context, roomID, actorID string, kind UpdateKind, payload string) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	r := m.room(roomID)
	r.seq++
	u := StoredUpdate{Seq: r.seq, Kind: kind, ActorID: actorID, Payload: payload, CreatedAt: time.Now().UTC()}
	if kind == KindSnapshot {
		r.snapshot, r.increments = &u, nil
		return u.Seq, nil
	}
	r.increments = append(r.increments, u)
	return u.Seq, nil
}

// LoadReplay returns the room's newest snapshot, the increments after it, and its seq.
func (m *MemoryStore) LoadReplay(_ context.Context, roomID string) (Replay, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	r, ok := m.rooms[roomID]
	if !ok {
		return Replay{}, nil
	}
	return Replay{Snapshot: r.snapshot, Increments: slices.Clone(r.increments), Seq: r.seq}, nil
}

// TrimUpdates drops increments at or below baseSeq.
func (m *MemoryStore) TrimUpdates(_ context.Context, roomID string, baseSeq int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if r, ok := m.rooms[roomID]; ok {
		r.increments = slices.DeleteFunc(r.increments, func(u StoredUpdate) bool { return u.Seq <= baseSeq })
	}
	return nil
}

// Reset drops the room's state and returns a seq above every seq it held.
func (m *MemoryStore) Reset(_ context.Context, roomID string) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	r := m.room(roomID)
	r.seq++
	r.snapshot, r.increments = nil, nil
	return r.seq, nil
}
