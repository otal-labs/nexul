package mentions

import (
	"cmp"
	"slices"
	"strings"
	"time"
	"unicode"
)

// Match values say where a ticket or doc hit's words were found.
const (
	MatchTitle = "title"
	MatchText  = "text"
)

// recencyDays is the age at which recent activity's boost has halved (ADR 0171).
const recencyDays = 7

type rankedHit struct {
	SearchHit
	match string
	score float64
}

// rankHits sorts hits as search shows them (ADR 0171): a title holding every word first, then a text match, and within
// each by relevance, doubled when every word is a whole word, times up to double again for recent activity.
func rankHits(hits []SearchHit, words []string, now time.Time) []rankedHit {
	out := make([]rankedHit, 0, len(hits))
	for _, h := range hits {
		match, matched := MatchText, h.Snippet
		if everyWord(words, tokens(h.Title), strings.HasPrefix) {
			match, matched = MatchTitle, h.Title
		}
		score := -h.Rank
		if everyWord(words, tokens(matched), func(token, word string) bool { return token == word }) {
			score *= 2
		}
		age := max(now.Sub(h.UpdatedAt).Hours()/24, 0)
		score *= 1 + recencyDays/(recencyDays+age)
		out = append(out, rankedHit{SearchHit: h, match: match, score: score})
	}
	slices.SortStableFunc(out, func(a, b rankedHit) int {
		if a.match != b.match {
			return cmp.Compare(sectionOrder(a.match), sectionOrder(b.match))
		}
		return cmp.Compare(b.score, a.score)
	})
	return out
}

func sectionOrder(match string) int {
	if match == MatchTitle {
		return 0
	}
	return 1
}

// tokens splits text as FTS5's tokenizer does, on anything that is not a letter or a digit, lowercased.
func tokens(text string) []string {
	return strings.FieldsFunc(strings.ToLower(text), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
}

// everyWord reports whether each word matches some token by match(token, word); no words never match.
func everyWord(words, toks []string, match func(token, word string) bool) bool {
	if len(words) == 0 {
		return false
	}
	for _, w := range words {
		if !slices.ContainsFunc(toks, func(t string) bool { return match(t, w) }) {
			return false
		}
	}
	return true
}

var snippetMarks = strings.NewReplacer("*", "", "`", "", "#", "", ">", "", `\`, "")

// plainSnippet drops a snippet's markdown marks and line breaks so it reads as one line.
func plainSnippet(s string) string {
	return strings.Join(strings.Fields(snippetMarks.Replace(s)), " ")
}
