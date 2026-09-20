package ids

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew_IsUUIDv7(t *testing.T) {
	id, err := uuid.Parse(New())
	require.NoError(t, err)
	assert.Equal(t, uuid.Version(7), id.Version())
}

func TestNew_IsUnique(t *testing.T) {
	assert.NotEqual(t, New(), New())
}
