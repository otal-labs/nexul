package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/otal-labs/nexul/internal/access"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/permissions"
	"github.com/otal-labs/nexul/internal/platform/storage/sqlcgen"
)

var _ access.Repo = (*AccessRepo)(nil)

// AccessRepo persists permission_overwrites: one reusable per-user override row shared across every resource type.
type AccessRepo struct {
	db *sql.DB
	w  *Serializer
	q  *sqlcgen.Queries
}

func (r *AccessRepo) Get(ctx context.Context, resourceType, resourceID, userID string) (*access.Overwrite, error) {
	row, err := r.q.GetOverwrite(ctx, sqlcgen.GetOverwriteParams{
		ResourceType: resourceType, ResourceID: resourceID, UserID: userID,
	})
	if err != nil {
		return nil, fmt.Errorf("get overwrite %s/%s/%s: %w", resourceType, resourceID, userID, notFoundIfNoRows(err))
	}
	return toOverwrite(row)
}

// GetMany returns userID's overwrite on each of resourceIDs that has one, keyed by resource id, in one read.
// ListByUser returns every overwrite userID holds on a resourceType, keyed by resource id, in one read.
func (r *AccessRepo) ListByUser(ctx context.Context, resourceType, userID string) (map[string]*access.Overwrite, error) {
	rows, err := r.q.ListUserOverwritesOfType(ctx, sqlcgen.ListUserOverwritesOfTypeParams{ResourceType: resourceType, UserID: userID})
	if err != nil {
		return nil, fmt.Errorf("list %s overwrites of %s: %w", resourceType, userID, err)
	}
	out := make(map[string]*access.Overwrite, len(rows))
	for _, row := range rows {
		ow, err := toOverwrite(row)
		if err != nil {
			return nil, err
		}
		out[ow.ResourceID] = ow
	}
	return out, nil
}

func (r *AccessRepo) GetMany(ctx context.Context, resourceType string, resourceIDs []string, userID string) (map[string]*access.Overwrite, error) {
	out := make(map[string]*access.Overwrite, len(resourceIDs))
	if len(resourceIDs) == 0 {
		return out, nil
	}
	rows, err := r.q.ListOverwritesForUser(ctx, sqlcgen.ListOverwritesForUserParams{
		ResourceType: resourceType, UserID: userID, ResourceIds: resourceIDs,
	})
	if err != nil {
		return nil, fmt.Errorf("get overwrites %s for %s: %w", resourceType, userID, err)
	}
	for _, row := range rows {
		ow, err := toOverwrite(row)
		if err != nil {
			return nil, err
		}
		out[ow.ResourceID] = ow
	}
	return out, nil
}

func (r *AccessRepo) ListByResource(ctx context.Context, resourceType, resourceID string) ([]*access.Overwrite, error) {
	rows, err := r.q.ListOverwritesByResource(ctx, sqlcgen.ListOverwritesByResourceParams{
		ResourceType: resourceType, ResourceID: resourceID,
	})
	if err != nil {
		return nil, fmt.Errorf("list overwrites %s/%s: %w", resourceType, resourceID, err)
	}
	return toOverwrites(rows)
}

func (r *AccessRepo) Set(ctx context.Context, resourceType, resourceID, userID string, allow, deny permissions.Set, events ...eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		q := r.q.WithTx(tx)
		if len(allow) == 0 && len(deny) == 0 {
			n, err := q.DeleteOverwrite(ctx, sqlcgen.DeleteOverwriteParams{
				ResourceType: resourceType, ResourceID: resourceID, UserID: userID,
			})
			if err != nil {
				return fmt.Errorf("delete overwrite %s/%s/%s: %w", resourceType, resourceID, userID, err)
			}
			if n == 0 {
				return fmt.Errorf("delete overwrite %s/%s/%s: %w", resourceType, resourceID, userID, apperrs.ErrNotFound)
			}
			return insertOutboxRows(ctx, tx, events)
		}
		n, err := q.UpsertOverwrite(ctx, sqlcgen.UpsertOverwriteParams{
			ResourceType: resourceType, ResourceID: resourceID, UserID: userID,
			Allow: setJSON(allow), Deny: setJSON(deny), Now: time.Now().Unix(),
		})
		if err != nil {
			return fmt.Errorf("set overwrite %s/%s/%s: %w", resourceType, resourceID, userID, classifyWriteErr(err))
		}
		if n == 0 {
			return fmt.Errorf("set overwrite %s/%s/%s: %w", resourceType, resourceID, userID, apperrs.ErrConflict)
		}
		return insertOutboxRows(ctx, tx, events)
	})
}

func (r *AccessRepo) DeleteByResource(ctx context.Context, resourceType, resourceID string) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		if err := r.q.WithTx(tx).DeleteOverwritesByResource(ctx, sqlcgen.DeleteOverwritesByResourceParams{
			ResourceType: resourceType, ResourceID: resourceID,
		}); err != nil {
			return fmt.Errorf("delete overwrites %s/%s: %w", resourceType, resourceID, err)
		}
		return nil
	})
}

func (r *AccessRepo) HasAllowAny(ctx context.Context, resourceType, userID string, action permissions.Action) (bool, error) {
	// The quotes make the substring match exact: "docs:read" never matches inside "docs:readx".
	n, err := r.q.CountOverwriteAllowAny(ctx, sqlcgen.CountOverwriteAllowAnyParams{
		ResourceType: resourceType, UserID: userID, QuotedAction: `"` + string(action) + `"`,
	})
	if err != nil {
		return false, fmt.Errorf("has allow %s for %s/%s: %w", action, resourceType, userID, err)
	}
	return n > 0, nil
}

func toOverwrite(row sqlcgen.PermissionOverwrite) (*access.Overwrite, error) {
	allow, err := parseSet(row.Allow)
	if err != nil {
		return nil, fmt.Errorf("decode allow for %s/%s/%s: %w", row.ResourceType, row.ResourceID, row.UserID, err)
	}
	deny, err := parseSet(row.Deny)
	if err != nil {
		return nil, fmt.Errorf("decode deny for %s/%s/%s: %w", row.ResourceType, row.ResourceID, row.UserID, err)
	}
	return &access.Overwrite{
		ResourceType: row.ResourceType,
		ResourceID:   row.ResourceID,
		UserID:       row.UserID,
		Allow:        allow,
		Deny:         deny,
		CreatedAt:    time.Unix(row.CreatedAt, 0).UTC(),
		UpdatedAt:    time.Unix(row.UpdatedAt, 0).UTC(),
	}, nil
}

func toOverwrites(rows []sqlcgen.PermissionOverwrite) ([]*access.Overwrite, error) {
	var out []*access.Overwrite
	for _, row := range rows {
		ow, err := toOverwrite(row)
		if err != nil {
			return nil, err
		}
		out = append(out, ow)
	}
	return out, nil
}

// setJSON is the stored form of a permission set; marshalling a string slice cannot fail.
func setJSON(s permissions.Set) string {
	b, _ := json.Marshal(s)
	return string(b)
}

func parseSet(text string) (permissions.Set, error) {
	var s permissions.Set
	if err := json.Unmarshal([]byte(text), &s); err != nil {
		return nil, err
	}
	return s, nil
}
