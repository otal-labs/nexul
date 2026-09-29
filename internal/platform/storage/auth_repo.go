package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/otal-labs/nexul/internal/auth"
	"github.com/otal-labs/nexul/internal/platform/crypto"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/storage/sqlcgen"
)

var (
	_ auth.UserStore      = (*UsersRepo)(nil)
	_ auth.AllowlistStore = (*AllowlistRepo)(nil)
	_ auth.SettingsStore  = (*SettingsRepo)(nil)
)

type UsersRepo struct {
	db *sql.DB
	w  *Serializer
	q  *sqlcgen.Queries
}

// UpsertUser syncs a known identity, and the user's profile when that identity is their first, else creates the user.
func (r *UsersRepo) UpsertUser(ctx context.Context, id *auth.Identity) (*auth.User, bool, error) {
	created := false
	var persisted *auth.User
	err := r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		q := r.q.WithTx(tx)
		existing, err := q.GetUserByIdentity(ctx, sqlcgen.GetUserByIdentityParams{
			Provider: string(id.Provider), ProviderUserID: id.ProviderUserID,
		})
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("query user: %w", err)
		}
		now := time.Now()
		if err != nil {
			created = true
			if err := insertUserWithIdentity(ctx, q, id, now); err != nil {
				return err
			}
			persisted, err = r.getByProvider(ctx, q, id.Provider, id.ProviderUserID)
			return err
		}
		if err := q.SyncIdentity(ctx, sqlcgen.SyncIdentityParams{
			Login: id.Login, Name: id.Name, AvatarUrl: id.AvatarURL, Provider: string(id.Provider), ProviderUserID: id.ProviderUserID,
		}); err != nil {
			return fmt.Errorf("sync identity: %w", err)
		}
		if err := q.SyncUserFromFirstIdentity(ctx, sqlcgen.SyncUserFromFirstIdentityParams{
			Login: id.Login, Name: id.Name, AvatarUrl: id.AvatarURL, UpdatedAt: now.Unix(), UserID: existing.ID, Provider: string(id.Provider),
		}); err != nil {
			return fmt.Errorf("update user: %w", err)
		}
		persisted, err = r.getByProvider(ctx, q, id.Provider, id.ProviderUserID)
		return err
	})
	if err != nil {
		return nil, false, err
	}
	return persisted, created, nil
}

// CreateFirstUser admits the first identity while holding the shared write serializer.
func (r *UsersRepo) CreateFirstUser(ctx context.Context, id *auth.Identity, events ...eventbus.OutboxEvent) (*auth.User, error) {
	var persisted *auth.User
	err := r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		q := r.q.WithTx(tx)
		count, err := q.CountUsers(ctx)
		if err != nil {
			return fmt.Errorf("count users: %w", err)
		}
		if count != 0 {
			return apperrs.ErrConflict
		}
		if err := insertUserWithIdentity(ctx, q, id, time.Now()); err != nil {
			return err
		}
		if err := insertOutboxRows(ctx, tx, events); err != nil {
			return fmt.Errorf("write first-user event: %w", err)
		}
		persisted, err = r.getByProvider(ctx, q, id.Provider, id.ProviderUserID)
		return err
	})
	if err != nil {
		return nil, err
	}
	return persisted, nil
}

// insertUserWithIdentity creates the user row from the identity's profile and the identity row that resolves it.
func insertUserWithIdentity(ctx context.Context, q *sqlcgen.Queries, id *auth.Identity, now time.Time) error {
	if err := q.InsertUser(ctx, sqlcgen.InsertUserParams{
		ID: id.UserID, Login: id.Login, Name: id.Name, AvatarUrl: id.AvatarURL, CreatedAt: now.Unix(), UpdatedAt: now.Unix(),
	}); err != nil {
		return fmt.Errorf("insert user: %w", classifyWriteErr(err))
	}
	return insertIdentity(ctx, q, id, now)
}

