package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/storage/sqlcgen"
	"github.com/otal-labs/nexul/internal/topology"
)

var _ topology.Repo = (*TopologyRepo)(nil)

type TopologyRepo struct {
	db *sql.DB
	w  *Serializer
	q  *sqlcgen.Queries
}

func (r *TopologyRepo) Get(ctx context.Context, environment string) (*topology.Canvas, error) {
	raw, err := r.q.GetTopologyCanvas(ctx, environment)
	if err != nil {
		return nil, fmt.Errorf("get topology %s: %w", environment, notFoundIfNoRows(err))
	}
	var c topology.Canvas
	if err := json.Unmarshal(raw, &c); err != nil {
		return nil, fmt.Errorf("unmarshal topology %s: %w", environment, err)
	}
	return &c, nil
}

func (r *TopologyRepo) Save(ctx context.Context, environment string, c *topology.Canvas, evts ...eventbus.OutboxEvent) error {
	raw, err := json.Marshal(c)
	if err != nil {
		return fmt.Errorf("marshal topology %s: %w", environment, err)
	}
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		err := r.q.WithTx(tx).SaveTopology(ctx, sqlcgen.SaveTopologyParams{
			Environment:   environment,
			SchemaVersion: int64(c.SchemaVersion),
			Canvas:        raw,
			UpdatedAt:     time.Now().Unix(),
		})
		if err != nil {
			return fmt.Errorf("save topology %s: %w", environment, err)
		}
		for _, evt := range evts {
			if err := insertOutboxRow(ctx, tx, evt.ID, evt.Topic, evt.Payload); err != nil {
				return err
			}
		}
		return nil
	})
}
