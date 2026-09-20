package workspace

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func notifServe(t *testing.T, h http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	var r io.Reader
	if body != "" {
		r = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, r)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

// notifAuthedHandler resolves the current user to u1.
func notifAuthedHandler(s *NotificationService) http.Handler {
	return NewNotificationHandler(s, func(r *http.Request) string { return "u1" }).Routes()
}

// notifUnauthHandler resolves no current user (defensive check).
func notifUnauthHandler(s *NotificationService) http.Handler {
	return NewNotificationHandler(s, func(r *http.Request) string { return "" }).Routes()
}

func TestNotificationHandler_List(t *testing.T) {
	t.Run("unauthorized without current user", func(t *testing.T) {
		repo := newFakeNotifRepo()
		s := newTestNotifService(repo, newFakeNotifUsers())
		rec := notifServe(t, notifUnauthHandler(s), http.MethodGet, "/api/notifications", "")
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})
	t.Run("lists the current user's notifications", func(t *testing.T) {
		repo := newFakeNotifRepo()
		repo.create(t, mkNotif("n1", "u1"))
		repo.create(t, mkNotif("n2", "u2"))
		s := newTestNotifService(repo, newFakeNotifUsers())
		rec := notifServe(t, notifAuthedHandler(s), http.MethodGet, "/api/notifications?limit=10", "")
		require.Equal(t, http.StatusOK, rec.Code)
		var ns []*Notification
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &ns))
		require.Len(t, ns, 1)
		assert.Equal(t, "n1", ns[0].ID)
	})
	t.Run("invalid limit is treated as default", func(t *testing.T) {
		repo := newFakeNotifRepo()
		repo.create(t, mkNotif("n1", "u1"))
		s := newTestNotifService(repo, newFakeNotifUsers())
		rec := notifServe(t, notifAuthedHandler(s), http.MethodGet, "/api/notifications?limit=abc", "")
		assert.Equal(t, http.StatusOK, rec.Code)
	})
}

func TestNotificationHandler_UnreadCount(t *testing.T) {
	repo := newFakeNotifRepo()
	repo.create(t, mkNotif("n1", "u1"))
	s := newTestNotifService(repo, newFakeNotifUsers())
	rec := notifServe(t, notifAuthedHandler(s), http.MethodGet, "/api/notifications/unread-count", "")
	require.Equal(t, http.StatusOK, rec.Code)
	var out map[string]int
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &out))
	assert.Equal(t, 1, out["count"])
}

func TestNotificationHandler_MarkRead(t *testing.T) {
	t.Run("marks the notification read", func(t *testing.T) {
		repo := newFakeNotifRepo()
		repo.create(t, mkNotif("n1", "u1"))
		s := newTestNotifService(repo, newFakeNotifUsers())
		rec := notifServe(t, notifAuthedHandler(s), http.MethodPost, "/api/notifications/n1/read", "")
		assert.Equal(t, http.StatusNoContent, rec.Code)
		ns, err := s.List(context.Background(), "u1", 0)
		require.NoError(t, err)
		assert.True(t, ns[0].Read)
	})
	t.Run("someone else's notification is not found", func(t *testing.T) {
		repo := newFakeNotifRepo()
		repo.create(t, mkNotif("n1", "u2"))
		s := newTestNotifService(repo, newFakeNotifUsers())
		rec := notifServe(t, notifAuthedHandler(s), http.MethodPost, "/api/notifications/n1/read", "")
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
	t.Run("unauthorized without current user", func(t *testing.T) {
		repo := newFakeNotifRepo()
		s := newTestNotifService(repo, newFakeNotifUsers())
		rec := notifServe(t, notifUnauthHandler(s), http.MethodPost, "/api/notifications/n1/read", "")
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})
}

func TestNotificationHandler_MarkAllRead(t *testing.T) {
	t.Run("marks all of the user's notifications read", func(t *testing.T) {
		repo := newFakeNotifRepo()
		repo.create(t, mkNotif("n1", "u1"))
		repo.create(t, mkNotif("n2", "u1"))
		repo.create(t, mkNotif("n3", "u2"))
		s := newTestNotifService(repo, newFakeNotifUsers())
		rec := notifServe(t, notifAuthedHandler(s), http.MethodPost, "/api/notifications/read-all", "")
		assert.Equal(t, http.StatusNoContent, rec.Code)
		ns, err := s.List(context.Background(), "u1", 0)
		require.NoError(t, err)
		for _, n := range ns {
			assert.True(t, n.Read)
		}
		other, err := s.List(context.Background(), "u2", 0)
		require.NoError(t, err)
		for _, n := range other {
			assert.False(t, n.Read)
		}
	})
}
