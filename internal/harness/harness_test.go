package harness

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCapDetail_CutsOnARuneBoundary(t *testing.T) {
	long := strings.Repeat("é", MaxActivityDetail)
	got := CapDetail(long)
	assert.LessOrEqual(t, len(got), MaxActivityDetail)
	assert.True(t, strings.HasSuffix(got, "é"), "never ends mid-rune")
	assert.Equal(t, "short", CapDetail("short"))
}

func TestPreview_FlattensAndTruncates(t *testing.T) {
	assert.Equal(t, "a b c", Preview("a\n  b\t c", 10))
	assert.Equal(t, "abc…", Preview("abcdef", 3))
	assert.Equal(t, "", Preview("  \n ", 5))
}
