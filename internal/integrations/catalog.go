package integrations

import (
	"context"
	"fmt"
)

// PublishCatalog publishes each topic whose schema differs from its latest stored version as the next version,
// leaving earlier versions in place; an unchanged catalog writes nothing. schemas maps topic to schema text.
func (s *Service) PublishCatalog(ctx context.Context, schemas map[string]string) error {
	published, err := s.cfg.Schemas.Catalog(ctx)
	if err != nil {
		return fmt.Errorf("read published schemas: %w", err)
	}
	latest := make(map[string]SchemaEntry, len(published))
	for _, e := range published {
		if e.Version > latest[e.Topic].Version {
			latest[e.Topic] = e
		}
	}
	now := s.cfg.Now().UTC()
	for topic, schema := range schemas {
		prev, ok := latest[topic]
		if ok && prev.Schema == schema {
			continue
		}
		if err := s.cfg.Schemas.Publish(ctx, SchemaEntry{Topic: topic, Version: prev.Version + 1, Schema: schema, CreatedAt: now}); err != nil {
			return err
		}
	}
	return nil
}
