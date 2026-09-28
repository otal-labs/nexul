package auth

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
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

// fakeUserStore is an in-memory UserStore; byKey maps a provider identity to its user, identities keeps the rows.
type fakeUserStore struct {
	mu         sync.Mutex
	byID       map[string]*User
	byKey      map[string]string
	identities map[string]*Identity
	seq        int
}

func newFakeUserStore() *fakeUserStore {
	return &fakeUserStore{byID: map[string]*User{}, byKey: map[string]string{}, identities: map[string]*Identity{}}
}

func userKey(p Provider, pid string) string { return string(p) + ":" + pid }

func (f *fakeUserStore) UpsertUser(_ context.Context, id *Identity) (*User, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	key := userKey(id.Provider, id.ProviderUserID)
	if userID, ok := f.byKey[key]; ok {
		stored := f.identities[key]
		stored.Login, stored.Name, stored.AvatarURL = id.Login, id.Name, id.AvatarURL
		existing := f.byID[userID]
		// Mirrors the repo: the user's profile follows the identity they were created with, never a later link.
		f.mu.Unlock()
		first, _ := f.ListIdentities(context.Background(), userID)
		f.mu.Lock()
		if len(first) > 0 && userKey(first[0].Provider, first[0].ProviderUserID) == key {
			existing.Login, existing.Name, existing.AvatarURL = id.Login, id.Name, id.AvatarURL
		}
		return existing, false, nil
	}
	if id.UserID == "" {
		f.seq++
		id.UserID = fmt.Sprintf("user-%d", f.seq)
	}
	return f.insert(id), true, nil
}

// insert creates the user row from the identity, keeping the caller's copy of the identity untouched.
func (f *fakeUserStore) insert(id *Identity) *User {
	u := &User{ID: id.UserID, Login: id.Login, Name: id.Name, AvatarURL: id.AvatarURL, AccountStatus: AccountActive}
	cp := *id
	f.byKey[userKey(id.Provider, id.ProviderUserID)] = u.ID
	f.identities[userKey(id.Provider, id.ProviderUserID)] = &cp
	f.byID[u.ID] = u
	return u
}

func (f *fakeUserStore) CreateFirstUser(_ context.Context, id *Identity, _ ...eventbus.OutboxEvent) (*User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.byID) != 0 {
		return nil, apperrs.ErrConflict
	}
	if id.UserID == "" {
		f.seq++
		id.UserID = fmt.Sprintf("user-%d", f.seq)
	}
	return f.insert(id), nil
}

func (f *fakeUserStore) GetUserByProvider(_ context.Context, provider Provider, providerUserID string) (*User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	id, ok := f.byKey[userKey(provider, providerUserID)]
	if !ok {
		return nil, apperrs.ErrNotFound
	}
	return f.byID[id], nil
}

func (f *fakeUserStore) ListIdentities(_ context.Context, userID string) ([]Identity, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []Identity
	for _, id := range f.identities {
		if id.UserID == userID {
			out = append(out, *id)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if !out[i].CreatedAt.Equal(out[j].CreatedAt) {
			return out[i].CreatedAt.Before(out[j].CreatedAt)
		}
		return out[i].Provider < out[j].Provider
	})
	return out, nil
}

func (f *fakeUserStore) LinkIdentity(_ context.Context, id *Identity, _ ...eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	key := userKey(id.Provider, id.ProviderUserID)
	if owner, ok := f.byKey[key]; ok && owner == id.UserID {
		return fmt.Errorf("%w: that %s account is already linked to you", apperrs.ErrConflict, id.Provider)
	}
	if _, ok := f.byKey[key]; ok {
		return fmt.Errorf("%w: that %s account is already linked to another user", apperrs.ErrConflict, id.Provider)
	}
	for _, existing := range f.identities {
		if existing.UserID == id.UserID && existing.Provider == id.Provider {
			return fmt.Errorf("%w: a %s account is already linked; unlink it first", apperrs.ErrConflict, id.Provider)
		}
	}
	cp := *id
	f.byKey[key] = id.UserID
	f.identities[key] = &cp
	return nil
}

