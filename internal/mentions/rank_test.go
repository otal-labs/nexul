package mentions

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRankHits(t *testing.T) {
	now := time.Date(2026, 10, 27, 12, 0, 0, 0, time.UTC)
	today, lastYear := now.Add(-time.Hour), now.AddDate(-1, 0, 0)
	tests := []struct {
		name  string
		query string
		hits  []SearchHit
		want  []string
	}{
		{"a title holding every word ranks above a better text match", "deploy",
			[]SearchHit{{ID: "text", Title: "Board lag", Rank: -9, Snippet: "deploy", UpdatedAt: today}, {ID: "title", Title: "Deploy fails", Rank: -1, UpdatedAt: lastYear}},
			[]string{"title", "text"}},
		{"a title holding only some words is a text match", "deploy runner",
			[]SearchHit{{ID: "half", Title: "Deploy fails", Rank: -5, Snippet: "the runner", UpdatedAt: today}, {ID: "full", Title: "Runner deploy", Rank: -1, UpdatedAt: today}},
			[]string{"full", "half"}},
		{"a whole word outranks the same relevance as a prefix", "log",
			[]SearchHit{{ID: "prefix", Title: "Login times out", Rank: -2, UpdatedAt: today}, {ID: "whole", Title: "Log viewer", Rank: -2, UpdatedAt: today}},
			[]string{"whole", "prefix"}},
		{"recent activity outranks the same relevance a year old", "log",
			[]SearchHit{{ID: "old", Title: "Log viewer", Rank: -2, UpdatedAt: lastYear}, {ID: "new", Title: "Log shipper", Rank: -2, UpdatedAt: today}},
			[]string{"new", "old"}},
		{"far better relevance still beats recency", "log",
			[]SearchHit{{ID: "new", Title: "Log shipper", Rank: -1, UpdatedAt: today}, {ID: "old", Title: "Log viewer", Rank: -5, UpdatedAt: lastYear}},
			[]string{"old", "new"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ranked := rankHits(tt.hits, tokens(tt.query), now)
			ids := make([]string, 0, len(ranked))
			for _, h := range ranked {
				ids = append(ids, h.ID)
			}
			assert.Equal(t, tt.want, ids)
		})
	}
}

func TestRankHits_NamesEachSection(t *testing.T) {
	ranked := rankHits([]SearchHit{{ID: "a", Title: "Deploy fails"}, {ID: "b", Title: "Board lag", Snippet: "deploy"}}, tokens("deploy"), time.Now())
	require.Len(t, ranked, 2)
	assert.Equal(t, MatchTitle, ranked[0].match)
	assert.Equal(t, MatchText, ranked[1].match)
}

func TestPlainSnippet_ReadsAsOneLine(t *testing.T) {
	assert.Equal(t, "Context the websocket keeps closing", plainSnippet("# Context\n\nthe **websocket** keeps `closing`"))
}
