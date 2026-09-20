package auth

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// fakeGitHub implements GitHubClient without network access.
type fakeGitHub struct {
	token  string
	user   *GitHubUser
	exErr  error
	logErr error
}

func (f *fakeGitHub) Exchange(_ context.Context, code string) (string, error) {
	if f.exErr != nil {
		return "", f.exErr
	}
	if code == "bad-code" {
		return "", apperrs.ErrUnauthorized
	}
	return f.token, nil
}

func (f *fakeGitHub) FetchUser(_ context.Context, _ string) (*GitHubUser, error) {
	if f.logErr != nil {
		return nil, f.logErr
	}
	if f.user == nil {
		return nil, apperrs.ErrUnauthorized
	}
	return f.user, nil
}

// fakeUserStore is an in-memory UserStore.
type fakeUserStore struct {
	mu    sync.Mutex
	byID  map[string]*User
	byKey map[string]string
	seq   int
}

func newFakeUserStore() *fakeUserStore {
	return &fakeUserStore{byID: map[string]*User{}, byKey: map[string]string{}}
}

func userKey(p Provider, pid string) string { return string(p) + ":" + pid }

func (f *fakeUserStore) UpsertUser(_ context.Context, u *User) (*User, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	key := userKey(u.Provider, u.ProviderUserID)
	if id, ok := f.byKey[key]; ok {
		existing := f.byID[id]
		existing.Login = u.Login
		existing.Name = u.Name
		existing.AvatarURL = u.AvatarURL
		return existing, false, nil
	}
	if u.ID == "" {
		f.seq++
		u.ID = fmt.Sprintf("user-%d", f.seq)
	}
	cp := *u
	f.byKey[key] = cp.ID
	f.byID[cp.ID] = &cp
	return &cp, true, nil
}

func (f *fakeUserStore) GetUserByID(_ context.Context, id string) (*User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	u, ok := f.byID[id]
	if !ok {
		return nil, apperrs.ErrNotFound
	}
	return u, nil
}

func (f *fakeUserStore) GetUserByLogin(_ context.Context, login string) (*User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, u := range f.byID {
		if u.Login == login {
			return u, nil
		}
	}
	return nil, apperrs.ErrNotFound
}

func (f *fakeUserStore) ListUsers(_ context.Context) ([]*User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]*User, 0, len(f.byID))
	for _, u := range f.byID {
		out = append(out, u)
	}
	return out, nil
}

func (f *fakeUserStore) CanCreateWorkspaceExists(_ context.Context) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, u := range f.byID {
		if u.CanCreateWorkspace {
			return true, nil
		}
	}
	return false, nil
}

func (f *fakeUserStore) SetCanCreateWorkspace(_ context.Context, id string, can bool) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	u, ok := f.byID[id]
	if !ok {
		return apperrs.ErrNotFound
	}
	u.CanCreateWorkspace = can
	return nil
}

func (f *fakeUserStore) MarkFirstLoginDone(_ context.Context, id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	u, ok := f.byID[id]
	if !ok {
		return apperrs.ErrNotFound
	}
	u.FirstLoginDone = true
	return nil
}

func (f *fakeUserStore) SetProfileOverride(_ context.Context, id string, displayName, avatarOverrideURL *string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	u, ok := f.byID[id]
	if !ok {
		return apperrs.ErrNotFound
	}
	u.DisplayName = displayName
	u.AvatarOverrideURL = avatarOverrideURL
	return nil
}

// fakeAllowlist is an in-memory AllowlistStore.
type fakeAllowlist struct {
	mu  sync.Mutex
	set map[string]struct{}
}

func newFakeAllowlist() *fakeAllowlist {
	return &fakeAllowlist{set: map[string]struct{}{}}
}

func (f *fakeAllowlist) Add(_ context.Context, login string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.set[login]; ok {
		return apperrs.ErrConflict
	}
	f.set[login] = struct{}{}
	return nil
}

func (f *fakeAllowlist) Remove(_ context.Context, login string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.set[login]; !ok {
		return apperrs.ErrNotFound
	}
	delete(f.set, login)
	return nil
}

func (f *fakeAllowlist) Contains(_ context.Context, login string) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	_, ok := f.set[login]
	return ok, nil
}

func (f *fakeAllowlist) List(_ context.Context) ([]string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]string, 0, len(f.set))
	for l := range f.set {
		out = append(out, l)
	}
	sort.Strings(out)
	return out, nil
}

// fakeSettings is an in-memory SettingsStore.
type fakeSettings struct {
	mu sync.Mutex
	st Settings
}

func newFakeSettings() *fakeSettings {
	return &fakeSettings{st: Settings{SettingsVersion: 1, MentionChipTemplate: "{ticket.Ticket} {ticket.Status}"}}
}

func (f *fakeSettings) Get(_ context.Context) (Settings, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.st, nil
}

func (f *fakeSettings) Set(_ context.Context, instanceURL string) (Settings, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.st.InstanceURL = instanceURL
	f.st.SettingsVersion++
	return f.st, nil
}

func (f *fakeSettings) SetGitHubOAuth(_ context.Context, clientID, clientSecret string) (Settings, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.st.GitHubOAuthClientID = clientID
	f.st.GitHubOAuthClientSecret = clientSecret
	return f.st, nil
}

func (f *fakeSettings) SetProviderOAuth(_ context.Context, provider Provider, clientID, clientSecret string) (Settings, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	switch provider {
	case ProviderGoogle:
		f.st.GoogleOAuthClientID, f.st.GoogleOAuthClientSecret = clientID, clientSecret
	case ProviderDiscord:
		f.st.DiscordOAuthClientID, f.st.DiscordOAuthClientSecret = clientID, clientSecret
	default:
		return Settings{}, apperrs.ErrInvalid
	}
	return f.st, nil
}

