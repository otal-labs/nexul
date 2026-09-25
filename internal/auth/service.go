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
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/oauthx"
)

const (
	authorizeURL         = "https://github.com/login/oauth/authorize"
	tokenURL             = "https://github.com/login/oauth/access_token"
	userURL              = "https://api.github.com/user"
	githubSearchUsersURL = "https://api.github.com/search/users"
	stateCookie          = "nexul_oauth_state"
	stateMaxAge          = 10 * time.Minute
	defaultTokenTTL      = 24 * time.Hour
)

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
		return "", fmt.Errorf("%w: GitHub OAuth is not configured", apperrs.ErrInvalid)
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
	// MentionLayout gates SetMentionChipTemplate on workspaces:write; nil fails closed with ErrForbidden.
	MentionLayout MentionLayoutGate
	// DefaultWorkspace binds the wizard's completing user to the default workspace; wired later via SetDefaultWorkspace.
	DefaultWorkspace DefaultWorkspaceBinder
	// PendingInvites resolves pending invites the moment a User row is created; wired later, same as DefaultWorkspace.
	PendingInvites PendingInviteResolver
	// ConnectorApps seeds github's app-level OAuth registration during Bootstrap, before Settings is reachable (ADR 0017).
	ConnectorApps ConnectorAppSeeder
	// GitHubApp checks the pasted App credentials against GitHub before Bootstrap stores them; nil skips (tests).
	GitHubApp GitHubAppVerifier
	Now       func() time.Time
	TokenTTL  time.Duration
	// DevLogin enables GitHub-free session minting at /auth/dev-login for local dev; must be false in production.
	DevLogin bool
}

// Service is the auth use-case layer: OAuth, session tokens, and the user/onboarding/allowlist use-cases.
type Service struct {
	cfg Config
	// httpClient is the shared transport reused per request (T3), avoiding a new *http.Client per login.
	httpClient *http.Client
	// searchURL is GitHub's user-search endpoint, overridable in tests like HTTPGitHubClient.userURL.
	searchURL string
}

// NewService wires the auth use-cases.
func NewService(cfg Config) *Service {
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	if cfg.TokenTTL <= 0 {
		cfg.TokenTTL = defaultTokenTTL
	}
	return &Service{cfg: cfg, httpClient: &http.Client{Timeout: 15 * time.Second}, searchURL: githubSearchUsersURL}
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
		return Settings{}, fmt.Errorf("%w: %s OAuth is not configured", apperrs.ErrInvalid, provider)
	}
	st, err := s.cfg.Settings.Get(ctx)
	if err != nil {
		return Settings{}, fmt.Errorf("get settings: %w", err)
	}
	if !st.ProviderConfigured(provider) {
		return Settings{}, fmt.Errorf("%w: %s OAuth is not configured", apperrs.ErrInvalid, provider)
	}
	return st, nil
}

// SetDefaultWorkspace wires tenancy's default-workspace binder, since the root builds auth before tenancy.
func (s *Service) SetDefaultWorkspace(b DefaultWorkspaceBinder) {
	s.cfg.DefaultWorkspace = b
}

// SetPendingInviteResolver wires tenancy's pending-invite resolver, same reason as SetDefaultWorkspace.
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

// SetProviderOAuth stores or clears sign-in credentials (owner, ADR 0040); GitHub is refused, half-filled input rejected.
func (s *Service) SetProviderOAuth(ctx context.Context, userID string, provider Provider, clientID, clientSecret string) (Settings, error) {
	if _, ok := signInProviders[provider]; !ok || provider == ProviderGitHub {
		return Settings{}, fmt.Errorf("%w: %q is not a configurable sign-in provider", apperrs.ErrInvalid, provider)
	}
	if err := s.requireCanCreateWorkspace(ctx, userID); err != nil {
		return Settings{}, err
	}
	clientID = strings.TrimSpace(clientID)
	clientSecret = strings.TrimSpace(clientSecret)
	if (clientID == "") != (clientSecret == "") {
		return Settings{}, fmt.Errorf("%w: client ID and client secret must be set together (or both empty to disable %s sign-in)", apperrs.ErrInvalid, provider)
	}
	return s.cfg.Settings.SetProviderOAuth(ctx, provider, clientID, clientSecret)
}

// DevLoginEnabled reports whether the GitHub-free dev session endpoint is wired, gating /auth/dev-login registration.
func (s *Service) DevLoginEnabled() bool {
	return s.cfg.DevLogin
}

