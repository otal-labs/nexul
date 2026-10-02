package agent

import (
	"fmt"
	"log/slog"
	"strings"
)

// InlinedMemory is one always-included memory carried in full inside a turn's request block.
type InlinedMemory struct {
	Title string
	Body  string
}

// InlineLimits are the per-memory and per-turn character ceilings the inlined memories must fit.
type InlineLimits struct {
	PerMemory int
	PerRun    int
}

// DefaultInlineLimits returns the ceilings declared beside MaxPromptChars.
func DefaultInlineLimits() InlineLimits {
	return InlineLimits{PerMemory: MaxMemoryChars, PerRun: MaxInlinedMemoryChars}
}

// renderMemories renders items in order as one block, and whether every memory and their total fit limits.
func renderMemories(items []InlinedMemory, limits InlineLimits) (string, bool) {
	var b strings.Builder
	total := 0
	fits := true
	for i, m := range items {
		total += len(m.Body)
		fits = fits && len(m.Body) <= limits.PerMemory
		if i > 0 {
			b.WriteString("\n\n")
		}
		fmt.Fprintf(&b, "### %s\n%s", m.Title, m.Body)
	}
	return b.String(), fits && total <= limits.PerRun
}

// InlineMemoriesTrimmed renders items as one block, never refusing them: over a ceiling, it drops items from the
// end until the rest fits, truncating a lone oversized survivor as a last resort, and logs the trim at warn with a
// "[... trimmed to fit ...]" note appended. It is for a project's standing always-included memories, sent best-effort.
func InlineMemoriesTrimmed(items []InlinedMemory, limits InlineLimits, log *slog.Logger) string {
	block, fits := renderMemories(items, limits)
	if fits {
		return block
	}
	kept := trimToFit(items, limits)
	block, fits = renderMemories(kept, limits)
	if !fits {
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
		if _, fits := renderMemories(kept, limits); fits {
			return kept
		}
		kept = kept[:len(kept)-1]
	}
	if len(kept) == 1 && len(kept[0].Body) > limits.PerMemory {
		kept[0] = InlinedMemory{Title: kept[0].Title, Body: kept[0].Body[:limits.PerMemory]}
	}
	return kept
}
