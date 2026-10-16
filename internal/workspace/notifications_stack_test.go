package workspace

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
)

type fakeStackUnwatchers struct {
	byStack map[string][]string
	err     error
}

func (f *fakeStackUnwatchers) ListUnwatcherIDs(_ context.Context, stackID string) ([]string, error) {
	return f.byStack[stackID], f.err
}

func stackOutagePayload(overrides map[string]any) map[string]any {
	payload := map[string]any{
		"workspace_id": "ws-1", "project_id": "p-1", "stack_id": "st-1", "stack_name": "acme-web",
		"service": "api", "exit_code": 137, "oom": true,
	}
	for k, v := range overrides {
		payload[k] = v
	}
	return payload
}

func newStackNotifService(repo *fakeNotifRepo, access *fakeAccessChecker, unwatchers *fakeStackUnwatchers) *NotificationService {
	members := &fakeMemberStore{byWorkspace: map[string][]string{"ws-1": {"alice", "bob", "carol"}}}
	return newTestNotifServiceWith(repo, newFakeNotifUsers(), members, access).WithStackUnwatchers(unwatchers)
}

func TestHandleServiceStopped(t *testing.T) {
	t.Run("tells every member who reads the stack and has not opted out", func(t *testing.T) {
		repo := newFakeNotifRepo()
		s := newStackNotifService(repo, &fakeAccessChecker{}, &fakeStackUnwatchers{byStack: map[string][]string{"st-1": {"bob"}}})

		require.NoError(t, HandleServiceStopped(t.Context(), s, notifEvFor(t, "service.container_stopped", stackOutagePayload(nil))))

		assert.Empty(t, repo.notifsFor("bob"), "an opt-out is not told")
		for _, user := range []string{"alice", "carol"} {
			require.Len(t, repo.notifsFor(user), 1)
			n := repo.notifsFor(user)[0]
			assert.Equal(t, KindServiceStopped, n.Kind)
			assert.Equal(t, SubjectStack, n.SubjectType)
			assert.Equal(t, "st-1", n.SubjectID)
			assert.Equal(t, "ws-1", n.WorkspaceID)
			assert.Equal(t, "api crashed on acme-web: exit 137, out of memory", n.SubjectTitle)
		}
	})

	t.Run("skips someone who cannot read the stack when the event fires", func(t *testing.T) {
		repo := newFakeNotifRepo()
		access := &fakeAccessChecker{denyStack: map[string]bool{"carol:st-1": true}}
		s := newStackNotifService(repo, access, &fakeStackUnwatchers{})

		require.NoError(t, HandleServiceStopped(t.Context(), s, notifEvFor(t, "service.container_stopped", stackOutagePayload(nil))))

		assert.Empty(t, repo.notifsFor("carol"))
		assert.Len(t, repo.notifsFor("alice"), 1)
	})

	t.Run("words the exit by what the container did", func(t *testing.T) {
		cases := []struct {
			name      string
			overrides map[string]any
			want      string
		}{
			{"plain exit", map[string]any{"oom": false, "exit_code": 1}, "api crashed on acme-web: exit 1"},
			{"only an out-of-memory kill seen", map[string]any{"oom": true, "exit_code": 0}, "api crashed on acme-web: out of memory"},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				repo := newFakeNotifRepo()
				s := newStackNotifService(repo, &fakeAccessChecker{}, &fakeStackUnwatchers{})

				require.NoError(t, HandleServiceStopped(t.Context(), s, notifEvFor(t, "service.container_stopped", stackOutagePayload(tc.overrides))))

				assert.Equal(t, tc.want, repo.notifsFor("alice")[0].SubjectTitle)
			})
		}
	})

	t.Run("tells nobody about a stack outside every project", func(t *testing.T) {
		repo := newFakeNotifRepo()
		s := newStackNotifService(repo, &fakeAccessChecker{}, &fakeStackUnwatchers{})
		payload := stackOutagePayload(map[string]any{"workspace_id": "", "project_id": ""})

		require.NoError(t, HandleServiceStopped(t.Context(), s, notifEvFor(t, "service.container_stopped", payload)))

		assert.Empty(t, repo.topics())
	})

	t.Run("a redelivered event adds nothing", func(t *testing.T) {
		repo := newFakeNotifRepo()
		s := newStackNotifService(repo, &fakeAccessChecker{}, &fakeStackUnwatchers{})
		ev := notifEvFor(t, "service.container_stopped", stackOutagePayload(nil))

		require.NoError(t, HandleServiceStopped(t.Context(), s, ev))
		require.NoError(t, HandleServiceStopped(t.Context(), s, ev))

		assert.Len(t, repo.notifsFor("alice"), 1)
	})

	t.Run("a malformed or stackless payload is fatal", func(t *testing.T) {
		s := newStackNotifService(newFakeNotifRepo(), &fakeAccessChecker{}, &fakeStackUnwatchers{})

		assert.ErrorIs(t, HandleServiceStopped(t.Context(), s, eventbus.Event{ID: "e1", Payload: []byte(`{`)}), apperrs.ErrFatal)
		assert.ErrorIs(t, HandleServiceStopped(t.Context(), s, notifEvFor(t, "service.container_stopped", map[string]any{})), apperrs.ErrFatal)
	})

	t.Run("a failed opt-out read fails the delivery so it retries", func(t *testing.T) {
		boom := errors.New("db down")
		s := newStackNotifService(newFakeNotifRepo(), &fakeAccessChecker{}, &fakeStackUnwatchers{err: boom})

		err := HandleServiceStopped(t.Context(), s, notifEvFor(t, "service.container_stopped", stackOutagePayload(nil)))

		assert.ErrorIs(t, err, boom)
	})
}

