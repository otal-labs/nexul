package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/otal-labs/nexul/ee/sso"
	"github.com/otal-labs/nexul/internal/auth"
	"github.com/otal-labs/nexul/internal/platform/crypto"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/ids"
	"github.com/otal-labs/nexul/internal/platform/storage/sqlcgen"
	"github.com/otal-labs/nexul/internal/tenancy"
)

var (
	_ sso.Repo     = (*SSORepo)(nil)
	_ sso.Accounts = (*SSORepo)(nil)
)

// isSSOProvider reports whether a sign-in identity was made by single sign-on, OIDC or SAML, the providers stored by
// their protocol's name.
func isSSOProvider(provider string) bool {
	return provider == sso.ProtocolOIDC || provider == sso.ProtocolSAML
}

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
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		q := r.q.WithTx(tx)
		if err := q.SetSSOSettings(ctx, params); err != nil {
			return fmt.Errorf("set sso settings: %w", err)
		}
		revoked, err := endSessionsOutsideSSO(ctx, q, keepUsers)
		if err != nil {
			return err
		}
		return enqueueWorkspaceOutbox(ctx, tx, append(evts, revoked...))
	})
}

// endSessionsOutsideSSO deletes every session not started through single sign-on outside keepUsers' and returns a
// session.revoked event for each.
func endSessionsOutsideSSO(ctx context.Context, q *sqlcgen.Queries, keepUsers []string) ([]eventbus.OutboxEvent, error) {
	rows, err := q.ListSessionsOutsideSSO(ctx)
	if err != nil {
		return nil, fmt.Errorf("list sessions outside sso: %w", err)
	}
	ended := []string{}
	evts := []eventbus.OutboxEvent{}
	for _, row := range rows {
		if slices.Contains(keepUsers, row.UserID) {
			continue
		}
		ended = append(ended, row.ID)
		evts = append(evts, sessionRevokedEvent(row))
	}
	if len(ended) == 0 {
		return nil, nil
	}
	if err := q.DeleteSessionsByID(ctx, ended); err != nil {
		return nil, fmt.Errorf("end sessions outside sso: %w", err)
	}
	return evts, nil
}

func sessionRevokedEvent(row sqlcgen.Session) eventbus.OutboxEvent {
	return eventbus.OutboxEvent{ID: ids.New(), Topic: auth.TopicSessionRevoked, Payload: auth.SessionChangedEvent{
		SessionID: row.ID, UserID: row.UserID, Client: auth.SessionClient(row.Client), Platform: row.Platform, Label: row.Label,
	}}
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

// Delete removes the settings and every grant, with its events in the same transaction.
func (r *SSORepo) Delete(ctx context.Context, evts ...eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		q := r.q.WithTx(tx)
		if err := q.DeleteSSOSettings(ctx); err != nil {
			return fmt.Errorf("delete sso settings: %w", err)
		}
		if err := q.DeleteSSOGrants(ctx); err != nil {
			return fmt.Errorf("delete sso grants: %w", err)
		}
		return enqueueWorkspaceOutbox(ctx, tx, evts)
	})
}

// BySubject finds the account a provider's issuer-scoped subject is linked to.
func (r *SSORepo) BySubject(ctx context.Context, provider, key string) (sso.Account, error) {
	row, err := r.q.GetUserByIdentity(ctx, sqlcgen.GetUserByIdentityParams{Provider: provider, ProviderUserID: key})
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
	return &auth.Identity{UserID: userID, Provider: auth.Provider(id.Provider), ProviderUserID: id.Key, Login: id.Email, Name: id.Name, AvatarURL: id.AvatarURL}
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

// SaveGrant stores the person's refresh token sealed, replacing the one an earlier sign-in left.
func (r *SSORepo) SaveGrant(ctx context.Context, userID, refreshToken string, at time.Time) error {
	sealed, err := crypto.Encrypt(r.encKey, []byte(refreshToken))
	if err != nil {
		return fmt.Errorf("encrypt sso refresh token: %w", err)
	}
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		if err := r.q.WithTx(tx).UpsertSSOGrant(ctx, sqlcgen.UpsertSSOGrantParams{UserID: userID, RefreshToken: []byte(sealed), CheckedAt: at.Unix()}); err != nil {
			return fmt.Errorf("save sso grant: %w", err)
		}
		return nil
	})
}

