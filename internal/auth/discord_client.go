package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

const (
	discordAuthorizeURL = "https://discord.com/oauth2/authorize"
	discordTokenURL     = "https://discord.com/api/oauth2/token"
	discordUserURL      = "https://discord.com/api/users/@me"
	discordAvatarCDN    = "https://cdn.discordapp.com/avatars/%s/%s.png"
)

// HTTPDiscordClient talks to Discord's OAuth + /users/@me (ADR 0040); keyed by snowflake id, login is the verified email.
type HTTPDiscordClient struct {
	client       *http.Client
	clientID     string
	clientSecret string
	redirectURI  string
	tokenURL     string
	userURL      string
}

// NewHTTPDiscordClient wires a real Discord client; redirectURI must equal the one sent on authorize.
func NewHTTPDiscordClient(clientID, clientSecret, redirectURI string, hc *http.Client) *HTTPDiscordClient {
	return &HTTPDiscordClient{
		client:       hc,
		clientID:     clientID,
		clientSecret: clientSecret,
		redirectURI:  redirectURI,
		tokenURL:     discordTokenURL,
		userURL:      discordUserURL,
	}
}

// Exchange trades an authorization code for an access token.
func (c *HTTPDiscordClient) Exchange(ctx context.Context, code string) (string, error) {
	return exchangeCode(ctx, c.client, "discord", c.tokenURL, c.clientID, c.clientSecret, c.redirectURI, code)
}

// FetchUser fetches /users/@me (identify + email scopes); an unverified email is refused, the allowlist matches on it.
func (c *HTTPDiscordClient) FetchUser(ctx context.Context, accessToken string) (user *ProviderUser, err error) {
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
		ID         string `json:"id"`
		Username   string `json:"username"`
		GlobalName string `json:"global_name"`
		Avatar     string `json:"avatar"`
		Email      string `json:"email"`
		Verified   bool   `json:"verified"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("fetch user: %w", apperrs.Retryable(err))
	}
	if body.ID == "" || body.Email == "" {
		return nil, fmt.Errorf("%w: discord returned an incomplete user (is the email scope granted?)", apperrs.ErrUnauthorized)
	}
	if !body.Verified {
		return nil, fmt.Errorf("%w: discord account email is not verified", apperrs.ErrUnauthorized)
	}
	name := body.GlobalName
	if name == "" {
		name = body.Username
	}
	avatar := ""
	if body.Avatar != "" {
		avatar = fmt.Sprintf(discordAvatarCDN, body.ID, body.Avatar)
	}
	return &ProviderUser{
		ID:        body.ID,
		Login:     strings.ToLower(body.Email),
		Name:      name,
		AvatarURL: avatar,
	}, nil
}
