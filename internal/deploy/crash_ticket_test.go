package deploy

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
)

type filedTicket struct{ project, title, body string }

type fakeDesk struct {
	filed    []filedTicket
	comments map[string][]string
	closed   map[string]bool
	fileErr  error
	openErr  error
	postErr  error
}

func newFakeDesk() *fakeDesk {
	return &fakeDesk{comments: map[string][]string{}, closed: map[string]bool{}}
}

func (f *fakeDesk) File(_ context.Context, projectID, title, body string) (string, error) {
	if f.fileErr != nil {
		return "", f.fileErr
	}
	f.filed = append(f.filed, filedTicket{projectID, title, body})
	return "t" + string(rune('0'+len(f.filed))), nil
}

func (f *fakeDesk) Open(_ context.Context, id string) (bool, error) {
	return !f.closed[id], f.openErr
}

func (f *fakeDesk) Comment(_ context.Context, id, body string) error {
	if f.postErr != nil {
		return f.postErr
	}
	f.comments[id] = append(f.comments[id], body)
	return nil
}

type fakeCrashLinks struct {
	links  map[string]string
	getErr error
	setErr error
}

func (f *fakeCrashLinks) Get(_ context.Context, stackID, subject string) (string, error) {
	if f.getErr != nil {
		return "", f.getErr
	}
	id, ok := f.links[stackID+"/"+subject]
	if !ok {
		return "", apperrs.ErrNotFound
	}
	return id, nil
}

func (f *fakeCrashLinks) Set(_ context.Context, stackID, subject, ticketID string) error {
	if f.setErr != nil {
		return f.setErr
	}
	f.links[stackID+"/"+subject] = ticketID
	return nil
}

type crashFixture struct {
	s      *Service
	desk   *fakeDesk
	links  *fakeCrashLinks
	stacks *fakeStackRepo
}

func newCrashFixture(t *testing.T, mutate ...func(*Stack)) crashFixture {
	t.Helper()
	stacks := newFakeStackRepo()
	stack := validStack()
	stack.CrashTickets = true
	for _, m := range mutate {
		m(&stack)
	}
	requireStack(t, stacks, stack)
	s := newTestServiceWith(newFakeRepo(), stacks, newFakeContainerRepo(), newFakeProjects(), newFakeBus())
	f := crashFixture{s: s, desk: newFakeDesk(), links: &fakeCrashLinks{links: map[string]string{}}, stacks: stacks}
	s.SetCrashTickets(f.desk, f.links)
	return f
}

func outageEvent(t *testing.T, topic string, mutate ...func(*OutageEvent)) eventbus.Event {
	t.Helper()
	back := time.Date(2026, 9, 1, 12, 5, 0, 0, time.UTC)
	p := OutageEvent{
		ProjectID: "proj-1", StackID: "svc-1", StackName: "api", Service: "web", Machine: "10.0.0.1:22", OutageID: "o-1",
		ExitCode: 137, OOM: true, StartedAt: time.Date(2026, 9, 1, 12, 2, 0, 0, time.UTC),
	}
	if topic == TopicContainerRecovered {
		p.EndedAt, p.RestartCount = &back, 2
	}
	for _, m := range mutate {
		m(&p)
	}
	raw, err := json.Marshal(p)
	require.NoError(t, err)
	return eventbus.Event{ID: p.OutageID + topic, Topic: topic, Payload: raw}
}

func TestHandleContainerStopped_FilesOneTicketPerCrashedService(t *testing.T) {
	f := newCrashFixture(t)
	require.NoError(t, f.s.HandleContainerStopped(t.Context(), outageEvent(t, TopicContainerStopped)))

	require.Len(t, f.desk.filed, 1)
	got := f.desk.filed[0]
	assert.Equal(t, "proj-1", got.project)
	assert.Equal(t, "web crashed: exit 137, out of memory", got.title)
	for _, want := range []string{"`web`", "`api`", "exit 137, out of memory", "2026-09-01T12:02:00Z", "o-1", "`stack_get`", "`logs`"} {
		assert.Contains(t, got.body, want)
	}
	assert.Equal(t, "t1", f.links.links["svc-1/web"])
}

func TestHandleContainerStopped_Reasons(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*OutageEvent)
		title  string
	}{
		{"a plain exit", func(p *OutageEvent) { p.OOM = false }, "web crashed: exit 137"},
		{"an out-of-memory kill seen before its exit", func(p *OutageEvent) { p.ExitCode = 0 }, "web crashed: out of memory"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newCrashFixture(t)
			require.NoError(t, f.s.HandleContainerStopped(t.Context(), outageEvent(t, TopicContainerStopped, tt.mutate)))
			require.Len(t, f.desk.filed, 1)
			assert.Equal(t, tt.title, f.desk.filed[0].title)
		})
	}
}

func TestHandleContainerStopped_ARepeatCommentsWhileTheTicketIsOpen(t *testing.T) {
	f := newCrashFixture(t)
	require.NoError(t, f.s.HandleContainerStopped(t.Context(), outageEvent(t, TopicContainerStopped)))
	require.NoError(t, f.s.HandleContainerStopped(t.Context(), outageEvent(t, TopicContainerStopped, func(p *OutageEvent) { p.OutageID = "o-2" })))

	assert.Len(t, f.desk.filed, 1)
	require.Len(t, f.desk.comments["t1"], 1)
	assert.Contains(t, f.desk.comments["t1"][0], "crashed again")
	assert.Contains(t, f.desk.comments["t1"][0], "o-2")

	require.NoError(t, f.s.HandleContainerStopped(t.Context(), outageEvent(t, TopicContainerStopped, func(p *OutageEvent) { p.Service = "worker"; p.OutageID = "o-3" })))
	assert.Len(t, f.desk.filed, 2, "another service has its own ticket")
}

