package storage

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/platform/eventbus"
)

func TestLicenseRepo_SetReplaceDelete_KeepsOneKeyAndItsEvents(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := t.Context()

	key, err := s.License.Key(ctx)
	require.NoError(t, err)
	assert.Empty(t, key, "a fresh instance holds no key")

	evt := func(id string) eventbus.OutboxEvent {
		return eventbus.OutboxEvent{ID: id, Topic: "instance.license_changed", Payload: map[string]string{"id": id}}
	}
	require.NoError(t, s.License.SetKey(ctx, "first", "u1", time.Unix(100, 0), evt("e1")))
	require.NoError(t, s.License.SetKey(ctx, "second", "u2", time.Unix(200, 0), evt("e2")))
	key, err = s.License.Key(ctx)
	require.NoError(t, err)
	assert.Equal(t, "second", key)

	require.NoError(t, s.License.DeleteKey(ctx, evt("e3")))
	key, err = s.License.Key(ctx)
	require.NoError(t, err)
	assert.Empty(t, key)

	pending, err := s.Outbox.Unpublished(ctx, 10)
	require.NoError(t, err)
	require.Len(t, pending, 3)
	assert.Equal(t, []string{"e1", "e2", "e3"}, []string{pending[0].ID, pending[1].ID, pending[2].ID})
}
