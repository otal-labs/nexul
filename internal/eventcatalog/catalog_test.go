package eventcatalog

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAllTopics_NoDuplicates_SortedAndPopulated(t *testing.T) {
	topics := AllTopics()
	require.NotEmpty(t, topics)

	seen := map[string]bool{}
	for i, topic := range topics {
		assert.False(t, seen[topic], "duplicate topic %q", topic)
		seen[topic] = true
		if i > 0 {
			assert.Less(t, topics[i-1], topic, "topics must be sorted")
		}
	}
}