func insertIdentity(ctx context.Context, q *sqlcgen.Queries, id *auth.Identity, now time.Time) error {
	if err := q.InsertIdentity(ctx, sqlcgen.InsertIdentityParams{
		UserID: id.UserID, Provider: string(id.Provider), ProviderUserID: id.ProviderUserID,
		Login: id.Login, Name: id.Name, AvatarUrl: id.AvatarURL, CreatedAt: now.Unix(),
	}); err != nil {
		return fmt.Errorf("insert identity: %w", classifyWriteErr(err))
	}
	return nil
}

func (r *UsersRepo) ListIdentities(ctx context.Context, userID string) ([]auth.Identity, error) {
	rows, err := r.q.ListIdentitiesByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list identities of %s: %w", userID, err)
	}
	out := make([]auth.Identity, 0, len(rows))
	for _, row := range rows {
		out = append(out, auth.Identity{
			UserID: row.UserID, Provider: auth.Provider(row.Provider), ProviderUserID: row.ProviderUserID,
			Login: row.Login, Name: row.Name, AvatarURL: row.AvatarUrl, CreatedAt: time.Unix(row.CreatedAt, 0).UTC(),
		})
	}
	return out, nil
}

// LinkIdentity looks up the owner first so each conflict gets its own message; the PRIMARY KEY still guards the race.
func (r *UsersRepo) LinkIdentity(ctx context.Context, id *auth.Identity, events ...eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		q := r.q.WithTx(tx)
		owner, err := q.GetUserByIdentity(ctx, sqlcgen.GetUserByIdentityParams{Provider: string(id.Provider), ProviderUserID: id.ProviderUserID})
		if err == nil && owner.ID == id.UserID {
			return fmt.Errorf("%w: that %s account is already linked to you", apperrs.ErrConflict, id.Provider)
		}
		if err == nil {
			return fmt.Errorf("%w: that %s account is already linked to another user", apperrs.ErrConflict, id.Provider)
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("query identity: %w", err)
		}
		if err := insertIdentity(ctx, q, id, time.Now()); err != nil {
			if errors.Is(err, apperrs.ErrConflict) {
				return fmt.Errorf("%w: a %s account is already linked; unlink it first", apperrs.ErrConflict, id.Provider)
			}
			return err
		}
		return insertOutboxRows(ctx, tx, events)
	})
}

func (r *UsersRepo) UnlinkIdentity(ctx context.Context, userID string, provider auth.Provider, events ...eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		q := r.q.WithTx(tx)
		n, err := q.CountIdentitiesByUser(ctx, userID)
		if err != nil {
			return fmt.Errorf("count identities of %s: %w", userID, err)
		}
		if n <= 1 {
			return fmt.Errorf("%w: this is your only sign-in", apperrs.ErrConflict)
		}
		deleted, err := q.DeleteIdentity(ctx, sqlcgen.DeleteIdentityParams{UserID: userID, Provider: string(provider)})
		if err != nil {
			return fmt.Errorf("delete %s identity of %s: %w", provider, userID, err)
		}
		if deleted == 0 {
			return fmt.Errorf("%w: no %s account is linked", apperrs.ErrNotFound, provider)
		}
		return insertOutboxRows(ctx, tx, events)
	})
}

func (r *UsersRepo) GetUserByID(ctx context.Context, id string) (*auth.User, error) {
	row, err := r.q.GetUserByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get user %s: %w", id, notFoundIfNoRows(err))
	}
	return toUser(row), nil
}

func (r *UsersRepo) GetUserByProvider(ctx context.Context, provider auth.Provider, providerUserID string) (*auth.User, error) {
	return r.getByProvider(ctx, r.q, provider, providerUserID)
}

func (r *UsersRepo) CanCreateWorkspaceExists(ctx context.Context) (bool, error) {
	n, err := r.q.CountCanCreateWorkspace(ctx)
	if err != nil {
		return false, fmt.Errorf("count instance admins: %w", err)
	}
	return n > 0, nil
}

