package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/otal-labs/nexul/internal/auth"
	"github.com/otal-labs/nexul/internal/platform/storage/sqlcgen"
)

var _ auth.ConnectCodeStore = (*ConnectCodesRepo)(nil)

// ConnectCodesRepo keeps only the SHA-256 hash of a connect code, like SetupCodesRepo, so a dump cannot sign a phone in.
type ConnectCodesRepo struct {
	db *sql.DB
	w  *Serializer
	q  *sqlcgen.Queries
}

// ReplaceConnectCode drops the user's earlier code and stores hash, in one transaction.
func (r *ConnectCodesRepo) ReplaceConnectCode(ctx context.Context, userID, hash string, createdAt, expiresAt time.Time) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		q := r.q.WithTx(tx)
		if err := q.DeleteConnectCodesByUser(ctx, userID); err != nil {
			return fmt.Errorf("delete connect codes of %s: %w", userID, err)
		}
		err := q.CreateConnectCode(ctx, sqlcgen.CreateConnectCodeParams{
			CodeHash: hash, UserID: userID, CreatedAt: createdAt.Unix(), ExpiresAt: expiresAt.Unix(),
		})
		if err != nil {
			return fmt.Errorf("insert connect code: %w", classifyWriteErr(err))
		}
		return nil
	})
}

// ConsumeConnectCode deletes the live code named by hash and returns its user; a missing or expired code is ErrNotFound.
func (r *ConnectCodesRepo) ConsumeConnectCode(ctx context.Context, hash string, now time.Time) (string, error) {
	var userID string
	err := r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		q := r.q.WithTx(tx)
		var err error
		userID, err = q.GetLiveConnectCodeUser(ctx, sqlcgen.GetLiveConnectCodeUserParams{CodeHash: hash, ExpiresAt: now.Unix()})
		if err != nil {
			return fmt.Errorf("get connect code: %w", notFoundIfNoRows(err))
		}
		if err := q.DeleteConnectCode(ctx, hash); err != nil {
			return fmt.Errorf("delete connect code: %w", err)
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	return userID, nil
}
