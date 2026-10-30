// Package textedit applies find-and-replace edits to a body, so a change never needs the whole body resent.
package textedit

import (
	"fmt"
	"strings"
)

// Edit replaces the one place Find occurs in the text with Replace.
type Edit struct {
	Find    string `json:"find" jsonschema:"The text to find."`
	Replace string `json:"replace" jsonschema:"Its replacement; empty deletes."`
}

// Apply runs edits in order and fails on the first empty or not-exactly-once Find, returning no half-edited text.
func Apply(text string, edits []Edit) (string, error) {
	for i, e := range edits {
		n := occurrences(text, e.Find)
		switch {
		case e.Find == "":
			return "", fmt.Errorf("edit %d: find is empty", i+1)
		case n == 0:
			return "", fmt.Errorf("edit %d: find matches nothing; read the current text and retry", i+1)
		case n > 1:
			return "", fmt.Errorf("edit %d: find matches %d places; include more surrounding text so it matches once", i+1, n)
		}
		text = strings.Replace(text, e.Find, e.Replace, 1)
	}
	return text, nil
}

// occurrences counts find in text including overlapping ones, so "aa" in "aaa" is two places, not one.
func occurrences(text, find string) int {
	if find == "" {
		return 0
	}
	n := 0
	for i := 0; ; {
		j := strings.Index(text[i:], find)
		if j < 0 {
			return n
		}
		n++
		i += j + 1
	}
}
