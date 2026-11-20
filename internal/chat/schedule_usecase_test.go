package chat

import (
	"context"
	"io"
	"log/slog"
	"slices"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/mcptool"
	"github.com/otal-labs/nexul/internal/platform/paging"
	"github.com/otal-labs/nexul/internal/platform/wake"
)

func (f *fakeRepo) CreateScheduledMessage(_ context.Context, m *ScheduledMessage, evts ...eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	cp := *m
	f.scheduled[m.ID] = &cp
	f.events = append(f.events, evts...)
	return nil
}

func (f *fakeRepo) GetScheduledMessage(_ context.Context, id string) (*ScheduledMessage, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	m, ok := f.scheduled[id]
	if !ok {
		return nil, apperrs.ErrNotFound
	}
	cp := *m
	return &cp, nil
}

func (f *fakeRepo) UpdateScheduledMessage(_ context.Context, m *ScheduledMessage, evts ...eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.scheduled[m.ID]; !ok {
		return apperrs.ErrNotFound
	}
	cp := *m
	f.scheduled[m.ID] = &cp
	f.events = append(f.events, evts...)
	return nil
}

func (f *fakeRepo) DeleteScheduledMessage(_ context.Context, id string, evts ...eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.scheduled[id]; !ok {
		return apperrs.ErrNotFound
	}
	delete(f.scheduled, id)
	f.events = append(f.events, evts...)
	return nil
}

func (f *fakeRepo) PageScheduledMessages(_ context.Context, conversationID, authorID string, w paging.Window) ([]*ScheduledMessage, int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []*ScheduledMessage
	for _, m := range f.scheduled {
		if m.ConversationID == conversationID && m.AuthorID == authorID {
			cp := *m
			out = append(out, &cp)
		}
	}
	slices.SortFunc(out, func(a, b *ScheduledMessage) int { return a.SendAt.Compare(b.SendAt) })
	w = w.Clamped()
	return out[min(w.Offset, len(out)):min(w.Offset+w.Limit, len(out))], len(out), nil
}

func (f *fakeRepo) ScheduledMessagesDue(_ context.Context, now time.Time, limit int) ([]*ScheduledMessage, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []*ScheduledMessage
	for _, m := range f.scheduled {
		if !m.SendAt.After(now) {
			cp := *m
			out = append(out, &cp)
		}
	}
	slices.SortFunc(out, func(a, b *ScheduledMessage) int { return a.SendAt.Compare(b.SendAt) })
	return out[:min(limit, len(out))], nil
}

func (f *fakeRepo) NextScheduledMessage(context.Context) (time.Time, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var next time.Time
	for _, m := range f.scheduled {
		if next.IsZero() || m.SendAt.Before(next) {
			next = m.SendAt
		}
	}
	return next, !next.IsZero(), nil
}

func (f *fakeRepo) SendScheduledMessage(_ context.Context, m *ScheduledMessage, msg *Message, next time.Time, evts ...eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	cur, ok := f.scheduled[m.ID]
	if !ok || !cur.SendAt.Equal(m.SendAt) || cur.Body != m.Body || cur.Repeats != m.Repeats {
		return apperrs.ErrConflict
	}
	delete(f.scheduled, m.ID)
	if !next.IsZero() {
		cur.SendAt = next
		f.scheduled[m.ID] = cur
	}
	cp := *msg
	f.messages[msg.ID] = &cp
	f.events = append(f.events, evts...)
	return nil
}

// sentAndWaiting counts the posted messages and the scheduled ones still waiting, under the lock the loop writes under.
func (f *fakeRepo) sentAndWaiting() [2]int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return [2]int{len(f.messages), len(f.scheduled)}
}

var quietLog = slog.New(slog.NewTextHandler(io.Discard, nil))