func (r *UsersRepo) SetCanCreateWorkspace(ctx context.Context, id string, can bool) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		n, err := r.q.WithTx(tx).SetCanCreateWorkspace(ctx, sqlcgen.SetCanCreateWorkspaceParams{
			CanCreateWorkspace: int64(boolInt(can)), UpdatedAt: time.Now().Unix(), ID: id,
		})
		if err != nil {
			return fmt.Errorf("set can_create_workspace %s: %w", id, err)
		}
		if n == 0 {
			return fmt.Errorf("set can_create_workspace %s: %w", id, apperrs.ErrNotFound)
		}
		return nil
	})
}

func (r *UsersRepo) SetAccountStatus(ctx context.Context, id string, status auth.AccountStatus, events ...eventbus.OutboxEvent) error {
	if !validAccountStatus(status) {
		return fmt.Errorf("%w: invalid account status %q", apperrs.ErrInvalid, status)
	}
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		q := r.q.WithTx(tx)
		user, err := q.GetUserByID(ctx, id)
		if err != nil {
			return fmt.Errorf("get account %s: %w", id, notFoundIfNoRows(err))
		}
		if err := protectLastActiveAdmin(ctx, q, user, status); err != nil {
			return err
		}
		now := time.Now().Unix()
		if status == auth.AccountRemoved {
			if err := removeAccountAccess(ctx, q, id, now); err != nil {
				return err
			}
		}
		n, err := q.SetAccountStatus(ctx, sqlcgen.SetAccountStatusParams{
			AccountStatus: string(status), UpdatedAt: now, ID: id,
		})
		if err != nil {
			return fmt.Errorf("set account status %s: %w", id, err)
		}
		if n == 0 {
			return fmt.Errorf("set account status %s: %w", id, apperrs.ErrNotFound)
		}
		return enqueueAccountEvents(ctx, tx, id, events)
	})
}

func validAccountStatus(status auth.AccountStatus) bool {
	return status == auth.AccountActive || status == auth.AccountDisabled || status == auth.AccountRemoved
}

func protectLastActiveAdmin(ctx context.Context, q *sqlcgen.Queries, user sqlcgen.User, status auth.AccountStatus) error {
	if user.CanCreateWorkspace == 0 || user.AccountStatus != string(auth.AccountActive) || status == auth.AccountActive {
		return nil
	}
	activeAdmins, err := q.CountActiveAdmins(ctx)
	if err != nil {
		return fmt.Errorf("count active instance admins: %w", err)
	}
	if activeAdmins <= 1 {
		return fmt.Errorf("%w: cannot change the last active instance admin", apperrs.ErrConflict)
	}
	return nil
}

func removeAccountAccess(ctx context.Context, q *sqlcgen.Queries, id string, now int64) error {
	steps := []struct {
		name string
		run  func() error
	}{
		{"memberships", func() error { return q.DeleteAccountMemberships(ctx, id) }},
		{"overwrites", func() error { return q.DeleteAccountOverwrites(ctx, id) }},
		{"tokens", func() error {
			return q.RevokeAccountPATs(ctx, sqlcgen.RevokeAccountPATsParams{RevokedAt: sql.NullInt64{Int64: now, Valid: true}, UserID: id})
		}},
		{"pairing defaults", func() error { return q.DeleteAccountPairingDefaults(ctx, id) }},
		{"pairing computers", func() error { return q.DeleteAccountPairingComputers(ctx, id) }},
		{"invitations", func() error { return q.DeleteAccountInvitations(ctx, id) }},
	}
	for _, step := range steps {
		if err := step.run(); err != nil {
			return fmt.Errorf("remove account %s %s: %w", id, step.name, err)
		}
	}
	if _, err := q.SetCanCreateWorkspace(ctx, sqlcgen.SetCanCreateWorkspaceParams{CanCreateWorkspace: 0, UpdatedAt: now, ID: id}); err != nil {
		return fmt.Errorf("clear account admin %s: %w", id, err)
	}
	return nil
}

