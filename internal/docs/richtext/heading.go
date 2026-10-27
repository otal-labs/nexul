package richtext

import (
	"strings"
	"unicode"
)

var headingMarks = strings.NewReplacer(`\`, "", "*", "", "`", "")

// HeadingAbove is the text of the heading over the first markdown line holding a word that starts with one of words,
// so a search hit can open the doc at it; "" when that line comes before every heading, or no line holds one.
func HeadingAbove(markdown string, words []string) string {
	if len(words) == 0 {
		return ""
	}
	heading, fenced := "", false
	for line := range strings.Lines(markdown) {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			fenced = !fenced
		}
		if text, ok := atxHeading(trimmed); ok && !fenced {
			heading = text
		}
		if holdsWordPrefix(trimmed, words) {
			return heading
		}
	}
	return ""
}

// atxHeading reads a "## Text" line's text without its marks; a "#tag" with no space after the hashes is not a heading.
func atxHeading(line string) (string, bool) {
	text := strings.TrimLeft(line, "#")
	level := len(line) - len(text)
	if level == 0 || level > 6 || (text != "" && text[0] != ' ') {
		return "", false
	}
	return strings.TrimSpace(headingMarks.Replace(strings.TrimRight(strings.TrimSpace(text), "# "))), true
}

func holdsWordPrefix(line string, words []string) bool {
	tokens := strings.FieldsFunc(strings.ToLower(line), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	for _, token := range tokens {
		for _, w := range words {
			if strings.HasPrefix(token, strings.ToLower(w)) {
				return true
			}
		}
	}
	return false
}
