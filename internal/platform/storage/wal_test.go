package storage_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/platform/storage/testutil/migrated"
)

func TestOpenDB_Connection_CapsTheWALAt32MiB(t *testing.T) {
	var limit int
	require.NoError(t, migrated.DB(t).QueryRow(`PRAGMA journal_size_limit`).Scan(&limit))
	assert.Equal(t, 32<<20, limit)
}