// DevLogin mints a session for a fixed local identity, skipping the GitHub round trip, via Login's upsert-then-sign.
func (s *Service) DevLogin(ctx context.Context) (string, error) {
	identity := &User{
		ID:             newUserID(),
		Provider:       ProviderDev,
		ProviderUserID: "dev",
		Login:          "dev",
		Name:           "Dev User",
	}
	user, err := s.findOrCreateLoginUser(ctx, identity)
	if err != nil {
		return "", err
	}
	return s.Sign(user.ID)
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

// StartInvitationOAuth validates a bearer invitation and creates a state-bound OAuth handoff.
func (s *Service) StartInvitationOAuth(ctx context.Context, provider Provider, rawToken string) (*InvitationOAuthStart, error) {
	if s.cfg.Invitations == nil || s.cfg.OAuthHandoffs == nil {
		return nil, fmt.Errorf("%w: invitation OAuth is unavailable", apperrs.ErrInvalid)
	}
	if strings.TrimSpace(rawToken) == "" {
		return nil, invalidInvitationError()
	}
	if _, ok := signInProviders[provider]; !ok {
		return nil, invalidInvitationError()
	}
	invitation, err := s.cfg.Invitations.GetInvitationByToken(ctx, rawToken, s.cfg.Now())
	if err != nil {
		return nil, classifyInvitationError(err)
	}
	state, err := s.NewState()
	if err != nil {
		return nil, err
	}
	stateHash := hashCredential(state)
	url, err := s.AuthorizeURLFor(ctx, provider, state)
	if err != nil {
		return nil, err
	}
	handoff := &OAuthHandoff{ID: newUserID(), InvitationID: invitation.InvitationID, OAuthStateHash: stateHash, Provider: provider, CreatedAt: s.cfg.Now(), ExpiresAt: s.cfg.Now().Add(stateMaxAge)}
	if err := s.cfg.OAuthHandoffs.StartOAuthHandoff(ctx, handoff); err != nil {
		return nil, fmt.Errorf("start invitation OAuth: %w", err)
	}
	return &InvitationOAuthStart{URL: url, State: state, CookieName: invitationStateCookie(stateHash)}, nil
}

// CompleteInvitationOAuth authenticates an invitation handoff without creating or mutating a User.
func (s *Service) CompleteInvitationOAuth(ctx context.Context, provider Provider, state, code string) (string, error) {
	if s.cfg.OAuthHandoffs == nil {
		return "", invalidInvitationError()
	}
	if strings.TrimSpace(state) == "" || strings.TrimSpace(code) == "" {
		return "", invalidInvitationError()
	}
	client, err := s.providerClient(ctx, provider)
	if err != nil {
		return "", err
	}
	accessToken, err := client.Exchange(ctx, code)
	if err != nil {
		return "", err
	}
	providerUser, err := client.FetchUser(ctx, accessToken)
	if err != nil {
		return "", err
	}
	existingID := ""
	if user, lookupErr := s.cfg.Users.GetUserByProvider(ctx, provider, providerUser.ID); lookupErr == nil {
		existingID = user.ID
	} else if !errors.Is(lookupErr, apperrs.ErrNotFound) {
		return "", fmt.Errorf("find invitation user: %w", lookupErr)
	}
	acceptance, err := newCredential()
	if err != nil {
		return "", err
	}
	identity := OAuthHandoffIdentity{Provider: provider, ProviderUserID: providerUser.ID, Login: providerUser.Login, Name: providerUser.Name, AvatarURL: providerUser.AvatarURL, ExistingUserID: existingID}
	if _, err := s.cfg.OAuthHandoffs.CompleteOAuthCallback(ctx, hashCredential(state), hashCredential(acceptance), identity, s.cfg.Now().Add(stateMaxAge), s.cfg.Now()); err != nil {
		return "", classifyInvitationError(err)
	}
	return acceptance, nil
}

// PrepareAuthenticatedAcceptance creates an acceptance handoff for an active user without OAuth.
func (s *Service) PrepareAuthenticatedAcceptance(ctx context.Context, userID, rawToken string) (*InvitationAcceptance, error) {
	if s.cfg.Invitations == nil || s.cfg.OAuthHandoffs == nil {
		return nil, fmt.Errorf("%w: invitation acceptance is unavailable", apperrs.ErrInvalid)
	}
	user, err := s.cfg.Users.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if !accountIsActive(user.AccountStatus) {
		return nil, apperrs.ErrUnauthorized
	}
	details, err := s.cfg.Invitations.GetInvitationByToken(ctx, rawToken, s.cfg.Now())
	if err != nil {
		return nil, classifyInvitationError(err)
	}
	state, err := s.NewState()
	if err != nil {
		return nil, err
	}
	acceptance, err := newCredential()
	if err != nil {
		return nil, err
	}
	handoff := &OAuthHandoff{ID: newUserID(), InvitationID: details.InvitationID, OAuthStateHash: hashCredential(state), Provider: user.Provider, CreatedAt: s.cfg.Now(), ExpiresAt: s.cfg.Now().Add(stateMaxAge)}
	if err := s.cfg.OAuthHandoffs.StartOAuthHandoff(ctx, handoff); err != nil {
		return nil, err
	}
	identity := OAuthHandoffIdentity{Provider: user.Provider, ProviderUserID: user.ProviderUserID, Login: user.Login, Name: user.Name, AvatarURL: user.AvatarURL, ExistingUserID: user.ID}
	if _, err := s.cfg.OAuthHandoffs.CompleteOAuthCallback(ctx, hashCredential(state), hashCredential(acceptance), identity, s.cfg.Now().Add(stateMaxAge), s.cfg.Now()); err != nil {
		return nil, classifyInvitationError(err)
	}
	details.AuthenticatedUser = &InvitationAuthenticatedUser{ID: user.ID, Provider: user.Provider, Login: user.Login, Name: user.Name, AvatarURL: user.AvatarURL}
	return s.acceptanceDetails(acceptance, details)
}

// GetInvitationAcceptance resolves a short-lived acceptance credential without mutating membership.
func (s *Service) GetInvitationAcceptance(ctx context.Context, acceptance string) (*InvitationAcceptance, error) {
	if s.cfg.Invitations == nil || s.cfg.OAuthHandoffs == nil {
		return nil, invalidInvitationError()
	}
	handoff, err := s.cfg.OAuthHandoffs.GetOAuthHandoffByAcceptanceHash(ctx, hashCredential(acceptance), s.cfg.Now())
	if err != nil {
		return nil, classifyInvitationError(err)
	}
	details, err := s.cfg.Invitations.GetInvitationByAcceptance(ctx, hashCredential(acceptance), s.cfg.Now())
	if err != nil {
		return nil, classifyInvitationError(err)
	}
	return s.acceptanceDetails(acceptance, detailsWithHandoff(details, handoff))
}

func (s *Service) acceptanceDetails(acceptance string, details *InvitationAcceptance) (*InvitationAcceptance, error) {
	if details == nil {
		return nil, invalidInvitationError()
	}
	details.AcceptanceToken = acceptance
	return details, nil
}

func detailsWithHandoff(details *InvitationAcceptance, handoff *OAuthHandoff) *InvitationAcceptance {
	if handoff == nil {
		return details
	}
	details.AuthenticatedUser = &InvitationAuthenticatedUser{ID: handoff.ExistingUserID, Provider: handoff.Provider, Login: handoff.Login, Name: handoff.Name, AvatarURL: handoff.AvatarURL}
	if handoff.AdmittedUserID != "" {
		details.AuthenticatedUser.ID = handoff.AdmittedUserID
	}
	return details
}

// RedeemInvitation admits the callback identity and returns a normal session token.
func (s *Service) RedeemInvitation(ctx context.Context, acceptance string) (InvitationRedeemResult, error) {
	if s.cfg.Invitations == nil || s.cfg.OAuthHandoffs == nil {
		return InvitationRedeemResult{}, invalidInvitationError()
	}
	handoff, err := s.cfg.OAuthHandoffs.GetOAuthHandoffByAcceptanceHash(ctx, hashCredential(acceptance), s.cfg.Now())
	if err != nil {
		return InvitationRedeemResult{}, classifyInvitationError(err)
	}
	if handoff.ProviderUserID == "" {
		return InvitationRedeemResult{}, invalidInvitationError()
	}
	identity := InvitationIdentity{ID: newUserID(), Provider: handoff.Provider, ProviderUserID: handoff.ProviderUserID, Login: handoff.Login, Name: handoff.Name, AvatarURL: handoff.AvatarURL}
	if handoff.ExistingUserID != "" {
		identity.ID = handoff.ExistingUserID
	}
	admission, err := s.cfg.Invitations.RedeemInvitation(ctx, hashCredential(acceptance), identity, s.cfg.Now())
	if err != nil {
		return InvitationRedeemResult{}, classifyInvitationError(err)
	}
	token, err := s.Sign(admission.UserID)
	if err != nil {
		return InvitationRedeemResult{}, err
	}
	return InvitationRedeemResult{Token: token, WorkspaceIDs: admission.WorkspaceIDs}, nil
}

func newCredential() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate credential: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func hashCredential(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func invitationStateCookie(stateHash string) string {
	return "nexul_invite_oauth_" + stateHash[:16]
}

func invalidInvitationError() error {
	return fmt.Errorf("%w: invitation is invalid or expired", apperrs.ErrNotFound)
}

func classifyInvitationError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, apperrs.ErrNotFound) {
		return invalidInvitationError()
	}
	if errors.Is(err, apperrs.ErrInvalid) {
		message := strings.ToLower(err.Error())
		if strings.Contains(message, "invitation") || strings.Contains(message, "credential") || strings.Contains(message, "acceptance") {
			return invalidInvitationError()
		}
	}
	return err
}

