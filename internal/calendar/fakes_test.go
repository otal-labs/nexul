package calendar

import (
	"context"
	"fmt"
	"slices"
	"sync"
	"testing"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/paging"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

var fixedNow = at("2026-11-01T08:00:00Z")

// fakeRepo is an in-memory Repo that keeps every event it was handed.
type fakeRepo struct {
	mu        sync.Mutex
	meetings  map[string]*Meeting
	events    []eventbus.OutboxEvent
	due       []DueTicket
	reminded  map[string]string
	saveErr   error
	saveCalls int
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{meetings: map[string]*Meeting{}, reminded: map[string]string{}}
}

func (f *fakeRepo) Get(_ context.Context, id string) (*Meeting, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	m, ok := f.meetings[id]
	if !ok {
		return nil, apperrs.ErrNotFound
	}
	return m.clone(), nil
}

func (f *fakeRepo) sorted(keep func(*Meeting) bool) []*Meeting {
	var out []*Meeting
	for _, m := range f.meetings {
		if keep(m) {
			out = append(out, m.clone())
		}
	}
	slices.SortFunc(out, func(a, b *Meeting) int {
		if c := a.StartsAt.Compare(b.StartsAt); c != 0 {
			return c
		}
		return compareStrings(a.ID, b.ID)
	})
	return out
}

func compareStrings(a, b string) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}

func (f *fakeRepo) ListVisible(_ context.Context, workspaceID, userID string, r RangeFilter) ([]*Meeting, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.sorted(func(m *Meeting) bool {
		return m.WorkspaceID == workspaceID && m.Sees(userID) && m.StartsAt.Before(r.To) && (m.EndsAt.After(r.From) || m.Repeating())
	}), nil
}

func (f *fakeRepo) PageVisible(_ context.Context, workspaceID, userID string, _ RangeFilter, w paging.Window) ([]*Meeting, int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	all := f.sorted(func(m *Meeting) bool { return m.WorkspaceID == workspaceID && m.Sees(userID) })
	end := min(w.Offset+w.Limit, len(all))
	if w.Offset >= len(all) {
		return []*Meeting{}, len(all), nil
	}
	return all[w.Offset:end], len(all), nil
}

func (f *fakeRepo) Replacements(_ context.Context, seriesIDs []string) ([]*Meeting, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.sorted(func(m *Meeting) bool { return m.SeriesID != "" && slices.Contains(seriesIDs, m.SeriesID) }), nil
}

func (f *fakeRepo) Save(_ context.Context, ms []*Meeting, evts ...eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.saveCalls++
	if f.saveErr != nil {
		return f.saveErr
	}
	for _, m := range ms {
		f.meetings[m.ID] = m.clone()
	}
	f.events = append(f.events, evts...)
	return nil
}

func (f *fakeRepo) SaveReply(_ context.Context, meetingID, userID string, reply Reply, at time.Time, evts ...eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	m := f.meetings[meetingID]
	for i := range m.Invitees {
		if m.Invitees[i].UserID == userID {
			m.Invitees[i].Reply, m.Invitees[i].RepliedAt = reply, &at
			f.events = append(f.events, evts...)
			return nil
		}
	}
	return apperrs.ErrNotFound
}

func (f *fakeRepo) NextReminder(context.Context) (time.Time, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var next *time.Time
	for _, m := range f.meetings {
		if m.NextRemindAt != nil && (next == nil || m.NextRemindAt.Before(*next)) {
			next = m.NextRemindAt
		}
	}
	if next == nil {
		return time.Time{}, false, nil
	}
	return *next, true, nil
}

func (f *fakeRepo) RemindersDue(_ context.Context, now time.Time) ([]*Meeting, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.sorted(func(m *Meeting) bool { return m.NextRemindAt != nil && !m.NextRemindAt.After(now) }), nil
}