func TestScheduledMessage_NextAfter(t *testing.T) {
	london, err := time.LoadLocation("Europe/London")
	require.NoError(t, err)
	tests := []struct {
		name     string
		starts   time.Time
		repeats  Repeat
		timezone string
		after    time.Time
		want     time.Time
		wantOK   bool
	}{
		{"a one-off never comes round again", fixedNow, RepeatNone, "UTC", fixedNow, time.Time{}, false},
		{"daily keeps its clock time across the clocks going forward",
			time.Date(2026, 3, 28, 9, 0, 0, 0, london), RepeatDaily, "Europe/London", time.Date(2026, 3, 28, 9, 0, 0, 0, london),
			time.Date(2026, 3, 29, 9, 0, 0, 0, london), true},
		{"weekly lands seven days on", fixedNow, RepeatWeekly, "UTC", fixedNow, fixedNow.AddDate(0, 0, 7), true},
		{"monthly on the 31st skips the months without one",
			time.Date(2026, 1, 31, 9, 0, 0, 0, time.UTC), RepeatMonthly, "UTC", time.Date(2026, 1, 31, 9, 0, 0, 0, time.UTC),
			time.Date(2026, 3, 31, 9, 0, 0, 0, time.UTC), true},
		{"a long past skips straight to the first send ahead",
			time.Date(2020, 1, 1, 9, 0, 0, 0, time.UTC), RepeatDaily, "UTC", fixedNow, time.Date(2026, 8, 27, 9, 0, 0, 0, time.UTC), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &ScheduledMessage{StartsAt: tt.starts.UTC(), Repeats: tt.repeats, Timezone: tt.timezone}
			got, ok := m.NextAfter(tt.after)
			assert.Equal(t, tt.wantOK, ok)
			assert.True(t, tt.want.Equal(got), "want %s, got %s", tt.want, got)
		})
	}
}

func TestService_ScheduleMessage_Refusals(t *testing.T) {
	ctx := t.Context()
	repo := newFakeRepo()
	s := newTestService(repo)
	s.SetStanding(fakeStanding{mute: map[string][]string{"u-2": {"c-1"}}})
	repo.conversations["c-1"] = &Conversation{ID: "c-1", WorkspaceID: "w-1", Kind: KindChannel, Name: "eng"}
	later := fixedNow.Add(time.Hour)
	tests := []struct {
		name   string
		author string
		in     ScheduleInput
		want   error
	}{
		{"an empty body", "u-1", ScheduleInput{Body: "  ", SendAt: later}, apperrs.ErrInvalid},
		{"a send time already past", "u-1", ScheduleInput{Body: "hi", SendAt: fixedNow}, apperrs.ErrInvalid},
		{"an unknown repeat", "u-1", ScheduleInput{Body: "hi", SendAt: later, Repeats: "hourly"}, apperrs.ErrInvalid},
		{"an unknown timezone", "u-1", ScheduleInput{Body: "hi", SendAt: later, Timezone: "Mars/Olympus"}, apperrs.ErrInvalid},
		{"a channel the author may not post in", "u-2", ScheduleInput{Body: "hi", SendAt: later}, apperrs.ErrForbidden},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := s.ScheduleMessage(ctx, "c-1", tt.author, tt.in)
			require.ErrorIs(t, err, tt.want)
			assert.Empty(t, repo.scheduled)
		})
	}
}

func TestService_ScheduledMessage_OnlyItsAuthorSeesOrChangesIt(t *testing.T) {
	ctx := t.Context()
	repo := newFakeRepo()
	s := newTestService(repo)
	repo.conversations["c-1"] = &Conversation{ID: "c-1", WorkspaceID: "w-1", Kind: KindChannel, Name: "eng"}
	m, err := s.ScheduleMessage(ctx, "c-1", "u-1", ScheduleInput{Body: "Heads up for tomorrow", SendAt: fixedNow.Add(time.Hour)})
	require.NoError(t, err)

	others, total, err := s.PageScheduledMessages(ctx, "c-1", "u-2", paging.Window{})
	require.NoError(t, err)
	assert.Empty(t, others)
	assert.Zero(t, total)
	body := "taken over"
	_, err = s.UpdateScheduledMessage(ctx, m.ID, "u-2", ScheduleChange{Body: &body})
	require.ErrorIs(t, err, apperrs.ErrNotFound)
	require.ErrorIs(t, s.CancelScheduledMessage(ctx, m.ID, "u-2"), apperrs.ErrNotFound)
	_, err = s.SendScheduledMessageNow(ctx, m.ID, "u-2")
	require.ErrorIs(t, err, apperrs.ErrNotFound)

	own, total, err := s.PageScheduledMessages(ctx, "c-1", "u-1", paging.Window{})
	require.NoError(t, err)
	require.Equal(t, 1, total)
	assert.Equal(t, "Heads up for tomorrow", own[0].Body)
	assert.Empty(t, repo.messages)
}

