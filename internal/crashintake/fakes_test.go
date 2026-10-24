package crashintake

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
)

// fakeDesk holds tickets by body, so Find matches the fingerprint line the way the real search-then-filter does.
type fakeDesk struct {
	mu      sync.Mutex
	tickets []fakeTicket
	filed   []Bug
	findErr error
	fileErr error
}

type fakeTicket struct {
	Match
	body string
}

func (d *fakeDesk) Find(_ context.Context, _ string, fingerprint string) ([]Match, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.findErr != nil {
		return nil, d.findErr
	}
	var out []Match
	for _, t := range d.tickets {
		if strings.Contains(t.body, FingerprintLine+fingerprint) {
			out = append(out, t.Match)
		}
	}
	return out, nil
}

func (d *fakeDesk) File(_ context.Context, bug Bug) (string, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.fileErr != nil {
		return "", d.fileErr
	}
	id := fmt.Sprintf("t%d", len(d.tickets)+1)
	d.tickets = append(d.tickets, fakeTicket{Match: Match{ID: id, CreatedAt: time.Unix(int64(len(d.tickets)+1000), 0)}, body: bug.Body})
	d.filed = append(d.filed, bug)
	return id, nil
}

func (d *fakeDesk) add(m Match, fingerprint string) {
	d.tickets = append(d.tickets, fakeTicket{Match: m, body: FingerprintLine + fingerprint})
}

// fakeNotes keeps notes per ticket and file name, counting posts and replaces.
type fakeNotes struct {
	mu       sync.Mutex
	notes    map[string]Note
	posts    int
	replaces int
	err      error
}

func newFakeNotes() *fakeNotes { return &fakeNotes{notes: map[string]Note{}} }

func (n *fakeNotes) Find(_ context.Context, ticketID, name string) (Note, bool, error) {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.err != nil {
		return Note{}, false, n.err
	}
	note, ok := n.notes[ticketID+"/"+name]
	return note, ok, nil
}

func (n *fakeNotes) Post(_ context.Context, ticketID, name, markdown string) error {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.posts++
	n.notes[ticketID+"/"+name] = Note{ID: ticketID + "/" + name, Markdown: markdown}
	return nil
}

func (n *fakeNotes) Replace(_ context.Context, _, noteID, markdown string) error {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.replaces++
	n.notes[noteID] = Note{ID: noteID, Markdown: markdown}
	return nil
}

func (n *fakeNotes) markdown(ticketID, name string) string {
	return n.notes[ticketID+"/"+name].Markdown
}

// fakeFixes answers MergedInto per PR number.
type fakeFixes struct {
	in   map[int]bool
	errs map[int]error
}

func (f fakeFixes) MergedInto(_ context.Context, _, _ string, number int, _ string) (bool, error) {
	if err := f.errs[number]; err != nil {
		return false, err
	}
	return f.in[number], nil
}
