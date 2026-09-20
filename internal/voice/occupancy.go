package voice

import (
	"sort"
	"sync"
)

// occupancyStore is voice's in-memory conversationID -> occupant-set map (ephemeral, no DB writes).
type occupancyStore struct {
	mu    sync.Mutex
	rooms map[string]map[string]Occupant // conversationID -> identity -> Occupant
}

func newOccupancyStore() *occupancyStore {
	return &occupancyStore{rooms: map[string]map[string]Occupant{}}
}

// join adds/updates one occupant, returning the room's new full list and whether membership actually changed.
func (s *occupancyStore) join(room string, o Occupant) ([]Occupant, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	occupants, ok := s.rooms[room]
	if !ok {
		occupants = map[string]Occupant{}
		s.rooms[room] = occupants
	}
	if existing, ok := occupants[o.Identity]; ok && existing == o {
		return snapshotRoom(occupants), false
	}
	occupants[o.Identity] = o
	return snapshotRoom(occupants), true
}

// leave removes one occupant, dropping the room entirely once it's empty.
func (s *occupancyStore) leave(room, identity string) ([]Occupant, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	occupants, ok := s.rooms[room]
	if !ok {
		return nil, false
	}
	if _, ok := occupants[identity]; !ok {
		return snapshotRoom(occupants), false
	}
	delete(occupants, identity)
	remaining := snapshotRoom(occupants)
	if len(occupants) == 0 {
		delete(s.rooms, room)
	}
	return remaining, true
}

// clear empties one room outright (room_finished), reporting whether it had any tracked occupants to clear.
func (s *occupancyStore) clear(room string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.rooms[room]; !ok {
		return false
	}
	delete(s.rooms, room)
	return true
}

// room returns one room's current occupant list.
func (s *occupancyStore) room(conversationID string) []Occupant {
	s.mu.Lock()
	defer s.mu.Unlock()
	return snapshotRoom(s.rooms[conversationID])
}

// snapshot feeds the initial-render endpoint (GET /api/voice/occupancy).
func (s *occupancyStore) snapshot() map[string][]Occupant {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make(map[string][]Occupant, len(s.rooms))
	for room, occupants := range s.rooms {
		out[room] = snapshotRoom(occupants)
	}
	return out
}

// isEmpty reports whether any room currently has tracked occupants.
func (s *occupancyStore) isEmpty() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.rooms) == 0
}

// reconcile replaces state from a fresh LiveKit snapshot, returning the conversation IDs that actually changed.
func (s *occupancyStore) reconcile(rooms map[string][]Occupant) []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	var changed []string
	seen := make(map[string]bool, len(rooms))
	for room, occupants := range rooms {
		seen[room] = true
		next := make(map[string]Occupant, len(occupants))
		for _, o := range occupants {
			next[o.Identity] = o
		}
		if !equalRooms(s.rooms[room], next) {
			changed = append(changed, room)
		}
		if len(next) == 0 {
			delete(s.rooms, room)
			continue
		}
		s.rooms[room] = next
	}
	for room := range s.rooms {
		if !seen[room] {
			delete(s.rooms, room)
			changed = append(changed, room)
		}
	}
	return changed
}

func snapshotRoom(occupants map[string]Occupant) []Occupant {
	out := make([]Occupant, 0, len(occupants))
	for _, o := range occupants {
		out = append(out, o)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Identity < out[j].Identity })
	return out
}

func equalRooms(a, b map[string]Occupant) bool {
	if len(a) != len(b) {
		return false
	}
	for id, o := range a {
		if b[id] != o {
			return false
		}
	}
	return true
}
