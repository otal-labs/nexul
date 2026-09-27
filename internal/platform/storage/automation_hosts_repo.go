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

var _ automations.HostRepo = (*AutomationHostsRepo)(nil)

type AutomationHostsRepo struct {
	db *sql.DB
	w  *Serializer
	q  *sqlcgen.Queries
}

func (r *AutomationHostsRepo) GetHost(ctx context.Context, id string) (*automations.Host, error) {
	row, err := r.q.GetAutomationHost(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get automations host %s: %w", id, notFoundIfNoRows(err))
	}
	return toHost(row), nil
}

func (r *AutomationHostsRepo) GetHostByName(ctx context.Context, name string) (*automations.Host, error) {
	row, err := r.q.GetAutomationHostByName(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("get automations host named %s: %w", name, notFoundIfNoRows(err))
	}
	return toHost(row), nil
}

func (r *AutomationHostsRepo) ListHosts(ctx context.Context) ([]automations.Host, error) {
	rows, err := r.q.ListAutomationHosts(ctx)
	if err != nil {
		return nil, fmt.Errorf("list automations hosts: %w", err)
	}
	out := make([]automations.Host, 0, len(rows))
	for _, row := range rows {
		out = append(out, *toHost(row))
	}
	return out, nil
}

func (r *AutomationHostsRepo) TouchHost(ctx context.Context, id string, at time.Time) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		n, err := r.q.WithTx(tx).TouchAutomationHost(ctx, sqlcgen.TouchAutomationHostParams{LastSeen: at.Unix(), ID: id})
		if err != nil {
			return fmt.Errorf("touch automations host %s: %w", id, err)
		}
		if n == 0 {
			return fmt.Errorf("touch automations host %s: %w", id, apperrs.ErrNotFound)
		}
		return nil
	})
}

func (r *AutomationHostsRepo) CreateHostEnrollment(ctx context.Context, e *automations.HostEnrollmentCode) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		q := r.q.WithTx(tx)
		if err := q.PruneAutomationHostEnrollmentCodes(ctx, e.CreatedAt.Unix()); err != nil {
			return fmt.Errorf("prune automations host enrollment codes: %w", err)
		}
		err := q.CreateAutomationHostEnrollmentCode(ctx, sqlcgen.CreateAutomationHostEnrollmentCodeParams{
			CodeHash: e.CodeHash, Name: e.Name, Machine: e.Machine, CreatedAt: e.CreatedAt.Unix(), ExpiresAt: e.ExpiresAt.Unix(),
		})
		if err != nil {
			return fmt.Errorf("insert automations host enrollment code: %w", classifyWriteErr(err))
		}
		return nil
	})
}

func (r *AutomationHostsRepo) GetHostEnrollment(ctx context.Context, codeHash string, now time.Time) (*automations.HostEnrollmentCode, error) {
	row, err := r.q.GetAutomationHostEnrollmentCode(ctx, sqlcgen.GetAutomationHostEnrollmentCodeParams{CodeHash: codeHash, ExpiresAt: now.Unix()})
	if err != nil {
		return nil, fmt.Errorf("get automations host enrollment code: %w", notFoundIfNoRows(err))
	}
	return &automations.HostEnrollmentCode{
		CodeHash: row.CodeHash, Name: row.Name, Machine: row.Machine,
		CreatedAt: time.Unix(row.CreatedAt, 0).UTC(), ExpiresAt: time.Unix(row.ExpiresAt, 0).UTC(),
	}, nil
}

func (r *AutomationHostsRepo) EnrollHost(ctx context.Context, codeHash string, h *automations.Host, credentialHash string, now time.Time) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		q := r.q.WithTx(tx)
		n, err := q.ConsumeAutomationHostEnrollmentCode(ctx, sqlcgen.ConsumeAutomationHostEnrollmentCodeParams{CodeHash: codeHash, ExpiresAt: now.Unix()})
		if err != nil {
			return fmt.Errorf("consume automations host enrollment code: %w", err)
		}
		if n == 0 {
			return fmt.Errorf("consume automations host enrollment code: %w", apperrs.ErrNotFound)
		}
		err = q.CreateAutomationHost(ctx, sqlcgen.CreateAutomationHostParams{
			ID: h.ID, Name: h.Name, Machine: h.Machine, Os: h.OS, Arch: h.Arch, Version: h.Version,
			LastSeen: h.LastSeen.Unix(), CreatedAt: h.CreatedAt.Unix(),
		})
		if err != nil {
			return fmt.Errorf("insert automations host %s: %w", h.Name, classifyWriteErr(err))
		}
		err = q.CreateAutomationHostCredential(ctx, sqlcgen.CreateAutomationHostCredentialParams{
			CredentialHash: credentialHash, HostID: h.ID, HostName: h.Name, CreatedAt: now.Unix(),
		})
		if err != nil {
			return fmt.Errorf("insert automations host %s credential: %w", h.Name, classifyWriteErr(err))
		}
		return nil
	})
}

func (r *AutomationHostsRepo) GetHostCredential(ctx context.Context, hash string) (*automations.HostCredential, error) {
	row, err := r.q.GetAutomationHostCredential(ctx, hash)
	if err != nil {
		return nil, fmt.Errorf("get automations host credential: %w", notFoundIfNoRows(err))
	}
	return toHostCredential(row), nil
}

func (r *AutomationHostsRepo) LiveHostCredential(ctx context.Context, hostID string) (*automations.HostCredential, error) {
	row, err := r.q.GetLiveAutomationHostCredential(ctx, hostID)
	if err != nil {
		return nil, fmt.Errorf("get live credential of automations host %s: %w", hostID, notFoundIfNoRows(err))
	}
	return toHostCredential(row), nil
}

func (r *AutomationHostsRepo) RemoveHost(ctx context.Context, id string, at time.Time) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		q := r.q.WithTx(tx)
		err := q.RevokeAutomationHostCredentials(ctx, sqlcgen.RevokeAutomationHostCredentialsParams{
			RevokedAt: sql.NullInt64{Int64: at.Unix(), Valid: true}, HostID: id,
		})
		if err != nil {
			return fmt.Errorf("revoke automations host %s credentials: %w", id, err)
		}
		n, err := q.DeleteAutomationHost(ctx, id)
		if err != nil {
			return fmt.Errorf("delete automations host %s: %w", id, err)
		}
		if n == 0 {
			return fmt.Errorf("delete automations host %s: %w", id, apperrs.ErrNotFound)
		}
		return nil
	})
}

func toHost(row sqlcgen.AutomationHost) *automations.Host {
	return &automations.Host{
		ID: row.ID, Name: row.Name, Machine: row.Machine, OS: row.Os, Arch: row.Arch, Version: row.Version,
		LastSeen: time.Unix(row.LastSeen, 0).UTC(), CreatedAt: time.Unix(row.CreatedAt, 0).UTC(),
	}
}

func toHostCredential(row sqlcgen.AutomationHostCredential) *automations.HostCredential {
	return &automations.HostCredential{Hash: row.CredentialHash, HostID: row.HostID, HostName: row.HostName, Revoked: row.RevokedAt.Valid}
}
