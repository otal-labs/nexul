package workspace

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

func meetingPayload(extra map[string]any) map[string]any {
	p := map[string]any{
		"meeting":  map[string]any{"id": "m-1", "workspace_id": "ws-1", "organiser_id": "u1", "title": "Standup"},
		"actor_id": "u1", "user_ids": []string{"u1", "u2", "u3"}, "invited_ids": []string{},
	}
	for k, v := range extra {
		p[k] = v
	}
	return p
}

func TestHandleMeetingEvents(t *testing.T) {
	tests := []struct {
		name      string
		topic     string
		handle    func(*testing.T, *NotificationService, map[string]any) error
		payload   map[string]any
		wantKind  Kind
		wantTitle string
		wantTo    []string
	}{
		{"an invite tells only who was just invited", "meeting.updated", func(t *testing.T, s *NotificationService, p map[string]any) error {
			return HandleMeetingInvited(t.Context(), s, notifEvFor(t, "meeting.updated", p))
		}, meetingPayload(map[string]any{"invited_ids": []string{"u3"}}), KindMeetingInvited, "Invited to Standup", []string{"u3"}},
		{"a cancellation tells everyone but who cancelled", "meeting.cancelled", func(t *testing.T, s *NotificationService, p map[string]any) error {
			return HandleMeetingCancelled(t.Context(), s, notifEvFor(t, "meeting.cancelled", p))
		}, meetingPayload(nil), KindMeetingCancelled, "Standup was cancelled", []string{"u2", "u3"}},
		{"a reminder tells every attendee how soon", "meeting.reminder_due", func(t *testing.T, s *NotificationService, p map[string]any) error {
			return HandleMeetingReminder(t.Context(), s, notifEvFor(t, "meeting.reminder_due", p))
		}, meetingPayload(map[string]any{"actor_id": "", "occurrence_starts_at": notifFixedNow.Add(10 * time.Minute)}),
			KindMeetingReminder, "Standup starts in 10 minutes", []string{"u1", "u2", "u3"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newFakeNotifRepo()
			s := newTestNotifService(repo, newFakeNotifUsers(notifUser("u1", "alice"), notifUser("u2", "bob"), notifUser("u3", "carol")))

			require.NoError(t, tt.handle(t, s, tt.payload))

			for _, uid := range []string{"u1", "u2", "u3"} {
				ns := repo.notifsFor(uid)
				if !containsString(tt.wantTo, uid) {
					assert.Empty(t, ns, uid)
					continue
				}
				require.Len(t, ns, 1, uid)
				assert.Equal(t, tt.wantKind, ns[0].Kind)
				assert.Equal(t, SubjectMeeting, ns[0].SubjectType)
				assert.Equal(t, "m-1", ns[0].SubjectID)
				assert.Equal(t, tt.wantTitle, ns[0].SubjectTitle)
				assert.Equal(t, "ws-1", ns[0].WorkspaceID)
			}
		})
	}
}

func containsString(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

func TestHandleMeetingInvited_NobodyNewlyInvitedIsNoNotice(t *testing.T) {
	repo := newFakeNotifRepo()
	s := newTestNotifService(repo, newFakeNotifUsers(notifUser("u2", "bob")))
	require.NoError(t, HandleMeetingInvited(t.Context(), s, notifEvFor(t, "meeting.updated", meetingPayload(nil))))
	assert.Empty(t, repo.notifsFor("u2"))
}

func TestHandleMeetingEvents_ABrokenPayloadIsFatal(t *testing.T) {
	repo := newFakeNotifRepo()
	s := newTestNotifService(repo, newFakeNotifUsers())
	for _, handle := range []func() error{
		func() error {
			return HandleMeetingInvited(t.Context(), s, notifEvFor(t, "meeting.created", map[string]any{"meeting": "nope"}))
		},
		func() error {
			return HandleMeetingCancelled(t.Context(), s, notifEvFor(t, "meeting.cancelled", map[string]any{"meeting": map[string]any{}}))
		},
		func() error {
			return HandleMeetingReminder(t.Context(), s, notifEvFor(t, "meeting.reminder_due", map[string]any{"meeting": 1}))
		},
		func() error {
			return HandleDueDateReminder(t.Context(), s, notifEvFor(t, "due_date.reminder_due", map[string]any{"ticket_id": 1}))
		},
		func() error {
			return HandleDueDateReminder(t.Context(), s, notifEvFor(t, "due_date.reminder_due", map[string]any{}))
		},
	} {
		require.ErrorIs(t, handle(), apperrs.ErrFatal)
	}
}

func TestHandleDueDateReminder_TellsTheDeveloper(t *testing.T) {
	repo := newFakeNotifRepo()
	s := newTestNotifService(repo, newFakeNotifUsers(notifUser("u2", "bob")))
	ev := notifEvFor(t, "due_date.reminder_due", map[string]any{
		"ticket_id": "t-1", "project_id": "p-1", "workspace_id": "ws-1", "key": "REF-7", "title": "Ship it", "due_on": "2026-11-03", "user_ids": []string{"u2"},
	})

	require.NoError(t, HandleDueDateReminder(t.Context(), s, ev))
	require.NoError(t, HandleDueDateReminder(t.Context(), s, ev), "a redelivery adds nothing")

	require.Len(t, repo.notifsFor("u2"), 1)
	n := repo.notifsFor("u2")[0]
	assert.Equal(t, KindTicketDueSoon, n.Kind)
	assert.Equal(t, SubjectTicket, n.SubjectType)
	assert.Equal(t, "REF-7 Ship it is due tomorrow", n.SubjectTitle)
}

func TestStartsIn(t *testing.T) {
	for d, want := range map[time.Duration]string{
		-time.Minute:                 "starts now",
		0:                            "starts now",
		30 * time.Second:             "starts in 1 minute",
		10 * time.Minute:             "starts in 10 minutes",
		59*time.Minute + time.Second: "starts in 1 hour",
		61 * time.Minute:             "starts in 1 hour",
		5 * time.Hour:                "starts in 5 hours",
		3 * 24 * time.Hour:           "starts in 3 days",
	} {
		assert.Equal(t, want, startsIn(d), d.String())
	}
}

func TestHandleMeetingReminder_LeavesOutWhoLeftTheWorkspace(t *testing.T) {
	repo := newFakeNotifRepo()
	members := &fakeMemberStore{byWorkspace: map[string][]string{"ws-1": {"u1", "u2"}}}
	s := newTestNotifServiceWith(repo, newFakeNotifUsers(notifUser("u1", "alice"), notifUser("u2", "bob"), notifUser("u3", "carol")), members, &fakeAccessChecker{})

	require.NoError(t, HandleMeetingReminder(t.Context(), s, notifEvFor(t, "meeting.reminder_due", meetingPayload(map[string]any{"actor_id": ""}))))

	assert.Len(t, repo.notifsFor("u1"), 1)
	assert.Len(t, repo.notifsFor("u2"), 1)
	assert.Empty(t, repo.notifsFor("u3"), "carol is no longer in the workspace")

	members.err = errors.New("db down")
	require.Error(t, HandleMeetingReminder(t.Context(), s, notifEvFor(t, "meeting.reminder_due", meetingPayload(map[string]any{"actor_id": ""}))))
}
