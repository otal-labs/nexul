// Package wordcount sizes a text edit as the words added and removed, the figures a version's change size shows.
package wordcount

import "strings"

// Changed returns how many whitespace-separated words after has that before lacks, and how many before has that
// after lacks. The matching words at both ends are skipped first, so a small edit in a long text builds a small
// table, as the web diff does; ponytail: the middle is still O(n*m), so two long texts that differ throughout stay slow.
func Changed(before, after string) (added, removed int) {
	a, b := strings.Fields(before), strings.Fields(after)
	start := 0
	for start < len(a) && start < len(b) && a[start] == b[start] {
		start++
	}
	endA, endB := len(a), len(b)
	for endA > start && endB > start && a[endA-1] == b[endB-1] {
		endA--
		endB--
	}
	a, b = a[start:endA], b[start:endB]
	common := commonWords(a, b)
	return len(b) - common, len(a) - common
}

// commonWords is the length of the longest common subsequence of a and b, kept in one row of the table.
func commonWords(a, b []string) int {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	row := make([]int, len(b)+1)
	for _, wa := range a {
		diag := 0
		for j, wb := range b {
			above := row[j+1]
			row[j+1] = max(above, row[j])
			if wa == wb {
				row[j+1] = diag + 1
			}
			diag = above
		}
	}
	return row[len(b)]
}