func (f *fakeUserStore) UnlinkIdentity(_ context.Context, userID string, provider Provider, _ ...eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	var keys []string
	for key, id := range f.identities {
		if id.UserID == userID {
			keys = append(keys, key)
		}
	}
	if len(keys) <= 1 {
		return fmt.Errorf("%w: this is your only sign-in", apperrs.ErrConflict)
	}
	for _, key := range keys {
		if f.identities[key].Provider != provider {
			continue
		}
		delete(f.identities, key)
		delete(f.byKey, key)
		return nil
	}
	return fmt.Errorf("%w: no %s account is linked", apperrs.ErrNotFound, provider)
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

func (f *fakeUserStore) SetAccountStatus(_ context.Context, id string, status AccountStatus, _ ...eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	u, ok := f.byID[id]
	if !ok {
		return apperrs.ErrNotFound
	}
	u.AccountStatus = status
	return nil
}

func (f *fakeUserStore) CountUsers(_ context.Context) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.byID), nil
}

func (f *fakeUserStore) CountActiveAdmins(_ context.Context) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	count := 0
	for _, u := range f.byID {
		if u.CanCreateWorkspace && u.AccountStatus == AccountActive {
			count++
		}
	}
	return count, nil
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
	return &fakeSettings{st: Settings{SettingsVersion: 1}}
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

// fakePATStore is an in-memory PATStore that mimics the real repo's contract: only hashes are stored and revoke is user-scoped + idempotent-once.
type fakePATStore struct {
	mu     sync.Mutex
	seq    int
	byID   map[string]*PersonalAccessToken
	outbox []eventbus.OutboxEvent
	getErr error
}

func newFakePATStore() *fakePATStore {
	return &fakePATStore{byID: map[string]*PersonalAccessToken{}}
}

func (f *fakePATStore) Create(_ context.Context, p *PersonalAccessToken, evts ...eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.outbox = append(f.outbox, evts...)
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

func (f *fakePATStore) Revoke(_ context.Context, id, userID string, evts ...eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	p, ok := f.byID[id]
	if !ok || p.UserID != userID || p.RevokedAt != nil {
		return apperrs.ErrNotFound
	}
	now := time.Unix(1_700_000_100, 0)
	p.RevokedAt = &now
	f.outbox = append(f.outbox, evts...)
	return nil
}

func (f *fakePATStore) GetActiveForComputer(_ context.Context, userID, computerID string) (*PersonalAccessToken, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.getErr != nil {
		return nil, f.getErr
	}
	for _, p := range f.byID {
		if p.UserID == userID && p.ComputerID == computerID && p.RevokedAt == nil {
			cp := *p
			return &cp, nil
		}
	}
	return nil, apperrs.ErrNotFound
}

func (f *fakePATStore) topics() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]string, 0, len(f.outbox))
	for _, e := range f.outbox {
		out = append(out, e.Topic)
	}
	return out
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
// fakeSessionStore is an in-memory SessionStore with the real repo's contract: hashes only, user-scoped deletes.
type fakeSessionStore struct {
	mu       sync.Mutex
	byID     map[string]*Session
	outbox   []eventbus.OutboxEvent
	touches  int
	touchErr error
}

func newFakeSessionStore() *fakeSessionStore {
	return &fakeSessionStore{byID: map[string]*Session{}}
}

func (f *fakeSessionStore) CreateSession(_ context.Context, s *Session, evts ...eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	cp := *s
	f.byID[s.ID] = &cp
	f.outbox = append(f.outbox, evts...)
	return nil
}

func (f *fakeSessionStore) GetSessionByHash(_ context.Context, hash string) (*Session, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, s := range f.byID {
		if s.TokenHash == hash {
			cp := *s
			return &cp, nil
		}
	}
	return nil, apperrs.ErrNotFound
}

