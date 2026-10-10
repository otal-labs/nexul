package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/otal-labs/nexul/internal/auth"
	"github.com/otal-labs/nexul/internal/platform/crypto"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/storage/sqlcgen"
)

// GitHubManifestsRepo keeps callback hashes and commits the generated credentials together.
type GitHubManifestsRepo struct {
	db     *sql.DB
	w      *Serializer
	q      *sqlcgen.Queries
	encKey []byte
}

func (r *GitHubManifestsRepo) Start(ctx context.Context, stateHash, initiatorHash string, expiresAt, now time.Time) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		q := r.q.WithTx(tx)
		if err := q.DeleteExpiredGitHubManifests(ctx, now.Unix()); err != nil {
			return err
		}
		return q.StartGitHubManifest(ctx, sqlcgen.StartGitHubManifestParams{StateHash: stateHash, InitiatorHash: initiatorHash, ExpiresAt: expiresAt.Unix()})
	})
}

func (r *GitHubManifestsRepo) Consume(ctx context.Context, stateHash, initiatorHash string, now time.Time) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		n, err := r.q.WithTx(tx).ConsumeGitHubManifest(ctx, sqlcgen.ConsumeGitHubManifestParams{StateHash: stateHash, InitiatorHash: initiatorHash, ExpiresAt: now.Unix()})
		if err != nil {
			return err
		}
		if n != 1 {
			return fmt.Errorf("%w: GitHub setup is expired, already used, or belongs to another setup pass", apperrs.ErrInvalid)
		}
		return nil
	})
}

func (r *GitHubManifestsRepo) Save(ctx context.Context, app auth.GitHubAppCredentials) error {
	secret, err := crypto.Encrypt(r.encKey, []byte(app.ClientSecret))
	if err != nil {
		return err
	}
	key, err := crypto.Encrypt(r.encKey, []byte(app.PrivateKey))
	if err != nil {
		return err
	}
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		q := r.q.WithTx(tx)
		n, err := q.CountUsers(ctx)
		if err != nil {
			return err
		}
		if n != 0 {
			return apperrs.ErrConflict
		}
		n, err = q.SetSettingsGitHubOAuth(ctx, sqlcgen.SetSettingsGitHubOAuthParams{GithubOauthClientID: app.ClientID, GithubOauthClientSecret: secret, UpdatedAt: time.Now().Unix()})
		if err != nil {
			return err
		}
		if n != 1 {
			return apperrs.ErrNotFound
		}
		if err := q.SetConnectorAppConfig(ctx, sqlcgen.SetConnectorAppConfigParams{ConnectorID: "github", ClientID: app.ClientID, ClientSecret: secret, AppSlug: app.Slug}); err != nil {
			return err
		}
		_, err = q.SetConnectorAppPrivateKey(ctx, sqlcgen.SetConnectorAppPrivateKeyParams{ConnectorID: "github", PrivateKey: key})
		return err
	})
}
