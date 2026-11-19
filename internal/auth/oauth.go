package auth

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
	"sync"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/ids"
)

const (
	oauthAccessTTL        = time.Hour
	oauthCodeTTL          = time.Minute
	oauthCodeCap          = 1000
	oauthRefreshPrefix    = "depr_"
	oauthClientNameMax    = 100
	oauthRedirectURIsMax  = 5
	oauthRedirectURIMax   = 2000
	oauthRegistrations    = 60
	oauthRegistrationSpan = time.Hour
	oauthDefaultName      = "MCP client"
)

// OAuthError is an RFC 6749 error the token and registration endpoints answer with.
type OAuthError struct {
	Code        string
	Description string
}

func (e *OAuthError) Error() string { return e.Code + ": " + e.Description }

func oauthErr(code, description string) error {
	return &OAuthError{Code: code, Description: description}
}

var (
	errOAuthOff       = fmt.Errorf("%w: mcp sign-in is not configured", apperrs.ErrNotFound)
	errOAuthBadGrant  = oauthErr("invalid_grant", "the code or refresh token is invalid, expired or already used")
	errOAuthBadClient = oauthErr("invalid_client", "unknown client_id")
)

// OAuthRequest is an authorize request's parameters, as the consent page passes them on.
type OAuthRequest struct {
	ClientID            string `json:"client_id"`
	RedirectURI         string `json:"redirect_uri"`
	ResponseType        string `json:"response_type"`
	CodeChallenge       string `json:"code_challenge"`
	CodeChallengeMethod string `json:"code_challenge_method"`
	State               string `json:"state"`
}

// OAuthConsent is what the consent page shows: who asks, and where approving sends the person back to.
type OAuthConsent struct {
	ClientName  string `json:"client_name"`
	RedirectURI string `json:"redirect_uri"`
}

// OAuthTokens is a token endpoint answer; the raw tokens are returned once and only their hashes kept.
type OAuthTokens struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
}

type oauthCode struct {
	clientID    string
	userID      string
	redirectURI string
	// requested is the redirect_uri as the authorize request sent it, which the token request must repeat.
	requested string
	challenge string
	expires   time.Time
}

// ponytail: codes live in memory for their minute, so a restart mid-sign-in means approving again.
type oauthCodes struct {
	mu sync.Mutex
	m  map[string]oauthCode
}

