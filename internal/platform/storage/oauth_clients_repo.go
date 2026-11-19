package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/otal-labs/nexul/internal/auth"
	"github.com/otal-labs/nexul/internal/platform/storage/sqlcgen"
)

var _ auth.OAuthClientStore = (*OAuthClientsRepo)(nil)

// OAuthClientsRepo keeps the MCP clients that registered themselves through dynamic client registration.
type OAuthClientsRepo struct {
	db *sql.DB
	w  *Serializer
	q  *sqlcgen.Queries
}

func (r *OAuthClientsRepo) CreateOAuthClient(ctx context.Context, c auth.OAuthClient) error {
	uris, err := json.Marshal(c.RedirectURIs)
	if err != nil {
		return fmt.Errorf("encode redirect uris: %w", err)
	}
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		err := r.q.WithTx(tx).CreateOAuthClient(ctx, sqlcgen.CreateOAuthClientParams{
			ID: c.ID, Name: c.Name, RedirectUris: string(uris), CreatedAt: c.CreatedAt.Unix(),
		})
		if err != nil {
			return fmt.Errorf("insert oauth client %s: %w", c.ID, classifyWriteErr(err))
		}
		return nil
	})
}

func (r *OAuthClientsRepo) GetOAuthClient(ctx context.Context, id string) (*auth.OAuthClient, error) {
	row, err := r.q.GetOAuthClient(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get oauth client %s: %w", id, notFoundIfNoRows(err))
	}
	var uris []string
	if err := json.Unmarshal([]byte(row.RedirectUris), &uris); err != nil {
		return nil, fmt.Errorf("decode redirect uris of %s: %w", id, err)
	}
	return &auth.OAuthClient{ID: row.ID, Name: row.Name, RedirectURIs: uris, CreatedAt: time.Unix(row.CreatedAt, 0).UTC()}, nil
}
