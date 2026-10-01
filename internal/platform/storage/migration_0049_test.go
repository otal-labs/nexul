package storage

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMigration0049_ReadNotificationsCountFromCreation_UnreadStayUnread(t *testing.T) {
	db := migrateBefore(t, "0049")
	_, err := db.Exec(`
INSERT INTO users (id, login, created_at, updated_at) VALUES ('u1', 'onik97', 0, 0);
INSERT INTO notifications (id, user_id, workspace_id, kind, subject_type, subject_id, subject_title, read, created_at) VALUES
    ('n-read', 'u1', 'workspace-default', 'doc.updated', 'doc', 'd-1', 'Spec', 1, 1000),
    ('n-unread', 'u1', 'workspace-default', 'doc.updated', 'doc', 'd-2', 'Plan', 0, 2000);
`)
	require.NoError(t, err)

	require.NoError(t, Migrate(db), "0049 and every later migration apply on top, as an upgrade would")
	s := New(db, testEncKey)
	ctx := t.Context()

	ns, err := s.Notifications.List(ctx, "u1", "", 50)
	require.NoError(t, err)
	byID := map[string]*time.Time{}
	for _, n := range ns {
		byID[n.ID] = n.ReadAt
	}
	require.Contains(t, byID, "n-read")
	require.NotNil(t, byID["n-read"])
	assert.Equal(t, time.Unix(1000, 0).UTC(), *byID["n-read"], "an unknown read time counts from creation")
	assert.Nil(t, byID["n-unread"])

	read, _, err := s.Notifications.DeleteExpired(ctx, time.Unix(1001, 0), time.Unix(0, 0))
	require.NoError(t, err)
	assert.Equal(t, int64(1), read, "a row read before the upgrade ages out like any read one")
}