func TestHandleServiceRecovered(t *testing.T) {
	t.Run("tells the stack's watchers the service came back", func(t *testing.T) {
		repo := newFakeNotifRepo()
		s := newStackNotifService(repo, &fakeAccessChecker{}, &fakeStackUnwatchers{byStack: map[string][]string{"st-1": {"bob"}}})

		require.NoError(t, HandleServiceRecovered(t.Context(), s, notifEvFor(t, "service.container_recovered", stackOutagePayload(nil))))

		assert.Empty(t, repo.notifsFor("bob"))
		require.Len(t, repo.notifsFor("alice"), 1)
		n := repo.notifsFor("alice")[0]
		assert.Equal(t, KindServiceRecovered, n.Kind)
		assert.Equal(t, "api came back on acme-web", n.SubjectTitle)
	})

	t.Run("a stackless payload is fatal", func(t *testing.T) {
		s := newStackNotifService(newFakeNotifRepo(), &fakeAccessChecker{}, &fakeStackUnwatchers{})

		err := HandleServiceRecovered(t.Context(), s, notifEvFor(t, "service.container_recovered", map[string]any{}))

		assert.ErrorIs(t, err, apperrs.ErrFatal)
	})
}

func errorRatePayload(overrides map[string]any) map[string]any {
	payload := map[string]any{
		"workspace_id": "ws-1", "project_id": "p-1", "stack_id": "st-1", "stack_name": "acme-web",
		"peak_percent": 12.34, "limit_percent": 5, "window_minutes": 5,
	}
	for k, v := range overrides {
		payload[k] = v
	}
	return payload
}