func (f *fakeSessionStore) ListSessionsByUser(_ context.Context, userID string) ([]Session, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]Session, 0)
	for _, s := range f.byID {
		if s.UserID == userID {
			out = append(out, *s)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func (f *fakeSessionStore) DeleteSession(_ context.Context, id, userID string, evts ...eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	s, ok := f.byID[id]
	if !ok || s.UserID != userID {
		return apperrs.ErrNotFound
	}
	delete(f.byID, id)
	f.outbox = append(f.outbox, evts...)
	return nil
}

func (f *fakeSessionStore) DeleteOtherSessions(_ context.Context, userID, keepID string, evts ...eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for id, s := range f.byID {
		if s.UserID == userID && id != keepID {
			delete(f.byID, id)
		}
	}
	f.outbox = append(f.outbox, evts...)
	return nil
}

func (f *fakeSessionStore) TouchSession(_ context.Context, id string, lastActive time.Time, ip string, expiresAt time.Time) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.touches++
	if f.touchErr != nil {
		return f.touchErr
	}
	if s, ok := f.byID[id]; ok {
		s.LastActiveAt, s.IP, s.ExpiresAt = lastActive, ip, expiresAt
	}
	return nil
}

func (f *fakeSessionStore) DeleteExpiredSessions(_ context.Context, userID string, now time.Time) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for id, s := range f.byID {
		if s.UserID == userID && !now.Before(s.ExpiresAt) {
			delete(f.byID, id)
		}
	}
	return nil
}

func (f *fakeSessionStore) topics() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]string, 0, len(f.outbox))
	for _, e := range f.outbox {
		out = append(out, e.Topic)
	}
	return out
}

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

type fakeInvitationGate struct {
	invitation *InvitationAcceptance
	token      string
	acceptance string
	admission  InvitationAdmission
	err        error
}

func (f *fakeInvitationGate) GetInvitationByToken(_ context.Context, token string, _ time.Time) (*InvitationAcceptance, error) {
	if f.err != nil {
		return nil, f.err
	}
	if token != f.token {
		return nil, apperrs.ErrNotFound
	}
	return f.invitation, nil
}

func (f *fakeInvitationGate) GetInvitationByAcceptance(_ context.Context, acceptanceHash string, _ time.Time) (*InvitationAcceptance, error) {
	if f.err != nil {
		return nil, f.err
	}
	if acceptanceHash != hashCredential(f.acceptance) {
		return nil, apperrs.ErrNotFound
	}
	return f.invitation, nil
}

func (f *fakeInvitationGate) RedeemInvitation(_ context.Context, acceptanceHash string, _ InvitationIdentity, _ time.Time, _ ...eventbus.OutboxEvent) (InvitationAdmission, error) {
	if f.err != nil {
		return InvitationAdmission{}, f.err
	}
	if acceptanceHash != hashCredential(f.acceptance) {
		return InvitationAdmission{}, apperrs.ErrNotFound
	}
	return f.admission, nil
}

type fakeOAuthHandoffStore struct {
	handoff *OAuthHandoff
}

func (f *fakeOAuthHandoffStore) StartOAuthHandoff(_ context.Context, handoff *OAuthHandoff) error {
	f.handoff = handoff
	return nil
}

func (f *fakeOAuthHandoffStore) CompleteOAuthCallback(_ context.Context, stateHash, acceptanceHash string, identity OAuthHandoffIdentity, expiresAt, _ time.Time) (*OAuthHandoff, error) {
	if f.handoff == nil || f.handoff.OAuthStateHash != stateHash {
		return nil, apperrs.ErrNotFound
	}
	f.handoff.OAuthStateHash = ""
	f.handoff.AcceptanceHash = acceptanceHash
	f.handoff.ProviderUserID = identity.ProviderUserID
	f.handoff.Login = identity.Login
	f.handoff.Name = identity.Name
	f.handoff.AvatarURL = identity.AvatarURL
	f.handoff.ExistingUserID = identity.ExistingUserID
	f.handoff.ExpiresAt = expiresAt
	return f.handoff, nil
}

