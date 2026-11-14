package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/otal-labs/nexul/ee/sso"
	"github.com/otal-labs/nexul/internal/auth"
	"github.com/otal-labs/nexul/internal/platform/crypto"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/ids"
	"github.com/otal-labs/nexul/internal/platform/storage/sqlcgen"
	"github.com/otal-labs/nexul/internal/tenancy"
)

var (
	_ sso.Repo     = (*SSORepo)(nil)
	_ sso.Accounts = (*SSORepo)(nil)
)

// ssoProvider is the provider name an OIDC sign-in identity is stored under.
const ssoProvider = "oidc"

// SSORepo stores the instance's one single sign-on provider, its client secret sealed, and the accounts it signs in.
type SSORepo struct {
	db     *sql.DB
	w      *Serializer
	q      *sqlcgen.Queries
	encKey []byte
}

// Get returns the zero Settings when the instance has none.
func (r *SSORepo) Get(ctx context.Context) (sso.Settings, error) {
	row, err := r.q.GetSSOSettings(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return sso.Settings{}, nil
	}
	if err != nil {
		return sso.Settings{}, fmt.Errorf("get sso settings: %w", err)
	}
	secret, err := crypto.Decrypt(r.encKey, row.ClientSecret)
	if err != nil {
		return sso.Settings{}, fmt.Errorf("decrypt sso client secret: %w", err)
	}
	return sso.Settings{
		IssuerURL: row.IssuerUrl, ClientID: row.ClientID, ClientSecret: string(secret), DisplayName: row.DisplayName,
		JoinRoleID: row.JoinRoleID, Only: row.SsoOnly != 0, UpdatedBy: row.UpdatedBy, UpdatedAt: time.Unix(row.UpdatedAt, 0).UTC(),
	}, nil
}

// Set replaces the settings, with its events in the same transaction.
func (r *SSORepo) Set(ctx context.Context, s sso.Settings, evts ...eventbus.OutboxEvent) error {
	params, err := r.settingsParams(s)
	if err != nil {
		return err
	}
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		if err := r.q.WithTx(tx).SetSSOSettings(ctx, params); err != nil {
			return fmt.Errorf("set sso settings: %w", err)
		}
		return enqueueWorkspaceOutbox(ctx, tx, evts)
	})
}

// TurnOnOnly saves the settings and deletes every session not started through single sign-on outside keepUsers', with a
// session.revoked event per session, in one transaction.
func (r *SSORepo) TurnOnOnly(ctx context.Context, s sso.Settings, keepUsers []string, evts ...eventbus.OutboxEvent) error {
	params, err := r.settingsParams(s)
	if err != nil {
		return err
	}
	keep := make(map[string]bool, len(keepUsers))
	for _, id := range keepUsers {
		keep[id] = true
	}
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		q := r.q.WithTx(tx)
		if err := q.SetSSOSettings(ctx, params); err != nil {
			return fmt.Errorf("set sso settings: %w", err)
		}
		rows, err := q.ListSessionsOutsideSSO(ctx)
		if err != nil {
			return fmt.Errorf("list sessions outside sso: %w", err)
		}
		ended := []string{}
		for _, row := range rows {
			if keep[row.UserID] {
				continue
			}
			ended = append(ended, row.ID)
			evts = append(evts, eventbus.OutboxEvent{ID: ids.New(), Topic: auth.TopicSessionRevoked, Payload: auth.SessionChangedEvent{
				SessionID: row.ID, UserID: row.UserID, Client: auth.SessionClient(row.Client), Platform: row.Platform, Label: row.Label,
			}})
		}
		if len(ended) > 0 {
			if err := q.DeleteSessionsByID(ctx, ended); err != nil {
				return fmt.Errorf("end sessions outside sso: %w", err)
			}
		}
		return enqueueWorkspaceOutbox(ctx, tx, evts)
	})
}

func (r *SSORepo) settingsParams(s sso.Settings) (sqlcgen.SetSSOSettingsParams, error) {
	sealed, err := crypto.Encrypt(r.encKey, []byte(s.ClientSecret))
	if err != nil {
		return sqlcgen.SetSSOSettingsParams{}, fmt.Errorf("encrypt sso client secret: %w", err)
	}
	return sqlcgen.SetSSOSettingsParams{
		IssuerUrl: s.IssuerURL, ClientID: s.ClientID, ClientSecret: sealed, DisplayName: s.DisplayName,
		JoinRoleID: s.JoinRoleID, SsoOnly: boolToInt(s.Only), UpdatedBy: s.UpdatedBy, UpdatedAt: s.UpdatedAt.Unix(),
	}, nil
}

