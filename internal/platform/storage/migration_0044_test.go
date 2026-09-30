package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMigration0044_DocsFromBeforeItReadBackUnlockedAndCanBeLocked(t *testing.T) {
	db := migrateBefore(t, "0044")
	_, err := db.Exec(`INSERT INTO docs (id, title, body, version, created_at, updated_at) VALUES ('d-old', 'Spec', '', 3, 1, 1)`)
	require.NoError(t, err)

	require.NoError(t, Migrate(db), "0044 and every later migration apply on top, as an upgrade would")
	s := New(db, testEncKey)

	got, err := s.Docs.GetByID(t.Context(), "d-old")
	require.NoError(t, err)
	assert.False(t, got.Locked, "an existing doc stays editable")
	assert.Equal(t, 3, got.Version, "its version is untouched")

	require.NoError(t, s.Docs.SetLocked(t.Context(), "d-old", true))
	got, err = s.Docs.GetByID(t.Context(), "d-old")
	require.NoError(t, err)
	assert.True(t, got.Locked)
}
