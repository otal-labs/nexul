//go:build !windows

package install

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStartDetached_StartsTheProcessWithItsLog(t *testing.T) {
	log := filepath.Join(t.TempDir(), "logs", "nexul-upgrade.log")
	require.NoError(t, startDetached("true", nil, log))
	assert.FileExists(t, log)
	require.Error(t, startDetached(filepath.Join(t.TempDir(), "absent"), nil, log))
}
