package agent

import (
	"fmt"
	"log/slog"
	"strings"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// InlinedMemory is one memory carried in full inside a run's request block.
type InlinedMemory struct {
	Title string
	Body  string
}

// InlineLimits are the per-memory and per-run character ceilings a selection must fit.
type InlineLimits struct {
	PerMemory int
	PerRun    int
}

// DefaultInlineLimits returns the ceilings declared beside MaxPromptChars.
func DefaultInlineLimits() InlineLimits {
	return InlineLimits{PerMemory: MaxMemoryChars, PerRun: MaxInlinedMemoryChars}
}

// InlineTotals is what a selection adds up to; the run dialog shows them when a selection is refused.
type InlineTotals struct {
	Memories     int    `json:"memories"`
	Chars        int    `json:"chars"`
	LargestChars int    `json:"largest_chars"`
	LargestTitle string `json:"largest_title"`
}

// OverCeilingError refuses a selection over a ceiling, carrying the totals and the limits it broke.
type OverCeilingError struct {
	Totals InlineTotals
	Limits InlineLimits
}

func (e *OverCeilingError) Error() string {
	if e.Totals.LargestChars > e.Limits.PerMemory {
		return fmt.Sprintf("memory %q is %d characters, over the %d per-memory ceiling", e.Totals.LargestTitle, e.Totals.LargestChars, e.Limits.PerMemory)
	}
	return fmt.Sprintf("the %d selected memories total %d characters, over the %d per-run ceiling", e.Totals.Memories, e.Totals.Chars, e.Limits.PerRun)
}

func (e *OverCeilingError) Unwrap() error { return apperrs.ErrInvalid }

// InlineMemories renders items in order as one block, refusing the whole selection when any memory or the
// total is over limits; the totals come back either way so a caller can show them.
func InlineMemories(items []InlinedMemory, limits InlineLimits) (string, InlineTotals, error) {
	totals := InlineTotals{Memories: len(items)}
	var b strings.Builder
	for i, m := range items {
		n := len(m.Body)
		totals.Chars += n
		if n > totals.LargestChars {
			totals.LargestChars, totals.LargestTitle = n, m.Title
		}
		if i > 0 {
			b.WriteString("\n\n")
		}
		fmt.Fprintf(&b, "### %s\n%s", m.Title, m.Body)
	}
	if totals.LargestChars > limits.PerMemory || totals.Chars > limits.PerRun {
		return "", totals, &OverCeilingError{Totals: totals, Limits: limits}
	}
	return b.String(), totals, nil
}

// InlineMemoriesTrimmed renders like InlineMemories, but never refuses a selection: over a ceiling, it drops
// items from the end until the rest fits, truncating a lone oversized survivor as a last resort, and logs the
// trim at warn with a "[... trimmed to fit ...]" note appended. It is for material a turn sends best-effort
// (a project's standing always-included memories); InlineMemories' hard refusal stays for a run the user
// hand-picked, where dropping one silently would be the wrong call.
func InlineMemoriesTrimmed(items []InlinedMemory, limits InlineLimits, log *slog.Logger) string {
	block, _, err := InlineMemories(items, limits)
	if err == nil {
		return block
	}
	kept := trimToFit(items, limits)
	block, _, err = InlineMemories(kept, limits)
	if err != nil {
		log.Warn("agent: always-included memories still over ceiling after trimming, dropping them all",
			"total", len(items), "per_memory_limit", limits.PerMemory, "per_run_limit", limits.PerRun)
		return ""
	}
	log.Warn("agent: always-included memories trimmed to fit ceiling",
		"inlined", len(kept), "total", len(items), "per_memory_limit", limits.PerMemory, "per_run_limit", limits.PerRun)
	return block + fmt.Sprintf("\n\n[always-included memories trimmed to fit: %d of %d inlined]", len(kept), len(items))
}

// ponytail: trimToFit drops trailing items by a greedy trim, then truncates a lone survivor's body; not
// relevance-ranked, matching fitMemoriesIndex and fitContext's trim style.
func trimToFit(items []InlinedMemory, limits InlineLimits) []InlinedMemory {
	kept := append([]InlinedMemory(nil), items...)
	for len(kept) > 1 {
		if _, _, err := InlineMemories(kept, limits); err == nil {
			return kept
		}
		kept = kept[:len(kept)-1]
	}
	if len(kept) == 1 && len(kept[0].Body) > limits.PerMemory {
		kept[0] = InlinedMemory{Title: kept[0].Title, Body: kept[0].Body[:limits.PerMemory]}
	}
	return kept
}
