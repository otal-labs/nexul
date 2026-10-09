package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

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
	// Redemption resolves the user through an identity, so the signed-in user's first one stands in for the OAuth round trip.
	first, err := s.firstIdentity(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	handoff := &OAuthHandoff{ID: newUserID(), InvitationID: details.InvitationID, OAuthStateHash: hashCredential(state), Provider: first.Provider, CreatedAt: s.cfg.Now(), ExpiresAt: s.cfg.Now().Add(stateMaxAge)}
	if err := s.cfg.OAuthHandoffs.StartOAuthHandoff(ctx, handoff); err != nil {
		return nil, err
	}
	identity := OAuthHandoffIdentity{Provider: first.Provider, ProviderUserID: first.ProviderUserID, Login: user.Login, Name: user.Name, AvatarURL: user.AvatarURL, ExistingUserID: user.ID}
	if _, err := s.cfg.OAuthHandoffs.CompleteOAuthCallback(ctx, hashCredential(state), hashCredential(acceptance), identity, s.cfg.Now().Add(stateMaxAge), s.cfg.Now()); err != nil {
		return nil, classifyInvitationError(err)
	}
	details.AuthenticatedUser = &InvitationAuthenticatedUser{ID: user.ID, Provider: first.Provider, Login: user.Login, Name: user.Name, AvatarURL: user.AvatarURL}
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
	token, err := s.CreateSession(ctx, admission.UserID)
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
