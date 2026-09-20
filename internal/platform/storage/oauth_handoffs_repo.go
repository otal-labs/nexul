package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/otal-labs/nexul/internal/auth"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/storage/sqlcgen"
)

var _ auth.OAuthHandoffStore = (*OAuthHandoffsRepo)(nil)

type OAuthHandoffsRepo struct {
	db *sql.DB
	w  *Serializer
	q  *sqlcgen.Queries
}

func (r *OAuthHandoffsRepo) StartOAuthHandoff(ctx context.Context, handoff *auth.OAuthHandoff) error {
	if handoff == nil || handoff.ID == "" || handoff.InvitationID == "" || handoff.Provider == "" || !validTokenHash(handoff.OAuthStateHash) {
		return fmt.Errorf("%w: OAuth handoff fields are required", apperrs.ErrInvalid)
	}
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		err := r.q.WithTx(tx).CreateOAuthHandoff(ctx, sqlcgen.CreateOAuthHandoffParams{
			ID: handoff.ID, InvitationID: handoff.InvitationID,
			OauthStateHash: sql.NullString{String: handoff.OAuthStateHash, Valid: true},
			Provider:       string(handoff.Provider), CreatedAt: handoff.CreatedAt.Unix(), ExpiresAt: handoff.ExpiresAt.Unix(),
		})
		if err != nil {
			return fmt.Errorf("create OAuth handoff %s: %w", handoff.ID, classifyWriteErr(err))
		}
		return nil
	})
}

func (r *OAuthHandoffsRepo) CompleteOAuthCallback(ctx context.Context, oauthStateHash, acceptanceHash string, identity auth.OAuthHandoffIdentity, expiresAt time.Time) (*auth.OAuthHandoff, error) {
	if !validTokenHash(oauthStateHash) || !validTokenHash(acceptanceHash) || identity.Provider == "" || identity.ProviderUserID == "" {
		return nil, fmt.Errorf("%w: OAuth handoff completion is invalid", apperrs.ErrInvalid)
	}
	var handoff *auth.OAuthHandoff
	err := r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		q := r.q.WithTx(tx)
		row, err := q.GetOAuthHandoffByState(ctx, sql.NullString{String: oauthStateHash, Valid: true})
		if errors.Is(err, sql.ErrNoRows) {
			return apperrs.ErrNotFound
		}
		if err != nil {
			return fmt.Errorf("get OAuth handoff by state: %w", err)
		}
		if row.Provider != string(identity.Provider) || row.ExpiresAt <= time.Now().Unix() {
			return apperrs.ErrNotFound
		}
		changed, err := q.CompleteOAuthHandoffTransition(ctx, sqlcgen.CompleteOAuthHandoffTransitionParams{
			AcceptanceHash: sql.NullString{String: acceptanceHash, Valid: true},
			ProviderUserID: sql.NullString{String: identity.ProviderUserID, Valid: true},
			Login:          sql.NullString{String: identity.Login, Valid: identity.Login != ""},
			Name:           sql.NullString{String: identity.Name, Valid: identity.Name != ""},
			AvatarUrl:      sql.NullString{String: identity.AvatarURL, Valid: identity.AvatarURL != ""},
			ExistingUserID: sql.NullString{String: identity.ExistingUserID, Valid: identity.ExistingUserID != ""},
			ExpiresAt:      expiresAt.Unix(),
			OauthStateHash: sql.NullString{String: oauthStateHash, Valid: true},
			ExpiresAt_2:    time.Now().Unix(),
		})
		if err != nil {
			return fmt.Errorf("complete OAuth handoff: %w", err)
		}
		if changed != 1 {
			return apperrs.ErrNotFound
		}
		row.OauthStateHash = sql.NullString{}
		row.AcceptanceHash = sql.NullString{String: acceptanceHash, Valid: true}
		row.ProviderUserID = sql.NullString{String: identity.ProviderUserID, Valid: true}
		row.Login = sql.NullString{String: identity.Login, Valid: identity.Login != ""}
		row.Name = sql.NullString{String: identity.Name, Valid: identity.Name != ""}
		row.AvatarUrl = sql.NullString{String: identity.AvatarURL, Valid: identity.AvatarURL != ""}
		row.ExistingUserID = sql.NullString{String: identity.ExistingUserID, Valid: identity.ExistingUserID != ""}
		row.ExpiresAt = expiresAt.Unix()
		handoff = toOAuthHandoff(row)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return handoff, nil
}