func TestHandleErrorRateCrossed(t *testing.T) {
	t.Run("tells every member who reads the stack and has not opted out", func(t *testing.T) {
		repo := newFakeNotifRepo()
		access := &fakeAccessChecker{denyStack: map[string]bool{"carol:st-1": true}}
		s := newStackNotifService(repo, access, &fakeStackUnwatchers{byStack: map[string][]string{"st-1": {"bob"}}})

		require.NoError(t, HandleErrorRateCrossed(t.Context(), s, notifEvFor(t, "stack.error_rate_crossed", errorRatePayload(nil))))

		assert.Empty(t, repo.notifsFor("bob"), "an opt-out is not told")
		assert.Empty(t, repo.notifsFor("carol"), "someone who cannot read the stack is not told")
		require.Len(t, repo.notifsFor("alice"), 1)
		n := repo.notifsFor("alice")[0]
		assert.Equal(t, KindErrorRateCrossed, n.Kind)
		assert.Equal(t, SubjectStack, n.SubjectType)
		assert.Equal(t, "st-1", n.SubjectID)
		assert.Equal(t, "5xx at 12.3% on acme-web over the last 5 minutes, above 5%", n.SubjectTitle)
	})

	t.Run("tells nobody about a stack outside every project", func(t *testing.T) {
		repo := newFakeNotifRepo()
		s := newStackNotifService(repo, &fakeAccessChecker{}, &fakeStackUnwatchers{})

		require.NoError(t, HandleErrorRateCrossed(t.Context(), s, notifEvFor(t, "stack.error_rate_crossed", errorRatePayload(map[string]any{"workspace_id": "", "project_id": ""}))))

		assert.Empty(t, repo.topics())
	})

	t.Run("a redelivered event adds nothing", func(t *testing.T) {
		repo := newFakeNotifRepo()
		s := newStackNotifService(repo, &fakeAccessChecker{}, &fakeStackUnwatchers{})
		ev := notifEvFor(t, "stack.error_rate_crossed", errorRatePayload(nil))

		require.NoError(t, HandleErrorRateCrossed(t.Context(), s, ev))
		require.NoError(t, HandleErrorRateCrossed(t.Context(), s, ev))

		assert.Len(t, repo.notifsFor("alice"), 1)
	})

	t.Run("a malformed or stackless payload is fatal", func(t *testing.T) {
		s := newStackNotifService(newFakeNotifRepo(), &fakeAccessChecker{}, &fakeStackUnwatchers{})

		assert.ErrorIs(t, HandleErrorRateCrossed(t.Context(), s, eventbus.Event{ID: "e1", Payload: []byte(`{`)}), apperrs.ErrFatal)
		assert.ErrorIs(t, HandleErrorRateCrossed(t.Context(), s, notifEvFor(t, "stack.error_rate_crossed", map[string]any{})), apperrs.ErrFatal)
	})

	t.Run("a failed opt-out read fails the delivery so it retries", func(t *testing.T) {
		boom := errors.New("db down")
		s := newStackNotifService(newFakeNotifRepo(), &fakeAccessChecker{}, &fakeStackUnwatchers{err: boom})

		err := HandleErrorRateCrossed(t.Context(), s, notifEvFor(t, "stack.error_rate_crossed", errorRatePayload(nil)))

		assert.ErrorIs(t, err, boom)
	})
}

func TestHandleErrorRateRecovered(t *testing.T) {
	t.Run("tells the stack's watchers the share is back under the limit", func(t *testing.T) {
		repo := newFakeNotifRepo()
		s := newStackNotifService(repo, &fakeAccessChecker{}, &fakeStackUnwatchers{byStack: map[string][]string{"st-1": {"bob"}}})

		require.NoError(t, HandleErrorRateRecovered(t.Context(), s, notifEvFor(t, "stack.error_rate_recovered", errorRatePayload(nil))))

		assert.Empty(t, repo.notifsFor("bob"))
		require.Len(t, repo.notifsFor("alice"), 1)
		n := repo.notifsFor("alice")[0]
		assert.Equal(t, KindErrorRateRecovered, n.Kind)
		assert.Equal(t, "5xx on acme-web is back under 5%", n.SubjectTitle)
	})

	t.Run("a stackless payload is fatal", func(t *testing.T) {
		s := newStackNotifService(newFakeNotifRepo(), &fakeAccessChecker{}, &fakeStackUnwatchers{})

		err := HandleErrorRateRecovered(t.Context(), s, notifEvFor(t, "stack.error_rate_recovered", map[string]any{}))

		assert.ErrorIs(t, err, apperrs.ErrFatal)
	})
}
