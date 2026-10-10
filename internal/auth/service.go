// Package auth implements the identity and session use-cases (ADR 0019).
package auth

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/freshdns"
	"github.com/otal-labs/nexul/internal/platform/logging"
	"github.com/otal-labs/nexul/internal/platform/oauthx"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

const (
	authorizeURL         = "https://github.com/login/oauth/authorize"
	tokenURL             = "https://github.com/login/oauth/access_token"
	userURL              = "https://api.github.com/user"
	githubSearchUsersURL = "https://api.github.com/search/users"
	stateCookie          = "nexul_oauth_state"
	stateMaxAge          = 10 * time.Minute
)

// Sign-in refusals the browser callback tells apart; each still reads as ErrUnauthorized (or ErrInvalid) to JSON callers.
var (
	errInvitationRequired = fmt.Errorf("%w: invitation required", apperrs.ErrUnauthorized)
	errAccountDisabled    = fmt.Errorf("%w: account is disabled", apperrs.ErrUnauthorized)
	errAccountRemoved     = fmt.Errorf("%w: account was removed", apperrs.ErrUnauthorized)
	errOAuthNotConfigured = fmt.Errorf("%w: OAuth is not configured", apperrs.ErrInvalid)
)

func oauthNotConfigured(provider string) error {
	return fmt.Errorf("%w for %s", errOAuthNotConfigured, provider)
}

// ProviderClient is the slice of an OAuth provider's API the service calls, fakeable in tests.
type ProviderClient interface {
	Exchange(ctx context.Context, code string) (string, error)
	FetchUser(ctx context.Context, accessToken string) (*ProviderUser, error)
}

// GitHubClient is ProviderClient under its original name.
type GitHubClient = ProviderClient

// HTTPGitHubClient talks to GitHub's OAuth + user endpoints over HTTP.
type HTTPGitHubClient struct {
	client       *http.Client
	clientID     string
	clientSecret string
	tokenURL     string
	userURL      string
}

// NewHTTPGitHubClient wires a real GitHub client; an empty clientID/secret makes Exchange fail, reported as a 503.
func NewHTTPGitHubClient(clientID, clientSecret string, hc *http.Client) *HTTPGitHubClient {
	return &HTTPGitHubClient{
		client:       hc,
		clientID:     clientID,
		clientSecret: clientSecret,
		tokenURL:     tokenURL,
		userURL:      userURL,
	}
}

// Exchange trades an authorization code for an access token.
func (c *HTTPGitHubClient) Exchange(ctx context.Context, code string) (string, error) {
	if c.clientID == "" || c.clientSecret == "" {
		return "", oauthNotConfigured("GitHub")
	}
	form := url.Values{
		"client_id":     {c.clientID},
		"client_secret": {c.clientSecret},
		"code":          {code},
	}
	var body struct {
		AccessToken string `json:"access_token"`
		Error       string `json:"error"`
	}
	if _, err := oauthx.PostForm(ctx, c.client, c.tokenURL, form, "", "", "application/json", &body); err != nil {
		return "", fmt.Errorf("exchange code: %w", apperrs.Retryable(err))
	}
	if body.Error != "" {
		return "", fmt.Errorf("%w: github: %s", apperrs.ErrUnauthorized, body.Error)
	}
	if body.AccessToken == "" {
		return "", fmt.Errorf("%w: github returned no access token", apperrs.ErrUnauthorized)
	}
	return body.AccessToken, nil
}