func TestService_SendDueScheduledMessages(t *testing.T) {
	due := fixedNow.Add(-time.Minute)
	tests := []struct {
		name     string
		repeats  Repeat
		starts   time.Time
		mute     bool
		wantPost bool
		// wantNext is where the schedule moves to; zero means it is gone.
		wantNext time.Time
	}{
		{"a one-off posts and is gone", RepeatNone, due, false, true, time.Time{}},
		{"a daily one posts and moves a day on", RepeatDaily, due, false, true, due.AddDate(0, 0, 1)},
		{"a weekly one missed for a month posts once and skips to the next week ahead", RepeatWeekly, due.AddDate(0, 0, -28), false, true, due.AddDate(0, 0, 7)},
		{"one its author may no longer post posts nothing and is dropped", RepeatDaily, due, true, false, time.Time{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newFakeRepo()
			s := newTestService(repo)
			standing := fakeStanding{}
			if tt.mute {
				standing.mute = map[string][]string{"u-1": {"c-1"}}
			}
			s.SetStanding(standing)
			repo.conversations["c-1"] = &Conversation{ID: "c-1", WorkspaceID: "w-1", Kind: KindChannel, Name: "eng"}
			sendAt := due
			if tt.repeats == RepeatWeekly {
				sendAt = tt.starts
			}
			repo.scheduled["sm-1"] = &ScheduledMessage{ID: "sm-1", ConversationID: "c-1", AuthorID: "u-1", Body: "Standup in 10 @Agent",
				StartsAt: tt.starts, SendAt: sendAt, Repeats: tt.repeats, Timezone: "UTC"}

			_, err := s.SendDueScheduledMessages(t.Context(), quietLog)
			require.NoError(t, err)

			assert.Len(t, repo.messages, map[bool]int{true: 1, false: 0}[tt.wantPost])
			for _, msg := range repo.messages {
				assert.Equal(t, "u-1", msg.AuthorID)
				assert.Equal(t, AuthorUser, msg.AuthorKind)
				assert.Equal(t, []Mention{{Kind: MentionAgent, Handle: AgentHandle}}, msg.Mentions)
			}
			left, ok := repo.scheduled["sm-1"]
			if tt.wantNext.IsZero() {
				assert.False(t, ok, "the schedule should be gone")
				return
			}
			require.True(t, ok)
			assert.True(t, tt.wantNext.Equal(left.SendAt), "want next %s, got %s", tt.wantNext, left.SendAt)
		})
	}
}

func TestService_SendScheduledMessageNow_KeepsARepeatsNextSend(t *testing.T) {
	ctx := t.Context()
	repo := newFakeRepo()
	s := newTestService(repo)
	repo.conversations["c-1"] = &Conversation{ID: "c-1", WorkspaceID: "w-1", Kind: KindChannel, Name: "eng"}
	m, err := s.ScheduleMessage(ctx, "c-1", "u-1", ScheduleInput{Body: "Weekly notes", SendAt: fixedNow.Add(time.Hour), Repeats: RepeatWeekly})
	require.NoError(t, err)

	msg, err := s.SendScheduledMessageNow(ctx, m.ID, "u-1")
	require.NoError(t, err)

	assert.Equal(t, "Weekly notes", msg.Body)
	require.Contains(t, repo.scheduled, m.ID)
	assert.True(t, m.SendAt.Equal(repo.scheduled[m.ID].SendAt))
}