func (f *fakeOAuthHandoffStore) GetOAuthHandoffByAcceptanceHash(_ context.Context, acceptanceHash string, _ time.Time) (*OAuthHandoff, error) {
	if f.handoff == nil || f.handoff.AcceptanceHash != acceptanceHash {
		return nil, apperrs.ErrNotFound
	}
	return f.handoff, nil
}

func (f *fakeOAuthHandoffStore) CompleteOAuthRedemption(_ context.Context, _, admittedUserID string, now time.Time) error {
	if f.handoff == nil || f.handoff.AdmittedUserID != "" && f.handoff.AdmittedUserID != admittedUserID {
		return apperrs.ErrNotFound
	}
	f.handoff.AdmittedUserID = admittedUserID
	f.handoff.CompletedAt = &now
	return nil
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

// fakeSetupCodes is an in-memory SetupCodeStore holding at most one code, like the real one.
type fakeSetupCodes struct {
	mu      sync.Mutex
	hash    string
	expires time.Time
	err     error
}

func (f *fakeSetupCodes) ReplaceSetupCode(_ context.Context, hash string, _, expiresAt time.Time) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return f.err
	}
	f.hash, f.expires = hash, expiresAt
	return nil
}

func (f *fakeSetupCodes) SetupCodeValid(_ context.Context, hash string, now time.Time) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return false, f.err
	}
	return f.hash != "" && f.hash == hash && now.Before(f.expires), nil
}

func (f *fakeSetupCodes) ClearSetupCodes(context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return f.err
	}
	f.hash = ""
	return nil
}

// fakePublicAddress answers PublicAddressLookup without the network.
type fakePublicAddress struct{ addr PublicAddress }

func (f fakePublicAddress) PublicAddress(context.Context) PublicAddress { return f.addr }

func newTestService(gh GitHubClient, users UserStore, allowlist AllowlistStore, settings SettingsStore) *Service {
	return NewService(Config{
		SetupCodes:       &fakeSetupCodes{},
		PublicAddress:    fakePublicAddress{addr: PublicAddress{IPv4: "203.0.113.7"}},
		Secret:           []byte("test-secret"),
		GitHub:           gh,
		Users:            users,
		Allowlist:        allowlist,
		Settings:         settings,
		DefaultWorkspace: &fakeDefaultWorkspace{},
		PendingInvites:   &fakePendingInviteResolver{},
		Now:              func() time.Time { return time.Unix(1_700_000_000, 0) },
		Sessions:         newFakeSessionStore(),
	})
}

// sign mints a browser session for userID the way every sign-in path does, returning the raw bearer.
func sign(s *Service, userID string) (string, error) {
	return s.CreateSession(context.Background(), userID)
}

// verify resolves a raw bearer to the user it acts as, the way RequireAuth does.
func verify(s *Service, token string) (string, error) {
	user, _, err := s.AuthenticateSession(context.Background(), token, "")
	if err != nil {
		return "", err
	}
	return user.ID, nil
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

// fakeConnectCodes is an in-memory ConnectCodeStore holding one code per user, like the real one.
type fakeConnectCodes struct {
	mu      sync.Mutex
	byHash  map[string]fakeConnectCode
	err     error
	replace int
}

type fakeConnectCode struct {
	userID  string
	expires time.Time
}

func newFakeConnectCodes() *fakeConnectCodes {
	return &fakeConnectCodes{byHash: map[string]fakeConnectCode{}}
}

func (f *fakeConnectCodes) ReplaceConnectCode(_ context.Context, userID, hash string, _, expiresAt time.Time) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return f.err
	}
	for h, c := range f.byHash {
		if c.userID == userID {
			delete(f.byHash, h)
		}
	}
	f.byHash[hash] = fakeConnectCode{userID: userID, expires: expiresAt}
	f.replace++
	return nil
}

func (f *fakeConnectCodes) ConsumeConnectCode(_ context.Context, hash string, now time.Time) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return "", f.err
	}
	c, ok := f.byHash[hash]
	if !ok || !now.Before(c.expires) {
		return "", apperrs.ErrNotFound
	}
	delete(f.byHash, hash)
	return c.userID, nil
}
