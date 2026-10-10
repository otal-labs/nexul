package auth

import (
	"context"
	"crypto/hmac"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/ids"
)

const (
	// linkStatePrefix lets the shared provider callback tell a link-mode state from a sign-in one.
	linkStatePrefix = "link."
	// linkStateMACLabel namespaces the state's MAC so nothing else signed with the secret ever verifies as one.
	linkStateMACLabel = "identity-link."
)

// linkStateClaims names the user who started a link, signed, so the public callback never takes that from a cookie.
type linkStateClaims struct {
	UserID   string   `json:"uid"`
	Provider Provider `json:"p"`
	Nonce    string   `json:"n"`
	Exp      int64    `json:"exp"`
}

// IsLinkState reports whether an OAuth callback state was minted by StartIdentityLink.
func IsLinkState(state string) bool {
	return strings.HasPrefix(state, linkStatePrefix)
}

func (s *Service) signLinkState(userID string, provider Provider) (string, error) {
	nonce, err := s.NewState()
	if err != nil {
		return "", err
	}
	payload, err := json.Marshal(linkStateClaims{UserID: userID, Provider: provider, Nonce: nonce, Exp: s.cfg.Now().Add(stateMaxAge).Unix()})
	if err != nil {
		return "", fmt.Errorf("sign link state: %w", err)
	}
	enc := base64.RawURLEncoding.EncodeToString(payload)
	return linkStatePrefix + enc + "." + s.mac(linkStateMACLabel+enc), nil
}

func (s *Service) verifyLinkState(state string, provider Provider) (string, error) {
	enc, sig, ok := strings.Cut(strings.TrimPrefix(state, linkStatePrefix), ".")
	if !ok || !hmac.Equal([]byte(sig), []byte(s.mac(linkStateMACLabel+enc))) {
		return "", fmt.Errorf("%w: link state is not valid", apperrs.ErrUnauthorized)
	}
	payload, err := base64.RawURLEncoding.DecodeString(enc)
	if err != nil {
		return "", fmt.Errorf("%w: link state is not valid", apperrs.ErrUnauthorized)
	}
	var claims linkStateClaims
	if err := json.Unmarshal(payload, &claims); err != nil || claims.UserID == "" || claims.Provider != provider {
		return "", fmt.Errorf("%w: link state is not valid", apperrs.ErrUnauthorized)
	}
	if s.cfg.Now().Unix() >= claims.Exp {
		return "", fmt.Errorf("%w: the link expired, start it again", apperrs.ErrUnauthorized)
	}
	return claims.UserID, nil
}

// SignInProviders is the composition root's read for the Team list, which checks who may see each account itself.
func (s *Service) SignInProviders(ctx context.Context) (map[string][]Provider, error) {
	providers, err := s.cfg.Users.ListIdentityProviders(ctx)
	if err != nil {
		return nil, fmt.Errorf("sign-in providers: %w", err)
	}
	return providers, nil
}

// ListIdentities returns the caller's sign-in accounts, oldest first.
func (s *Service) ListIdentities(ctx context.Context, userID string) ([]Identity, error) {
	identities, err := s.cfg.Users.ListIdentities(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list identities: %w", err)
	}
	return identities, nil
}

func (s *Service) firstIdentity(ctx context.Context, userID string) (Identity, error) {
	identities, err := s.ListIdentities(ctx, userID)
	if err != nil {
		return Identity{}, err
	}
	if len(identities) == 0 {
		return Identity{}, fmt.Errorf("%w: account has no sign-in identity", apperrs.ErrUnauthorized)
	}
	return identities[0], nil
}

// StartIdentityLink runs provider's OAuth in link mode: the authorize URL plus the signed state naming the user.
func (s *Service) StartIdentityLink(ctx context.Context, userID string, provider Provider) (authorizeURL, state string, err error) {
	if userID == "" {
		return "", "", apperrs.ErrUnauthorized
	}
	if _, ok := signInProviders[provider]; !ok {
		return "", "", fmt.Errorf("%w: unknown sign-in provider %q", apperrs.ErrInvalid, provider)
	}
	state, err = s.signLinkState(userID, provider)
	if err != nil {
		return "", "", err
	}
	authorizeURL, err = s.AuthorizeURLFor(ctx, provider, state)
	if err != nil {
		return "", "", err
	}
	return authorizeURL, state, nil
}

// CompleteIdentityLink attaches the provider account behind code to the user the link state names.
func (s *Service) CompleteIdentityLink(ctx context.Context, provider Provider, state, code string) error {
	userID, err := s.verifyLinkState(state, provider)
	if err != nil {
		return err
	}
	user, err := s.cfg.Users.GetUserByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("get user %s: %w", userID, err)
	}
	if !accountIsActive(user.AccountStatus) {
		return fmt.Errorf("%w: account is not active", apperrs.ErrUnauthorized)
	}
	client, err := s.providerClient(ctx, provider)
	if err != nil {
		return err
	}
	accessToken, err := client.Exchange(ctx, code)
	if err != nil {
		return err
	}
	pu, err := client.FetchUser(ctx, accessToken)
	if err != nil {
		return err
	}
	identity := providerIdentity(user.ID, provider, pu)
	identity.CreatedAt = s.cfg.Now()
	if err := s.cfg.Users.LinkIdentity(ctx, identity, identityEvent(TopicIdentityLinked, *identity)); err != nil {
		return fmt.Errorf("link %s identity: %w", provider, err)
	}
	return nil
}

// UnlinkIdentity detaches one of the caller's sign-in accounts; the store refuses the last one.
func (s *Service) UnlinkIdentity(ctx context.Context, userID string, provider Provider) error {
	identities, err := s.ListIdentities(ctx, userID)
	if err != nil {
		return err
	}
	i := slices.IndexFunc(identities, func(id Identity) bool { return id.Provider == provider })
	if i < 0 {
		return fmt.Errorf("%w: no %s account is linked", apperrs.ErrNotFound, provider)
	}
	if err := s.cfg.Users.UnlinkIdentity(ctx, userID, provider, identityEvent(TopicIdentityUnlinked, identities[i])); err != nil {
		return fmt.Errorf("unlink %s identity: %w", provider, err)
	}
	return nil
}

func identityEvent(topic string, id Identity) eventbus.OutboxEvent {
	return eventbus.OutboxEvent{ID: ids.New(), Topic: topic, Payload: IdentityChangedEvent{UserID: id.UserID, Provider: id.Provider, Login: id.Login}}
}
