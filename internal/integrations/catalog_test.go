package integrations

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/eventcatalog"
)

func TestPublishCatalogSeedsAllTopics(t *testing.T) {
	f := newFakeData()
	svc := newTestServiceWithOwner(f, true)
	require.NoError(t, svc.PublishCatalog(context.Background()))

	entries, err := svc.Catalog(context.Background())
	require.NoError(t, err)
	require.Len(t, entries, len(catalogSchemas))

	// Every schema must be valid JSON and carry a draft-2020-12 marker.
	for _, entry := range entries {
		_, ok := catalogSchemas[entry.Topic]
		assert.True(t, ok, "catalog contains unexpected topic %s", entry.Topic)
		assert.Equal(t, 1, entry.Version)
		var doc map[string]any
		require.NoError(t, json.Unmarshal([]byte(entry.Schema), &doc), "invalid schema JSON for %s", entry.Topic)
		assert.Contains(t, doc, "$schema")
		assert.Equal(t, "object", doc["type"])
	}
}

func TestPublishCatalogIdempotent(t *testing.T) {
	f := newFakeData()
	svc := newTestServiceWithOwner(f, true)
	require.NoError(t, svc.PublishCatalog(context.Background()))
	require.NoError(t, svc.PublishCatalog(context.Background()))

	entries, err := svc.Catalog(context.Background())
	require.NoError(t, err)
	assert.Len(t, entries, len(catalogSchemas))
}

func TestCatalogSchemasAreCompleteContracts(t *testing.T) {
	// Guard against an empty catalog regressing the published surface: the
	// catalog is the frozen public contract (ADR 0044), so it must keep every topic
	// documented. Additive changes add rows; this floor just catches an empty
	// wipe.
	assert.GreaterOrEqual(t, len(catalogSchemas), 20)
}

func TestCatalogSchemasCoverEveryPublishedTopic(t *testing.T) {
	// The schema catalog must reach full coverage of eventcatalog's
	// enumerable topic set, not lag it the way the hand-maintained list did
	// before (23/41 at the time of ticket 07). A new domain topic with no
	// schema here fails this test instead of silently shipping undocumented.
	for _, topic := range eventcatalog.AllTopics() {
		_, ok := catalogSchemas[topic]
		assert.True(t, ok, "missing event schema for published topic %s", topic)
	}
}
