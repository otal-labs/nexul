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

// UpsertUser inserts or syncs provider-sourced columns; owner/onboarding flags are never touched by a re-login.
func (r *UsersRepo) UpsertUser(ctx context.Context, u *auth.User) (*auth.User, bool, error) {
	created := false
	var persisted *auth.User
	err := r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		q := r.q.WithTx(tx)
		existingID, err := q.GetUserIDByProvider(ctx, sqlcgen.GetUserIDByProviderParams{
			Provider: string(u.Provider), ProviderUserID: u.ProviderUserID,
		})
		newUser := errors.Is(err, sql.ErrNoRows)
		if err != nil && !newUser {
			return fmt.Errorf("query user: %w", err)
		}
		now := time.Now().Unix()
		if newUser {
			created = true
			if err := q.InsertUser(ctx, sqlcgen.InsertUserParams{
				ID: u.ID, Provider: string(u.Provider), ProviderUserID: u.ProviderUserID,
				Login: u.Login, Name: u.Name, AvatarUrl: u.AvatarURL, CreatedAt: now, UpdatedAt: now,
			}); err != nil {
				return fmt.Errorf("insert user: %w", err)
			}
		}
		if !newUser {
			if err := q.SyncUser(ctx, sqlcgen.SyncUserParams{
				Login: u.Login, Name: u.Name, AvatarUrl: u.AvatarURL, UpdatedAt: now, ID: existingID,
			}); err != nil {
				return fmt.Errorf("update user: %w", err)
			}
		}
		p, err := r.getByProvider(ctx, q, u.Provider, u.ProviderUserID)
		if err != nil {
			return err
		}
		persisted = p
		return nil
	})
	if err != nil {
		return nil, false, err
	}
	return persisted, created, nil
}

func (r *UsersRepo) GetUserByID(ctx context.Context, id string) (*auth.User, error) {
	row, err := r.q.GetUserByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get user %s: %w", id, notFoundIfNoRows(err))
	}
	return toUser(row), nil
}

func (r *UsersRepo) GetUserByProvider(ctx context.Context, provider auth.Provider, providerUserID string) (*auth.User, error) {
	return userRow(r.q.GetUserByProvider(ctx, sqlcgen.GetUserByProviderParams{Provider: string(provider), ProviderUserID: providerUserID}))
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

func (r *UsersRepo) SetAccountStatus(ctx context.Context, id string, status auth.AccountStatus) error {
	if status != auth.AccountActive && status != auth.AccountDisabled && status != auth.AccountRemoved {
		return fmt.Errorf("%w: invalid account status %q", apperrs.ErrInvalid, status)
	}
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		q := r.q.WithTx(tx)
		user, err := q.GetUserByID(ctx, id)
		if err != nil {
			return fmt.Errorf("get account %s: %w", id, notFoundIfNoRows(err))
		}
		if user.CanCreateWorkspace != 0 && user.AccountStatus == string(auth.AccountActive) && status != auth.AccountActive {
			activeAdmins, err := q.CountActiveAdmins(ctx)
			if err != nil {
				return fmt.Errorf("count active instance admins: %w", err)
			}
			if activeAdmins <= 1 {
				return fmt.Errorf("%w: cannot change the last active instance admin", apperrs.ErrConflict)
			}
		}
		n, err := q.SetAccountStatus(ctx, sqlcgen.SetAccountStatusParams{
			AccountStatus: string(status), UpdatedAt: time.Now().Unix(), ID: id,
		})
		if err != nil {
			return fmt.Errorf("set account status %s: %w", id, err)
		}
		if n == 0 {
			return fmt.Errorf("set account status %s: %w", id, apperrs.ErrNotFound)
		}
		return nil
	})
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
func (r *UsersRepo) SetProfileOverride(ctx context.Context, id string, displayName, avatarOverrideURL *string) error {
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
		return nil
	})
}

func (r *UsersRepo) getByProvider(ctx context.Context, q *sqlcgen.Queries, provider auth.Provider, providerUserID string) (*auth.User, error) {
	return userRow(q.GetUserByProvider(ctx, sqlcgen.GetUserByProviderParams{
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
		Provider:           auth.Provider(row.Provider),
		ProviderUserID:     row.ProviderUserID,
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

// SetMentionChipTemplate persists the chip template; doesn't bump SettingsVersion (that tracks the instance URL only).
func (r *SettingsRepo) SetMentionChipTemplate(ctx context.Context, template string) (auth.Settings, error) {
	return r.update(ctx, "set mention chip template", func(q *sqlcgen.Queries) (int64, error) {
		return q.SetSettingsMentionChipTemplate(ctx, sqlcgen.SetSettingsMentionChipTemplateParams{
			MentionChipTemplate: template, UpdatedAt: time.Now().Unix(),
		})
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
		MentionChipTemplate:  row.MentionChipTemplate,
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
