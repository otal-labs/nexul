package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/otal-labs/nexul/internal/integrations"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/storage/sqlcgen"
)

var _ integrations.InstallStore = (*IntegrationInstallsRepo)(nil)
var _ integrations.TokenStore = (*IntegrationTokensRepo)(nil)

// IntegrationInstallsRepo persists integration installs (consent records).
type IntegrationInstallsRepo struct {
	db *sql.DB
	w  *Serializer
	q  *sqlcgen.Queries
}

func (r *IntegrationInstallsRepo) Create(ctx context.Context, install *integrations.Install) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		scopes, err := json.Marshal(install.Scopes)
		if err != nil {
			return fmt.Errorf("marshal scopes: %w", err)
		}
		err = r.q.WithTx(tx).CreateIntegrationInstall(ctx, sqlcgen.CreateIntegrationInstallParams{
			ID:            install.ID,
			Name:          install.Name,
			TrustTier:     string(install.TrustTier),
			WebhookUrl:    install.WebhookURL,
			WebhookSecret: install.WebhookSecret,
			Scopes:        string(scopes),
			CreatedBy:     install.CreatedBy,
			CreatedAt:     install.CreatedAt.Unix(),
			RevokedAt:     nullUnixPtr(install.RevokedAt),
		})
		if err != nil {
			return fmt.Errorf("insert install %s: %w", install.ID, classifyWriteErr(err))
		}
		return nil
	})
}

func (r *IntegrationInstallsRepo) GetByID(ctx context.Context, id string) (*integrations.Install, error) {
	row, err := r.q.GetIntegrationInstall(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get install %s: %w", id, notFoundIfNoRows(err))
	}
	install, err := toInstall(row)
	if err != nil {
		return nil, fmt.Errorf("get install %s: %w", id, err)
	}
	return install, nil
}

func (r *IntegrationInstallsRepo) List(ctx context.Context) ([]*integrations.Install, error) {
	rows, err := r.q.ListIntegrationInstalls(ctx)
	if err != nil {
		return nil, fmt.Errorf("list installs: %w", err)
	}
	var out []*integrations.Install
	for _, row := range rows {
		install, err := toInstall(row)
		if err != nil {
			return nil, fmt.Errorf("scan install: %w", err)
		}
		out = append(out, install)
	}
	return out, nil
}

func (r *IntegrationInstallsRepo) Revoke(ctx context.Context, id string) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		n, err := r.q.WithTx(tx).RevokeIntegrationInstall(ctx, sqlcgen.RevokeIntegrationInstallParams{
			RevokedAt: nullUnixNow(), ID: id,
		})
		if err != nil {
			return fmt.Errorf("revoke install %s: %w", id, err)
		}
		if n == 0 {
			return fmt.Errorf("revoke install %s: %w", id, apperrs.ErrNotFound)
		}
		return nil
	})
}

func toInstall(row sqlcgen.IntegrationInstall) (*integrations.Install, error) {
	install := &integrations.Install{
		ID:            row.ID,
		Name:          row.Name,
		TrustTier:     integrations.TrustTier(row.TrustTier),
		WebhookURL:    row.WebhookUrl,
		WebhookSecret: row.WebhookSecret,
		CreatedBy:     row.CreatedBy,
		CreatedAt:     time.Unix(row.CreatedAt, 0).UTC(),
	}
	if err := json.Unmarshal([]byte(row.Scopes), &install.Scopes); err != nil {
		return nil, fmt.Errorf("decode scopes for install %s: %w", install.ID, err)
	}
	if row.RevokedAt.Valid {
		t := time.Unix(row.RevokedAt.Int64, 0).UTC()
		install.RevokedAt = &t
	}
	return install, nil
}

// IntegrationTokensRepo persists scoped integration tokens. Only the hash of
// the raw token is stored.
type IntegrationTokensRepo struct {
	db *sql.DB
	w  *Serializer
	q  *sqlcgen.Queries
}

func (r *IntegrationTokensRepo) Create(ctx context.Context, token *integrations.IntegrationToken) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		err := r.q.WithTx(tx).CreateIntegrationToken(ctx, sqlcgen.CreateIntegrationTokenParams{
			ID:        token.ID,
			InstallID: token.InstallID,
			TokenHash: token.TokenHash,
			Prefix:    token.Prefix,
			CreatedAt: token.CreatedAt.Unix(),
		})
		if err != nil {
			return fmt.Errorf("insert token %s: %w", token.ID, classifyWriteErr(err))
		}
		return nil
	})
}

func (r *IntegrationTokensRepo) GetByHash(ctx context.Context, hash string) (*integrations.IntegrationToken, error) {
	row, err := r.q.GetIntegrationTokenByHash(ctx, hash)
	if err != nil {
		return nil, fmt.Errorf("get token: %w", notFoundIfNoRows(err))
	}
	return toIntegrationToken(row), nil
}

func (r *IntegrationTokensRepo) ListByInstall(ctx context.Context, installID string) ([]*integrations.IntegrationToken, error) {
	rows, err := r.q.ListIntegrationTokensByInstall(ctx, installID)
	if err != nil {
		return nil, fmt.Errorf("list tokens: %w", err)
	}
	var out []*integrations.IntegrationToken
	for _, row := range rows {
		out = append(out, toIntegrationToken(row))
	}
	return out, nil
}

func (r *IntegrationTokensRepo) RevokeByInstall(ctx context.Context, installID string) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		err := r.q.WithTx(tx).RevokeIntegrationTokensByInstall(ctx, sqlcgen.RevokeIntegrationTokensByInstallParams{
			RevokedAt: nullUnixNow(), InstallID: installID,
		})
		if err != nil {
			return fmt.Errorf("revoke install %s tokens: %w", installID, err)
		}
		return nil
	})
}

func (r *IntegrationTokensRepo) TouchLastUsed(ctx context.Context, id string) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		err := r.q.WithTx(tx).TouchIntegrationTokenLastUsed(ctx, sqlcgen.TouchIntegrationTokenLastUsedParams{
			LastUsedAt: nullUnixNow(), ID: id,
		})
		if err != nil {
			return fmt.Errorf("touch token %s: %w", id, err)
		}
		return nil
	})
}

func toIntegrationToken(row sqlcgen.IntegrationToken) *integrations.IntegrationToken {
	token := &integrations.IntegrationToken{
		ID:        row.ID,
		InstallID: row.InstallID,
		TokenHash: row.TokenHash,
		Prefix:    row.Prefix,
		CreatedAt: time.Unix(row.CreatedAt, 0).UTC(),
	}
	if row.LastUsedAt.Valid {
		t := time.Unix(row.LastUsedAt.Int64, 0).UTC()
		token.LastUsedAt = &t
	}
	if row.RevokedAt.Valid {
		t := time.Unix(row.RevokedAt.Int64, 0).UTC()
		token.RevokedAt = &t
	}
	return token
}

// nullUnixPtr converts an optional timestamp to the column's nullable param type; nil stays SQL NULL.
func nullUnixPtr(t *time.Time) sql.NullInt64 {
	if t == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: t.Unix(), Valid: true}
}

// nullUnixNow is nullUnixPtr(now) for the revoke/touch writes that always stamp the current time.
func nullUnixNow() sql.NullInt64 {
	return sql.NullInt64{Int64: time.Now().Unix(), Valid: true}
}
