package logging

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTee_WritesEachRecordAtOrAboveTheLevelAsOneJSONLine(t *testing.T) {
	var out bytes.Buffer
	logger := NewTee("warn", &out)

	logger.Info("below the level")
	logger.With("runner", "alpha").Warn("slow", "ms", 7)
	logger.Error("down")

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	require.Len(t, lines, 2)
	assert.Contains(t, lines[0], `"level":"WARN"`)
	assert.Contains(t, lines[0], `"runner":"alpha"`)
	assert.Contains(t, lines[0], `"ms":7`)
	assert.Contains(t, lines[1], `"msg":"down"`)
}
