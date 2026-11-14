package sso

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/otal-labs/nexul/ee/license"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/ids"
	"github.com/otal-labs/nexul/internal/platform/permissions"
	"github.com/otal-labs/nexul/internal/platform/secretref"
)

const maxDisplayName = 40

// Gate checks the caller holds an instance permission in any workspace.
type Gate interface {
	RequireAnywhere(ctx context.Context, action permissions.Action) error
}

// Licenses is the check every enterprise feature asks before it runs.
type Licenses interface {
	Enabled(ctx context.Context, f license.Feature) bool
}

// Secrets resolves a client secret that is a secret reference; a nil Secrets takes each stored value as it is.
type Secrets interface {
	Value(ctx context.Context, stored string) (string, error)
	Read(ctx context.Context, stored string) (string, error)
	Refresh(ctx context.Context, stale string) (string, bool)
}

// Config wires the single sign-on use-cases.
type Config struct {
	Repo     Repo
	Accounts Accounts
	Licenses Licenses
	Gate     Gate
	Secrets  Secrets
	// InstanceURL is the instance's public address, which the provider's redirect URI is built on.
	InstanceURL func(context.Context) (string, error)
	// OwnerHasPasskey reports whether an instance Owner can still sign in with a passkey, which turning SSO-only on needs.
	OwnerHasPasskey func(context.Context) (bool, error)
	// Secret keys the nonce and PKCE verifier each sign-in derives from its state.
	Secret []byte
	// Connect reaches the provider; nil dials it over HTTP.
	Connect Connect
	Now     func() time.Time
}

// Service is the single sign-on use-case layer the HTTP gateway, the MCP tool and the sign-in callback call.
type Service struct {
	cfg Config
}

// NewService wires the use-cases over cfg.
func NewService(cfg Config) *Service {
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	if cfg.Connect == nil {
		cfg.Connect = dial(&http.Client{Timeout: 15 * time.Second}, cfg.Secret, cfg.Now)
	}
	return &Service{cfg: cfg}
}

func (s *Service) require(ctx context.Context, action permissions.Action) error {
	if s.cfg.Gate == nil {
		return permissions.Ungated(ctx)
	}
	return s.cfg.Gate.RequireAnywhere(ctx, action)
}

func (s *Service) unlocked(ctx context.Context) bool {
	return s.cfg.Licenses.Enabled(ctx, license.FeatureSSO)
}

func (s *Service) redirectURI(ctx context.Context) (string, error) {
	base, err := s.cfg.InstanceURL(ctx)
	if err != nil {
		return "", fmt.Errorf("read instance url: %w", err)
	}
	return strings.TrimSuffix(base, "/") + "/auth/oidc/callback", nil
}

// Public is what the sign-in page shows: the button only while the key unlocks sso and a provider is set, and every
// other method hidden only while the switch is on. A failed read counts as off.
func (s *Service) Public(ctx context.Context) Public {
	st, err := s.cfg.Repo.Get(ctx)
	if err != nil || !st.Configured() || !s.unlocked(ctx) {
		return Public{}
	}
	return Public{Active: true, Name: st.DisplayName, Only: st.Only}
}

// OnlySSO reports whether every other sign-in method is off right now.
func (s *Service) OnlySSO(ctx context.Context) bool {
	return s.Public(ctx).Only
}

// View returns the settings as they stand, whatever the license says; it needs instance:read.
func (s *Service) View(ctx context.Context) (View, error) {
	if err := s.require(ctx, permissions.InstanceRead); err != nil {
		return View{}, err
	}
	st, err := s.cfg.Repo.Get(ctx)
	if err != nil {
		return View{}, err
	}
	return s.view(ctx, st)
}

func (s *Service) view(ctx context.Context, st Settings) (View, error) {
	redirect, err := s.redirectURI(ctx)
	if err != nil {
		return View{}, err
	}
	unlocked := s.unlocked(ctx)
	return View{
		Unlocked: unlocked, Configured: st.Configured(), IssuerURL: st.IssuerURL, ClientID: st.ClientID, SecretSet: st.ClientSecret != "",
		DisplayName: st.DisplayName, JoinRoleID: st.JoinRoleID, Only: st.Only, RedirectURI: redirect, Active: unlocked && st.Configured(),
	}, nil
}