func enqueueAccountEvents(ctx context.Context, tx *sql.Tx, id string, events []eventbus.OutboxEvent) error {
	for _, event := range events {
		if err := insertOutboxRow(ctx, tx, event.ID, event.Topic, event.Payload); err != nil {
			return fmt.Errorf("write account event %s: %w", id, err)
		}
	}
	return nil
}

func (r *UsersRepo) CountUsers(ctx context.Context) (int, error) {
	n, err := r.q.CountUsers(ctx)
	if err != nil {
		return 0, fmt.Errorf("count users: %w", err)
	}
	return int(n), nil
}

func (r *UsersRepo) CountActiveAdmins(ctx context.Context) (int, error) {
	n, err := r.q.CountActiveAdmins(ctx)
	if err != nil {
		return 0, fmt.Errorf("count active instance admins: %w", err)
	}
	return int(n), nil
}

func (r *UsersRepo) MarkFirstLoginDone(ctx context.Context, id string) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		n, err := r.q.WithTx(tx).MarkFirstLoginDone(ctx, sqlcgen.MarkFirstLoginDoneParams{
			UpdatedAt: time.Now().Unix(), ID: id,
		})
		if err != nil {
			return fmt.Errorf("mark first login %s: %w", id, err)
		}
		if n == 0 {
			return fmt.Errorf("mark first login %s: %w", id, apperrs.ErrNotFound)
		}
		return nil
	})
}

// SetProfileOverride sets the manual profile columns, separate from the GitHub sync; nil clears back to SQL NULL.
func (r *UsersRepo) SetProfileOverride(ctx context.Context, id string, displayName, avatarOverrideURL *string, events ...eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		n, err := r.q.WithTx(tx).SetProfileOverride(ctx, sqlcgen.SetProfileOverrideParams{
			DisplayName: nullStringPtr(displayName), AvatarOverrideUrl: nullStringPtr(avatarOverrideURL),
			UpdatedAt: time.Now().Unix(), ID: id,
		})
		if err != nil {
			return fmt.Errorf("set profile override %s: %w", id, err)
		}
		if n == 0 {
			return fmt.Errorf("set profile override %s: %w", id, apperrs.ErrNotFound)
		}
		return enqueueAccountEvents(ctx, tx, id, events)
	})
}

func (r *UsersRepo) getByProvider(ctx context.Context, q *sqlcgen.Queries, provider auth.Provider, providerUserID string) (*auth.User, error) {
	return userRow(q.GetUserByIdentity(ctx, sqlcgen.GetUserByIdentityParams{
		Provider: string(provider), ProviderUserID: providerUserID,
	}))
}

// GetUserByLogin resolves a user by provider login, case-insensitive (GitHub usernames are case-insensitive).
func (r *UsersRepo) GetUserByLogin(ctx context.Context, login string) (*auth.User, error) {
	return userRow(r.q.GetUserByLogin(ctx, login))
}

// ListUsers returns every workspace member, used by the notifications fan-out.
func (r *UsersRepo) ListUsers(ctx context.Context) ([]*auth.User, error) {
	rows, err := r.q.ListUsers(ctx)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	var out []*auth.User
	for _, row := range rows {
		out = append(out, toUser(row))
	}
	return out, nil
}

// userRow wraps a single-user query result the way every UsersRepo lookup but GetUserByID does (no id in the message).
func userRow(row sqlcgen.User, err error) (*auth.User, error) {
	if err != nil {
		return nil, fmt.Errorf("get user: %w", notFoundIfNoRows(err))
	}
	return toUser(row), nil
}

func toUser(row sqlcgen.User) *auth.User {
	u := &auth.User{
		ID:                 row.ID,
		Login:              row.Login,
		Name:               row.Name,
		AvatarURL:          row.AvatarUrl,
		CanCreateWorkspace: row.CanCreateWorkspace != 0,
		FirstLoginDone:     row.FirstLoginDone != 0,
		AccountStatus:      auth.AccountStatus(row.AccountStatus),
		CreatedAt:          time.Unix(row.CreatedAt, 0).UTC(),
		UpdatedAt:          time.Unix(row.UpdatedAt, 0).UTC(),
	}
	if row.DisplayName.Valid {
		u.DisplayName = &row.DisplayName.String
	}
	if row.AvatarOverrideUrl.Valid {
		u.AvatarOverrideURL = &row.AvatarOverrideUrl.String
	}
	return u
}

