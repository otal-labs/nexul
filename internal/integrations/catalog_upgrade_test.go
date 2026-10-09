package integrations_test

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/eventcatalog"
	"github.com/otal-labs/nexul/internal/integrations"
	"github.com/otal-labs/nexul/internal/platform/eventbus/testutil"
)

// TestPublishCatalog_OnAnInstanceSeededWithHandWrittenSchemas_AddsOneVersionPerChangedTopicOnce boots an instance
// whose event_schemas holds the catalog's hand-written texts, the state every instance was in before the schemas were
// generated: the first boot adds exactly one version per changed topic, and the second adds none.
func TestPublishCatalog_OnAnInstanceSeededWithHandWrittenSchemas_AddsOneVersionPerChangedTopicOnce(t *testing.T) {
	raw, err := os.ReadFile("testdata/handwritten_schemas.json")
	require.NoError(t, err)
	var handWritten map[string]string
	require.NoError(t, json.Unmarshal(raw, &handWritten))
	store := testutil.NewStore(t)
	for topic, text := range handWritten {
		require.NoError(t, store.EventSchemas.Publish(t.Context(), integrations.SchemaEntry{Topic: topic, Version: 1, Schema: text, CreatedAt: time.Now().UTC()}))
	}
	svc := integrations.NewService(integrations.Config{Schemas: store.EventSchemas, Now: func() time.Time { return time.Now().UTC() }})
	generated := eventcatalog.Schemas()

	require.NoError(t, svc.PublishCatalog(t.Context(), generated))
	afterFirst, err := svc.Catalog(t.Context())
	require.NoError(t, err)
	require.NoError(t, svc.PublishCatalog(t.Context(), generated))
	afterSecond, err := svc.Catalog(t.Context())
	require.NoError(t, err)

	versions := map[string]map[int]string{}
	for _, e := range afterFirst {
		if versions[e.Topic] == nil {
			versions[e.Topic] = map[int]string{}
		}
		versions[e.Topic][e.Version] = e.Schema
	}
	added := 0
	for topic, text := range generated {
		old, seeded := handWritten[topic]
		if !seeded {
			added++
			assert.Equal(t, map[int]string{1: text}, versions[topic], "a new topic starts at version 1")
			continue
		}
		if old == text {
			assert.Equal(t, map[int]string{1: old}, versions[topic], "%s is unchanged", topic)
			continue
		}
		added++
		assert.Equal(t, map[int]string{1: old, 2: text}, versions[topic], "%s keeps its old text and gains the generated one", topic)
	}
	assert.Len(t, afterFirst, len(handWritten)+added, "no row beyond one per new or changed topic")
	assert.Len(t, afterSecond, len(afterFirst), "a second boot publishes nothing")
}