// Delete removes the settings, with its events in the same transaction.
func (r *SSORepo) Delete(ctx context.Context, evts ...eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		if err := r.q.WithTx(tx).DeleteSSOSettings(ctx); err != nil {
			return fmt.Errorf("delete sso settings: %w", err)
		}
		return enqueueWorkspaceOutbox(ctx, tx, evts)
	})
}

// BySubject finds the account an issuer-scoped subject is linked to.
func (r *SSORepo) BySubject(ctx context.Context, key string) (sso.Account, error) {
	row, err := r.q.GetUserByIdentity(ctx, sqlcgen.GetUserByIdentityParams{Provider: ssoProvider, ProviderUserID: key})
	if err != nil {
		return sso.Account{}, fmt.Errorf("get user by sso subject: %w", notFoundIfNoRows(err))
	}
	return sso.Account{UserID: row.ID, Status: row.AccountStatus}, nil
}

// ByEmail finds the account whose Google or Discord sign-in, a verified email, is that address.
func (r *SSORepo) ByEmail(ctx context.Context, email string) (sso.Account, error) {
	row, err := r.q.FindUserByEmailIdentity(ctx, email)
	if err != nil {
		return sso.Account{}, fmt.Errorf("get user by email: %w", notFoundIfNoRows(err))
	}
	return sso.Account{UserID: row.ID, Status: row.AccountStatus}, nil
}

func ssoIdentity(userID string, id sso.Identity) *auth.Identity {
	return &auth.Identity{UserID: userID, Provider: ssoProvider, ProviderUserID: id.Key, Login: id.Email, Name: id.Name, AvatarURL: id.AvatarURL}
}

// Link attaches the subject to the account as an identity, like Profile's link does for the other providers.
func (r *SSORepo) Link(ctx context.Context, userID string, id sso.Identity) error {
	identity := ssoIdentity(userID, id)
	evt := eventbus.OutboxEvent{ID: ids.New(), Topic: auth.TopicIdentityLinked, Payload: auth.IdentityChangedEvent{UserID: userID, Provider: identity.Provider, Login: identity.Login}}
	return (&UsersRepo{db: r.db, w: r.w, q: r.q}).LinkIdentity(ctx, identity, evt)
}

// Create makes the account and joins it to the default workspace with roleID, in one transaction, with the events an
// invitation's redemption publishes.
func (r *SSORepo) Create(ctx context.Context, userID string, id sso.Identity, roleID string, at time.Time) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		q := r.q.WithTx(tx)
		if err := insertUserWithIdentity(ctx, q, ssoIdentity(userID, id), at); err != nil {
			return fmt.Errorf("admit sso user: %w", err)
		}
		if err := q.AddInvitationMember(ctx, sqlcgen.AddInvitationMemberParams{UserID: userID, WorkspaceID: tenancy.DefaultWorkspaceID, CreatedAt: at.Unix()}); err != nil {
			return fmt.Errorf("add sso membership: %w", classifyWriteErr(err))
		}
		if err := addMemberRole(ctx, q, tenancy.DefaultWorkspaceID, userID, roleID, at); err != nil {
			return err
		}
		return enqueueWorkspaceOutbox(ctx, tx, []eventbus.OutboxEvent{
			{ID: ids.New(), Topic: auth.TopicAccountAdmitted, Payload: auth.AccountLifecycleEvent{AccountID: userID}},
			{ID: ids.New(), Topic: tenancy.TopicWorkspaceMemberAdded, Payload: tenancy.MemberEvent{UserID: userID, WorkspaceID: tenancy.DefaultWorkspaceID, RoleIDsAdded: []string{roleID}}},
		})
	})
}

// CanJoinAs reports whether roleID is a role of the default workspace that is not its Owner role.
func (r *SSORepo) CanJoinAs(ctx context.Context, roleID string) (bool, error) {
	isOwner, err := r.q.GetSSOJoinRole(ctx, sqlcgen.GetSSOJoinRoleParams{ID: roleID, WorkspaceID: tenancy.DefaultWorkspaceID})
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("get join role: %w", err)
	}
	return isOwner == 0, nil
}