// Login is LoginWith for GitHub.
func (s *Service) Login(ctx context.Context, code string) (string, error) {
	return s.LoginWith(ctx, ProviderGitHub, code)
}

// LoginWith exchanges a code for identity, admits only known active users or the first instance user, and mints a token.
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
	user, err := s.findOrCreateLoginUser(ctx, &User{
		ID:             newUserID(),
		Provider:       provider,
		ProviderUserID: pu.ID,
		Login:          pu.Login,
		Name:           pu.Name,
		AvatarURL:      pu.AvatarURL,
	})
	if err != nil {
		return "", err
	}
	return s.Sign(user.ID)
}

func (s *Service) findOrCreateLoginUser(ctx context.Context, identity *User) (*User, error) {
	user, err := s.cfg.Users.GetUserByProvider(ctx, identity.Provider, identity.ProviderUserID)
	if err == nil {
		if !accountIsActive(user.AccountStatus) {
			return nil, fmt.Errorf("%w: account is not active", apperrs.ErrUnauthorized)
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
		return nil, fmt.Errorf("%w: invitation required", apperrs.ErrUnauthorized)
	}
	user, err = s.cfg.Users.CreateFirstUser(ctx, identity, eventbus.OutboxEvent{ID: newUserID(), Topic: TopicAccountAdmitted, Payload: AccountLifecycleEvent{AccountID: identity.ID}})
	if err != nil {
		if errors.Is(err, apperrs.ErrConflict) {
			return nil, fmt.Errorf("%w: invitation required", apperrs.ErrUnauthorized)
		}
		return nil, fmt.Errorf("create first user: %w", err)
	}
	return user, nil
}

func accountIsActive(status AccountStatus) bool {
	return status == "" || status == AccountActive
}

// Sign mints a session token for a user ID, valid for TokenTTL (ADR 0041).
func (s *Service) Sign(userID string) (string, error) {
	payload, err := json.Marshal(tokenClaims{
		UserID: userID,
		Exp:    s.cfg.Now().Add(s.cfg.TokenTTL).Unix(),
	})
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}
	enc := base64.RawURLEncoding.EncodeToString(payload)
	return enc + "." + s.mac(enc), nil
}

