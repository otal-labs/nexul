package hostcred

import (
	"encoding/base64"
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

func TestToken_OnlyItsOwnKeyAndAnUntouchedPayloadPass(t *testing.T) {
	key := []byte("0123456789abcdef0123456789abcdef")
	type claims struct {
		Server string `json:"server"`
	}
	token, err := SignToken(claims{Server: "https://nexul.example.com"}, key)
	require.NoError(t, err)

	var got claims
	require.NoError(t, ParseToken(token, key, &got))
	assert.Equal(t, "https://nexul.example.com", got.Server)

	parts := strings.Split(token, ".")
	swapped := parts[0] + "." + base64.RawURLEncoding.EncodeToString([]byte(`{"server":"https://evil.example.com"}`)) + "." + parts[2]
	var peeked claims
	require.NoError(t, PeekToken(swapped, &peeked), "an installer reads where to go without the key")
	assert.Equal(t, "https://evil.example.com", peeked.Server)
	for name, bad := range map[string]string{"a swapped payload": swapped, "two segments": parts[0] + "." + parts[1], "not base64": token + "!", "empty": ""} {
		assert.ErrorIs(t, ParseToken(bad, key, &got), ErrBadToken, name)
	}
	assert.ErrorIs(t, ParseToken(token, []byte("another instance's key"), &got), ErrBadToken)
}

func TestComputerCommand_CarriesAnotherSiteAndReleaseOnlyWhenSet(t *testing.T) {
	assert.Equal(t, "curl -fsSL https://nexul.io/computer.sh | sh -s -- eyJ.x.y", ComputerCommand(DefaultSite, "", "eyJ.x.y"))
	assert.Equal(t, "curl -fsSL http://10.0.0.5:8000/computer.sh | NEXUL_INSTALL_URL=http://10.0.0.5:8000/install.sh NEXUL_RELEASE_URL=http://10.0.0.5:8000 sh -s -- eyJ.x.y",
		ComputerCommand("http://10.0.0.5:8000", "http://10.0.0.5:8000", "eyJ.x.y"))
}
