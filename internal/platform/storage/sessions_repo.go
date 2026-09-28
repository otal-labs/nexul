package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/otal-labs/nexul/internal/auth"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/storage/sqlcgen"
	"github.com/otal-labs/nexul/internal/push"
)

var (
	_ auth.SessionStore = (*SessionsRepo)(nil)
	_ push.TokenStore   = (*SessionsRepo)(nil)
)

// SessionsRepo persists only the SHA-256 hash of a session token, like PATsRepo, so a leaked dump cannot be replayed.
type SessionsRepo struct {
	db *sql.DB
	w  *Serializer
	q  *sqlcgen.Queries
}

func (r *SessionsRepo) CreateSession(ctx context.Context, s *auth.Session, evts ...eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		err := r.q.WithTx(tx).CreateSession(ctx, sqlcgen.CreateSessionParams{
			ID: s.ID, UserID: s.UserID, TokenHash: s.TokenHash, Client: string(s.Client),
			Platform: s.Platform, Label: s.Label, Ip: s.IP,
			CreatedAt: s.CreatedAt.Unix(), LastActiveAt: s.LastActiveAt.Unix(), ExpiresAt: s.ExpiresAt.Unix(),
		})
		if err != nil {
			return fmt.Errorf("insert session %s: %w", s.ID, classifyWriteErr(err))
		}
		return insertOutboxRows(ctx, tx, evts)
	})
}

func (r *SessionsRepo) GetSessionByHash(ctx context.Context, hash string) (*auth.Session, error) {
	row, err := r.q.GetSessionByHash(ctx, hash)
	if err != nil {
		return nil, fmt.Errorf("get session: %w", notFoundIfNoRows(err))
	}
	return toSession(row), nil
}

func (r *SessionsRepo) ListSessionsByUser(ctx context.Context, userID string) ([]auth.Session, error) {
	rows, err := r.q.ListSessionsByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list sessions: %w", err)
	}
	out := make([]auth.Session, 0, len(rows))
	for _, row := range rows {
		out = append(out, *toSession(row))
	}
	return out, nil
}

func (r *SessionsRepo) DeleteSession(ctx context.Context, id, userID string, evts ...eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		n, err := r.q.WithTx(tx).DeleteSession(ctx, sqlcgen.DeleteSessionParams{ID: id, UserID: userID})
		if err != nil {
			return fmt.Errorf("delete session %s: %w", id, err)
		}
		if n == 0 {
			return fmt.Errorf("delete session %s: %w", id, apperrs.ErrNotFound)
		}
		return insertOutboxRows(ctx, tx, evts)
	})
}

func (r *SessionsRepo) DeleteOtherSessions(ctx context.Context, userID, keepID string, evts ...eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		if err := r.q.WithTx(tx).DeleteOtherSessions(ctx, sqlcgen.DeleteOtherSessionsParams{UserID: userID, ID: keepID}); err != nil {
			return fmt.Errorf("delete other sessions of %s: %w", userID, err)
		}
		return insertOutboxRows(ctx, tx, evts)
	})
}

func (r *SessionsRepo) TouchSession(ctx context.Context, id string, lastActive time.Time, ip string, expiresAt time.Time) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		err := r.q.WithTx(tx).TouchSession(ctx, sqlcgen.TouchSessionParams{
			LastActiveAt: lastActive.Unix(), Ip: ip, ExpiresAt: expiresAt.Unix(), ID: id,
		})
		if err != nil {
			return fmt.Errorf("touch session %s: %w", id, err)
		}
		return nil
	})
}

func (r *SessionsRepo) DeleteExpiredSessions(ctx context.Context, userID string, now time.Time) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		err := r.q.WithTx(tx).DeleteExpiredSessions(ctx, sqlcgen.DeleteExpiredSessionsParams{UserID: userID, ExpiresAt: now.Unix()})
		if err != nil {
			return fmt.Errorf("delete expired sessions of %s: %w", userID, err)
		}
		return nil
	})
}

// SetSessionPushToken stores the token, or NULL for an empty one; user-scoped like every session write.
func (r *SessionsRepo) SetSessionPushToken(ctx context.Context, id, userID, token string) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		n, err := r.q.WithTx(tx).SetSessionPushToken(ctx, sqlcgen.SetSessionPushTokenParams{
			PushToken: sql.NullString{String: token, Valid: token != ""}, ID: id, UserID: userID,
		})
		if err != nil {
			return fmt.Errorf("set push token on session %s: %w", id, err)
		}
		if n == 0 {
			return fmt.Errorf("set push token on session %s: %w", id, apperrs.ErrNotFound)
		}
		return nil
	})
}

// ClearPushToken is the push sender's DeviceNotRegistered path; a row already gone is not an error.
func (r *SessionsRepo) ClearPushToken(ctx context.Context, sessionID, userID string) error {
	err := r.SetSessionPushToken(ctx, sessionID, userID, "")
	if errors.Is(err, apperrs.ErrNotFound) {
		return nil
	}
	return err
}

// ListPushTargets returns the live phone sessions holding a token for any of the users.
func (r *SessionsRepo) ListPushTargets(ctx context.Context, userIDs []string) ([]push.Target, error) {
	if len(userIDs) == 0 {
		return nil, nil
	}
	rows, err := r.q.ListPushTargets(ctx, sqlcgen.ListPushTargetsParams{Now: time.Now().Unix(), UserIds: userIDs})
	if err != nil {
		return nil, fmt.Errorf("list push targets: %w", err)
	}
	out := make([]push.Target, 0, len(rows))
	for _, row := range rows {
		out = append(out, push.Target{SessionID: row.ID, UserID: row.UserID, Token: row.PushToken.String})
	}
	return out, nil
}

func toSession(row sqlcgen.Session) *auth.Session {
	return &auth.Session{
		ID: row.ID, UserID: row.UserID, TokenHash: row.TokenHash, Client: auth.SessionClient(row.Client),
		Platform: row.Platform, Label: row.Label, IP: row.Ip,
		CreatedAt:    time.Unix(row.CreatedAt, 0).UTC(),
		LastActiveAt: time.Unix(row.LastActiveAt, 0).UTC(),
		ExpiresAt:    time.Unix(row.ExpiresAt, 0).UTC(),
	}
}