// Verify validates a session token and returns the user ID it names.
func (s *Service) Verify(token string) (string, error) {
	enc, sig, ok := strings.Cut(token, ".")
	if !ok {
		return "", apperrs.ErrUnauthorized
	}
	if !hmac.Equal([]byte(sig), []byte(s.mac(enc))) {
		return "", apperrs.ErrUnauthorized
	}
	payload, err := base64.RawURLEncoding.DecodeString(enc)
	if err != nil {
		return "", apperrs.ErrUnauthorized
	}
	var claims tokenClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return "", apperrs.ErrUnauthorized
	}
	if s.cfg.Now().Unix() >= claims.Exp {
		return "", apperrs.ErrUnauthorized
	}
	if claims.UserID == "" {
		return "", apperrs.ErrUnauthorized
	}
	return claims.UserID, nil
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
	adminExists, err := s.cfg.Users.CanCreateWorkspaceExists(ctx)
	if err != nil {
		return nil, fmt.Errorf("check instance admin: %w", err)
	}
	st := &OnboardingStatus{User: user}
	if !adminExists && !user.CanCreateWorkspace {
		st.NeedsOwnerWizard = true
		return st, nil
	}
	st.NeedsFirstLoginWizard = !user.FirstLoginDone
	return st, nil
}

// GitHubAppVerifier is the live "does this App exist and accept this secret" check, adapted at the composition root.
type GitHubAppVerifier interface {
	VerifyGitHubApp(ctx context.Context, clientID, clientSecret, appSlug string) error
	// VerifyGitHubAppCheck runs one named half ("slug" or "secret") so /setup can light each row separately.
	VerifyGitHubAppCheck(ctx context.Context, check, clientID, clientSecret, appSlug string) error
}

// CompleteOwnerWizard binds the caller to the workspace's Owner role, grants can_create_workspace; a repeat conflicts.
func (s *Service) CompleteOwnerWizard(ctx context.Context, userID, instanceURL string) error {
	if err := validateInstanceURL(instanceURL); err != nil {
		return err
	}
	user, err := s.cfg.Users.GetUserByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("get user %s: %w", userID, err)
	}
	adminExists, err := s.cfg.Users.CanCreateWorkspaceExists(ctx)
	if err != nil {
		return fmt.Errorf("check instance admin: %w", err)
	}
	// Idempotent for the admin themself: a double-submitted finish re-runs the grant instead of conflicting.
	if adminExists && !user.CanCreateWorkspace {
		return fmt.Errorf("%w: this instance already has an owner", apperrs.ErrConflict)
	}
	if _, err := s.cfg.Settings.Set(ctx, instanceURL); err != nil {
		return fmt.Errorf("set instance url: %w", err)
	}
	// Bind to the pre-seeded default workspace as its Owner-role member instead of creating a second, duplicate workspace.
	if err := s.cfg.DefaultWorkspace.BindDefaultWorkspaceOwner(ctx, user.ID); err != nil {
		return fmt.Errorf("bind default workspace owner: %w", err)
	}
	if err := s.cfg.Users.SetCanCreateWorkspace(ctx, user.ID, true); err != nil {
		return fmt.Errorf("grant can_create_workspace: %w", err)
	}
	if err := s.cfg.Users.MarkFirstLoginDone(ctx, user.ID); err != nil {
		return fmt.Errorf("mark first login: %w", err)
	}
	return nil
}

// CompleteFirstLogin only completes the profile; identity stays provider-sourced.
func (s *Service) CompleteFirstLogin(ctx context.Context, userID string) error {
	return s.cfg.Users.MarkFirstLoginDone(ctx, userID)
}

