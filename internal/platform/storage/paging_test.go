package storage

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// pageAll reads a list page by page until it runs out, failing on a repeated id or a total that changes between
// pages, and returns every id in the order the pages gave them.
func pageAll(t *testing.T, limit int, page func(offset, limit int) (ids []string, total int)) []string {
	t.Helper()
	seen := map[string]bool{}
	var out []string
	firstTotal := -1
	for offset := 0; ; offset += limit {
		ids, total := page(offset, limit)
		if firstTotal < 0 {
			firstTotal = total
		}
		require.Equal(t, firstTotal, total, "total at offset %d", offset)
		for _, id := range ids {
			require.False(t, seen[id], "%s came back twice", id)
			seen[id] = true
			out = append(out, id)
		}
		if len(ids) < limit {
			require.Len(t, out, total, "the pages hold exactly total items")
			return out
		}
	}
}
