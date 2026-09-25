package tickets

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
)

func (f *fakeRepo) linkedLocked(id string) *LinkedTicket {
	t, ok := f.tickets[id]
	if !ok {
		return nil
	}
	return &LinkedTicket{ID: t.ID, ProjectID: t.ProjectID, Number: t.Number, Title: t.Title, Status: t.Status, Done: f.doneStatuses[t.Status]}
}

func (f *fakeRepo) ListLinkEnds(_ context.Context, id string) ([]LinkEnd, []LinkEnd, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.ticketLinkErr != nil {
		return nil, nil, f.ticketLinkErr
	}
	var from, to []LinkEnd
	for _, l := range f.ticketLinks {
		if l.TicketID == id {
			from = append(from, LinkEnd{Kind: l.Kind, Ticket: f.linkedLocked(l.TargetID)})
		}
		if l.TargetID == id {
			to = append(to, LinkEnd{Kind: l.Kind, Ticket: f.linkedLocked(l.TicketID)})
		}
	}
	return from, to, nil
}

func (f *fakeRepo) BlockerIDs(_ context.Context, id string) ([]string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.ticketLinkErr != nil {
		return nil, f.ticketLinkErr
	}
	var out []string
	for _, l := range f.ticketLinks {
		if l.TicketID == id && l.Kind == LinkBlockedBy {
			out = append(out, l.TargetID)
		}
	}
	return out, nil
}

func (f *fakeRepo) CreateWithLink(ctx context.Context, t *Ticket, link TicketLink, evts ...eventbus.OutboxEvent) error {
	if f.ticketLinkErr != nil {
		return f.ticketLinkErr
	}
	if err := f.Create(ctx, t, evts...); err != nil {
		return err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.ticketLinks = append(f.ticketLinks, link)
	return nil
}

func (f *fakeRepo) PutLink(_ context.Context, link TicketLink, evts ...eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.ticketLinkErr != nil {
		return f.ticketLinkErr
	}
	if link.Kind == LinkFoundIn {
		f.dropLocked(link)
	}
	f.ticketLinks = append(f.ticketLinks, link)
	f.events = append(f.events, evts...)
	return nil
}

func (f *fakeRepo) DeleteLink(_ context.Context, link TicketLink, evts ...eventbus.OutboxEvent) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.ticketLinkErr != nil {
		return false, f.ticketLinkErr
	}
	if !f.dropLocked(link) {
		return false, nil
	}
	f.events = append(f.events, evts...)
	return true, nil
}

func (f *fakeRepo) dropLocked(link TicketLink) bool {
	kept := f.ticketLinks[:0]
	removed := false
	for _, l := range f.ticketLinks {
		match := l.TicketID == link.TicketID && l.Kind == link.Kind && (link.Kind == LinkFoundIn || l.TargetID == link.TargetID)
		if match {
			removed = true
			continue
		}
		kept = append(kept, l)
	}
	f.ticketLinks = kept
	return removed
}

func (f *fakeRepo) UnclearedBlockers(_ context.Context) (map[string][]LinkedTicket, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.ticketLinkErr != nil {
		return nil, f.ticketLinkErr
	}
	out := map[string][]LinkedTicket{}
	for _, l := range f.ticketLinks {
		blocker := f.linkedLocked(l.TargetID)
		if l.Kind != LinkBlockedBy || blocker == nil || blocker.Done {
			continue
		}
		out[l.TicketID] = append(out[l.TicketID], *blocker)
	}
	return out, nil
}

func seedTickets(t *testing.T, s *Service, titles ...string) []*Ticket {
	t.Helper()
	out := make([]*Ticket, 0, len(titles))
	for _, title := range titles {
		created, err := s.Create(t.Context(), "p-1", title, "", "", "")
		require.NoError(t, err)
		out = append(out, created)
	}
	return out
}

func linkTopics(evts []eventbus.OutboxEvent) []string {
	var out []string
	for _, e := range evts {
		if e.Topic == TopicLinkCreated || e.Topic == TopicLinkDeleted {
			out = append(out, e.Topic)
		}
	}
	return out
}

func TestAddBlocker_InvalidInput_ReturnsError(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo)
	ts := seedTickets(t, s, "frontend")
	tests := []struct {
		name      string
		id        string
		blockerID string
		want      error
	}{
		{"empty id", "", ts[0].ID, apperrs.ErrInvalid},
		{"empty blocker", ts[0].ID, " ", apperrs.ErrInvalid},
		{"self", ts[0].ID, ts[0].ID, apperrs.ErrInvalid},
		{"missing ticket", "nope", ts[0].ID, apperrs.ErrNotFound},
		{"missing blocker", ts[0].ID, "nope", apperrs.ErrNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := s.AddBlocker(t.Context(), tt.id, tt.blockerID)
			require.ErrorIs(t, err, tt.want)
		})
	}
	assert.Empty(t, repo.ticketLinks)
}

