package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/otal-labs/nexul/internal/automations"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/storage/sqlcgen"
)

var _ automations.VersionsRepo = (*AutomationVersionsRepo)(nil)

// AutomationVersionsRepo stores code versions; one pending/active row per automation is enforced by DB indexes.
type AutomationVersionsRepo struct {
	db *sql.DB
	w  *Serializer
	q  *sqlcgen.Queries
}

func (r *AutomationVersionsRepo) InsertActive(ctx context.Context, v *automations.Version) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		q := r.q.WithTx(tx)
		seq, err := nextVersionSequence(ctx, q, v.AutomationID)
		if err != nil {
			return err
		}
		v.Sequence = seq
		if err := q.InsertAutomationVersion(ctx, insertAutomationVersionParams(v)); err != nil {
			return classifyWriteErr(err)
		}
		return nil
	})
}

func (r *AutomationVersionsRepo) ReplacePending(ctx context.Context, v *automations.Version) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		q := r.q.WithTx(tx)
		// Computed before the discard so a replaced pending version's number is never handed out again.
		seq, err := nextVersionSequence(ctx, q, v.AutomationID)
		if err != nil {
			return err
		}
		v.Sequence = seq
		if err := q.DeletePendingAutomationVersion(ctx, sqlcgen.DeletePendingAutomationVersionParams{
			AutomationID: v.AutomationID, Status: string(automations.VersionPending),
		}); err != nil {
			return fmt.Errorf("discard previous pending version for automation %s: %w", v.AutomationID, err)
		}
		if err := q.InsertAutomationVersion(ctx, insertAutomationVersionParams(v)); err != nil {
			return classifyWriteErr(err)
		}
		return nil
	})
}

func (r *AutomationVersionsRepo) Activate(ctx context.Context, automationID, versionID string) (*automations.Version, error) {
	var result *automations.Version
	err := r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		q := r.q.WithTx(tx)
		count, err := q.CountAutomationVersionForAutomation(ctx, sqlcgen.CountAutomationVersionForAutomationParams{
			ID: versionID, AutomationID: automationID,
		})
		if err != nil {
			return fmt.Errorf("check version %s: %w", versionID, err)
		}
		if count == 0 {
			return apperrs.ErrNotFound
		}
		// Deactivate the current active row first so the "one active" unique index is never transiently violated.
		if err := q.DeactivateAutomationVersions(ctx, sqlcgen.DeactivateAutomationVersionsParams{
			NewStatus: string(automations.VersionInactive), AutomationID: automationID, OldStatus: string(automations.VersionActive),
		}); err != nil {
			return fmt.Errorf("deactivate current version for automation %s: %w", automationID, err)
		}
		if err := q.SetAutomationVersionStatus(ctx, sqlcgen.SetAutomationVersionStatusParams{
			Status: string(automations.VersionActive), ID: versionID,
		}); err != nil {
			return classifyWriteErr(err)
		}
		row, err := q.GetAutomationVersionByID(ctx, versionID)
		if err != nil {
			return fmt.Errorf("read activated version %s: %w", versionID, err)
		}
		result = toAutomationVersion(row)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (r *AutomationVersionsRepo) Get(ctx context.Context, automationID, versionID string) (*automations.Version, error) {
	row, err := r.q.GetAutomationVersion(ctx, sqlcgen.GetAutomationVersionParams{ID: versionID, AutomationID: automationID})
	if err != nil {
		return nil, fmt.Errorf("get version %s: %w", versionID, notFoundIfNoRows(err))
	}
	return toAutomationVersion(row), nil
}

func (r *AutomationVersionsRepo) ListByAutomation(ctx context.Context, automationID string) ([]automations.Version, error) {
	rows, err := r.q.ListAutomationVersionsByAutomation(ctx, automationID)
	if err != nil {
		return nil, fmt.Errorf("list versions for automation %s: %w", automationID, err)
	}
	var out []automations.Version
	for _, row := range rows {
		out = append(out, *toAutomationVersion(row))
	}
	return out, nil
}

func (r *AutomationVersionsRepo) Pending(ctx context.Context, automationID string) (*automations.Version, error) {
	return r.byStatus(ctx, automationID, automations.VersionPending)
}

func (r *AutomationVersionsRepo) Active(ctx context.Context, automationID string) (*automations.Version, error) {
	return r.byStatus(ctx, automationID, automations.VersionActive)
}

func (r *AutomationVersionsRepo) byStatus(ctx context.Context, automationID string, status automations.VersionStatus) (*automations.Version, error) {
	row, err := r.q.GetAutomationVersionByStatus(ctx, sqlcgen.GetAutomationVersionByStatusParams{
		AutomationID: automationID, Status: string(status),
	})
	if err != nil {
		return nil, fmt.Errorf("get %s version for automation %s: %w", status, automationID, notFoundIfNoRows(err))
	}
	return toAutomationVersion(row), nil
}

// nextVersionSequence runs inside the caller's transaction so the read and the following insert can never race.
func nextVersionSequence(ctx context.Context, q *sqlcgen.Queries, automationID string) (int, error) {
	max, err := q.MaxAutomationVersionSequence(ctx, automationID)
	if err != nil {
		return 0, fmt.Errorf("compute next sequence for automation %s: %w", automationID, err)
	}
	n, ok, err := optionalInt(max)
	if err != nil {
		return 0, fmt.Errorf("compute next sequence for automation %s: %w", automationID, err)
	}
	if !ok {
		return 1, nil
	}
	return int(n) + 1, nil
}

func insertAutomationVersionParams(v *automations.Version) sqlcgen.InsertAutomationVersionParams {
	return sqlcgen.InsertAutomationVersionParams{
		ID:           v.ID,
		AutomationID: v.AutomationID,
		Sequence:     int64(v.Sequence),
		Code:         []byte(v.Code),
		PusherID:     v.PusherID,
		Message:      v.Message,
		Status:       string(v.Status),
		CreatedAt:    v.CreatedAt.Unix(),
	}
}

func toAutomationVersion(row sqlcgen.AutomationVersion) *automations.Version {
	return &automations.Version{
		ID:           row.ID,
		AutomationID: row.AutomationID,
		Sequence:     int(row.Sequence),
		Code:         string(row.Code),
		PusherID:     row.PusherID,
		Message:      row.Message,
		Status:       automations.VersionStatus(row.Status),
		CreatedAt:    time.Unix(row.CreatedAt, 0).UTC(),
	}
}