// Update saves the settings; it needs instance:write and a key that unlocks sso, and refuses a provider it cannot reach,
// a secret reference that does not resolve, a role nobody may join as, or SSO-only with no Owner passkey to fall back on.
func (s *Service) Update(ctx context.Context, p Patch) (View, error) {
	if err := s.require(ctx, permissions.InstanceWrite); err != nil {
		return View{}, err
	}
	if !s.unlocked(ctx) {
		return View{}, errNeedsLicense
	}
	cur, err := s.cfg.Repo.Get(ctx)
	if err != nil {
		return View{}, err
	}
	next := Settings{
		IssuerURL: trim(p.IssuerURL, cur.IssuerURL), ClientID: trim(p.ClientID, cur.ClientID), ClientSecret: cur.ClientSecret,
		DisplayName: trim(p.DisplayName, cur.DisplayName), JoinRoleID: trim(p.JoinRoleID, cur.JoinRoleID), Only: cur.Only,
	}
	if secret := trim(p.ClientSecret, ""); secret != "" {
		next.ClientSecret = secret
	}
	if p.Only != nil {
		next.Only = *p.Only
	}
	if err := s.validate(ctx, cur, next); err != nil {
		return View{}, err
	}
	next.UpdatedBy, next.UpdatedAt = actorID(ctx), s.cfg.Now()
	evt := s.changed(ChangedEvent{DisplayName: next.DisplayName, IssuerURL: next.IssuerURL, Only: next.Only, ChangedBy: next.UpdatedBy}, next.UpdatedAt)
	if err := s.cfg.Repo.Set(ctx, next, evt); err != nil {
		return View{}, err
	}
	return s.view(ctx, next)
}

func (s *Service) validate(ctx context.Context, cur, next Settings) error {
	if !next.Configured() {
		return fmt.Errorf("%w: issuer_url, client_id, client_secret, display_name and join_role_id are all needed", apperrs.ErrInvalid)
	}
	if utf8.RuneCountInString(next.DisplayName) > maxDisplayName {
		return fmt.Errorf("%w: display_name is at most %d characters", apperrs.ErrInvalid, maxDisplayName)
	}
	if err := checkIssuer(next.IssuerURL); err != nil {
		return err
	}
	if err := secretref.Check("client secret", next.ClientSecret); err != nil {
		return err
	}
	if next.JoinRoleID != cur.JoinRoleID {
		ok, err := s.cfg.Accounts.CanJoinAs(ctx, next.JoinRoleID)
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("%w: join_role_id must be a role of the default workspace other than Owner; workspace_list with the default workspace id lists its roles", apperrs.ErrInvalid)
		}
	}
	if next.IssuerURL != cur.IssuerURL || next.ClientSecret != cur.ClientSecret {
		if err := s.reach(ctx, next); err != nil {
			return err
		}
	}
	if next.Only && !cur.Only {
		return s.requireOwnerPasskey(ctx)
	}
	return nil
}

// readSecret returns the secret a stored value stands for, asking the manager for a reference being saved and refusing
// one that does not resolve, naming only why.
func (s *Service) readSecret(ctx context.Context, stored string) (string, error) {
	kind, ref := secretref.Parse(stored)
	if s.cfg.Secrets == nil || kind != secretref.Reference {
		return stored, nil
	}
	v, err := s.cfg.Secrets.Read(ctx, stored)
	if err != nil {
		return "", fmt.Errorf("%w: client secret: %s", apperrs.ErrInvalid, secretref.Reason(secretref.ManagerName(ref.Scheme), err))
	}
	return v, nil
}

// reach connects to the provider with what is about to be saved, so a wrong issuer or a reference that does not resolve
// is refused with its reason now rather than at somebody's sign-in.
func (s *Service) reach(ctx context.Context, st Settings) error {
	secret, err := s.readSecret(ctx, st.ClientSecret)
	if err != nil {
		return err
	}
	_, err = s.connect(ctx, st, secret, true)
	return err
}