func TestAddBlocker_WouldFormCycle_ReturnsConflict(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo)
	ts := seedTickets(t, s, "a", "b", "c")
	_, err := s.AddBlocker(t.Context(), ts[0].ID, ts[1].ID)
	require.NoError(t, err)
	_, err = s.AddBlocker(t.Context(), ts[1].ID, ts[2].ID)
	require.NoError(t, err)

	_, err = s.AddBlocker(t.Context(), ts[2].ID, ts[0].ID)
	require.ErrorIs(t, err, apperrs.ErrConflict)
	assert.Contains(t, err.Error(), "cycle")

	_, err = s.AddBlocker(t.Context(), ts[1].ID, ts[0].ID)
	require.ErrorIs(t, err, apperrs.ErrConflict)
	assert.Len(t, repo.ticketLinks, 2)
}

func TestAddBlocker_ShowsBothDirectionsAndClearsWhenBlockerIsDone(t *testing.T) {
	repo := newFakeRepo()
	repo.doneStatuses = map[Status]bool{StatusDone: true}
	s := newTestService(repo)
	ts := seedTickets(t, s, "frontend /books", "backend /books")

	set, err := s.AddBlocker(t.Context(), ts[0].ID, ts[1].ID)
	require.NoError(t, err)
	require.Len(t, set.BlockedBy, 1)
	assert.Equal(t, ts[1].ID, set.BlockedBy[0].ID)
	assert.True(t, set.Blocked)
	assert.Equal(t, []string{TopicLinkCreated}, linkTopics(repo.events))

	again, err := s.AddBlocker(t.Context(), ts[0].ID, ts[1].ID)
	require.NoError(t, err)
	assert.Len(t, again.BlockedBy, 1)
	assert.Len(t, linkTopics(repo.events), 1, "a duplicate is a no-op")

	reverse, err := s.Links(t.Context(), ts[1].ID)
	require.NoError(t, err)
	require.Len(t, reverse.Blocks, 1)
	assert.Equal(t, ts[0].ID, reverse.Blocks[0].ID)

	uncleared, err := s.UnclearedBlockers(t.Context())
	require.NoError(t, err)
	assert.Len(t, uncleared[ts[0].ID], 1)

	_, err = s.UpdateStatus(t.Context(), ts[1].ID, StatusDone)
	require.NoError(t, err)
	cleared, err := s.Links(t.Context(), ts[0].ID)
	require.NoError(t, err)
	assert.False(t, cleared.Blocked)
	uncleared, err = s.UnclearedBlockers(t.Context())
	require.NoError(t, err)
	assert.Empty(t, uncleared)
}

func TestUpdateStatus_BlockedTicket_StillMoves(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo)
	ts := seedTickets(t, s, "blocked", "blocker")
	_, err := s.AddBlocker(t.Context(), ts[0].ID, ts[1].ID)
	require.NoError(t, err)

	moved, err := s.UpdateStatus(t.Context(), ts[0].ID, StatusInProgress)
	require.NoError(t, err)
	assert.Equal(t, StatusInProgress, moved.Status)
}

func TestRemoveBlocker(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo)
	ts := seedTickets(t, s, "a", "b")

	_, err := s.RemoveBlocker(t.Context(), ts[0].ID, "")
	require.ErrorIs(t, err, apperrs.ErrInvalid)
	_, err = s.RemoveBlocker(t.Context(), "nope", ts[1].ID)
	require.ErrorIs(t, err, apperrs.ErrNotFound)

	_, err = s.AddBlocker(t.Context(), ts[0].ID, ts[1].ID)
	require.NoError(t, err)
	set, err := s.RemoveBlocker(t.Context(), ts[0].ID, ts[1].ID)
	require.NoError(t, err)
	assert.Empty(t, set.BlockedBy)
	assert.Equal(t, []string{TopicLinkCreated, TopicLinkDeleted}, linkTopics(repo.events))

	_, err = s.RemoveBlocker(t.Context(), ts[0].ID, ts[1].ID)
	require.NoError(t, err)
	assert.Len(t, linkTopics(repo.events), 2, "removing a missing link publishes nothing")
}