// nullStringPtr converts an optional override value to the column's nullable param type; nil clears back to SQL NULL.
func nullStringPtr(s *string) sql.NullString {
	if s == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: *s, Valid: true}
}

type AllowlistRepo struct {
	db *sql.DB
	w  *Serializer
	q  *sqlcgen.Queries
}

func (r *AllowlistRepo) Add(ctx context.Context, login string) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		err := r.q.WithTx(tx).AddAllowlistMember(ctx, sqlcgen.AddAllowlistMemberParams{
			Login: login, AddedAt: time.Now().Unix(),
		})
		if err != nil {
			if classifyWriteErr(err) == apperrs.ErrConflict {
				return fmt.Errorf("%w: %s is already on the allowlist", apperrs.ErrConflict, login)
			}
			return fmt.Errorf("add member %s: %w", login, err)
		}
		return nil
	})
}

func (r *AllowlistRepo) Remove(ctx context.Context, login string) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		n, err := r.q.WithTx(tx).RemoveAllowlistMember(ctx, login)
		if err != nil {
			return fmt.Errorf("remove member %s: %w", login, err)
		}
		if n == 0 {
			return fmt.Errorf("%w: %s is not on the allowlist", apperrs.ErrNotFound, login)
		}
		return nil
	})
}

func (r *AllowlistRepo) Contains(ctx context.Context, login string) (bool, error) {
	n, err := r.q.CountAllowlistMember(ctx, login)
	if err != nil {
		return false, fmt.Errorf("check allowlist %s: %w", login, err)
	}
	return n > 0, nil
}

func (r *AllowlistRepo) List(ctx context.Context) ([]string, error) {
	out, err := r.q.ListAllowlist(ctx)
	if err != nil {
		return nil, fmt.Errorf("list allowlist: %w", err)
	}
	return out, nil
}

type SettingsRepo struct {
	db     *sql.DB
	w      *Serializer
	encKey []byte
	q      *sqlcgen.Queries
}

func (r *SettingsRepo) Get(ctx context.Context) (auth.Settings, error) {
	return r.readSettings(ctx, r.q)
}

// Set bumps the settings version so connection tokens regenerate; OAuth columns untouched.
func (r *SettingsRepo) Set(ctx context.Context, instanceURL string) (auth.Settings, error) {
	return r.update(ctx, "set settings", func(q *sqlcgen.Queries) (int64, error) {
		return q.SetInstanceURL(ctx, sqlcgen.SetInstanceURLParams{InstanceUrl: instanceURL, UpdatedAt: time.Now().Unix()})
	})
}

// SetGitHubOAuth persists the GitHub OAuth App client ID/secret, encrypted at rest with the repo's injected key.
func (r *SettingsRepo) SetGitHubOAuth(ctx context.Context, clientID, clientSecret string) (auth.Settings, error) {
	encSecret, err := r.encryptSecret("github", clientSecret)
	if err != nil {
		return auth.Settings{}, err
	}
	return r.update(ctx, "set github oauth", func(q *sqlcgen.Queries) (int64, error) {
		return q.SetSettingsGitHubOAuth(ctx, sqlcgen.SetSettingsGitHubOAuthParams{
			GithubOauthClientID: clientID, GithubOauthClientSecret: encSecret, UpdatedAt: time.Now().Unix(),
		})
	})
}

// providerOAuthColumns maps each optional sign-in provider to its credential
// columns; the SQL below is built from this table, never from request input.
var providerOAuthColumns = map[auth.Provider][2]string{
	auth.ProviderGoogle:  {"google_oauth_client_id", "google_oauth_client_secret"},
	auth.ProviderDiscord: {"discord_oauth_client_id", "discord_oauth_client_secret"},
}

