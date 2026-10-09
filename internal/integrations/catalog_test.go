package integrations

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/eventcatalog"
)

func TestPublishCatalog_OnAFreshDatabase_SeedsEveryTopicAsVersionOne(t *testing.T) {
	schemas := eventcatalog.Schemas()
	svc := newTestServiceWithOwner(newFakeData(), true)
	require.NoError(t, svc.PublishCatalog(t.Context(), schemas))

	entries, err := svc.Catalog(t.Context())
	require.NoError(t, err)
	require.Len(t, entries, len(schemas))
	for _, entry := range entries {
		assert.Equal(t, schemas[entry.Topic], entry.Schema)
		assert.Equal(t, 1, entry.Version)
		var doc map[string]any
		require.NoError(t, json.Unmarshal([]byte(entry.Schema), &doc), "invalid schema JSON for %s", entry.Topic)
		assert.Equal(t, "https://json-schema.org/draft/2020-12/schema", doc["$schema"])
		assert.Equal(t, "object", doc["type"])
	}
}

// TestPublishCatalog_AfterASchemaEdit_PublishesTheNewTextAsTheNextVersion guards an upgraded instance: a topic seeded
// with an older schema text gets the current text as its next version, and an unchanged topic gets no new row.
func TestPublishCatalog_AfterASchemaEdit_PublishesTheNewTextAsTheNextVersion(t *testing.T) {
	schemas := map[string]string{"doc.created": `{"type":"object","properties":{"doc":{"type":"object"}}}`, "ticket.created": `{"type":"object"}`}
	svc := newTestServiceWithOwner(newFakeData(), true)
	require.NoError(t, svc.cfg.Schemas.Publish(t.Context(), SchemaEntry{Topic: "doc.created", Version: 1, Schema: `{"type":"object"}`}))
	require.NoError(t, svc.cfg.Schemas.Publish(t.Context(), SchemaEntry{Topic: "ticket.created", Version: 1, Schema: schemas["ticket.created"]}))

	require.NoError(t, svc.PublishCatalog(t.Context(), schemas))
	require.NoError(t, svc.PublishCatalog(t.Context(), schemas))

	entries, err := svc.Catalog(t.Context())
	require.NoError(t, err)
	versions := map[string][]SchemaEntry{}
	for _, e := range entries {
		versions[e.Topic] = append(versions[e.Topic], e)
	}
	require.Len(t, versions["doc.created"], 2, "the old version stays beside the new one")
	for _, e := range versions["doc.created"] {
		if e.Version == 2 {
			assert.Equal(t, schemas["doc.created"], e.Schema)
		}
	}
	require.Len(t, versions["ticket.created"], 1, "an unchanged schema publishes no new version")
	assert.Len(t, entries, 3)
}