func (f *fakeSettings) SetMentionChipTemplate(_ context.Context, template string) (Settings, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.st.MentionChipTemplate = template
	return f.st, nil
}

// fakeMentionLayoutGate is an in-memory MentionLayoutGate: allow defaults to true so tests that don't care about the workspaces:write gate aren't forced to wire it up; SetAllowed(false) exercises the forbidden path.
type fakeMentionLayoutGate struct {
	mu      sync.Mutex
	allowed bool
}

func newFakeMentionLayoutGate() *fakeMentionLayoutGate {
	return &fakeMentionLayoutGate{allowed: true}
}

func (f *fakeMentionLayoutGate) CanManageMentionLayout(_ context.Context, _ string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.allowed
}

func (f *fakeMentionLayoutGate) SetAllowed(allowed bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.allowed = allowed
}

// fakePATStore is an in-memory PATStore that mimics the real repo's contract: only hashes are stored and revoke is user-scoped + idempotent-once.
type fakePATStore struct {
	mu   sync.Mutex
	seq  int
	byID map[string]*PersonalAccessToken
}

func newFakePATStore() *fakePATStore {
	return &fakePATStore{byID: map[string]*PersonalAccessToken{}}
}

func (f *fakePATStore) Create(_ context.Context, p *PersonalAccessToken) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if p.ID == "" {
		f.seq++
		p.ID = fmt.Sprintf("pat-%d", f.seq)
	}
	if p.CreatedAt.IsZero() {
		p.CreatedAt = time.Unix(1_700_000_000, 0)
	}
	cp := *p
	f.byID[cp.ID] = &cp
	return nil
}

func (f *fakePATStore) GetByHash(_ context.Context, hash string) (*PersonalAccessToken, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, p := range f.byID {
		if p.TokenHash == hash {
			cp := *p
			return &cp, nil
		}
	}
	return nil, apperrs.ErrNotFound
}

func (f *fakePATStore) ListByUser(_ context.Context, userID string) ([]PersonalAccessToken, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]PersonalAccessToken, 0)
	for _, p := range f.byID {
		if p.UserID == userID {
			out = append(out, *p)
		}
	}
	return out, nil
}

func (f *fakePATStore) Revoke(_ context.Context, id, userID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	p, ok := f.byID[id]
	if !ok || p.UserID != userID || p.RevokedAt != nil {
		return apperrs.ErrNotFound
	}
	now := time.Unix(1_700_000_100, 0)
	p.RevokedAt = &now
	return nil
}

func (f *fakePATStore) TouchLastUsed(_ context.Context, id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if p, ok := f.byID[id]; ok {
		now := time.Unix(1_700_000_200, 0)
		p.LastUsedAt = &now
	}
	return nil
}

// fakeDefaultWorkspace is a minimal stand-in for the tenancy domain's BindDefaultWorkspaceOwner seam (ticket 11): it just records who was bound, mirroring what the real tenancy.Service would persist.
type fakeDefaultWorkspace struct {
	mu    sync.Mutex
	bound []string
	err   error
}

func (f *fakeDefaultWorkspace) BindDefaultWorkspaceOwner(_ context.Context, userID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return f.err
	}
	f.bound = append(f.bound, userID)
	return nil
}

// fakePendingInviteResolver is a minimal stand-in for the tenancy domain's ResolvePendingInvites seam: it just records which (login, userID) pairs were resolved, mirroring what the real tenancy.Service would persist.
type fakeConnectorAppSeeder struct {
	seeded  []string // [clientID, clientSecret, appSlug]
	seedErr error
}

func (f *fakeConnectorAppSeeder) SeedGitHubApp(_ context.Context, clientID, clientSecret, appSlug string) error {
	f.seeded = []string{clientID, clientSecret, appSlug}
	return f.seedErr
}

type fakePendingInviteResolver struct {
	mu         sync.Mutex
	resolved   [][2]string // [login, userID]
	resolveErr error
}

func (f *fakePendingInviteResolver) ResolvePendingInvites(_ context.Context, login, userID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.resolveErr != nil {
		return f.resolveErr
	}
	f.resolved = append(f.resolved, [2]string{login, userID})
	return nil
}

func newTestService(gh GitHubClient, users UserStore, allowlist AllowlistStore, settings SettingsStore) *Service {
	return NewService(Config{
		Secret:           []byte("test-secret"),
		GitHub:           gh,
		Users:            users,
		Allowlist:        allowlist,
		Settings:         settings,
		DefaultWorkspace: &fakeDefaultWorkspace{},
		PendingInvites:   &fakePendingInviteResolver{},
		MentionLayout:    newFakeMentionLayoutGate(),
		Now:              func() time.Time { return time.Unix(1_700_000_000, 0) },
		TokenTTL:         time.Hour,
	})
}

// newTestHarness wires a service with fresh in-memory fakes and returns them so tests can seed allowlist entries, owners, etc.; Configured()/AuthorizeURL read GitHub OAuth credentials from the returned *fakeSettings (T3/T5), so callers needing Configured() true must seed settings.st.GitHubOAuthClientID/Secret themselves.
func newTestHarness(gh GitHubClient) (*Service, *fakeUserStore, *fakeAllowlist, *fakeSettings) {
	users := newFakeUserStore()
	allowlist := newFakeAllowlist()
	settings := newFakeSettings()
	return newTestService(gh, users, allowlist, settings), users, allowlist, settings
}

func ghUser(id, login string) *GitHubUser {
	return &GitHubUser{ID: id, Login: login, Name: "Name " + login, AvatarURL: "https://avatar/" + login}
}
