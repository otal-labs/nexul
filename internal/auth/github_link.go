package auth

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// Codes the web reads to offer Connect GitHub instead of an error; the messages tell an agent what the person must do.
var (
	ErrGitHubNotConnected = apperrs.WithCode("github_not_connected", fmt.Errorf("%w: connect GitHub in Settings → Profile to see your repositories; Nexul lists only what your own GitHub account can open", apperrs.ErrForbidden))
	ErrGitHubReconnect    = apperrs.WithCode("github_reconnect", fmt.Errorf("%w: GitHub asked for your account to be connected again; reconnect GitHub in Settings → Profile", apperrs.ErrForbidden))
)

// githubTokenSkew refreshes a person's token this long before GitHub expires it, so a list never starts on a lapsing one.
const githubTokenSkew = time.Minute

// githubGranter is the GitHub client's token exchange that keeps the refresh token; other providers have none.
type githubGranter interface {
	ExchangeGrant(ctx context.Context, code string) (GitHubGrant, error)
	RefreshGrant(ctx context.Context, refreshToken string) (GitHubGrant, error)
}

// exchange trades code for a token, keeping GitHub's refresh token when the client hands one out.
func exchange(ctx context.Context, client ProviderClient, code string) (GitHubGrant, error) {
	if g, ok := client.(githubGranter); ok {
		return g.ExchangeGrant(ctx, code)
	}
	tok, err := client.Exchange(ctx, code)
	return GitHubGrant{AccessToken: tok}, err
}

func (s *Service) linkFromGrant(userID string, grant GitHubGrant) GitHubLink {
	now := s.cfg.Now().UTC()
	l := GitHubLink{UserID: userID, AccessToken: grant.AccessToken, RefreshToken: grant.RefreshToken, ConnectedAt: now}
	if grant.ExpiresIn > 0 {
		l.ExpiresAt = now.Add(grant.ExpiresIn)
	}
	if grant.RefreshExpiresIn > 0 {
		l.RefreshExpiresAt = now.Add(grant.RefreshExpiresIn)
	}
	return l
}

// keepGitHubLink stores the token a GitHub sign-in or Connect GitHub handed out, replacing any earlier one.
func (s *Service) keepGitHubLink(ctx context.Context, userID string, grant GitHubGrant) error {
	if s.cfg.GitHubLinks == nil {
		return nil
	}
	return s.cfg.GitHubLinks.SaveGitHubLink(ctx, s.linkFromGrant(userID, grant))
}

// GitHubToken is userID's own GitHub user token, refreshed when it has lapsed. It never falls back to another
// credential: no link is ErrGitHubNotConnected, and a refresh GitHub refuses marks the link ErrGitHubReconnect.
func (s *Service) GitHubToken(ctx context.Context, userID string) (string, error) {
	if s.cfg.GitHubLinks == nil || userID == "" {
		return "", ErrGitHubNotConnected
	}
	s.githubRefresh.Lock()
	defer s.githubRefresh.Unlock()
	link, err := s.cfg.GitHubLinks.GetGitHubLink(ctx, userID)
	if errors.Is(err, apperrs.ErrNotFound) {
		return "", ErrGitHubNotConnected
	}
	if err != nil {
		return "", err
	}
	if link.NeedsReconnect {
		return "", ErrGitHubReconnect
	}
	if link.ExpiresAt.IsZero() || s.cfg.Now().Before(link.ExpiresAt.Add(-githubTokenSkew)) {
		return link.AccessToken, nil
	}
	grant, err := s.refreshGitHub(ctx, link.RefreshToken)
	if errors.Is(err, apperrs.ErrUnauthorized) {
		return "", s.markReconnect(ctx, link)
	}
	if err != nil {
		return "", fmt.Errorf("refresh the GitHub token of %s: %w", userID, err)
	}
	fresh := s.linkFromGrant(userID, grant)
	fresh.ConnectedAt = link.ConnectedAt
	if err := s.cfg.GitHubLinks.SaveGitHubLink(ctx, fresh); err != nil {
		return "", err
	}
	return fresh.AccessToken, nil
}

// refreshGitHub asks GitHub for a fresh token; a missing refresh token or a client without refresh is a refusal.
func (s *Service) refreshGitHub(ctx context.Context, refreshToken string) (GitHubGrant, error) {
	if refreshToken == "" {
		return GitHubGrant{}, fmt.Errorf("%w: no refresh token", apperrs.ErrUnauthorized)
	}
	client, err := s.providerClient(ctx, ProviderGitHub)
	if err != nil {
		return GitHubGrant{}, err
	}
	g, ok := client.(githubGranter)
	if !ok {
		return GitHubGrant{}, fmt.Errorf("%w: the GitHub client cannot refresh", apperrs.ErrUnauthorized)
	}
	return g.RefreshGrant(ctx, refreshToken)
}

func (s *Service) markReconnect(ctx context.Context, link GitHubLink) error {
	spent := GitHubLink{UserID: link.UserID, NeedsReconnect: true, ConnectedAt: link.ConnectedAt}
	if err := s.cfg.GitHubLinks.SaveGitHubLink(ctx, spent); err != nil {
		return err
	}
	return ErrGitHubReconnect
}

// GitHubLinkStatus says whether userID's GitHub link lists their repositories, and as which GitHub account.
func (s *Service) GitHubLinkStatus(ctx context.Context, userID string) (GitHubLinkStatus, error) {
	if s.cfg.GitHubLinks == nil {
		return GitHubLinkStatus{State: GitHubLinkNone}, nil
	}
	link, err := s.cfg.GitHubLinks.GetGitHubLink(ctx, userID)
	if errors.Is(err, apperrs.ErrNotFound) {
		return GitHubLinkStatus{State: GitHubLinkNone}, nil
	}
	if err != nil {
		return GitHubLinkStatus{}, err
	}
	identities, err := s.ListIdentities(ctx, userID)
	if err != nil {
		return GitHubLinkStatus{}, err
	}
	st := GitHubLinkStatus{State: GitHubLinkConnected, ConnectedAt: &link.ConnectedAt}
	if i := slices.IndexFunc(identities, func(id Identity) bool { return id.Provider == ProviderGitHub }); i >= 0 {
		st.Login = identities[i].Login
	}
	if link.NeedsReconnect {
		st.State = GitHubLinkReconnect
	}
	return st, nil
}

// DisconnectGitHub forgets userID's GitHub token, so Nexul lists no repositories for them until they connect again.
func (s *Service) DisconnectGitHub(ctx context.Context, userID string) error {
	if userID == "" {
		return apperrs.ErrUnauthorized
	}
	if s.cfg.GitHubLinks == nil {
		return nil
	}
	return s.cfg.GitHubLinks.DeleteGitHubLink(ctx, userID)
}