// Check runs one named check of what is about to be saved, "issuer" (the provider answers at that address) or "secret"
// (the client secret, or the reference to it, reads), and stores nothing; it needs what Update does. The detail is a
// line for the row to show.
func (s *Service) Check(ctx context.Context, check string, p Patch) (string, error) {
	if err := s.require(ctx, permissions.InstanceWrite); err != nil {
		return "", err
	}
	if !s.unlocked(ctx) {
		return "", errNeedsLicense
	}
	cur, err := s.cfg.Repo.Get(ctx)
	if err != nil {
		return "", err
	}
	st := Settings{IssuerURL: trim(p.IssuerURL, cur.IssuerURL), ClientID: trim(p.ClientID, cur.ClientID), ClientSecret: cur.ClientSecret}
	if secret := trim(p.ClientSecret, ""); secret != "" {
		st.ClientSecret = secret
	}
	switch check {
	case "issuer":
		return s.checkIssuerReaches(ctx, st)
	case "secret":
		return s.checkSecret(ctx, st)
	}
	return "", fmt.Errorf("%w: check is issuer or secret", apperrs.ErrInvalid)
}

func (s *Service) checkIssuerReaches(ctx context.Context, st Settings) (string, error) {
	if err := checkIssuer(st.IssuerURL); err != nil {
		return "", err
	}
	if _, err := s.connect(ctx, st, "", true); err != nil {
		return "", err
	}
	u, _ := url.Parse(st.IssuerURL)
	return "Found the sign-in endpoints at " + u.Host, nil
}

func (s *Service) checkSecret(ctx context.Context, st Settings) (string, error) {
	if st.ClientSecret == "" {
		return "", fmt.Errorf("%w: client secret is needed", apperrs.ErrInvalid)
	}
	if err := secretref.Check("client secret", st.ClientSecret); err != nil {
		return "", err
	}
	if _, err := s.readSecret(ctx, st.ClientSecret); err != nil {
		return "", err
	}
	if kind, _ := secretref.Parse(st.ClientSecret); kind == secretref.Reference {
		return "Read from the secrets manager", nil
	}
	return "", nil
}

func (s *Service) requireOwnerPasskey(ctx context.Context) error {
	ok, err := s.cfg.OwnerHasPasskey(ctx)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("%w: only hides every other way in, so an instance Owner needs a passkey first, "+
			"or a broken provider would lock everyone out; add one under Profile, Passkeys", apperrs.ErrInvalid)
	}
	return nil
}

// Remove deletes the settings: the button goes, the other sign-in methods come back, and nobody's account changes. It
// needs instance:write, and works without a key so a lapsed license never traps a provider in place.
func (s *Service) Remove(ctx context.Context) (View, error) {
	if err := s.require(ctx, permissions.InstanceWrite); err != nil {
		return View{}, err
	}
	cur, err := s.cfg.Repo.Get(ctx)
	if err != nil {
		return View{}, err
	}
	if cur.IssuerURL == "" {
		return s.view(ctx, Settings{})
	}
	evt := s.changed(ChangedEvent{Removed: true, ChangedBy: actorID(ctx)}, s.cfg.Now())
	if err := s.cfg.Repo.Delete(ctx, evt); err != nil {
		return View{}, err
	}
	return s.view(ctx, Settings{})
}

// active returns the settings a sign-in runs on, or why nobody can sign in this way.
func (s *Service) active(ctx context.Context) (Settings, error) {
	st, err := s.cfg.Repo.Get(ctx)
	if err != nil {
		return Settings{}, err
	}
	if !st.Configured() || !s.unlocked(ctx) {
		return Settings{}, fmt.Errorf("%w: single sign-on is not on for this instance", apperrs.ErrInvalid)
	}
	return st, nil
}

// connect reaches the provider; fresh is for a check, which must not answer from a discovery made earlier.
func (s *Service) connect(ctx context.Context, st Settings, secret string, fresh bool) (IdP, error) {
	redirect, err := s.redirectURI(ctx)
	if err != nil {
		return nil, err
	}
	return s.cfg.Connect(ctx, ConnectConfig{Issuer: st.IssuerURL, ClientID: st.ClientID, ClientSecret: secret, RedirectURI: redirect, Fresh: fresh})
}

func (s *Service) clientSecret(ctx context.Context, stored string) (string, error) {
	if s.cfg.Secrets == nil {
		return stored, nil
	}
	secret, err := s.cfg.Secrets.Value(ctx, stored)
	if err != nil {
		return "", fmt.Errorf("resolve the single sign-on client secret: %w", err)
	}
	return secret, nil
}

