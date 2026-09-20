package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/otal-labs/nexul/internal/automations"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/storage/sqlcgen"
)

var _ automations.Repo = (*AutomationsRepo)(nil)

type AutomationsRepo struct {
	db *sql.DB
	w  *Serializer
	q  *sqlcgen.Queries
}

func (r *AutomationsRepo) Create(ctx context.Context, a *automations.Automation) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		subs, err := json.Marshal(a.Subscriptions)
		if err != nil {
			return err
		}
		scopes, err := json.Marshal(a.Scopes)
		if err != nil {
			return err
		}
		err = r.q.WithTx(tx).CreateAutomation(ctx, sqlcgen.CreateAutomationParams{
			ID:             a.ID,
			Name:           a.Name,
			Description:    a.Description,
			Kind:           string(a.Kind),
			Enabled:        int64(boolInt(a.Enabled)),
			Subscriptions:  subs,
			ConfigSchema:   nullableJSON(a.ConfigSchema),
			ConfigValues:   nullableJSON(a.ConfigValues),
			Scopes:         scopes,
			CreatedBy:      a.CreatedBy,
			TokenHash:      a.TokenHash,
			TokenPrefix:    a.TokenPrefix,
			TokenRevokedAt: tokenRevokedAtParam(a.TokenRevokedAt),
			CreatedAt:      a.CreatedAt.Unix(),
			UpdatedAt:      a.UpdatedAt.Unix(),
		})
		if err != nil {
			return classifyWriteErr(err)
		}
		return nil
	})
}

func (r *AutomationsRepo) Get(ctx context.Context, id string) (*automations.Automation, error) {
	row, err := r.q.GetAutomation(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get automation %s: %w", id, notFoundIfNoRows(err))
	}
	return toAutomation(row)
}

func (r *AutomationsRepo) GetByTokenHash(ctx context.Context, hash string) (*automations.Automation, error) {
	row, err := r.q.GetAutomationByTokenHash(ctx, hash)
	if err != nil {
		return nil, fmt.Errorf("get automation by token: %w", notFoundIfNoRows(err))
	}
	return toAutomation(row)
}

func (r *AutomationsRepo) List(ctx context.Context) ([]automations.Automation, error) {
	rows, err := r.q.ListAutomations(ctx)
	if err != nil {
		return nil, fmt.Errorf("list automations: %w", err)
	}
	var out []automations.Automation
	for _, row := range rows {
		a, err := toAutomation(row)
		if err != nil {
			return nil, fmt.Errorf("scan automation: %w", err)
		}
		out = append(out, *a)
	}
	return out, nil
}

func (r *AutomationsRepo) Update(ctx context.Context, a *automations.Automation) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		subs, err := json.Marshal(a.Subscriptions)
		if err != nil {
			return err
		}
		scopes, err := json.Marshal(a.Scopes)
		if err != nil {
			return err
		}
		n, err := r.q.WithTx(tx).UpdateAutomation(ctx, sqlcgen.UpdateAutomationParams{
			Name:           a.Name,
			Description:    a.Description,
			Kind:           string(a.Kind),
			Enabled:        int64(boolInt(a.Enabled)),
			Subscriptions:  subs,
			ConfigSchema:   nullableJSON(a.ConfigSchema),
			ConfigValues:   nullableJSON(a.ConfigValues),
			Scopes:         scopes,
			TokenHash:      a.TokenHash,
			TokenPrefix:    a.TokenPrefix,
			TokenRevokedAt: tokenRevokedAtParam(a.TokenRevokedAt),
			UpdatedAt:      a.UpdatedAt.Unix(),
			ID:             a.ID,
		})
		if err != nil {
			return classifyWriteErr(err)
		}
		if n == 0 {
			return apperrs.ErrNotFound
		}
		return nil
	})
}

func (r *AutomationsRepo) Delete(ctx context.Context, id string) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		n, err := r.q.WithTx(tx).DeleteAutomation(ctx, id)
		if err != nil {
			return err
		}
		if n == 0 {
			return apperrs.ErrNotFound
		}
		return nil
	})
}

func toAutomation(row sqlcgen.Automation) (*automations.Automation, error) {
	a := &automations.Automation{
		ID:          row.ID,
		Name:        row.Name,
		Description: row.Description,
		Kind:        automations.Kind(row.Kind),
		Enabled:     row.Enabled != 0,
		CreatedBy:   row.CreatedBy,
		TokenHash:   row.TokenHash,
		TokenPrefix: row.TokenPrefix,
		CreatedAt:   time.Unix(row.CreatedAt, 0).UTC(),
		UpdatedAt:   time.Unix(row.UpdatedAt, 0).UTC(),
	}
	var err error
	if a.Subscriptions, err = decodeStringList(row.Subscriptions); err != nil {
		return nil, fmt.Errorf("decode subscriptions for automation %s: %w", a.ID, err)
	}
	a.ConfigSchema = json.RawMessage(row.ConfigSchema)
	a.ConfigValues = json.RawMessage(row.ConfigValues)
	if a.Scopes, err = decodeStringList(row.Scopes); err != nil {
		return nil, fmt.Errorf("decode scopes for automation %s: %w", a.ID, err)
	}
	if row.TokenRevokedAt.Valid {
		t := time.Unix(row.TokenRevokedAt.Int64, 0).UTC()
		a.TokenRevokedAt = &t
	}
	return a, nil
}

// decodeStringList reads a NULL or empty list column as no entries instead of failing the whole row.
func decodeStringList(b []byte) ([]string, error) {
	if len(b) == 0 {
		return nil, nil
	}
	var out []string
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func nullableJSON(v json.RawMessage) []byte {
	if len(v) == 0 {
		return []byte(`{}`)
	}
	return v
}

// tokenRevokedAtParam builds the nullable column sqlc types as sql.NullInt64 from the domain's optional timestamp.
func tokenRevokedAtParam(t *time.Time) sql.NullInt64 {
	if t == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: t.Unix(), Valid: true}
}