// StaleGrants returns up to limit grants the provider has not confirmed since before, oldest first; a grant whose sealed
// token no longer opens (the key changed) comes back with an empty RefreshToken, so one bad row never blocks the rest.
func (r *SSORepo) StaleGrants(ctx context.Context, before time.Time, limit int) ([]sso.StoredGrant, error) {
	rows, err := r.q.ListStaleSSOGrants(ctx, sqlcgen.ListStaleSSOGrantsParams{CheckedAt: before.Unix(), Limit: int64(limit)})
	if err != nil {
		return nil, fmt.Errorf("list stale sso grants: %w", err)
	}
	out := make([]sso.StoredGrant, 0, len(rows))
	for _, row := range rows {
		token, err := crypto.Decrypt(r.encKey, string(row.RefreshToken))
		if err != nil {
			token = nil
		}
		out = append(out, sso.StoredGrant{UserID: row.UserID, RefreshToken: string(token)})
	}
	return out, nil
}

// ConfirmGrant records that the provider still honours the grant at at, keeping the refresh token unless a new one came back.
func (r *SSORepo) ConfirmGrant(ctx context.Context, userID, newRefreshToken string, at time.Time) error {
	params := sqlcgen.TouchSSOGrantParams{UserID: userID, CheckedAt: at.Unix()}
	if newRefreshToken != "" {
		sealed, err := crypto.Encrypt(r.encKey, []byte(newRefreshToken))
		if err != nil {
			return fmt.Errorf("encrypt sso refresh token: %w", err)
		}
		params.RefreshToken = []byte(sealed)
	}
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		if err := r.q.WithTx(tx).TouchSSOGrant(ctx, params); err != nil {
			return fmt.Errorf("confirm sso grant: %w", err)
		}
		return nil
	})
}

// DropGrant deletes the person's grant and leaves their sessions alone.
func (r *SSORepo) DropGrant(ctx context.Context, userID string) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		if err := r.q.WithTx(tx).DeleteSSOGrant(ctx, userID); err != nil {
			return fmt.Errorf("drop sso grant: %w", err)
		}
		return nil
	})
}

// RefuseGrant deletes the person's grant and ends every session of theirs started through single sign-on, each with its
// session.revoked event, in one transaction; it returns how many sessions ended.
func (r *SSORepo) RefuseGrant(ctx context.Context, userID string) (int, error) {
	ended := 0
	err := r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		q := r.q.WithTx(tx)
		rows, err := q.ListSSOSessionsByUser(ctx, userID)
		if err != nil {
			return fmt.Errorf("list sso sessions: %w", err)
		}
		if err := q.DeleteSSOGrant(ctx, userID); err != nil {
			return fmt.Errorf("delete sso grant: %w", err)
		}
		if len(rows) == 0 {
			return nil
		}
		sessionIDs := make([]string, 0, len(rows))
		evts := make([]eventbus.OutboxEvent, 0, len(rows))
		for _, row := range rows {
			sessionIDs = append(sessionIDs, row.ID)
			evts = append(evts, sessionRevokedEvent(row))
		}
		if err := q.DeleteSessionsByID(ctx, sessionIDs); err != nil {
			return fmt.Errorf("end sso sessions: %w", err)
		}
		ended = len(rows)
		return enqueueWorkspaceOutbox(ctx, tx, evts)
	})
	return ended, err
}

// MoveSignIn moves the single sign-on identity and grant of fromID, whose only sign-in it is, onto toID, then disables
// fromID and ends its sessions, all in one transaction; what fromID made stays with it.
func (r *SSORepo) MoveSignIn(ctx context.Context, fromID, toID, actorID string) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		q := r.q.WithTx(tx)
		from, moved, err := checkMove(ctx, q, fromID, toID)
		if err != nil {
			return err
		}
		if _, err := q.MoveSSOIdentity(ctx, sqlcgen.MoveSSOIdentityParams{ToUserID: toID, FromUserID: fromID}); err != nil {
			return fmt.Errorf("move sso identity: %w", err)
		}
		if err := q.DeleteSSOGrant(ctx, toID); err != nil {
			return fmt.Errorf("clear sso grant of %s: %w", toID, err)
		}
		if err := q.MoveSSOGrant(ctx, sqlcgen.MoveSSOGrantParams{ToUserID: toID, FromUserID: fromID}); err != nil {
			return fmt.Errorf("move sso grant: %w", err)
		}
		evts := []eventbus.OutboxEvent{
			{ID: ids.New(), Topic: auth.TopicIdentityLinked, Payload: auth.IdentityChangedEvent{UserID: toID, Provider: auth.Provider(moved.Provider), Login: moved.Login}},
		}
		if from.AccountStatus == string(auth.AccountActive) {
			if _, err := q.SetAccountStatus(ctx, sqlcgen.SetAccountStatusParams{AccountStatus: string(auth.AccountDisabled), UpdatedAt: time.Now().Unix(), ID: fromID}); err != nil {
				return fmt.Errorf("disable account %s: %w", fromID, err)
			}
			evts = append(evts, eventbus.OutboxEvent{ID: ids.New(), Topic: auth.TopicAccountDisabled, Payload: auth.AccountLifecycleEvent{AccountID: fromID, ActorID: actorID}})
		}
		revoked, err := endSessionsOf(ctx, q, fromID)
		if err != nil {
			return err
		}
		return enqueueWorkspaceOutbox(ctx, tx, append(evts, revoked...))
	})
}

