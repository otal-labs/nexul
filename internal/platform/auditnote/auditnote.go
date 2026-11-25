// Package auditnote lets a use-case name the computer a request acted on, so the audit row the adapter writes after
// it carries the computer, its owner and any command it started (the personal runners spec, Access and privacy rule 5).
package auditnote

import (
	"context"
	"sync"
)

// Note is what one request did to a computer; empty when it touched none.
type Note struct {
	ComputerID string
	OwnerID    string
	JobID      string
}

type holder struct {
	mu   sync.Mutex
	note Note
}

type key struct{}

// With gives ctx a note for the request it carries; Read reads it once the request is done.
func With(ctx context.Context) context.Context {
	return context.WithValue(ctx, key{}, &holder{})
}

// Computer records that the request acted on computerID, owned by ownerID; a later call wins.
func Computer(ctx context.Context, computerID, ownerID string) {
	h, ok := ctx.Value(key{}).(*holder)
	if !ok {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	h.note.ComputerID, h.note.OwnerID = computerID, ownerID
}

// Job records the command the request started on the computer it names.
func Job(ctx context.Context, computerID, ownerID, jobID string) {
	h, ok := ctx.Value(key{}).(*holder)
	if !ok {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	h.note = Note{ComputerID: computerID, OwnerID: ownerID, JobID: jobID}
}

// Read is the note the request left; empty without one.
func Read(ctx context.Context) Note {
	h, ok := ctx.Value(key{}).(*holder)
	if !ok {
		return Note{}
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.note
}