func TestMessagePost_SchedulesListsChangesAndCancels(t *testing.T) {
	s := newTestService(newFakeRepo())
	c, err := s.CreateChannel(t.Context(), "w-1", "u-1", "eng")
	require.NoError(t, err)

	out, err := callTool(as("u-1"), t, s, "message_post",
		`{"conversation_id":"`+c.ID+`","body":"Heads up","schedule":{"send_at":"2026-08-27T09:00:00Z","repeats":"weekly","timezone":"Europe/London"}}`)
	require.NoError(t, err)
	scheduled := out.(scheduledResult)
	assert.Equal(t, RepeatWeekly, scheduled.Repeats)

	out, err = callTool(as("u-1"), t, s, "message_post", `{"body":"Heads up: moved","schedule":{"id":"`+scheduled.ID+`"}}`)
	require.NoError(t, err)
	assert.Equal(t, "Heads up: moved", out.(scheduledResult).Body)
	assert.True(t, scheduled.SendAt.Equal(out.(scheduledResult).SendAt), "a body change keeps the send time")

	listed, err := callTool(as("u-1"), t, s, "message_list", `{"conversation_id":"`+c.ID+`","scheduled":true}`)
	require.NoError(t, err)
	require.Len(t, listed.(mcptool.Page[scheduledResult]).Items, 1)
	others, err := callTool(as("u-2"), t, s, "message_list", `{"conversation_id":"`+c.ID+`","scheduled":true}`)
	require.NoError(t, err)
	assert.Empty(t, others.(mcptool.Page[scheduledResult]).Items)

	_, err = callTool(as("u-1"), t, s, "message_post", `{"schedule":{"id":"`+scheduled.ID+`","cancel":true}}`)
	require.NoError(t, err)
	listed, err = callTool(as("u-1"), t, s, "message_list", `{"conversation_id":"`+c.ID+`","scheduled":true}`)
	require.NoError(t, err)
	assert.Empty(t, listed.(mcptool.Page[scheduledResult]).Items)
}

func TestMessagePost_ScheduleRefusals(t *testing.T) {
	s := newTestService(newFakeRepo())
	c, err := s.CreateChannel(t.Context(), "w-1", "u-1", "eng")
	require.NoError(t, err)
	target := `"conversation_id":"` + c.ID + `"`
	for name, args := range map[string]string{
		"cancel without an id":        `{` + target + `,"schedule":{"cancel":true}}`,
		"a send time that is no time": `{` + target + `,"body":"hi","schedule":{"send_at":"tomorrow"}}`,
		"a file with a schedule":      `{` + target + `,"body":"hi","file":{"name":"a.md","markdown":"x"},"schedule":{"send_at":"2026-08-27T09:00:00Z"}}`,
		"an id with a target":         `{` + target + `,"schedule":{"id":"sm-1","cancel":true}}`,
	} {
		t.Run(name, func(t *testing.T) {
			_, err := callTool(as("u-1"), t, s, "message_post", args)
			require.ErrorIs(t, err, apperrs.ErrInvalid)
		})
	}
}

func TestRunScheduleLoop_PostsWhenItFallsDueAndStopsWithItsContext(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		repo := newFakeRepo()
		s := newTestService(repo)
		s.now = time.Now
		repo.conversations["c-1"] = &Conversation{ID: "c-1", WorkspaceID: "w-1", Kind: KindChannel, Name: "eng"}
		_, err := s.ScheduleMessage(t.Context(), "c-1", "u-1", ScheduleInput{Body: "Heads up", SendAt: time.Now().Add(30 * time.Minute)})
		require.NoError(t, err)

		var commits wake.Broadcast
		ctx, cancel := context.WithCancel(t.Context())
		done := make(chan struct{})
		go func() {
			s.RunScheduleLoop(ctx, commits.Next, quietLog)
			close(done)
		}()

		synctest.Wait()
		assert.Equal(t, [2]int{0, 1}, repo.sentAndWaiting())
		time.Sleep(31 * time.Minute)
		synctest.Wait()
		assert.Equal(t, [2]int{1, 0}, repo.sentAndWaiting())

		cancel()
		<-done
	})
}
