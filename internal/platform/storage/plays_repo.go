package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/plays"

	"github.com/otal-labs/nexul/internal/platform/storage/sqlcgen"
)

var _ plays.Repo = (*PlaysRepo)(nil)

// PlaysRepo persists the plays domain's Play entity (ADR 0055).
type PlaysRepo struct {
	db *sql.DB
	w  *Serializer
	q  *sqlcgen.Queries
}

func (r *PlaysRepo) Create(ctx context.Context, p *plays.Play, evts ...eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		q := r.q.WithTx(tx)
		excluded, err := marshalStringList(p.ExcludedProjectIDs)
		if err != nil {
			return fmt.Errorf("encode excluded projects for play %s: %w", p.ID, err)
		}
		err = q.CreatePlay(ctx, sqlcgen.CreatePlayParams{
			ID: p.ID, WorkspaceID: p.WorkspaceID, Label: p.Label, Type: string(p.Type),
			Description: p.Description, Instructions: p.Instructions, Enabled: int64(boolInt(p.Enabled)),
			ShowWhenStage: nullStage(p.ShowWhenStage), ExcludedProjectIds: excluded,
			CreatedBy: p.CreatedBy, CreatedAt: p.CreatedAt.Unix(), UpdatedAt: p.UpdatedAt.Unix(),
		})
		if err != nil {
			return fmt.Errorf("insert play %s: %w", p.ID, classifyWriteErr(err))
		}
		return enqueuePlaysOutbox(ctx, tx, evts)
	})
}

func (r *PlaysRepo) Get(ctx context.Context, id string) (*plays.Play, error) {
	row, err := r.q.GetPlay(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get play %s: %w", id, notFoundIfNoRows(err))
	}
	return toPlay(row)
}

func (r *PlaysRepo) List(ctx context.Context, workspaceID string) ([]*plays.Play, error) {
	rows, err := r.q.ListPlays(ctx, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("list plays for workspace %s: %w", workspaceID, err)
	}
	var out []*plays.Play
	for _, row := range rows {
		p, err := toPlay(row)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, nil
}

func (r *PlaysRepo) Update(ctx context.Context, p *plays.Play, evts ...eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		q := r.q.WithTx(tx)
		excluded, err := marshalStringList(p.ExcludedProjectIDs)
		if err != nil {
			return fmt.Errorf("encode excluded projects for play %s: %w", p.ID, err)
		}
		n, err := q.UpdatePlay(ctx, sqlcgen.UpdatePlayParams{
			Label: p.Label, Description: p.Description, Instructions: p.Instructions,
			Enabled: int64(boolInt(p.Enabled)), ShowWhenStage: nullStage(p.ShowWhenStage),
			ExcludedProjectIds: excluded, UpdatedAt: p.UpdatedAt.Unix(), ID: p.ID,
		})
		if err != nil {
			return fmt.Errorf("update play %s: %w", p.ID, classifyWriteErr(err))
		}
		if n == 0 {
			return fmt.Errorf("update play %s: %w", p.ID, apperrs.ErrNotFound)
		}
		return enqueuePlaysOutbox(ctx, tx, evts)
	})
}

func (r *PlaysRepo) Delete(ctx context.Context, id string, evts ...eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		n, err := r.q.WithTx(tx).DeletePlay(ctx, id)
		if err != nil {
			return fmt.Errorf("delete play %s: %w", id, classifyWriteErr(err))
		}
		if n == 0 {
			return fmt.Errorf("delete play %s: %w", id, apperrs.ErrNotFound)
		}
		return enqueuePlaysOutbox(ctx, tx, evts)
	})
}

func enqueuePlaysOutbox(ctx context.Context, tx *sql.Tx, evts []eventbus.OutboxEvent) error {
	for _, evt := range evts {
		if err := insertOutboxRow(ctx, tx, evt.ID, evt.Topic, evt.Payload); err != nil {
			return err
		}
	}
	return nil
}

// marshalStringList always emits "[]" for an empty set, the same round-trip permissions.Set's MarshalJSON gives,
// rather than json.Marshal's bare "null" for a nil slice, which the column's NOT NULL text still accepts.
func marshalStringList(ids []string) (string, error) {
	if len(ids) == 0 {
		return "[]", nil
	}
	b, err := json.Marshal(ids)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func nullStage(s *plays.Stage) sql.NullString {
	if s == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: string(*s), Valid: true}
}

func toPlay(row sqlcgen.Play) (*plays.Play, error) {
	var excluded []string
	if err := json.Unmarshal([]byte(row.ExcludedProjectIds), &excluded); err != nil {
		return nil, fmt.Errorf("decode excluded projects for play %s: %w", row.ID, err)
	}
	var stage *plays.Stage
	if row.ShowWhenStage.Valid {
		s := plays.Stage(row.ShowWhenStage.String)
		stage = &s
	}
	return &plays.Play{
		ID: row.ID, WorkspaceID: row.WorkspaceID, Label: row.Label, Type: plays.Type(row.Type),
		Description: row.Description, Instructions: row.Instructions, Enabled: row.Enabled != 0,
		ShowWhenStage: stage, ExcludedProjectIDs: excluded, CreatedBy: row.CreatedBy,
		CreatedAt: time.Unix(row.CreatedAt, 0).UTC(), UpdatedAt: time.Unix(row.UpdatedAt, 0).UTC(),
	}, nil
}
