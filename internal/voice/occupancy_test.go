package voice

import (
	"reflect"
	"testing"
)

func TestOccupancyStore_JoinAndLeave(t *testing.T) {
	s := newOccupancyStore()

	occupants, changed := s.join("room-1", Occupant{Identity: "u1", Name: "Ada"})
	if !changed {
		t.Fatal("first join should report changed")
	}
	if want := []Occupant{{Identity: "u1", Name: "Ada"}}; !reflect.DeepEqual(occupants, want) {
		t.Errorf("occupants after join = %+v, want %+v", occupants, want)
	}
	if s.isEmpty() {
		t.Fatal("store should not be empty after a join")
	}

	// Re-joining the same identity+name is a no-op (e.g. a redelivered webhook).
	if _, changed := s.join("room-1", Occupant{Identity: "u1", Name: "Ada"}); changed {
		t.Error("re-joining an identical occupant should not report changed")
	}

	occupants, changed = s.join("room-1", Occupant{Identity: "u2", Name: "Grace"})
	if !changed {
		t.Fatal("second distinct join should report changed")
	}
	want := []Occupant{{Identity: "u1", Name: "Ada"}, {Identity: "u2", Name: "Grace"}}
	if !reflect.DeepEqual(occupants, want) {
		t.Errorf("occupants after second join = %+v, want %+v (sorted by identity)", occupants, want)
	}

	occupants, changed = s.leave("room-1", "u1")
	if !changed {
		t.Fatal("leave of a present occupant should report changed")
	}
	if want := []Occupant{{Identity: "u2", Name: "Grace"}}; !reflect.DeepEqual(occupants, want) {
		t.Errorf("occupants after leave = %+v, want %+v", occupants, want)
	}

	// Leaving an identity that was never there is a no-op.
	if _, changed := s.leave("room-1", "ghost"); changed {
		t.Error("leaving an absent identity should not report changed")
	}

	occupants, changed = s.leave("room-1", "u2")
	if !changed {
		t.Fatal("leaving the last occupant should report changed")
	}
	if len(occupants) != 0 {
		t.Errorf("occupants after emptying the room = %+v, want empty", occupants)
	}
	if !s.isEmpty() {
		t.Error("an emptied room should not remain in the store (ephemeral, absent when empty)")
	}
}

func TestOccupancyStore_Clear(t *testing.T) {
	s := newOccupancyStore()
	s.join("room-1", Occupant{Identity: "u1", Name: "Ada"})

	if !s.clear("room-1") {
		t.Error("clearing a room with occupants should report it had something to clear")
	}
	if !s.isEmpty() {
		t.Error("store should be empty after clear")
	}
	if s.clear("room-1") {
		t.Error("clearing an already-empty room should report nothing to clear")
	}
}

func TestOccupancyStore_Snapshot(t *testing.T) {
	s := newOccupancyStore()
	s.join("room-1", Occupant{Identity: "u1", Name: "Ada"})
	s.join("room-2", Occupant{Identity: "u2", Name: "Grace"})

	got := s.snapshot()
	want := map[string][]Occupant{
		"room-1": {{Identity: "u1", Name: "Ada"}},
		"room-2": {{Identity: "u2", Name: "Grace"}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("snapshot() = %+v, want %+v", got, want)
	}
}

func TestOccupancyStore_Reconcile(t *testing.T) {
	s := newOccupancyStore()
	s.join("room-1", Occupant{Identity: "u1", Name: "Ada"})
	s.join("room-2", Occupant{Identity: "u2", Name: "Grace"})

	// room-1 gains a second occupant (drift the poll should pick up), room-2
	// vanished entirely (its last participant left without a webhook
	// arriving), room-3 is new.
	changed := s.reconcile(map[string][]Occupant{
		"room-1": {{Identity: "u1", Name: "Ada"}, {Identity: "u3", Name: "Linus"}},
		"room-3": {{Identity: "u4", Name: "Barbara"}},
	})

	gotChanged := map[string]bool{}
	for _, room := range changed {
		gotChanged[room] = true
	}
	for _, room := range []string{"room-1", "room-2", "room-3"} {
		if !gotChanged[room] {
			t.Errorf("reconcile() changed set = %v, want it to include %q", changed, room)
		}
	}

	got := s.snapshot()
	want := map[string][]Occupant{
		"room-1": {{Identity: "u1", Name: "Ada"}, {Identity: "u3", Name: "Linus"}},
		"room-3": {{Identity: "u4", Name: "Barbara"}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("snapshot() after reconcile = %+v, want %+v", got, want)
	}
}

func TestOccupancyStore_Reconcile_NoChangeReportsNothing(t *testing.T) {
	s := newOccupancyStore()
	s.join("room-1", Occupant{Identity: "u1", Name: "Ada"})

	changed := s.reconcile(map[string][]Occupant{
		"room-1": {{Identity: "u1", Name: "Ada"}},
	})
	if len(changed) != 0 {
		t.Errorf("reconcile() with no drift changed = %v, want empty", changed)
	}
}