// checkMove refuses a move that would leave someone without a way in or give an account a second single sign-on, and
// returns the account moved from with its single sign-on identity.
func checkMove(ctx context.Context, q *sqlcgen.Queries, fromID, toID string) (sqlcgen.User, sqlcgen.UserIdentity, error) {
	from, err := q.GetUserByID(ctx, fromID)
	if err != nil {
		return sqlcgen.User{}, sqlcgen.UserIdentity{}, fmt.Errorf("get account %s: %w", fromID, notFoundIfNoRows(err))
	}
	to, err := q.GetUserByID(ctx, toID)
	if err != nil {
		return sqlcgen.User{}, sqlcgen.UserIdentity{}, fmt.Errorf("get account %s: %w", toID, notFoundIfNoRows(err))
	}
	if to.AccountStatus == string(auth.AccountRemoved) {
		return sqlcgen.User{}, sqlcgen.UserIdentity{}, fmt.Errorf("%w: the account to move single sign-on to was removed; restore it first", apperrs.ErrInvalid)
	}
	moved, err := onlySSOIdentity(ctx, q, fromID)
	if err != nil {
		return sqlcgen.User{}, sqlcgen.UserIdentity{}, err
	}
	if err := hasNoSSO(ctx, q, toID); err != nil {
		return sqlcgen.User{}, sqlcgen.UserIdentity{}, err
	}
	if err := protectLastActiveOwner(ctx, q, from, auth.AccountDisabled); err != nil {
		return sqlcgen.User{}, sqlcgen.UserIdentity{}, err
	}
	return from, moved, nil
}

// endSessionsOf deletes every session of the user and returns a session.revoked event for each.
func endSessionsOf(ctx context.Context, q *sqlcgen.Queries, userID string) ([]eventbus.OutboxEvent, error) {
	sessions, err := q.ListSessionsByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list sessions of %s: %w", userID, err)
	}
	if len(sessions) == 0 {
		return nil, nil
	}
	ended := make([]string, 0, len(sessions))
	evts := make([]eventbus.OutboxEvent, 0, len(sessions))
	for _, row := range sessions {
		ended = append(ended, row.ID)
		evts = append(evts, sessionRevokedEvent(row))
	}
	if err := q.DeleteSessionsByID(ctx, ended); err != nil {
		return nil, fmt.Errorf("end sessions of %s: %w", userID, err)
	}
	return evts, nil
}

// onlySSOIdentity is the account's single sign-on identity, refused unless that is its one sign-in.
func onlySSOIdentity(ctx context.Context, q *sqlcgen.Queries, userID string) (sqlcgen.UserIdentity, error) {
	identities, err := q.ListIdentitiesByUser(ctx, userID)
	if err != nil {
		return sqlcgen.UserIdentity{}, fmt.Errorf("list identities of %s: %w", userID, err)
	}
	if len(identities) != 1 || !isSSOProvider(identities[0].Provider) {
		return sqlcgen.UserIdentity{}, fmt.Errorf("%w: single sign-on moves only from an account whose one sign-in it is", apperrs.ErrInvalid)
	}
	return identities[0], nil
}

func hasNoSSO(ctx context.Context, q *sqlcgen.Queries, userID string) error {
	identities, err := q.ListIdentitiesByUser(ctx, userID)
	if err != nil {
		return fmt.Errorf("list identities of %s: %w", userID, err)
	}
	for _, id := range identities {
		if isSSOProvider(id.Provider) {
			return fmt.Errorf("%w: that account already signs in through single sign-on", apperrs.ErrConflict)
		}
	}
	return nil
}
