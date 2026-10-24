package crashintake

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"
)

// noteKind is one of the running notes the intake keeps on a ticket, each updated in place instead of posted again.
type noteKind struct {
	file    string
	outcome Outcome
	heading string
	summary string
}

var (
	seenAgain = noteKind{"crash-seen-again.md", OutcomeSeenAgain, "Seen again", "This crash was reported again."}
	oldBuild  = noteKind{"crash-old-build.md", OutcomeOldBuild, "Old build",
		"This crash was reported from builds that don't contain its fix yet, so no new bug was filed."}
	cantTell = noteKind{"crash-cant-tell.md", OutcomeCantTell, "Can't tell",
		"GitHub couldn't say whether these builds contain the fix, so no new bug was filed."}
)

const (
	reportsKey  = "Reports: "
	versionsKey = "Versions: "
	buildsKey   = "Builds: "
	lastSeenKey = "Last seen: "
	reasonKey   = "Reason: "
)

// tally is a running note's content, read back from the note itself so the note stays the one record.
type tally struct {
	reports  int
	versions []string
	builds   []string
	lastSeen string
	reason   string
}

// parseTally reads the lines render writes; a line someone edited away reads as empty and the count starts over.
func parseTally(markdown string) tally {
	var t tally
	for line := range strings.SplitSeq(markdown, "\n") {
		line = strings.TrimSpace(line)
		if v, ok := strings.CutPrefix(line, reportsKey); ok {
			t.reports, _ = strconv.Atoi(v) // an edited count that isn't a number restarts from zero
		}
		if v, ok := strings.CutPrefix(line, versionsKey); ok {
			t.versions = splitList(v)
		}
		if v, ok := strings.CutPrefix(line, buildsKey); ok {
			t.builds = splitList(v)
		}
		if v, ok := strings.CutPrefix(line, lastSeenKey); ok {
			t.lastSeen = v
		}
		if v, ok := strings.CutPrefix(line, reasonKey); ok {
			t.reason = v
		}
	}
	return t
}

func splitList(v string) []string {
	var out []string
	for item := range strings.SplitSeq(v, ",") {
		if item = strings.TrimSpace(item); item != "" {
			out = append(out, item)
		}
	}
	return out
}

func (t *tally) add(r Report, reason string, at time.Time) {
	t.reports += r.Reports()
	if !slices.Contains(t.versions, r.AppVersion) {
		t.versions = append(t.versions, r.AppVersion)
	}
	if build := short(r.BuildCommit); !slices.Contains(t.builds, build) {
		t.builds = append(t.builds, build)
	}
	t.lastSeen = at.Format(time.RFC3339)
	if reason != "" {
		t.reason = reason
	}
}

func (t tally) render(k noteKind) string {
	var b strings.Builder
	fmt.Fprintf(&b, "## %s\n\n%s\n\n", k.heading, k.summary)
	fmt.Fprintf(&b, "%s%d\n\n%s%s\n\n%s%s\n\n%s%s\n", reportsKey, t.reports, versionsKey, strings.Join(t.versions, ", "),
		buildsKey, strings.Join(t.builds, ", "), lastSeenKey, t.lastSeen)
	if k == cantTell && t.reason != "" {
		fmt.Fprintf(&b, "\n%s%s\n", reasonKey, t.reason)
	}
	return b.String()
}