// Bootstrap stores the instance URL and GitHub App credentials on a fresh instance, the one unauthenticated way in.
func (s *Service) Bootstrap(ctx context.Context, instanceURL, clientID, clientSecret, appSlug string) (Settings, error) {
	if err := validateInstanceURL(instanceURL); err != nil {
		return Settings{}, err
	}
	if err := s.bootstrapAllowed(ctx, clientID, clientSecret, appSlug); err != nil {
		return Settings{}, err
	}
	if s.cfg.GitHubApp != nil {
		if err := s.cfg.GitHubApp.VerifyGitHubApp(ctx, clientID, clientSecret, appSlug); err != nil {
			return Settings{}, err
		}
	}
	if _, err := s.cfg.Settings.Set(ctx, instanceURL); err != nil {
		return Settings{}, fmt.Errorf("set instance url: %w", err)
	}
	st, err := s.cfg.Settings.SetGitHubOAuth(ctx, clientID, clientSecret)
	if err != nil {
		return Settings{}, fmt.Errorf("set github oauth: %w", err)
	}
	// Same GitHub App backs the connectors' install flow, so seed its config now for Connect without a Settings visit.
	if s.cfg.ConnectorApps != nil {
		if err := s.cfg.ConnectorApps.SeedGitHubApp(ctx, clientID, clientSecret, appSlug); err != nil {
			return Settings{}, fmt.Errorf("seed github connector app: %w", err)
		}
	}
	return st, nil
}

// BootstrapVerify runs one named GitHub App check without storing anything, under Bootstrap's own guards so a live instance never proxies GitHub for strangers.
func (s *Service) BootstrapVerify(ctx context.Context, check, clientID, clientSecret, appSlug string) error {
	if err := s.bootstrapAllowed(ctx, clientID, clientSecret, appSlug); err != nil {
		return err
	}
	if s.cfg.GitHubApp == nil {
		return nil
	}
	return s.cfg.GitHubApp.VerifyGitHubAppCheck(ctx, check, clientID, clientSecret, appSlug)
}

// bootstrapAllowed is the shared gate for Bootstrap and BootstrapVerify: fields present, slug not the numeric App ID, and the instance still replaceable.
func (s *Service) bootstrapAllowed(ctx context.Context, clientID, clientSecret, appSlug string) error {
	if clientID == "" || clientSecret == "" || appSlug == "" {
		return fmt.Errorf("%w: client ID, client secret and app slug are required", apperrs.ErrInvalid)
	}
	// GitHub's install URL takes the URL slug, not the numeric "App ID"; a digits-only value is that ID pasted by mistake.
	if strings.Trim(appSlug, "0123456789") == "" {
		return fmt.Errorf("%w: that looks like the numeric App ID — the app slug is the name in your app's URL (github.com/apps/{slug})", apperrs.ErrInvalid)
	}
	return s.bootstrapReplaceable(ctx)
}

// bootstrapReplaceable fails ErrConflict once the instance is configured and a user has logged in.
func (s *Service) bootstrapReplaceable(ctx context.Context) error {
	st, err := s.cfg.Settings.Get(ctx)
	if err != nil {
		return fmt.Errorf("get settings: %w", err)
	}
	if !st.Configured() {
		return nil
	}
	reconfigurable, err := s.bootstrapReconfigurable(ctx)
	if err != nil {
		return err
	}
	if !reconfigurable {
		return fmt.Errorf("%w: this instance is already configured", apperrs.ErrConflict)
	}
	return nil
}

// InstanceURL returns the configured instance URL, or "" when none is stored or Settings is unwired.
func (s *Service) InstanceURL(ctx context.Context) string {
	if s.cfg.Settings == nil {
		return ""
	}
	st, err := s.cfg.Settings.Get(ctx)
	if err != nil {
		return ""
	}
	return st.InstanceURL
}

func (s *Service) secureCookie(ctx context.Context, requestTLS bool) bool {
	if requestTLS {
		return true
	}
	if s.cfg.Settings == nil {
		return false
	}
	settings, err := s.cfg.Settings.Get(ctx)
	if err != nil {
		return false
	}
	parsed, err := url.Parse(settings.InstanceURL)
	return err == nil && strings.EqualFold(parsed.Scheme, "https")
}