// SetProviderOAuth persists an optional sign-in provider's client ID/secret
// (ADR 0040), encrypted at rest exactly like SetGitHubOAuth; both empty disables it.
func (r *SettingsRepo) SetProviderOAuth(ctx context.Context, provider auth.Provider, clientID, clientSecret string) (auth.Settings, error) {
	cols, ok := providerOAuthColumns[provider]
	if !ok {
		return auth.Settings{}, fmt.Errorf("%w: no oauth columns for provider %q", apperrs.ErrInvalid, provider)
	}
	encSecret, err := r.encryptSecret(string(provider), clientSecret)
	if err != nil {
		return auth.Settings{}, err
	}
	op := "set " + string(provider) + " oauth"
	var st auth.Settings
	err = r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		q := r.q.WithTx(tx)
		// hand-written: sqlc cannot express column names chosen at runtime
		res, err := tx.ExecContext(ctx,
			fmt.Sprintf(`UPDATE instance_settings SET %s = ?, %s = ?, updated_at = ? WHERE id = 1`, cols[0], cols[1]),
			clientID, encSecret, time.Now().Unix())
		if err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}
		if n, _ := res.RowsAffected(); n == 0 {
			return fmt.Errorf("%s: %w", op, apperrs.ErrNotFound)
		}
		st, err = r.readSettings(ctx, q)
		return err
	})
	if err != nil {
		return auth.Settings{}, err
	}
	return st, nil
}

func (r *SettingsRepo) encryptSecret(provider, secret string) (string, error) {
	if secret == "" {
		return "", nil
	}
	enc, err := crypto.Encrypt(r.encKey, []byte(secret))
	if err != nil {
		return "", fmt.Errorf("encrypt %s oauth secret: %w", provider, err)
	}
	return enc, nil
}

// update runs one UPDATE against the singleton row and returns the resulting
// Settings read inside the same transaction.
func (r *SettingsRepo) update(ctx context.Context, op string, exec func(q *sqlcgen.Queries) (int64, error)) (auth.Settings, error) {
	var st auth.Settings
	err := r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		q := r.q.WithTx(tx)
		n, err := exec(q)
		if err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}
		if n == 0 {
			return fmt.Errorf("%s: %w", op, apperrs.ErrNotFound)
		}
		st, err = r.readSettings(ctx, q)
		return err
	})
	if err != nil {
		return auth.Settings{}, err
	}
	return st, nil
}

// readSettings scans the singleton row, decrypting whichever OAuth secrets are stored.
func (r *SettingsRepo) readSettings(ctx context.Context, q *sqlcgen.Queries) (auth.Settings, error) {
	row, err := q.GetSettings(ctx)
	if err != nil {
		return auth.Settings{}, fmt.Errorf("get settings: %w", err)
	}
	st := auth.Settings{
		InstanceURL:          row.InstanceUrl,
		SettingsVersion:      int(row.SettingsVersion),
		GitHubOAuthClientID:  row.GithubOauthClientID,
		GoogleOAuthClientID:  row.GoogleOauthClientID,
		DiscordOAuthClientID: row.DiscordOauthClientID,
	}
	if st.GitHubOAuthClientSecret, err = r.decryptSecret("github", row.GithubOauthClientSecret); err != nil {
		return auth.Settings{}, err
	}
	if st.GoogleOAuthClientSecret, err = r.decryptSecret("google", row.GoogleOauthClientSecret); err != nil {
		return auth.Settings{}, err
	}
	if st.DiscordOAuthClientSecret, err = r.decryptSecret("discord", row.DiscordOauthClientSecret); err != nil {
		return auth.Settings{}, err
	}
	return st, nil
}

func (r *SettingsRepo) decryptSecret(provider, enc string) (string, error) {
	if enc == "" {
		return "", nil
	}
	plain, err := crypto.Decrypt(r.encKey, enc)
	if err != nil {
		return "", fmt.Errorf("decrypt %s oauth secret: %w", provider, err)
	}
	return string(plain), nil
}
