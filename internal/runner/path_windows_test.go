//go:build windows

package runner

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadSavedPath_HoldsTheMachinesSystemFolder(t *testing.T) {
	saved, err := readSavedPath()
	require.NoError(t, err)
	system := strings.ToLower(filepath.Join(os.Getenv("SystemRoot"), "system32"))
	dirs := strings.Split(strings.ToLower(saved), string(filepath.ListSeparator))
	assert.Contains(t, dirs, system, "the machine's Path, expanded")
}