// VerifyInstanceURL is the bootstrap ticker's reachability row: this server fetches its own bootstrap-status through
// the pasted URL, proving DNS, the proxy, and TLS before the URL is stored and every callback gets derived from it.
// Gated like the other checks, so a live instance never fetches URLs on a stranger's behalf.
func (s *Service) VerifyInstanceURL(ctx context.Context, instanceURL string) (err error) {
	if err := validateInstanceURL(instanceURL); err != nil {
		return err
	}
	if err := s.bootstrapReplaceable(ctx); err != nil {
		return err
	}
	target := strings.TrimSuffix(strings.TrimSpace(instanceURL), "/") + "/api/auth/bootstrap-status"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return fmt.Errorf("%w: %v", apperrs.ErrInvalid, err)
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("%w: this server could not reach %s: %v", apperrs.ErrInvalid, instanceURL, err)
	}
	defer func() {
		err = errors.Join(err, resp.Body.Close())
	}()
	// ponytail: any Nexul answers this shape, not provably this one; add a per-process nonce if that ever matters.
	var body struct {
		Configured *bool `json:"configured"`
	}
	if resp.StatusCode != http.StatusOK || json.NewDecoder(io.LimitReader(resp.Body, 4096)).Decode(&body) != nil || body.Configured == nil {
		return fmt.Errorf("%w: %s answered, but not as a Nexul server (status %d) — check that the proxy forwards /api to it", apperrs.ErrInvalid, instanceURL, resp.StatusCode)
	}
	return nil
}

// bootstrapReconfigurable is true until the first login lands: a wrong App can be re-entered instead of bricking the instance.
func (s *Service) bootstrapReconfigurable(ctx context.Context) (bool, error) {
	users, err := s.cfg.Users.ListUsers(ctx)
	if err != nil {
		return false, fmt.Errorf("list users: %w", err)
	}
	return len(users) == 0, nil
}

// BootstrapReconfigurable reports whether Bootstrap may overwrite the stored App (no user has logged in yet).
func (s *Service) BootstrapReconfigurable(ctx context.Context) (bool, error) {
	return s.bootstrapReconfigurable(ctx)
}

// maxAvatarOverrideBytes caps a decoded avatar override data URI at ~10MB; there's no file-upload/object-storage here.
const maxAvatarOverrideBytes = 10 * 1024 * 1024

// UpdateProfileOverride sets the caller's display name/avatar override; empty clears back to provider-sourced.
func (s *Service) UpdateProfileOverride(ctx context.Context, userID, displayName, avatarOverrideURL string) (*User, error) {
	displayName = strings.TrimSpace(displayName)
	if avatarOverrideURL != "" {
		if err := validateAvatarOverrideURL(avatarOverrideURL); err != nil {
			return nil, err
		}
	}
	var namePtr, avatarPtr *string
	if displayName != "" {
		namePtr = &displayName
	}
	if avatarOverrideURL != "" {
		avatarPtr = &avatarOverrideURL
	}
	if err := s.cfg.Users.SetProfileOverride(ctx, userID, namePtr, avatarPtr); err != nil {
		return nil, fmt.Errorf("set profile override %s: %w", userID, err)
	}
	return s.cfg.Users.GetUserByID(ctx, userID)
}

// validateAvatarOverrideURL requires a base64 data: URI decoding to no more than maxAvatarOverrideBytes.
func validateAvatarOverrideURL(raw string) error {
	if !strings.HasPrefix(raw, "data:") {
		return fmt.Errorf("%w: avatar_override_url must be a data: URI", apperrs.ErrInvalid)
	}
	meta, data, ok := strings.Cut(strings.TrimPrefix(raw, "data:"), ",")
	if !ok || !strings.Contains(meta, "base64") {
		return fmt.Errorf("%w: avatar_override_url must be a base64-encoded data: URI", apperrs.ErrInvalid)
	}
	decoded, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return fmt.Errorf("%w: avatar_override_url has an invalid base64 payload", apperrs.ErrInvalid)
	}
	if len(decoded) > maxAvatarOverrideBytes {
		return fmt.Errorf("%w: avatar image exceeds the %dMB limit", apperrs.ErrInvalid, maxAvatarOverrideBytes/(1024*1024))
	}
	return nil
}

// UpdateInstanceURL changes the instance URL (owner only), bumping the version so new connection tokens carry it.
func (s *Service) UpdateInstanceURL(ctx context.Context, userID, instanceURL string) (Settings, error) {
	if err := validateInstanceURL(instanceURL); err != nil {
		return Settings{}, err
	}
	if err := s.requireCanCreateWorkspace(ctx, userID); err != nil {
		return Settings{}, err
	}
	return s.cfg.Settings.Set(ctx, instanceURL)
}

// SetMentionChipTemplate updates the mention chip layout, requiring workspaces:write; bad tokens pass through.
func (s *Service) SetMentionChipTemplate(ctx context.Context, userID, template string) (Settings, error) {
	if err := s.requireManageMentionLayout(ctx, userID); err != nil {
		return Settings{}, err
	}
	return s.cfg.Settings.SetMentionChipTemplate(ctx, template)
}

func (s *Service) requireManageMentionLayout(ctx context.Context, userID string) error {
	if s.cfg.MentionLayout == nil || !s.cfg.MentionLayout.CanManageMentionLayout(ctx, userID) {
		return fmt.Errorf("%w: workspaces:write required", apperrs.ErrForbidden)
	}
	return nil
}