// Start begins a sign-in: the provider's authorize URL and the state the callback must give back.
func (s *Service) Start(ctx context.Context) (authorizeURL, state string, err error) {
	st, err := s.active(ctx)
	if err != nil {
		return "", "", err
	}
	secret, err := s.clientSecret(ctx, st.ClientSecret)
	if err != nil {
		return "", "", err
	}
	idp, err := s.connect(ctx, st, secret, false)
	if err != nil {
		return "", "", err
	}
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", "", fmt.Errorf("generate sign-in state: %w", err)
	}
	state = hex.EncodeToString(b)
	return idp.AuthURL(state), state, nil
}

// Complete finishes a sign-in and returns the account to start a session for: the one already linked to the subject,
// else the one whose Google or Discord sign-in has the same verified email, else a new one that joins the default
// workspace as the settings' role.
func (s *Service) Complete(ctx context.Context, state, code string) (string, error) {
	st, err := s.active(ctx)
	if err != nil {
		return "", err
	}
	claims, err := s.exchange(ctx, st, state, code)
	if err != nil {
		return "", err
	}
	email := strings.TrimSpace(claims.Email)
	if claims.Subject == "" || email == "" || !claims.EmailVerified {
		return "", errUnverified
	}
	id := Identity{Key: st.IssuerURL + "#" + claims.Subject, Email: email, Name: claims.Name, AvatarURL: claims.Picture}
	return s.account(ctx, st, id)
}

// exchange trades the code for claims; a provider that rejects the client while its secret is a reference gets the
// reference read again (at most every 5 minutes, which Refresh enforces) and one retry.
func (s *Service) exchange(ctx context.Context, st Settings, state, code string) (Claims, error) {
	secret, err := s.clientSecret(ctx, st.ClientSecret)
	if err != nil {
		return Claims{}, err
	}
	idp, err := s.connect(ctx, st, secret, false)
	if err != nil {
		return Claims{}, err
	}
	claims, err := idp.Exchange(ctx, code, state)
	if !errors.Is(err, errClientRejected) || s.cfg.Secrets == nil {
		return claims, err
	}
	fresh, ok := s.cfg.Secrets.Refresh(ctx, secret)
	if !ok {
		return Claims{}, err
	}
	idp, rerr := s.connect(ctx, st, fresh, false)
	if rerr != nil {
		return Claims{}, err
	}
	return idp.Exchange(ctx, code, state)
}

func (s *Service) account(ctx context.Context, st Settings, id Identity) (string, error) {
	acct, err := s.cfg.Accounts.BySubject(ctx, id.Key)
	if err == nil {
		return admit(acct)
	}
	if !errors.Is(err, apperrs.ErrNotFound) {
		return "", err
	}
	acct, err = s.cfg.Accounts.ByEmail(ctx, id.Email)
	if err == nil {
		userID, aerr := admit(acct)
		if aerr != nil {
			return "", aerr
		}
		if err := s.cfg.Accounts.Link(ctx, userID, id); err != nil {
			return "", err
		}
		return userID, nil
	}
	if !errors.Is(err, apperrs.ErrNotFound) {
		return "", err
	}
	ok, err := s.cfg.Accounts.CanJoinAs(ctx, st.JoinRoleID)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", errNotReady
	}
	userID := ids.New()
	if err := s.cfg.Accounts.Create(ctx, userID, id, st.JoinRoleID, s.cfg.Now()); err != nil {
		return "", err
	}
	return userID, nil
}

// admit refuses an account that may not sign in, the way every other sign-in does.
func admit(a Account) (string, error) {
	if a.Status == "removed" {
		return "", errRemoved
	}
	if a.Status != "" && a.Status != "active" {
		return "", errDisabled
	}
	return a.UserID, nil
}

func (s *Service) changed(e ChangedEvent, at time.Time) eventbus.OutboxEvent {
	e.ChangedAt = at.UTC()
	return eventbus.OutboxEvent{ID: ids.New(), Topic: TopicChanged, Payload: e}
}

func actorID(ctx context.Context) string {
	a, _ := identity.ActorFromCtx(ctx)
	return a.ID
}
