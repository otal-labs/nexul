package wordcount

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestChanged_CountsWordsAddedAndRemoved(t *testing.T) {
	tests := []struct {
		name           string
		before, after  string
		added, removed int
	}{
		{"identical text changes nothing", "one two three", "one two three", 0, 0},
		{"whitespace alone changes nothing", "one  two\nthree", "one two three", 0, 0},
		{"an inserted word is one added", "one three", "one two three", 1, 0},
		{"a dropped word is one removed", "one two three", "one three", 0, 1},
		{"a replaced word is one of each", "one two three", "one 2 three", 1, 1},
		{"an empty before adds every word", "", "one two", 2, 0},
		{"an empty after removes every word", "one two", "", 0, 2},
		{"two empty texts change nothing", "", "", 0, 0},
		{"a tail edit is counted past the matching start", "a b c d", "a b c e f", 2, 1},
		{"a head edit is counted before the matching end", "x b c d", "y z b c d", 2, 1},
		{"a moved word counts once each way", "a b c", "b c a", 1, 1},
		{"a repeated word is matched once per occurrence", "a a b", "a b", 0, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			added, removed := Changed(tt.before, tt.after)
			assert.Equal(t, tt.added, added, "added")
			assert.Equal(t, tt.removed, removed, "removed")
		})
	}
}
