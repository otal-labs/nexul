package hostcred

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMatches_WrongOrEmpty_IsFalse(t *testing.T) {
	raw, hash, err := MintCredential("nxr_")
	require.NoError(t, err)
	assert.False(t, Matches(raw+"x", hash))
	assert.False(t, Matches("", hash))
	assert.False(t, Matches(raw, ""))
	assert.True(t, Matches(raw, hash))
}

func TestMintCode_HasPrefixAndThirtyTwoRandomBytes(t *testing.T) {
	raw, hash, err := MintCode()
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(raw, "nxe_"))
	assert.Len(t, strings.TrimPrefix(raw, "nxe_"), 43, "32 bytes base64url without padding")
	assert.Equal(t, Hash(raw), hash)
	assert.Len(t, hash, 64)

	other, _, err := MintCode()
	require.NoError(t, err)
	assert.NotEqual(t, raw, other)
}
