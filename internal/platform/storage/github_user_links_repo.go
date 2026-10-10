package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/otal-labs/nexul/internal/auth"
	"github.com/otal-labs/nexul/internal/platform/crypto"
	"github.com/otal-labs/nexul/internal/platform/storage/sqlcgen"
)

var _ auth.GitHubLinkStore = (*GitHubUserLinksRepo)(nil)

// GitHubUserLinksRepo stores each person's GitHub user token (ADR 0147), its tokens encrypted at rest.
type GitHubUserLinksRepo struct {
	db     *sql.DB
	w      *Serializer
	encKey []byte
	q      *sqlcgen.Queries
}

// GetGitHubLink returns userID's link with its tokens decrypted, ErrNotFound when they never connected GitHub.
func (r *GitHubUserLinksRepo) GetGitHubLink(ctx context.Context, userID string) (auth.GitHubLink, error) {
	row, err := r.q.GetGitHubUserLink(ctx, userID)
	if err != nil {
		return auth.GitHubLink{}, fmt.Errorf("get github link of %s: %w", userID, notFoundIfNoRows(err))
	}
	access, err := r.decrypt(row.AccessToken)
	if err != nil {
		return auth.GitHubLink{}, fmt.Errorf("decrypt github token of %s: %w", userID, err)
	}
	refresh, err := r.decrypt(row.RefreshToken)
	if err != nil {
		return auth.GitHubLink{}, fmt.Errorf("decrypt github refresh token of %s: %w", userID, err)
	}
	return auth.GitHubLink{
		UserID: row.UserID, AccessToken: access, RefreshToken: refresh,
		ExpiresAt: timeOrZero(row.ExpiresAt), RefreshExpiresAt: timeOrZero(row.RefreshExpiresAt),
		NeedsReconnect: row.NeedsReconnect != 0, ConnectedAt: time.Unix(row.ConnectedAt, 0).UTC(),
	}, nil
}

// SaveGitHubLink upserts l, encrypting its tokens before they reach SQL.
func (r *GitHubUserLinksRepo) SaveGitHubLink(ctx context.Context, l auth.GitHubLink) error {
	access, err := r.encrypt(l.AccessToken)
	if err != nil {
		return fmt.Errorf("encrypt github token of %s: %w", l.UserID, err)
	}
	refresh, err := r.encrypt(l.RefreshToken)
	if err != nil {
		return fmt.Errorf("encrypt github refresh token of %s: %w", l.UserID, err)
	}
	reconnect := int64(0)
	if l.NeedsReconnect {
		reconnect = 1
	}
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		err := r.q.WithTx(tx).SaveGitHubUserLink(ctx, sqlcgen.SaveGitHubUserLinkParams{
			UserID: l.UserID, AccessToken: access, RefreshToken: refresh,
			ExpiresAt: secondsOrZero(l.ExpiresAt), RefreshExpiresAt: secondsOrZero(l.RefreshExpiresAt),
			NeedsReconnect: reconnect, ConnectedAt: l.ConnectedAt.Unix(),
		})
		if err != nil {
			return fmt.Errorf("save github link of %s: %w", l.UserID, classifyWriteErr(err))
		}
		return nil
	})
}

// DeleteGitHubLink forgets userID's link; one never made is a no-op.
func (r *GitHubUserLinksRepo) DeleteGitHubLink(ctx context.Context, userID string) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		if err := r.q.WithTx(tx).DeleteGitHubUserLink(ctx, userID); err != nil {
			return fmt.Errorf("delete github link of %s: %w", userID, err)
		}
		return nil
	})
}

func (r *GitHubUserLinksRepo) encrypt(v string) (string, error) {
	if v == "" {
		return "", nil
	}
	return crypto.Encrypt(r.encKey, []byte(v))
}

func (r *GitHubUserLinksRepo) decrypt(v string) (string, error) {
	if v == "" {
		return "", nil
	}
	b, err := crypto.Decrypt(r.encKey, v)
	return string(b), err
}

func timeOrZero(sec int64) time.Time {
	if sec == 0 {
		return time.Time{}
	}
	return time.Unix(sec, 0).UTC()
}

func secondsOrZero(t time.Time) int64 {
	if t.IsZero() {
		return 0
	}
	return t.Unix()
}
