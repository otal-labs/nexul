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

func TestFrom_SameKeySameID_OtherKeyOtherID(t *testing.T) {
	assert.Equal(t, From("conv-1\x00item-1"), From("conv-1\x00item-1"))
	assert.NotEqual(t, From("conv-1\x00item-1"), From("conv-2\x00item-1"))
	id, err := uuid.Parse(From("conv-1\x00item-1"))
	require.NoError(t, err)
	assert.Equal(t, uuid.Version(5), id.Version())
}