// GenerateConnectionToken mints a connection token from the instance settings (owner only); carries server info only.
func (s *Service) GenerateConnectionToken(ctx context.Context, userID string) (*ConnectionToken, error) {
	if err := s.requireCanCreateWorkspace(ctx, userID); err != nil {
		return nil, err
	}
	st, err := s.cfg.Settings.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("get settings: %w", err)
	}
	if st.InstanceURL == "" {
		return nil, fmt.Errorf("%w: set the instance URL (owner wizard) before generating a connection token", apperrs.ErrInvalid)
	}
	token, err := s.signConnectionToken(st)
	if err != nil {
		return nil, err
	}
	return &ConnectionToken{
		Token:           token,
		InstanceURL:     st.InstanceURL,
		MCPURL:          mcpURLFor(st.InstanceURL),
		SettingsVersion: st.SettingsVersion,
		ExpiresAt:       s.cfg.Now().Add(connectionTokenTTL),
	}, nil
}

// CanCreateWorkspaceExists reports whether a user already holds can_create_workspace, via a consumer-side interface.
func (s *Service) CanCreateWorkspaceExists(ctx context.Context) (bool, error) {
	return s.cfg.Users.CanCreateWorkspaceExists(ctx)
}

// GetUserByID resolves a user record for consumers of the owner role.
func (s *Service) GetUserByID(ctx context.Context, id string) (*User, error) {
	return s.cfg.Users.GetUserByID(ctx, id)
}

// ListUsers returns every user record, for access's permissions modal user picker (via the access.Users interface).
func (s *Service) ListUsers(ctx context.Context) ([]*User, error) {
	return s.cfg.Users.ListUsers(ctx)
}

// GetAccount returns the caller's own account for an empty or own id; any other account only to an instance administrator.
func (s *Service) GetAccount(ctx context.Context, actorID, id string) (*User, error) {
	if id == "" || id == actorID {
		return s.Whoami(ctx, actorID)
	}
	if err := s.requireCanCreateWorkspace(ctx, actorID); err != nil {
		return nil, err
	}
	user, err := s.cfg.Users.GetUserByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get account %s: %w", id, err)
	}
	return user, nil
}

// UpdateAccountStatus moves an account to active or disabled; active reactivates a disabled account and restores a removed one.
func (s *Service) UpdateAccountStatus(ctx context.Context, actorID, targetID string, status AccountStatus) error {
	// The admin check comes first, so a non-admin cannot tell a real account id from an unknown one.
	if err := s.requireCanCreateWorkspace(ctx, actorID); err != nil {
		return err
	}
	target, err := s.cfg.Users.GetUserByID(ctx, targetID)
	if err != nil {
		return fmt.Errorf("get account %s: %w", targetID, err)
	}
	if status == AccountDisabled && target.AccountStatus == AccountDisabled {
		return nil
	}
	if status == AccountDisabled {
		return s.DisableAccount(ctx, actorID, target.ID)
	}
	if status != AccountActive {
		return fmt.Errorf("%w: account status must be active or disabled", apperrs.ErrInvalid)
	}
	if target.AccountStatus == AccountDisabled {
		return s.ReactivateAccount(ctx, actorID, target.ID)
	}
	if target.AccountStatus == AccountRemoved {
		return s.RestoreAccount(ctx, actorID, target.ID)
	}
	return nil
}

// ListAccounts returns every registered account to an active instance administrator.
func (s *Service) ListAccounts(ctx context.Context, actorID string) ([]*User, error) {
	if err := s.requireCanCreateWorkspace(ctx, actorID); err != nil {
		return nil, err
	}
	return s.cfg.Users.ListUsers(ctx)
}

// DisableAccount blocks authentication while preserving the account's memberships and credentials.
func (s *Service) DisableAccount(ctx context.Context, actorID, targetID string) error {
	return s.changeAccountStatus(ctx, actorID, targetID, AccountDisabled, AccountActive, TopicAccountDisabled)
}

// ReactivateAccount restores an intentionally disabled account.
func (s *Service) ReactivateAccount(ctx context.Context, actorID, targetID string) error {
	return s.changeAccountStatus(ctx, actorID, targetID, AccountActive, AccountDisabled, TopicAccountReactivated)
}

// RemoveAccount tombstones an account and removes its access credentials and memberships.
func (s *Service) RemoveAccount(ctx context.Context, actorID, targetID string) error {
	return s.changeAccountStatus(ctx, actorID, targetID, AccountRemoved, AccountActive, TopicAccountRemoved)
}

// RestoreAccount reactivates a removed account without restoring deleted access.
func (s *Service) RestoreAccount(ctx context.Context, actorID, targetID string) error {
	return s.changeAccountStatus(ctx, actorID, targetID, AccountActive, AccountRemoved, TopicAccountRestored)
}