func (r *OAuthHandoffsRepo) GetOAuthHandoffByAcceptanceHash(ctx context.Context, acceptanceHash string, now time.Time) (*auth.OAuthHandoff, error) {
	if !validTokenHash(acceptanceHash) {
		return nil, apperrs.ErrNotFound
	}
	row, err := r.q.GetOAuthHandoffByAcceptanceHash(ctx, sql.NullString{String: acceptanceHash, Valid: true})
	if errors.Is(err, sql.ErrNoRows) {
		return nil, apperrs.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get OAuth handoff by acceptance hash: %w", err)
	}
	if row.ExpiresAt <= now.Unix() {
		return nil, apperrs.ErrNotFound
	}
	return toOAuthHandoff(row), nil
}

func (r *OAuthHandoffsRepo) CompleteOAuthRedemption(ctx context.Context, acceptanceHash, admittedUserID string, now time.Time) error {
	if !validTokenHash(acceptanceHash) || admittedUserID == "" {
		return apperrs.ErrNotFound
	}
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		q := r.q.WithTx(tx)
		row, err := q.GetOAuthHandoffByAcceptanceHash(ctx, sql.NullString{String: acceptanceHash, Valid: true})
		if errors.Is(err, sql.ErrNoRows) {
			return apperrs.ErrNotFound
		}
		if err != nil {
			return fmt.Errorf("get OAuth handoff for completion: %w", err)
		}
		if row.ExpiresAt <= now.Unix() {
			return apperrs.ErrNotFound
		}
		if row.CompletedAt.Valid {
			if row.AdmittedUserID.Valid && row.AdmittedUserID.String == admittedUserID {
				return nil
			}
			return apperrs.ErrNotFound
		}
		changed, err := q.CompleteOAuthHandoffRedemption(ctx, sqlcgen.CompleteOAuthHandoffRedemptionParams{
			AdmittedUserID: sql.NullString{String: admittedUserID, Valid: true},
			CompletedAt:    sql.NullInt64{Int64: now.Unix(), Valid: true},
			AcceptanceHash: sql.NullString{String: acceptanceHash, Valid: true},
			ExpiresAt:      now.Unix(),
		})
		if err != nil {
			return fmt.Errorf("complete OAuth handoff redemption: %w", err)
		}
		if changed != 1 {
			return apperrs.ErrNotFound
		}
		return nil
	})
}

func toOAuthHandoff(row sqlcgen.InvitationOauthHandoff) *auth.OAuthHandoff {
	handoff := &auth.OAuthHandoff{
		ID: row.ID, InvitationID: row.InvitationID, Provider: auth.Provider(row.Provider),
		CreatedAt: time.Unix(row.CreatedAt, 0).UTC(), ExpiresAt: time.Unix(row.ExpiresAt, 0).UTC(),
	}
	if row.OauthStateHash.Valid {
		handoff.OAuthStateHash = row.OauthStateHash.String
	}
	if row.AcceptanceHash.Valid {
		handoff.AcceptanceHash = row.AcceptanceHash.String
	}
	if row.ProviderUserID.Valid {
		handoff.ProviderUserID = row.ProviderUserID.String
	}
	if row.Login.Valid {
		handoff.Login = row.Login.String
	}
	if row.Name.Valid {
		handoff.Name = row.Name.String
	}
	if row.AvatarUrl.Valid {
		handoff.AvatarURL = row.AvatarUrl.String
	}
	if row.ExistingUserID.Valid {
		handoff.ExistingUserID = row.ExistingUserID.String
	}
	if row.AdmittedUserID.Valid {
		handoff.AdmittedUserID = row.AdmittedUserID.String
	}
	if row.CompletedAt.Valid {
		completedAt := time.Unix(row.CompletedAt.Int64, 0).UTC()
		handoff.CompletedAt = &completedAt
	}
	return handoff
}
