package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/otal-labs/nexul/internal/connectors"
	"github.com/otal-labs/nexul/internal/platform/crypto"
	"github.com/otal-labs/nexul/internal/platform/storage/sqlcgen"
)

var _ connectors.AppConfigStore = (*ConnectorAppConfigRepo)(nil)

// ConnectorAppConfigRepo stores one instance-wide OAuth app registration per connector.
type ConnectorAppConfigRepo struct {
	db     *sql.DB
	w      *Serializer
	encKey []byte
	q      *sqlcgen.Queries
}

// GetAppConfig returns connectorID's config, decrypted, or a zero AppConfig if never set.
func (r *ConnectorAppConfigRepo) GetAppConfig(ctx context.Context, connectorID string) (connectors.AppConfig, error) {
	row, err := r.q.GetConnectorAppConfig(ctx, connectorID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return connectors.AppConfig{ConnectorID: connectorID}, nil
		}
		return connectors.AppConfig{}, fmt.Errorf("get connector app config %s: %w", connectorID, err)
	}
	c := connectors.AppConfig{ConnectorID: row.ConnectorID, ClientID: row.ClientID, BaseURL: row.BaseUrl, AppSlug: row.AppSlug}
	if row.ClientSecret != "" {
		secret, err := crypto.Decrypt(r.encKey, row.ClientSecret)
		if err != nil {
			return connectors.AppConfig{}, fmt.Errorf("decrypt client secret %s: %w", connectorID, err)
		}
		c.ClientSecret = string(secret)
	}
	return c, nil
}

// SetAppConfig upserts connectorID's config, encrypting the secret before it reaches SQL, for setup or rotation.
func (r *ConnectorAppConfigRepo) SetAppConfig(ctx context.Context, c connectors.AppConfig) error {
	encSecret := ""
	if c.ClientSecret != "" {
		var err error
		encSecret, err = crypto.Encrypt(r.encKey, []byte(c.ClientSecret))
		if err != nil {
			return fmt.Errorf("encrypt client secret %s: %w", c.ConnectorID, err)
		}
	}
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		err := r.q.WithTx(tx).SetConnectorAppConfig(ctx, sqlcgen.SetConnectorAppConfigParams{
			ConnectorID: c.ConnectorID, ClientID: c.ClientID, ClientSecret: encSecret, BaseUrl: c.BaseURL, AppSlug: c.AppSlug,
		})
		if err != nil {
			return fmt.Errorf("save connector app config %s: %w", c.ConnectorID, err)
		}
		return nil
	})
}