func (s *Service) changeAccountStatus(ctx context.Context, actorID, targetID string, status, expectedFrom AccountStatus, topic string) error {
	if err := s.requireCanCreateWorkspace(ctx, actorID); err != nil {
		return err
	}
	if strings.TrimSpace(targetID) == "" {
		return fmt.Errorf("%w: account id is required", apperrs.ErrInvalid)
	}
	target, err := s.cfg.Users.GetUserByID(ctx, targetID)
	if err != nil {
		return fmt.Errorf("get account %s: %w", targetID, err)
	}
	current := target.AccountStatus
	if current == "" {
		current = AccountActive
	}
	if status == AccountRemoved && (current == AccountDisabled || current == AccountActive) {
		expectedFrom = current
	}
	if current != expectedFrom {
		return fmt.Errorf("%w: account transition from %s to %s is not allowed", apperrs.ErrInvalid, current, status)
	}
	payload := AccountLifecycleEvent{AccountID: targetID, ActorID: actorID}
	return s.cfg.Users.SetAccountStatus(ctx, targetID, status, eventbus.OutboxEvent{ID: newUserID(), Topic: topic, Payload: payload})
}

// ListMembers returns the allowlist (owner only, ADR 0040).
func (s *Service) ListMembers(ctx context.Context, userID string) ([]string, error) {
	if err := s.requireCanCreateWorkspace(ctx, userID); err != nil {
		return nil, err
	}
	return s.cfg.Allowlist.List(ctx)
}

// AddMember adds a git provider username to the allowlist (owner only, ADR 0040).
func (s *Service) AddMember(ctx context.Context, userID, login string) error {
	if err := s.requireCanCreateWorkspace(ctx, userID); err != nil {
		return err
	}
	login = strings.ToLower(strings.TrimSpace(login))
	if login == "" {
		return fmt.Errorf("%w: username is required", apperrs.ErrInvalid)
	}
	return s.cfg.Allowlist.Add(ctx, login)
}

// RemoveMember removes a username from the allowlist (owner only); the user row is untouched, sessions live until TTL.
func (s *Service) RemoveMember(ctx context.Context, userID, login string) error {
	if err := s.requireCanCreateWorkspace(ctx, userID); err != nil {
		return err
	}
	return s.cfg.Allowlist.Remove(ctx, strings.ToLower(strings.TrimSpace(login)))
}

// IsLoginAllowlisted reports whether login can sign in, backing tenancy's AllowlistGate seam (ADR 0017).
func (s *Service) IsLoginAllowlisted(ctx context.Context, login string) (bool, error) {
	return s.cfg.Allowlist.Contains(ctx, strings.ToLower(strings.TrimSpace(login)))
}

// UserIDForLogin resolves login to an existing User's id, found=false if none (tenancy's UserLookupGate seam, ADR 0017).
func (s *Service) UserIDForLogin(ctx context.Context, login string) (string, bool, error) {
	u, err := s.cfg.Users.GetUserByLogin(ctx, strings.ToLower(strings.TrimSpace(login)))
	if err != nil {
		if errors.Is(err, apperrs.ErrNotFound) {
			return "", false, nil
		}
		return "", false, fmt.Errorf("resolve user for login %s: %w", login, err)
	}
	return u.ID, true, nil
}

// GrantCanCreateWorkspace grants targetID the can_create_workspace bit; actorID must already hold the bit.
func (s *Service) GrantCanCreateWorkspace(ctx context.Context, actorID, targetID string) error {
	if err := s.requireCanCreateWorkspace(ctx, actorID); err != nil {
		return err
	}
	if _, err := s.cfg.Users.GetUserByID(ctx, targetID); err != nil {
		return fmt.Errorf("get user %s: %w", targetID, err)
	}
	if err := s.cfg.Users.SetCanCreateWorkspace(ctx, targetID, true); err != nil {
		return fmt.Errorf("grant can_create_workspace to %s: %w", targetID, err)
	}
	return nil
}

// RevokeCanCreateWorkspace revokes targetID's can_create_workspace permission bit; actorID must hold the bit.
func (s *Service) RevokeCanCreateWorkspace(ctx context.Context, actorID, targetID string) error {
	if err := s.requireCanCreateWorkspace(ctx, actorID); err != nil {
		return err
	}
	if _, err := s.cfg.Users.GetUserByID(ctx, targetID); err != nil {
		return fmt.Errorf("get user %s: %w", targetID, err)
	}
	if err := s.cfg.Users.SetCanCreateWorkspace(ctx, targetID, false); err != nil {
		return fmt.Errorf("revoke can_create_workspace from %s: %w", targetID, err)
	}
	return nil
}

func (s *Service) requireCanCreateWorkspace(ctx context.Context, userID string) error {
	user, err := s.cfg.Users.GetUserByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("get user %s: %w", userID, err)
	}
	if !accountIsActive(user.AccountStatus) || !user.CanCreateWorkspace {
		return fmt.Errorf("%w: can_create_workspace permission required", apperrs.ErrForbidden)
	}
	return nil
}

// mac returns the HMAC-SHA256 of enc keyed by the shared secret, base64url.
func (s *Service) mac(enc string) string {
	m := hmac.New(sha256.New, s.cfg.Secret)
	_, _ = m.Write([]byte(enc))
	return base64.RawURLEncoding.EncodeToString(m.Sum(nil))
}

type tokenClaims struct {
	UserID string `json:"uid"`
	Exp    int64  `json:"exp"`
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