func (c *oauthCodes) put(now time.Time, code oauthCode) (string, error) {
	raw, err := newToken("")
	if err != nil {
		return "", err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	for k, v := range c.m {
		if !now.Before(v.expires) {
			delete(c.m, k)
		}
	}
	if len(c.m) >= oauthCodeCap {
		return "", fmt.Errorf("%w: too many sign-ins at once, try again in a minute", apperrs.ErrRateLimited)
	}
	code.expires = now.Add(oauthCodeTTL)
	c.m[hashToken(raw)] = code
	return raw, nil
}

// take hands a code out once; a replay or a code past its minute finds nothing.
func (c *oauthCodes) take(now time.Time, raw string) (oauthCode, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	key := hashToken(raw)
	code, ok := c.m[key]
	delete(c.m, key)
	if !ok || !now.Before(code.expires) {
		return oauthCode{}, false
	}
	return code, true
}

// OAuthEnabled reports whether MCP clients can sign in, so the discovery documents exist.
func (s *Service) OAuthEnabled() bool {
	return s.cfg.OAuthClients != nil && s.cfg.PATs != nil
}

// RegisterOAuthClient is dynamic client registration (RFC 7591): anyone may register, so it is throttled per address.
func (s *Service) RegisterOAuthClient(ctx context.Context, addr, name string, redirectURIs []string) (*OAuthClient, error) {
	if !s.OAuthEnabled() {
		return nil, errOAuthOff
	}
	now := s.cfg.Now()
	if s.registrations.blocked(addr, now) {
		return nil, fmt.Errorf("%w: too many client registrations, try again later", apperrs.ErrRateLimited)
	}
	if len(redirectURIs) == 0 || len(redirectURIs) > oauthRedirectURIsMax {
		return nil, oauthErr("invalid_redirect_uri", fmt.Sprintf("give between 1 and %d redirect_uris", oauthRedirectURIsMax))
	}
	for _, uri := range redirectURIs {
		if err := validRedirectURI(uri); err != nil {
			return nil, err
		}
	}
	s.registrations.fail(addr, now)
	client := OAuthClient{ID: ids.New(), Name: clientName(name), RedirectURIs: redirectURIs, CreatedAt: now.UTC()}
	if err := s.cfg.OAuthClients.CreateOAuthClient(ctx, client); err != nil {
		return nil, fmt.Errorf("register oauth client: %w", err)
	}
	return &client, nil
}

func clientName(name string) string {
	name = strings.TrimSpace(name)
	if r := []rune(name); len(r) > oauthClientNameMax {
		name = string(r[:oauthClientNameMax])
	}
	if name == "" {
		return oauthDefaultName
	}
	return name
}

// validRedirectURI admits https, http on loopback (RFC 8252) and an app's own scheme; never one a browser would run.
func validRedirectURI(raw string) error {
	bad := func(why string) error {
		return oauthErr("invalid_redirect_uri", fmt.Sprintf("%q %s", raw, why))
	}
	if len(raw) > oauthRedirectURIMax {
		return bad("is too long")
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" {
		return bad("is not an absolute URL")
	}
	if u.Fragment != "" || strings.Contains(raw, "#") {
		return bad("must not carry a fragment")
	}
	scheme := strings.ToLower(u.Scheme)
	switch scheme {
	case "javascript", "data", "vbscript", "file", "blob", "about", "ws", "wss":
		return bad("uses a scheme that cannot take a redirect")
	case "https":
		if u.Host == "" {
			return bad("has no host")
		}
		return nil
	case "http":
		if !isLoopbackHost(u.Hostname()) {
			return bad("must be https unless it is on localhost")
		}
		return nil
	}
	return nil
}

func isLoopbackHost(host string) bool {
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// OAuthConsent checks an authorize request before the consent page shows it; a bad client or redirect is never followed.
func (s *Service) OAuthConsent(ctx context.Context, req OAuthRequest) (*OAuthConsent, error) {
	client, redirect, err := s.checkAuthorize(ctx, req)
	if err != nil {
		return nil, err
	}
	return &OAuthConsent{ClientName: client.Name, RedirectURI: redirect}, nil
}

func (s *Service) checkAuthorize(ctx context.Context, req OAuthRequest) (*OAuthClient, string, error) {
	if !s.OAuthEnabled() {
		return nil, "", errOAuthOff
	}
	client, err := s.cfg.OAuthClients.GetOAuthClient(ctx, req.ClientID)
	if errors.Is(err, apperrs.ErrNotFound) {
		return nil, "", fmt.Errorf("%w: this app is not registered here; add the server to it again", apperrs.ErrInvalid)
	}
	if err != nil {
		return nil, "", fmt.Errorf("get oauth client: %w", err)
	}
	redirect, err := registeredRedirect(client, req.RedirectURI)
	if err != nil {
		return nil, "", err
	}
	if req.ResponseType != "code" {
		return nil, "", fmt.Errorf("%w: response_type must be code", apperrs.ErrInvalid)
	}
	if req.CodeChallengeMethod != "S256" || !validPKCEValue(req.CodeChallenge) {
		return nil, "", fmt.Errorf("%w: a PKCE code_challenge with method S256 is required", apperrs.ErrInvalid)
	}
	return client, redirect, nil
}

// registeredRedirect is the redirect_uri the client registered; an omitted one is allowed only when it registered one.
func registeredRedirect(client *OAuthClient, requested string) (string, error) {
	if requested == "" && len(client.RedirectURIs) == 1 {
		return client.RedirectURIs[0], nil
	}
	for _, uri := range client.RedirectURIs {
		if uri == requested {
			return uri, nil
		}
	}
	return "", fmt.Errorf("%w: redirect_uri is not one this app registered", apperrs.ErrInvalid)
}

// validPKCEValue is RFC 7636's shape for a verifier, and for an S256 challenge, which is 43 characters of base64url.
func validPKCEValue(v string) bool {
	if len(v) < 43 || len(v) > 128 {
		return false
	}
	for _, r := range v {
		ok := r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || strings.ContainsRune("-._~", r)
		if !ok {
			return false
		}
	}
	return true
}

// DecideOAuth answers the consent page with the client's redirect: a one-minute code for userID, or access_denied.
func (s *Service) DecideOAuth(ctx context.Context, userID string, req OAuthRequest, approve bool) (string, error) {
	if ViewerFromCtx(ctx) != nil {
		return "", fmt.Errorf("%w: stop viewing as someone else to sign an app in", apperrs.ErrForbidden)
	}
	_, redirect, err := s.checkAuthorize(ctx, req)
	if err != nil {
		return "", err
	}
	answer := url.Values{}
	if req.State != "" {
		answer.Set("state", req.State)
	}
	if !approve {
		answer.Set("error", "access_denied")
		return withQuery(redirect, answer), nil
	}
	code, err := s.oauthCodes.put(s.cfg.Now(), oauthCode{
		clientID: req.ClientID, userID: userID, redirectURI: redirect, requested: req.RedirectURI, challenge: req.CodeChallenge,
	})
	if err != nil {
		return "", err
	}
	answer.Set("code", code)
	return withQuery(redirect, answer), nil
}

func withQuery(redirect string, answer url.Values) string {
	sep := "?"
	if strings.Contains(redirect, "?") {
		sep = "&"
	}
	return redirect + sep + answer.Encode()
}

// ExchangeOAuthCode trades a code and its PKCE verifier for a grant: a token row named after the client.
func (s *Service) ExchangeOAuthCode(ctx context.Context, clientID, code, redirectURI, verifier string) (*OAuthTokens, error) {
	if !s.OAuthEnabled() {
		return nil, errOAuthOff
	}
	grant, ok := s.oauthCodes.take(s.cfg.Now(), code)
	if !ok || grant.clientID != clientID || grant.requested != redirectURI {
		return nil, errOAuthBadGrant
	}
	if !pkceMatches(verifier, grant.challenge) {
		return nil, oauthErr("invalid_grant", "code_verifier does not match the code_challenge")
	}
	client, err := s.cfg.OAuthClients.GetOAuthClient(ctx, clientID)
	if errors.Is(err, apperrs.ErrNotFound) {
		return nil, errOAuthBadClient
	}
	if err != nil {
		return nil, fmt.Errorf("get oauth client: %w", err)
	}
	user, err := s.cfg.Users.GetUserByID(ctx, grant.userID)
	if err != nil || !accountIsActive(user.AccountStatus) {
		return nil, errOAuthBadGrant
	}
	tokens, pat, err := s.newGrantTokens()
	if err != nil {
		return nil, err
	}
	pat.ID = newUserID()
	pat.UserID = grant.userID
	pat.Name = client.Name
	pat.OAuthClientID = clientID
	pat.CreatedAt = s.cfg.Now()
	if err := s.cfg.PATs.Create(ctx, &pat, tokenEvent(TopicTokenMinted, pat)); err != nil {
		return nil, fmt.Errorf("create grant: %w", err)
	}
	return tokens, nil
}

func pkceMatches(verifier, challenge string) bool {
	if !validPKCEValue(verifier) {
		return false
	}
	sum := sha256.Sum256([]byte(verifier))
	return subtle.ConstantTimeCompare([]byte(base64.RawURLEncoding.EncodeToString(sum[:])), []byte(challenge)) == 1
}

// RefreshOAuth rotates a grant: a new access and refresh token, and the old refresh token stops working.
func (s *Service) RefreshOAuth(ctx context.Context, clientID, refresh string) (*OAuthTokens, error) {
	if !s.OAuthEnabled() {
		return nil, errOAuthOff
	}
	if !strings.HasPrefix(refresh, oauthRefreshPrefix) {
		return nil, errOAuthBadGrant
	}
	tokens, next, err := s.newGrantTokens()
	if err != nil {
		return nil, err
	}
	pat, err := s.cfg.PATs.RotateByRefresh(ctx, hashToken(refresh), clientID, next)
	if errors.Is(err, apperrs.ErrNotFound) {
		return nil, errOAuthBadGrant
	}
	if err != nil {
		return nil, fmt.Errorf("rotate grant: %w", err)
	}
	user, err := s.cfg.Users.GetUserByID(ctx, pat.UserID)
	if err != nil || !accountIsActive(user.AccountStatus) {
		return nil, errOAuthBadGrant
	}
	return tokens, nil
}

// newGrantTokens mints a grant's raw tokens and the row fields that hold their hashes.
func (s *Service) newGrantTokens() (*OAuthTokens, PersonalAccessToken, error) {
	access, err := newToken(patPrefix)
	if err != nil {
		return nil, PersonalAccessToken{}, err
	}
	refresh, err := newToken(oauthRefreshPrefix)
	if err != nil {
		return nil, PersonalAccessToken{}, err
	}
	expires := s.cfg.Now().Add(oauthAccessTTL)
	pat := PersonalAccessToken{
		TokenHash: hashToken(access), RefreshHash: hashToken(refresh), Prefix: access[len(access)-6:], ExpiresAt: &expires,
	}
	tokens := &OAuthTokens{AccessToken: access, TokenType: "Bearer", ExpiresIn: int(oauthAccessTTL.Seconds()), RefreshToken: refresh}
	return tokens, pat, nil
}