func (f *fakeRepo) MarkReminded(_ context.Context, m *Meeting, sent time.Time, evts ...eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	stored := f.meetings[m.ID]
	if stored.NextRemindAt == nil || !stored.NextRemindAt.Equal(sent) {
		return nil
	}
	stored.NextRemindAt, stored.NextOccurrence = m.NextRemindAt, m.NextOccurrence
	f.events = append(f.events, evts...)
	return nil
}

func (f *fakeRepo) DueTickets(_ context.Context, fromDate, toDate string) ([]DueTicket, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []DueTicket
	for _, d := range f.due {
		if d.DueOn >= fromDate && d.DueOn <= toDate && f.reminded[d.TicketID] != d.DueOn {
			out = append(out, d)
		}
	}
	return out, nil
}

func (f *fakeRepo) MarkTicketReminded(_ context.Context, ticketID, dueOn string, _ time.Time, evts ...eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.reminded[ticketID] = dueOn
	f.events = append(f.events, evts...)
	return nil
}

func (f *fakeRepo) topics() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]string, len(f.events))
	for i, e := range f.events {
		out[i] = e.Topic
	}
	return out
}

func (f *fakeRepo) lastEvent(t *testing.T) eventbus.OutboxEvent {
	t.Helper()
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.events) == 0 {
		t.Fatal("no event was written")
	}
	return f.events[len(f.events)-1]
}

// fakeGate lets every member through a membership check and each person the actions listed for them.
type fakeGate struct {
	members []string
	held    map[string][]permissions.Action
}

func (g fakeGate) Require(ctx context.Context, workspaceID string, action permissions.Action) error {
	actor, _ := identity.ActorFromCtx(ctx)
	if !slices.Contains(g.members, actor.ID) {
		return fmt.Errorf("%w: workspace %s", apperrs.ErrNotFound, workspaceID)
	}
	if action == permissions.Member || slices.Contains(g.held[actor.ID], action) {
		return nil
	}
	return fmt.Errorf("%w: %s", apperrs.ErrForbidden, action)
}

type fakeChannels map[string]*Channel

func (f fakeChannels) Channel(_ context.Context, id string) (*Channel, error) {
	c, ok := f[id]
	if !ok {
		return nil, apperrs.ErrNotFound
	}
	return c, nil
}

type fakeMembers []string

func (f fakeMembers) ListMemberUserIDs(context.Context, string) ([]string, error) { return f, nil }

type harness struct {
	svc  *Service
	repo *fakeRepo
	now  time.Time
}

// newHarness wires a Service where alice and dave hold calendar:write, carol calendar:delete, and bob, erin nothing.
func newHarness() *harness {
	h := &harness{repo: newFakeRepo(), now: fixedNow}
	people := []string{"alice", "bob", "carol", "dave", "erin"}
	h.svc = NewService(Config{
		Repo: h.repo,
		Gate: fakeGate{members: people, held: map[string][]permissions.Action{
			"alice": {permissions.CalendarWrite}, "dave": {permissions.CalendarWrite}, "carol": {permissions.CalendarDelete},
		}},
		Channels: fakeChannels{
			"vc-open":    {ID: "vc-open", WorkspaceID: "ws-1", Name: "standup", Voice: true},
			"vc-private": {ID: "vc-private", WorkspaceID: "ws-1", Name: "leads", Voice: true, Private: true, MemberIDs: []string{"alice", "bob"}},
			"ch-text":    {ID: "ch-text", WorkspaceID: "ws-1", Name: "general"},
			"vc-other":   {ID: "vc-other", WorkspaceID: "ws-2", Name: "elsewhere", Voice: true},
		},
		Members: fakeMembers(people),
	})
	h.svc.now = func() time.Time { return h.now }
	return h
}

func as(t *testing.T, userID string) context.Context {
	return identity.WithActor(t.Context(), identity.Actor{ID: userID})
}

func standup() Details {
	return Details{
		Title: ptr("Standup"), StartsAt: ptr(at("2026-11-02T10:00:00Z")), EndsAt: ptr(at("2026-11-02T10:30:00Z")),
		InviteeIDs: ptr([]string{"bob"}),
	}
}
