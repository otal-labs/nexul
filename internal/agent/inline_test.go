package agent

import (
	"bytes"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInlineMemoriesTrimmed_UnderCeiling_ReturnsPlainBlockNoLog(t *testing.T) {
	var buf bytes.Buffer
	log := slog.New(slog.NewTextHandler(&buf, nil))
	block := InlineMemoriesTrimmed([]InlinedMemory{{Title: "A", Body: "short"}, {Title: "B", Body: "next"}}, DefaultInlineLimits(), log)
	assert.Equal(t, "### A\nshort\n\n### B\nnext", block)
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