// FetchUser fetches the user's identity; the numeric ID is the stable key, login/name/avatar sync every sign-in.
func (c *HTTPGitHubClient) FetchUser(ctx context.Context, accessToken string) (user *GitHubUser, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.userURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch user: %w", apperrs.Retryable(err))
	}
	defer func() {
		err = errors.Join(err, resp.Body.Close())
	}()
	var body struct {
		ID        int64  `json:"id"`
		Login     string `json:"login"`
		Name      string `json:"name"`
		AvatarURL string `json:"avatar_url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("fetch user: %w", apperrs.Retryable(err))
	}
	if body.ID == 0 || body.Login == "" {
		return nil, fmt.Errorf("%w: github returned an incomplete user", apperrs.ErrUnauthorized)
	}
	return &GitHubUser{
		ID:        strconv.FormatInt(body.ID, 10),
		Login:     body.Login,
		Name:      body.Name,
		AvatarURL: body.AvatarURL,
	}, nil
}

// Config wires the auth service.
type Config struct {
	Secret    []byte
	SPAOrigin string
	GitHub    ProviderClient
	// Google/Discord, like GitHub, are test/dev overrides; nil means build the client from Settings per request.
	Google        ProviderClient
	Discord       ProviderClient
	Users         UserStore
	Invitations   InvitationGate
	OAuthHandoffs OAuthHandoffStore
	Allowlist     AllowlistStore
	Settings      SettingsStore
	PATs          PATStore
	// Sessions stores one row per signed-in device (ADR 0083); every sign-in path mints through it.
	Sessions SessionStore
	// DefaultWorkspace binds the wizard's completing user to the default workspace; wired later via SetDefaultWorkspace.
	DefaultWorkspace DefaultWorkspaceBinder
	// PendingInvites resolves pending invites the moment a User row is created; wired later, same as DefaultWorkspace.
	PendingInvites PendingInviteResolver
	// ConnectorApps seeds github's app-level OAuth registration during Bootstrap, before Settings is reachable (ADR 0017).
	ConnectorApps ConnectorAppSeeder
	// GitHubApp checks the pasted App credentials against GitHub before Bootstrap stores them; nil skips (tests).
	GitHubApp           GitHubAppVerifier
	GitHubManifest      GitHubManifestConverter
	GitHubManifestStore GitHubManifestStore
	Now                 func() time.Time
	// DevLogin enables GitHub-free session minting at /auth/dev-login for local dev; must be false in production.
	DevLogin bool
	// SetupCodes stores the setup code's hash; EnrollDir is where the installer reads the code itself.
	SetupCodes SetupCodeStore
	EnrollDir  string
	// ConnectCodes stores connect code hashes; nil means phones cannot connect (tests without the store).
	ConnectCodes ConnectCodeStore
	// Local marks a desktop install, whose instance URL may stay http://localhost.
	Local bool
	// PublicAddress is overridable in tests; nil asks Cloudflare's trace endpoint.
	PublicAddress PublicAddressLookup
	// Permissions answers the instance-level checks; nil refuses every one of them.
	Permissions PermissionGate
	// Installations claims an App installation returned to the sign-in callback; nil sends it to the connectors page.
	Installations InstallationClaimer
}

// InstallationClaimer assigns the installation an install link led to when GitHub returns its installer (ADR 0144).
type InstallationClaimer interface {
	ClaimsState(state string) bool
	// ClaimInstallation returns the app path the browser lands on.
	ClaimInstallation(ctx context.Context, state, code, installationID string) (string, error)
}

// PermissionGate is the access domain's instance-level answer (ADR 0087): what a user holds in any workspace they belong to.
type PermissionGate interface {
	HoldsAnywhere(ctx context.Context, userID string, action permissions.Action) (bool, error)
	PermissionsAnywhere(ctx context.Context, userID string) ([]string, error)
}

// Service is the auth use-case layer: OAuth, device sessions, and the user/onboarding/allowlist use-cases.
type Service struct {
	cfg Config
	// httpClient is the shared transport reused per request (T3), avoiding a new *http.Client per login.
	httpClient *http.Client
	// probeClient checks the instance URL, a hostname usually created moments before, through freshdns.
	probeClient *http.Client
	// searchURL is GitHub's user-search endpoint, overridable in tests like HTTPGitHubClient.userURL.
	searchURL string
	unlocks   *failureLimiter
	exchanges *failureLimiter
}

// NewService wires the auth use-cases.
func NewService(cfg Config) *Service {
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	if cfg.PublicAddress == nil {
		cfg.PublicAddress = newCloudflareTrace()
	}
	return &Service{
		cfg: cfg, httpClient: &http.Client{Timeout: 15 * time.Second}, probeClient: freshdns.New().Client(15 * time.Second),
		searchURL: githubSearchUsersURL,
		unlocks:   newFailureLimiter(unlockMaxFailures, unlockWindow),
		exchanges: newFailureLimiter(exchangeMaxFailures, exchangeWindow),
	}
}

// providerClient builds a fresh client from live Settings each call (T3), except cfg.GitHub/Google test overrides.
func (s *Service) providerClient(ctx context.Context, provider Provider) (ProviderClient, error) {
	if provider == ProviderGitHub && s.cfg.GitHub != nil {
		return s.cfg.GitHub, nil
	}
	if provider == ProviderGoogle && s.cfg.Google != nil {
		return s.cfg.Google, nil
	}
	if provider == ProviderDiscord && s.cfg.Discord != nil {
		return s.cfg.Discord, nil
	}
	st, err := s.providerSettings(ctx, provider)
	if err != nil {
		return nil, err
	}
	id, secret := st.OAuthCredentials(provider)
	switch provider {
	case ProviderGoogle:
		return NewHTTPGoogleClient(id, secret, callbackFor(st.InstanceURL, provider), s.httpClient), nil
	case ProviderDiscord:
		return NewHTTPDiscordClient(id, secret, callbackFor(st.InstanceURL, provider), s.httpClient), nil
	}
	return NewHTTPGitHubClient(id, secret, s.httpClient), nil
}

// providerSpec is what differs per provider at the authorize step.
type providerSpec struct {
	authorizeURL string
	scope        string
	callbackPath string
}

// signInProviders is the registry of sign-in providers (ADR 0040); GitHub's callback keeps its original path.
// Sign-in scopes stay minimal: the GitHub connector's broader scope is its own authorize request against the same App, never widened here.
var signInProviders = map[Provider]providerSpec{
	ProviderGitHub:  {authorizeURL: authorizeURL, scope: "read:user", callbackPath: "/auth/callback"},
	ProviderGoogle:  {authorizeURL: googleAuthorizeURL, scope: "openid email profile", callbackPath: "/auth/google/callback"},
	ProviderDiscord: {authorizeURL: discordAuthorizeURL, scope: "identify email", callbackPath: "/auth/discord/callback"},
}

// callbackFor derives provider's redirect URI from the instance URL (T5); must match what's registered exactly.
func callbackFor(instanceURL string, provider Provider) string {
	return strings.TrimSuffix(instanceURL, "/") + signInProviders[provider].callbackPath
}

// providerSettings reads live Settings, failing ErrInvalid unless provider has stored sign-in credentials.
func (s *Service) providerSettings(ctx context.Context, provider Provider) (Settings, error) {
	if _, ok := signInProviders[provider]; !ok {
		return Settings{}, fmt.Errorf("%w: unknown sign-in provider %q", apperrs.ErrInvalid, provider)
	}
	if s.cfg.Settings == nil {
		return Settings{}, oauthNotConfigured(string(provider))
	}
	st, err := s.cfg.Settings.Get(ctx)
	if err != nil {
		return Settings{}, fmt.Errorf("get settings: %w", err)
	}
	if !st.ProviderConfigured(provider) {
		return Settings{}, oauthNotConfigured(string(provider))
	}
	return st, nil
}

// SetDefaultWorkspace wires tenancy's default-workspace binder, since the root builds auth before tenancy.
func (s *Service) SetDefaultWorkspace(b DefaultWorkspaceBinder) {
	s.cfg.DefaultWorkspace = b
}

// SetPendingInviteResolver wires tenancy's pending-invite resolver, same reason as SetDefaultWorkspace.
// SetInstallationClaimer wires the claim of an installation an install link led to.
func (s *Service) SetInstallationClaimer(c InstallationClaimer) {
	s.cfg.Installations = c
}

func (s *Service) SetPendingInviteResolver(r PendingInviteResolver) {
	s.cfg.PendingInvites = r
}

// SetInvitationGate wires invitation admission after the tenancy service is constructed.
func (s *Service) SetInvitationGate(g InvitationGate) {
	s.cfg.Invitations = g
}

// SetOAuthHandoffStore wires the state-bound invitation OAuth handoff store.
func (s *Service) SetOAuthHandoffStore(store OAuthHandoffStore) {
	s.cfg.OAuthHandoffs = store
}

// Configured drives the /auth/github 503 gate; reads live DB state, no restart needed.
func (s *Service) Configured(ctx context.Context) (bool, error) {
	if s.cfg.Settings == nil {
		return false, nil
	}
	st, err := s.cfg.Settings.Get(ctx)
	if err != nil {
		return false, fmt.Errorf("get settings: %w", err)
	}
	return st.Configured(), nil
}

// ProviderConfigured reports whether provider's credentials are stored (ADR 0040), driving each /auth/{provider} gate.
func (s *Service) ProviderConfigured(ctx context.Context, provider Provider) (bool, error) {
	if s.cfg.Settings == nil {
		return false, nil
	}
	st, err := s.cfg.Settings.Get(ctx)
	if err != nil {
		return false, fmt.Errorf("get settings: %w", err)
	}
	return st.ProviderConfigured(provider), nil
}

// SetProviderOAuth stores or clears sign-in credentials (instance:write, ADR 0040); GitHub is refused, half-filled input rejected.
func (s *Service) SetProviderOAuth(ctx context.Context, userID string, provider Provider, clientID, clientSecret string) (Settings, error) {
	if _, ok := signInProviders[provider]; !ok || provider == ProviderGitHub {
		return Settings{}, fmt.Errorf("%w: %q is not a configurable sign-in provider", apperrs.ErrInvalid, provider)
	}
	if err := s.requireAnywhere(ctx, userID, permissions.InstanceWrite); err != nil {
		return Settings{}, err
	}
	clientID = strings.TrimSpace(clientID)
	clientSecret = strings.TrimSpace(clientSecret)
	// Editing an enabled provider's client ID keeps its stored secret, which is never sent back to be re-entered.
	if clientID != "" && clientSecret == "" {
		st, err := s.cfg.Settings.Get(ctx)
		if err != nil {
			return Settings{}, fmt.Errorf("get settings: %w", err)
		}
		_, clientSecret = st.OAuthCredentials(provider)
	}
	if (clientID == "") != (clientSecret == "") {
		return Settings{}, fmt.Errorf("%w: client ID and client secret must be set together (or both empty to disable %s sign-in)", apperrs.ErrInvalid, provider)
	}
	return s.cfg.Settings.SetProviderOAuth(ctx, provider, clientID, clientSecret)
}

// DevLoginEnabled reports whether the GitHub-free dev session endpoint is wired, gating /auth/dev-login registration.
func (s *Service) DevLoginEnabled() bool {
	return s.cfg.DevLogin
}

// DevLogin mints a session for a fixed local identity, skipping the GitHub round trip, via Login's upsert-then-mint.
func (s *Service) DevLogin(ctx context.Context) (string, error) {
	identity := &Identity{UserID: newUserID(), Provider: ProviderDev, ProviderUserID: "dev", Login: "dev", Name: "Dev User"}
	user, err := s.findOrCreateLoginUser(ctx, identity)
	if err != nil {
		return "", err
	}
	return s.CreateSession(ctx, user.ID)
}

// AuthorizeURL builds the GitHub authorize redirect for a fresh state token.
func (s *Service) AuthorizeURL(ctx context.Context, state string) (string, error) {
	return s.AuthorizeURLFor(ctx, ProviderGitHub, state)
}

// AuthorizeURLFor reads the client ID live (T3) and derives redirect_uri (T5).
func (s *Service) AuthorizeURLFor(ctx context.Context, provider Provider, state string) (string, error) {
	st, err := s.providerSettings(ctx, provider)
	if err != nil {
		return "", err
	}
	spec := signInProviders[provider]
	clientID, _ := st.OAuthCredentials(provider)
	q := url.Values{
		"client_id":     {clientID},
		"redirect_uri":  {callbackFor(st.InstanceURL, provider)},
		"response_type": {"code"},
		"scope":         {spec.scope},
		"state":         {state},
	}
	return spec.authorizeURL + "?" + q.Encode(), nil
}

// NewState returns a cryptographically random CSRF state token.
func (s *Service) NewState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate oauth state: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// Login is LoginWith for GitHub.
func (s *Service) Login(ctx context.Context, code string) (string, error) {
	return s.LoginWith(ctx, ProviderGitHub, code)
}

// LoginWith exchanges a code for identity, admits only known active users or the first instance user, and mints a session.
func (s *Service) LoginWith(ctx context.Context, provider Provider, code string) (string, error) {
	client, err := s.providerClient(ctx, provider)
	if err != nil {
		return "", err
	}
	accessToken, err := client.Exchange(ctx, code)
	if err != nil {
		return "", err
	}
	pu, err := client.FetchUser(ctx, accessToken)
	if err != nil {
		return "", err
	}
	user, err := s.findOrCreateLoginUser(ctx, providerIdentity(newUserID(), provider, pu))
	if err != nil {
		return "", err
	}
	return s.CreateSession(ctx, user.ID)
}

func providerIdentity(userID string, provider Provider, pu *ProviderUser) *Identity {
	return &Identity{UserID: userID, Provider: provider, ProviderUserID: pu.ID, Login: pu.Login, Name: pu.Name, AvatarURL: pu.AvatarURL}
}

func (s *Service) findOrCreateLoginUser(ctx context.Context, identity *Identity) (*User, error) {
	user, err := s.cfg.Users.GetUserByProvider(ctx, identity.Provider, identity.ProviderUserID)
	if err == nil {
		if user.AccountStatus == AccountRemoved {
			return nil, errAccountRemoved
		}
		if !accountIsActive(user.AccountStatus) {
			return nil, errAccountDisabled
		}
		updated, _, err := s.cfg.Users.UpsertUser(ctx, identity)
		if err != nil {
			return nil, fmt.Errorf("sync user: %w", err)
		}
		return updated, nil
	}
	if !errors.Is(err, apperrs.ErrNotFound) {
		return nil, fmt.Errorf("find user by provider: %w", err)
	}
	count, err := s.cfg.Users.CountUsers(ctx)
	if err != nil {
		return nil, fmt.Errorf("count users: %w", err)
	}
	if count != 0 {
		return nil, errInvitationRequired
	}
	user, err = s.cfg.Users.CreateFirstUser(ctx, identity, eventbus.OutboxEvent{ID: newUserID(), Topic: TopicAccountAdmitted, Payload: AccountLifecycleEvent{AccountID: identity.UserID}})
	if err != nil {
		if errors.Is(err, apperrs.ErrConflict) {
			return nil, errInvitationRequired
		}
		return nil, fmt.Errorf("create first user: %w", err)
	}
	// Every pass already stops at the first user; this clears the code and the installer's file with it.
	if err := s.closeSetup(ctx); err != nil {
		logging.FromCtx(ctx).Warn("close first run", "error", err)
	}
	return user, nil
}

func accountIsActive(status AccountStatus) bool {
	return status == "" || status == AccountActive
}

// Whoami returns the caller's own account, so an MCP client can confirm which user its token acts as.
func (s *Service) Whoami(ctx context.Context, userID string) (*User, error) {
	if userID == "" {
		return nil, fmt.Errorf("%w: user is required", apperrs.ErrUnauthorized)
	}
	user, err := s.cfg.Users.GetUserByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get user %s: %w", userID, err)
	}
	return user, nil
}

// Me returns the user plus which onboarding wizard applies: owner wizard if fresh, first-login otherwise.
func (s *Service) Me(ctx context.Context, userID string) (*OnboardingStatus, error) {
	user, err := s.cfg.Users.GetUserByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get user %s: %w", userID, err)
	}
	owners, err := s.cfg.Users.ListActiveOwnerIDs(ctx)
	if err != nil {
		return nil, fmt.Errorf("list owners: %w", err)
	}
	held, err := s.permissionsAnywhere(ctx, userID)
	if err != nil {
		return nil, err
	}
	st := &OnboardingStatus{User: user, InstancePermissions: held}
	if len(owners) == 0 {
		st.NeedsOwnerWizard = true
		return st, nil
	}
	st.NeedsFirstLoginWizard = !user.FirstLoginDone
	return st, nil
}

// requireAnywhere refuses userID unless they hold action in a workspace they belong to (ADR 0087), which an Owner
// always does; nothing else grants instance-level power (ADR 0088).
func (s *Service) requireAnywhere(ctx context.Context, userID string, action permissions.Action) error {
	if s.cfg.Permissions == nil {
		return fmt.Errorf("%w: no permission gate wired", apperrs.ErrForbidden)
	}
	held, err := s.cfg.Permissions.HoldsAnywhere(ctx, userID, action)
	if err != nil {
		return fmt.Errorf("check %s: %w", action, err)
	}
	if !held {
		return fmt.Errorf("%w: %s required", apperrs.ErrForbidden, action)
	}
	return nil
}

func (s *Service) permissionsAnywhere(ctx context.Context, userID string) ([]string, error) {
	if s.cfg.Permissions == nil {
		return nil, nil
	}
	held, err := s.cfg.Permissions.PermissionsAnywhere(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list permissions of %s: %w", userID, err)
	}
	return held, nil
}

// mac returns the HMAC-SHA256 of enc keyed by the shared secret, base64url.
func (s *Service) mac(enc string) string {
	m := hmac.New(sha256.New, s.cfg.Secret)
	_, _ = m.Write([]byte(enc))
	return base64.RawURLEncoding.EncodeToString(m.Sum(nil))
}

func validateInstanceURL(raw string) error {
	raw = strings.TrimSpace(raw)
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return fmt.Errorf("%w: instance URL must be an absolute http(s) URL", apperrs.ErrInvalid)
	}
	return nil
}

func newUserID() string {
	id, err := uuid.NewV7()
	if err != nil {
		return uuid.NewString()
	}
	return id.String()
}
