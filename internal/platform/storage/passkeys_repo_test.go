package storage

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/auth"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
)

func TestPasskeysRepo_KeepsEachPersonsOwn(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	for _, u := range [][3]string{{"u1", "42", "alice"}, {"u2", "43", "bob"}} {
		_, _, err := s.Users.UpsertUser(t.Context(), newTestUser(u[0], u[1], u[2]))
		require.NoError(t, err)
	}
	at := time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC)
	p := &auth.Passkey{ID: "cred-1", UserID: "u1", Name: "Firefox on Linux", RPID: "nexul.example.com", Credential: []byte(`{"id":"x"}`), CreatedAt: at}
	added := eventbus.OutboxEvent{ID: "evt-1", Topic: auth.TopicPasskeyAdded, Payload: auth.PasskeyChangedEvent{UserID: "u1", PasskeyID: "cred-1"}}
	require.NoError(t, s.Passkeys.InsertPasskey(t.Context(), p, added))
	assert.ErrorIs(t, s.Passkeys.InsertPasskey(t.Context(), p), apperrs.ErrConflict, "a credential id is stored once")
	assert.Equal(t, []string{auth.TopicPasskeyAdded}, outboxTopicsOf(t, s))

	used := at.Add(time.Hour)
	require.NoError(t, s.Passkeys.UpdatePasskeyUse(t.Context(), "cred-1", []byte(`{"id":"y"}`), used))
	got, err := s.Passkeys.GetPasskey(t.Context(), "cred-1")
	require.NoError(t, err)
	assert.Equal(t, `{"id":"y"}`, string(got.Credential))
	require.NotNil(t, got.LastUsedAt)
	assert.True(t, used.Equal(*got.LastUsedAt))

	assert.ErrorIs(t, s.Passkeys.RenamePasskey(t.Context(), "u2", "cred-1", "Mine"), apperrs.ErrNotFound)
	assert.ErrorIs(t, s.Passkeys.DeletePasskey(t.Context(), "u2", "cred-1"), apperrs.ErrNotFound)
	require.NoError(t, s.Passkeys.RenamePasskey(t.Context(), "u1", "cred-1", "Work laptop"))
	listed, err := s.Passkeys.ListPasskeys(t.Context(), "u1")
	require.NoError(t, err)
	require.Len(t, listed, 1)
	assert.Equal(t, "Work laptop", listed[0].Name)
	others, err := s.Passkeys.ListPasskeys(t.Context(), "u2")
	require.NoError(t, err)
	assert.Empty(t, others)

	require.NoError(t, s.Passkeys.DeletePasskey(t.Context(), "u1", "cred-1"))
	_, err = s.Passkeys.GetPasskey(t.Context(), "cred-1")
	assert.ErrorIs(t, err, apperrs.ErrNotFound)
	assert.ErrorIs(t, s.Passkeys.UpdatePasskeyUse(t.Context(), "cred-1", nil, used), apperrs.ErrNotFound)
}