func TestSetFoundIn_InvalidInput_ReturnsError(t *testing.T) {
	s := newTestService(newFakeRepo())
	ts := seedTickets(t, s, "bug")
	tests := []struct {
		name     string
		id       string
		originID string
		unknown  bool
		want     error
	}{
		{"empty id", "", "x", false, apperrs.ErrInvalid},
		{"no origin and not unknown", ts[0].ID, "", false, apperrs.ErrInvalid},
		{"origin and unknown", ts[0].ID, "x", true, apperrs.ErrInvalid},
		{"self", ts[0].ID, ts[0].ID, false, apperrs.ErrInvalid},
		{"missing origin", ts[0].ID, "nope", false, apperrs.ErrNotFound},
		{"missing ticket", "nope", "", true, apperrs.ErrNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := s.SetFoundIn(t.Context(), tt.id, tt.originID, tt.unknown)
			require.ErrorIs(t, err, tt.want)
		})
	}
}

func TestSetFoundIn_ReplacesOriginAndShowsBugsFound(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo)
	ts := seedTickets(t, s, "bug", "origin", "other origin")

	set, err := s.SetFoundIn(t.Context(), ts[0].ID, "", true)
	require.NoError(t, err)
	assert.True(t, set.OriginUnknown)
	assert.Nil(t, set.FoundIn)

	set, err = s.SetFoundIn(t.Context(), ts[0].ID, ts[1].ID, false)
	require.NoError(t, err)
	assert.False(t, set.OriginUnknown)
	require.NotNil(t, set.FoundIn)
	assert.Equal(t, ts[1].ID, set.FoundIn.ID)
	assert.Equal(t, []string{TopicLinkCreated, TopicLinkDeleted, TopicLinkCreated}, linkTopics(repo.events))

	_, err = s.SetFoundIn(t.Context(), ts[0].ID, ts[1].ID, false)
	require.NoError(t, err)
	assert.Len(t, linkTopics(repo.events), 3, "setting the same origin is a no-op")

	origin, err := s.Links(t.Context(), ts[1].ID)
	require.NoError(t, err)
	require.Len(t, origin.BugsFound, 1)
	assert.Equal(t, ts[0].ID, origin.BugsFound[0].ID)

	_, err = s.SetFoundIn(t.Context(), ts[0].ID, ts[2].ID, false)
	require.NoError(t, err)
	origin, err = s.Links(t.Context(), ts[1].ID)
	require.NoError(t, err)
	assert.Empty(t, origin.BugsFound)
}

func TestRemoveFoundIn(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo)
	ts := seedTickets(t, s, "bug", "origin")

	_, err := s.RemoveFoundIn(t.Context(), " ")
	require.ErrorIs(t, err, apperrs.ErrInvalid)
	_, err = s.RemoveFoundIn(t.Context(), "nope")
	require.ErrorIs(t, err, apperrs.ErrNotFound)

	set, err := s.RemoveFoundIn(t.Context(), ts[0].ID)
	require.NoError(t, err)
	assert.Nil(t, set.FoundIn)
	assert.Empty(t, linkTopics(repo.events))

	_, err = s.SetFoundIn(t.Context(), ts[0].ID, ts[1].ID, false)
	require.NoError(t, err)
	set, err = s.RemoveFoundIn(t.Context(), ts[0].ID)
	require.NoError(t, err)
	assert.Nil(t, set.FoundIn)
	assert.False(t, set.OriginUnknown)
	assert.Equal(t, []string{TopicLinkCreated, TopicLinkDeleted}, linkTopics(repo.events))
}

func TestTicketLinks_RepoFailure_IsWrapped(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo)
	ts := seedTickets(t, s, "a", "b")
	boom := errors.New("boom")
	repo.ticketLinkErr = boom

	calls := map[string]func() error{
		"links":          func() error { _, err := s.Links(t.Context(), ts[0].ID); return err },
		"add blocker":    func() error { _, err := s.AddBlocker(t.Context(), ts[0].ID, ts[1].ID); return err },
		"remove blocker": func() error { _, err := s.RemoveBlocker(t.Context(), ts[0].ID, ts[1].ID); return err },
		"set found in":   func() error { _, err := s.SetFoundIn(t.Context(), ts[0].ID, ts[1].ID, false); return err },
		"remove found":   func() error { _, err := s.RemoveFoundIn(t.Context(), ts[0].ID); return err },
		"uncleared":      func() error { _, err := s.UnclearedBlockers(t.Context()); return err },
		"links missing":  func() error { _, err := s.Links(t.Context(), ""); return err },
	}
	for name, call := range calls {
		t.Run(name, func(t *testing.T) {
			require.Error(t, call())
		})
	}
	_, err := s.Links(t.Context(), ts[0].ID)
	require.ErrorIs(t, err, boom)
}
