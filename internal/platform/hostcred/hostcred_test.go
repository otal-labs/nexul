package hostcred

import (
	"os"
	"path/filepath"
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

func TestWriteCodeFile_ReplacesTheFileWholeAndPrivately(t *testing.T) {
	path := filepath.Join(t.TempDir(), "runner-instance")
	require.NoError(t, os.WriteFile(path, []byte("nxe_old"), 0o644))

	require.NoError(t, WriteCodeFile(path, "nxe_new"))

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "nxe_new", string(data))
	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())
	entries, err := os.ReadDir(filepath.Dir(path))
	require.NoError(t, err)
	assert.Len(t, entries, 1, "no temp file is left behind")
}

func TestWriteCodeFile_MissingDirectory_Fails(t *testing.T) {
	err := WriteCodeFile(filepath.Join(t.TempDir(), "missing", "runner-instance"), "nxe_new")
	require.Error(t, err)
}
