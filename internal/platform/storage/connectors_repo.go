package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/otal-labs/nexul/internal/connectors"
	"github.com/otal-labs/nexul/internal/platform/crypto"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/storage/sqlcgen"
)

var _ connectors.CredentialsStore = (*ConnectorsRepo)(nil)

// ConnectorsRepo stores one encrypted credential per connector: a singleton per connector, not multi-tenant.
type ConnectorsRepo struct {
	db     *sql.DB
	w      *Serializer
	encKey []byte
	q      *sqlcgen.Queries
}

// GetCredentials returns connectorID's stored credential, decrypting the
// tokens, or apperrs.ErrNotFound if that connector has never been connected.
func (r *ConnectorsRepo) GetCredentials(ctx context.Context, connectorID string) (connectors.Credentials, error) {
	row, err := r.q.GetConnectorCredentials(ctx, connectorID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return connectors.Credentials{}, apperrs.ErrNotFound
		}
		return connectors.Credentials{}, fmt.Errorf("get connector credentials %s: %w", connectorID, err)
	}

	c := connectors.Credentials{
		ConnectorID: row.ConnectorID,
		ConnectedBy: row.ConnectedBy,
	}
	access, err := crypto.Decrypt(r.encKey, row.AccessToken)
	if err != nil {
		return connectors.Credentials{}, fmt.Errorf("decrypt access token %s: %w", connectorID, err)
	}
	c.AccessToken = string(access)

	if row.RefreshToken != "" {
		refresh, err := crypto.Decrypt(r.encKey, row.RefreshToken)
		if err != nil {
			return connectors.Credentials{}, fmt.Errorf("decrypt refresh token %s: %w", connectorID, err)
		}
		c.RefreshToken = string(refresh)
	}

	if row.ManualFields != "" {
		manual, err := crypto.Decrypt(r.encKey, row.ManualFields)
		if err != nil {
			return connectors.Credentials{}, fmt.Errorf("decrypt manual fields %s: %w", connectorID, err)
		}
		if err := json.Unmarshal(manual, &c.ManualFields); err != nil {
			return connectors.Credentials{}, fmt.Errorf("decode manual fields %s: %w", connectorID, err)
		}
	}

	c.ExpiresAt = time.Unix(row.ExpiresAt, 0).UTC()
	c.ConnectedAt = time.Unix(row.ConnectedAt, 0).UTC()
	return c, nil
}

// SaveCredentials upserts connectorID's credential row, encrypting tokens before they reach SQL.
func (r *ConnectorsRepo) SaveCredentials(ctx context.Context, c connectors.Credentials) error {
	encAccess, err := crypto.Encrypt(r.encKey, []byte(c.AccessToken))
	if err != nil {
		return fmt.Errorf("encrypt access token %s: %w", c.ConnectorID, err)
	}
	encRefresh := ""
	if c.RefreshToken != "" {
		encRefresh, err = crypto.Encrypt(r.encKey, []byte(c.RefreshToken))
		if err != nil {
			return fmt.Errorf("encrypt refresh token %s: %w", c.ConnectorID, err)
		}
	}
	encManual := ""
	if len(c.ManualFields) > 0 {
		manual, err := json.Marshal(c.ManualFields)
		if err != nil {
			return fmt.Errorf("encode manual fields %s: %w", c.ConnectorID, err)
		}
		encManual, err = crypto.Encrypt(r.encKey, manual)
		if err != nil {
			return fmt.Errorf("encrypt manual fields %s: %w", c.ConnectorID, err)
		}
	}
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		err := r.q.WithTx(tx).SaveConnectorCredentials(ctx, sqlcgen.SaveConnectorCredentialsParams{
			ConnectorID:  c.ConnectorID,
			AccessToken:  encAccess,
			RefreshToken: encRefresh,
			ExpiresAt:    c.ExpiresAt.Unix(),
			ConnectedBy:  c.ConnectedBy,
			ConnectedAt:  c.ConnectedAt.Unix(),
			ManualFields: encManual,
		})
		if err != nil {
			return fmt.Errorf("save connector credentials %s: %w", c.ConnectorID, err)
		}
		return nil
	})
}

// DeleteCredentials removes connectorID's credential; a never-connected or already-disconnected connector is a no-op.
func (r *ConnectorsRepo) DeleteCredentials(ctx context.Context, connectorID string) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		err := r.q.WithTx(tx).DeleteConnectorCredentials(ctx, connectorID)
		if err != nil {
			return fmt.Errorf("delete connector credentials %s: %w", connectorID, err)
		}
		return nil
	})
}
