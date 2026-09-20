package automations

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteHostToken_EnvUnset_NoOp(t *testing.T) {
	t.Setenv(hostTokensPathEnv, "")
	require.NoError(t, writeHostToken("a1", "Ticket finished", "dat_raw"))
}

func TestWriteHostToken_CreatesFileWithEntry(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "tokens.json")
	t.Setenv(hostTokensPathEnv, path)

	require.NoError(t, writeHostToken("a1", "Ticket finished", "dat_raw1"))

	entries, err := readHostTokens(path)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, hostTokenEntry{ID: "a1", Name: "Ticket finished", Token: "dat_raw1"}, entries[0])
}

func TestWriteHostToken_UpsertsExistingEntryByID(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tokens.json")
	t.Setenv(hostTokensPathEnv, path)
	require.NoError(t, writeHostToken("a1", "Ticket finished", "dat_old"))

	require.NoError(t, writeHostToken("a2", "PR opened", "dat_2"))
	require.NoError(t, writeHostToken("a1", "Ticket finished (renamed)", "dat_new"))

	entries, err := readHostTokens(path)
	require.NoError(t, err)
	require.Len(t, entries, 2, "rotating a1's token must not duplicate it")
	assert.Equal(t, hostTokenEntry{ID: "a1", Name: "Ticket finished (renamed)", Token: "dat_new"}, entries[0])
	assert.Equal(t, hostTokenEntry{ID: "a2", Name: "PR opened", Token: "dat_2"}, entries[1])
}

func TestReadHostTokens_MissingFile_ReturnsEmpty(t *testing.T) {
	entries, err := readHostTokens(filepath.Join(t.TempDir(), "missing.json"))
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestReadHostTokens_MalformedFile_ReturnsError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tokens.json")
	require.NoError(t, os.WriteFile(path, []byte("not json"), 0o600))

	_, err := readHostTokens(path)
	require.Error(t, err)
}

func TestWriteHostToken_FileContentIsValidJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tokens.json")
	t.Setenv(hostTokensPathEnv, path)
	require.NoError(t, writeHostToken("a1", "Ticket finished", "dat_raw1"))

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	var raw []map[string]string
	require.NoError(t, json.Unmarshal(data, &raw))
	assert.Equal(t, "a1", raw[0]["id"])
}