func TestHandleContainerStopped_ADoneTicketIsReplaced(t *testing.T) {
	f := newCrashFixture(t)
	require.NoError(t, f.s.HandleContainerStopped(t.Context(), outageEvent(t, TopicContainerStopped)))
	f.desk.closed["t1"] = true
	require.NoError(t, f.s.HandleContainerStopped(t.Context(), outageEvent(t, TopicContainerStopped, func(p *OutageEvent) { p.OutageID = "o-2" })))

	assert.Len(t, f.desk.filed, 2)
	assert.Empty(t, f.desk.comments["t1"])
	assert.Equal(t, "t2", f.links.links["svc-1/web"])
}

func TestHandleContainerRecovered_CommentsAndLeavesTheTicketOpen(t *testing.T) {
	f := newCrashFixture(t)
	require.NoError(t, f.s.HandleContainerStopped(t.Context(), outageEvent(t, TopicContainerStopped)))
	require.NoError(t, f.s.HandleContainerRecovered(t.Context(), outageEvent(t, TopicContainerRecovered)))

	require.Len(t, f.desk.comments["t1"], 1)
	for _, want := range []string{"`web` came back at 2026-09-01T12:05:00Z", "down since 2026-09-01T12:02:00Z", "2 more times", "stays open"} {
		assert.Contains(t, f.desk.comments["t1"][0], want)
	}
	assert.False(t, f.desk.closed["t1"])
}

func TestCrashTickets_DoNothingWhere(t *testing.T) {
	tests := []struct {
		name   string
		stack  func(*Stack)
		detach bool
		event  func(*OutageEvent)
	}{
		{"the switch is off", func(s *Stack) { s.CrashTickets = false }, false, nil},
		{"the stack has no project", func(s *Stack) { s.ProjectID = "" }, false, nil},
		{"the event names a stack that is gone", nil, false, func(p *OutageEvent) { p.StackID = "gone" }},
		{"no ticket desk is wired", nil, true, nil},
	}
	for _, tt := range tests {
		for _, topic := range []string{TopicContainerStopped, TopicContainerRecovered} {
			t.Run(tt.name+" "+topic, func(t *testing.T) {
				var mutate []func(*Stack)
				if tt.stack != nil {
					mutate = append(mutate, tt.stack)
				}
				f := newCrashFixture(t, mutate...)
				f.links.links["svc-1/web"] = "t9"
				if tt.detach {
					f.s.SetCrashTickets(nil, nil)
				}
				ev := outageEvent(t, topic)
				if tt.event != nil {
					ev = outageEvent(t, topic, tt.event)
				}
				handle := f.s.HandleContainerStopped
				if topic == TopicContainerRecovered {
					handle = f.s.HandleContainerRecovered
				}
				require.NoError(t, handle(t.Context(), ev))
				assert.Empty(t, f.desk.filed)
				assert.Empty(t, f.desk.comments)
			})
		}
	}
}

func TestCrashTickets_ErrorPaths(t *testing.T) {
	boom := errors.New("boom")
	tests := []struct {
		name  string
		setup func(*crashFixture)
		topic string
	}{
		{"a ticket that cannot be filed", func(f *crashFixture) { f.desk.fileErr = boom }, TopicContainerStopped},
		{"a link that cannot be stored", func(f *crashFixture) { f.links.setErr = boom }, TopicContainerStopped},
		{"a link that cannot be read", func(f *crashFixture) { f.links.getErr = boom }, TopicContainerStopped},
		{"a ticket whose state cannot be read", func(f *crashFixture) { f.links.links["svc-1/web"] = "t9"; f.desk.openErr = boom }, TopicContainerStopped},
		{"a repeat that cannot be posted", func(f *crashFixture) { f.links.links["svc-1/web"] = "t9"; f.desk.postErr = boom }, TopicContainerStopped},
		{"a recovery that cannot be posted", func(f *crashFixture) { f.links.links["svc-1/web"] = "t9"; f.desk.postErr = boom }, TopicContainerRecovered},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newCrashFixture(t)
			tt.setup(&f)
			handle := f.s.HandleContainerStopped
			if tt.topic == TopicContainerRecovered {
				handle = f.s.HandleContainerRecovered
			}
			err := handle(t.Context(), outageEvent(t, tt.topic))
			require.ErrorIs(t, err, boom)
			assert.NotErrorIs(t, err, apperrs.ErrFatal, "worth retrying")
		})
	}

	t.Run("a payload that does not parse is fatal", func(t *testing.T) {
		f := newCrashFixture(t)
		err := f.s.HandleContainerStopped(t.Context(), eventbus.Event{Topic: TopicContainerStopped, Payload: []byte("{")})
		require.ErrorIs(t, err, apperrs.ErrFatal)
	})
	t.Run("a stack that cannot be read is retried", func(t *testing.T) {
		f := newCrashFixture(t)
		f.stacks.getErr = boom
		require.ErrorIs(t, f.s.HandleContainerStopped(t.Context(), outageEvent(t, TopicContainerStopped)), boom)
	})
}

func TestStack_CrashTicketsNeedAProject(t *testing.T) {
	s := validStack()
	s.CrashTickets, s.ProjectID = true, ""
	require.ErrorIs(t, s.Validate(), apperrs.ErrInvalid)
	s.CrashTickets = false
	require.NoError(t, s.Validate())
}
