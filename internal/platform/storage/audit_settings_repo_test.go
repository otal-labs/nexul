package storage

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/ee/auditlog"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
)

func TestAuditSettingsRepo_SavesSealsAndRemovesWithItsEvent(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := t.Context()
	evt := func(id string) eventbus.OutboxEvent {
		return eventbus.OutboxEvent{ID: id, Topic: auditlog.TopicSettingsChanged, Payload: map[string]string{"id": id}}
	}

	st, err := s.AuditSettings.Get(ctx)
	require.NoError(t, err)
	assert.Equal(t, auditlog.Settings{}, st, "a fresh instance has no setting")

	want := auditlog.Settings{
		RetentionDays: 365, StreamURL: "https://logs.example.com", StreamSecret: "s3cret", StreamEnabled: true,
		UpdatedBy: "u-owner", UpdatedAt: time.Unix(1000, 0).UTC(),
	}
	require.NoError(t, s.AuditSettings.Set(ctx, want, evt("e1")))
	got, err := s.AuditSettings.Get(ctx)
	require.NoError(t, err)
	assert.Equal(t, want, got)
	var stored string
	require.NoError(t, s.db.QueryRowContext(ctx, `SELECT stream_secret FROM audit_settings`).Scan(&stored))
	assert.NotContains(t, stored, "s3cret", "the secret is sealed at rest")

	require.NoError(t, s.AuditSettings.Set(ctx, auditlog.Settings{RetentionDays: 90, UpdatedAt: time.Unix(2000, 0).UTC()}))
	got, err = s.AuditSettings.Get(ctx)
	require.NoError(t, err)
	assert.Equal(t, auditlog.Settings{RetentionDays: 90, UpdatedAt: time.Unix(2000, 0).UTC()}, got, "no secret saves as none")

	require.NoError(t, s.AuditSettings.Delete(ctx, evt("e2")))
	got, err = s.AuditSettings.Get(ctx)
	require.NoError(t, err)
	assert.Equal(t, auditlog.Settings{}, got)
	assert.Equal(t, []string{auditlog.TopicSettingsChanged, auditlog.TopicSettingsChanged}, outboxTopics(t, s))
}
