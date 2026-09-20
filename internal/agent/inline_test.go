package agent

import (
	"bytes"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

func TestInlineMemories_RendersInOrderWithTotals(t *testing.T) {
	block, totals, err := InlineMemories([]InlinedMemory{
		{Title: "Working here", Body: "always"},
		{Title: "Deploy quirks", Body: "picked one"},
	}, DefaultInlineLimits())
	require.NoError(t, err)
	assert.Equal(t, "### Working here\nalways\n\n### Deploy quirks\npicked one", block)
	assert.Equal(t, InlineTotals{Memories: 2, Chars: 16, LargestChars: 10, LargestTitle: "Deploy quirks"}, totals)
}

func TestInlineMemories_Empty_IsEmptyBlock(t *testing.T) {
	block, totals, err := InlineMemories(nil, DefaultInlineLimits())
	require.NoError(t, err)
	assert.Empty(t, block)
	assert.Equal(t, InlineTotals{}, totals)
}

func TestInlineMemories_OverPerMemoryCeiling_RefusesWithTotals(t *testing.T) {
	limits := InlineLimits{PerMemory: 5, PerRun: 100}
	_, totals, err := InlineMemories([]InlinedMemory{{Title: "Small", Body: "ok"}, {Title: "Big", Body: "too long"}}, limits)
	var oc *OverCeilingError
	require.ErrorAs(t, err, &oc)
	assert.ErrorIs(t, err, apperrs.ErrInvalid)
	assert.Equal(t, InlineTotals{Memories: 2, Chars: 10, LargestChars: 8, LargestTitle: "Big"}, totals)
	assert.Equal(t, totals, oc.Totals)
	assert.Equal(t, limits, oc.Limits)
	assert.Equal(t, `memory "Big" is 8 characters, over the 5 per-memory ceiling`, err.Error())
}

func TestInlineMemories_OverPerRunCeiling_RefusesWithTotals(t *testing.T) {
	limits := InlineLimits{PerMemory: 10, PerRun: 12}
	_, totals, err := InlineMemories([]InlinedMemory{{Title: "A", Body: "1234567"}, {Title: "B", Body: "1234567"}}, limits)
	var oc *OverCeilingError
	require.True(t, errors.As(err, &oc))
	assert.Equal(t, 14, totals.Chars)
	assert.Equal(t, "the 2 selected memories total 14 characters, over the 12 per-run ceiling", err.Error())
}

func TestInlineMemories_DefaultLimits_MatchConstants(t *testing.T) {
	limits := DefaultInlineLimits()
	assert.Equal(t, MaxMemoryChars, limits.PerMemory)
	assert.Equal(t, MaxInlinedMemoryChars, limits.PerRun)
	_, _, err := InlineMemories([]InlinedMemory{{Title: "Huge", Body: strings.Repeat("x", MaxMemoryChars+1)}}, limits)
	require.Error(t, err)
}

func TestInlineMemoriesTrimmed_UnderCeiling_ReturnsPlainBlockNoLog(t *testing.T) {
	var buf bytes.Buffer
	log := slog.New(slog.NewTextHandler(&buf, nil))
	block := InlineMemoriesTrimmed([]InlinedMemory{{Title: "A", Body: "short"}}, DefaultInlineLimits(), log)
	assert.Equal(t, "### A\nshort", block)
	assert.Empty(t, buf.String(), "a selection that already fits never logs a trim")
}

func TestInlineMemoriesTrimmed_Empty_ReturnsEmptyNoLog(t *testing.T) {
	var buf bytes.Buffer
	log := slog.New(slog.NewTextHandler(&buf, nil))
	assert.Empty(t, InlineMemoriesTrimmed(nil, DefaultInlineLimits(), log))
	assert.Empty(t, buf.String())
}

func TestInlineMemoriesTrimmed_OverPerRunCeiling_DropsFromEndWithNoteAndLog(t *testing.T) {
	var buf bytes.Buffer
	log := slog.New(slog.NewTextHandler(&buf, nil))
	limits := InlineLimits{PerMemory: 10, PerRun: 15}
	items := []InlinedMemory{{Title: "First", Body: "1234567890"}, {Title: "Second", Body: "1234567890"}}

	block := InlineMemoriesTrimmed(items, limits, log)

	assert.Equal(t, "### First\n1234567890\n\n[always-included memories trimmed to fit: 1 of 2 inlined]", block)
	assert.Contains(t, buf.String(), "trimmed to fit ceiling")
	assert.Contains(t, buf.String(), "level=WARN")
	assert.Contains(t, buf.String(), "inlined=1")
	assert.Contains(t, buf.String(), "total=2")
}

func TestInlineMemoriesTrimmed_LoneSurvivorOverPerMemoryCeiling_TruncatesItsBody(t *testing.T) {
	var buf bytes.Buffer
	log := slog.New(slog.NewTextHandler(&buf, nil))
	limits := InlineLimits{PerMemory: 5, PerRun: 100}

	block := InlineMemoriesTrimmed([]InlinedMemory{{Title: "Solo", Body: "toolong"}}, limits, log)

	assert.Equal(t, "### Solo\ntoolo\n\n[always-included memories trimmed to fit: 1 of 1 inlined]", block)
	assert.Contains(t, buf.String(), "trimmed to fit ceiling")
}

func TestInlineMemoriesTrimmed_ImpossibleLimits_DropsAllWithLog(t *testing.T) {
	var buf bytes.Buffer
	log := slog.New(slog.NewTextHandler(&buf, nil))
	limits := InlineLimits{PerMemory: 5, PerRun: 3} // PerRun below PerMemory: no single memory can ever fit

	block := InlineMemoriesTrimmed([]InlinedMemory{{Title: "X", Body: "abcdef"}}, limits, log)

	assert.Empty(t, block)
	assert.Contains(t, buf.String(), "dropping them all")
}
