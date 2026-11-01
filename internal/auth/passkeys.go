package auth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/ids"
	"github.com/otal-labs/nexul/internal/platform/logging"
)

const (
	passkeyCeremonyTTL = 5 * time.Minute
	// passkeyCeremonyCap bounds what the public sign-in start can make the server hold.
	passkeyCeremonyCap = 1000
	// passkeyCeremoniesPerAddress keeps one address from filling the cap and blocking everyone else's passkey sign-in.
	passkeyCeremoniesPerAddress = 10
	passkeyNameMax              = 64
)

var (
	errPasskeysOff         = fmt.Errorf("%w: passkeys need https or localhost", apperrs.ErrInvalid)
	errPasskeyCeremonyGone = fmt.Errorf("%w: the passkey request expired, try again", apperrs.ErrUnauthorized)
	errPasskeyRefused      = fmt.Errorf("%w: that passkey isn't recognised here", apperrs.ErrUnauthorized)
)

type passkeyCeremony struct {
	session webauthn.SessionData
	// userID is who a registration is for; empty for a sign-in, which learns it from the passkey.
	userID  string
	origin  string
	addr    string
	expires time.Time
}

// ponytail: a scan of every ceremony on each start, fine under the cap; a heap if the cap ever grows.
type passkeyCeremonies struct {
	mu sync.Mutex
	m  map[string]passkeyCeremony
}

func newPasskeyCeremonies() *passkeyCeremonies {
	return &passkeyCeremonies{m: map[string]passkeyCeremony{}}
}

func (c *passkeyCeremonies) put(now time.Time, cer passkeyCeremony) (string, error) {
	id, err := newToken("pkc_")
	if err != nil {
		return "", err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	open := 0
	for k, v := range c.m {
		if !now.Before(v.expires) {
			delete(c.m, k)
			continue
		}
		if v.addr == cer.addr {
			open++
		}
	}
	if len(c.m) >= passkeyCeremonyCap || open >= passkeyCeremoniesPerAddress {
		return "", fmt.Errorf("%w: too many passkey requests at once, try again in a minute", apperrs.ErrRateLimited)
	}
	cer.expires = now.Add(passkeyCeremonyTTL)
	c.m[id] = cer
	return id, nil
}

// take hands a ceremony out once; a second finish with the same id, or one past its five minutes, finds nothing.
func (c *passkeyCeremonies) take(now time.Time, id string) (passkeyCeremony, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	cer, ok := c.m[id]
	delete(c.m, id)
	return cer, ok && now.Before(cer.expires)
}

// passkeyHost is the RP ID for an origin, or "" where browsers refuse WebAuthn: plain http off localhost, or an IP.
func passkeyHost(origin string) string {
	u, err := url.Parse(origin)
	if err != nil || u.Hostname() == "" || net.ParseIP(u.Hostname()) != nil {
		return ""
	}
	if u.Scheme == "https" || (u.Scheme == "http" && u.Hostname() == "localhost") {
		return u.Hostname()
	}
	return ""
}

// PasskeysAvailable reports whether a browser on origin can make and use passkeys here.
func (s *Service) PasskeysAvailable(origin string) bool {
	return s.cfg.Passkeys != nil && passkeyHost(origin) != ""
}

func (s *Service) relyingParty(origin string) (*webauthn.WebAuthn, error) {
	host := passkeyHost(origin)
	if s.cfg.Passkeys == nil || host == "" {
		return nil, errPasskeysOff
	}
	rp, err := webauthn.New(&webauthn.Config{RPID: host, RPDisplayName: "Nexul", RPOrigins: []string{strings.TrimSuffix(origin, "/")}})
	if err != nil {
		return nil, fmt.Errorf("passkey verifier for %s: %w", host, err)
	}
	return rp, nil
}

type passkeyUser struct {
	user     *User
	passkeys []Passkey
	creds    []webauthn.Credential
}

func (u passkeyUser) WebAuthnID() []byte   { return []byte(u.user.ID) }
func (u passkeyUser) WebAuthnName() string { return u.user.Login }
func (u passkeyUser) WebAuthnDisplayName() string {
	if u.user.Name == "" {
		return u.user.Login
	}
	return u.user.Name
}
func (u passkeyUser) WebAuthnCredentials() []webauthn.Credential { return u.creds }

func (s *Service) loadPasskeyUser(ctx context.Context, userID string) (passkeyUser, error) {
	user, err := s.cfg.Users.GetUserByID(ctx, userID)
	if err != nil {
		return passkeyUser{}, err
	}
	passkeys, err := s.cfg.Passkeys.ListPasskeys(ctx, userID)
	if err != nil {
		return passkeyUser{}, fmt.Errorf("list passkeys of %s: %w", userID, err)
	}
	creds := make([]webauthn.Credential, 0, len(passkeys))
	for _, p := range passkeys {
		var c webauthn.Credential
		if err := json.Unmarshal(p.Credential, &c); err != nil {
			return passkeyUser{}, fmt.Errorf("decode passkey %s: %w", p.ID, err)
		}
		creds = append(creds, c)
	}
	return passkeyUser{user: user, passkeys: passkeys, creds: creds}, nil
}

// ListPasskeys returns the caller's passkeys, oldest first.
func (s *Service) ListPasskeys(ctx context.Context, userID string) ([]Passkey, error) {
	if s.cfg.Passkeys == nil {
		return nil, nil
	}
	passkeys, err := s.cfg.Passkeys.ListPasskeys(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list passkeys: %w", err)
	}
	return passkeys, nil
}

// StartPasskeyRegistration answers the browser's creation options and the ceremony id the finish hands back.
func (s *Service) StartPasskeyRegistration(ctx context.Context, userID, origin string) (*protocol.PublicKeyCredentialCreationOptions, string, error) {
	rp, err := s.relyingParty(origin)
	if err != nil {
		return nil, "", err
	}
	user, err := s.loadPasskeyUser(ctx, userID)
	if err != nil {
		return nil, "", err
	}
	creation, session, err := rp.BeginRegistration(user,
		webauthn.WithAuthenticatorSelection(protocol.AuthenticatorSelection{UserVerification: protocol.VerificationRequired}),
		webauthn.WithResidentKeyRequirement(protocol.ResidentKeyRequirementRequired),
		webauthn.WithExclusions(webauthn.Credentials(user.creds).CredentialDescriptors()),
	)
	if err != nil {
		return nil, "", fmt.Errorf("begin passkey registration: %w", err)
	}
	id, err := s.ceremonies.put(s.cfg.Now(), passkeyCeremony{session: *session, userID: userID, origin: origin, addr: deviceFromCtx(ctx).IP})
	if err != nil {
		return nil, "", err
	}
	return &creation.Response, id, nil
}

// FinishPasskeyRegistration verifies the browser's new credential and stores it, named after the device that made it.
func (s *Service) FinishPasskeyRegistration(ctx context.Context, userID, origin, ceremonyID string, response []byte) (*Passkey, error) {
	cer, ok := s.ceremonies.take(s.cfg.Now(), ceremonyID)
	if !ok || cer.userID != userID || cer.origin != origin {
		return nil, errPasskeyCeremonyGone
	}
	rp, err := s.relyingParty(origin)
	if err != nil {
		return nil, err
	}
	parsed, err := protocol.ParseCredentialCreationResponseBytes(response)
	if err != nil {
		return nil, fmt.Errorf("%w: the browser's passkey response can't be read", apperrs.ErrInvalid)
	}
	user, err := s.loadPasskeyUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	cred, err := rp.CreateCredential(user, cer.session, parsed)
	if err != nil {
		return nil, fmt.Errorf("%w: the passkey couldn't be verified", apperrs.ErrInvalid)
	}
	blob, err := json.Marshal(cred)
	if err != nil {
		return nil, fmt.Errorf("encode passkey: %w", err)
	}
	p := &Passkey{
		ID: base64.RawURLEncoding.EncodeToString(cred.ID), UserID: userID, Name: deviceName(deviceFromCtx(ctx)),
		RPID: passkeyHost(origin), Credential: blob, CreatedAt: s.cfg.Now(),
	}
	if err := s.cfg.Passkeys.InsertPasskey(ctx, p, passkeyEvent(TopicPasskeyAdded, p)); err != nil {
		return nil, fmt.Errorf("store passkey: %w", err)
	}
	return p, nil
}

func deviceName(d Device) string {
	if d.Label == "" || d.Platform == "" || d.Platform == "Unknown" {
		return "Passkey"
	}
	return d.Label + " on " + d.Platform
}

// StartPasskeyLogin answers discoverable sign-in options: the browser offers whichever passkeys it holds for this domain.
func (s *Service) StartPasskeyLogin(ctx context.Context, origin string) (*protocol.PublicKeyCredentialRequestOptions, string, error) {
	rp, err := s.relyingParty(origin)
	if err != nil {
		return nil, "", err
	}
	assertion, session, err := rp.BeginDiscoverableLogin(webauthn.WithUserVerification(protocol.VerificationRequired))
	if err != nil {
		return nil, "", fmt.Errorf("begin passkey sign-in: %w", err)
	}
	id, err := s.ceremonies.put(s.cfg.Now(), passkeyCeremony{session: *session, origin: origin, addr: deviceFromCtx(ctx).IP})
	if err != nil {
		return nil, "", err
	}
	return &assertion.Response, id, nil
}

// FinishPasskeyLogin verifies the assertion, admits only an active account, and mints a session like any sign-in.
func (s *Service) FinishPasskeyLogin(ctx context.Context, origin, ceremonyID string, response []byte) (string, error) {
	cer, ok := s.ceremonies.take(s.cfg.Now(), ceremonyID)
	if !ok || cer.userID != "" || cer.origin != origin {
		return "", errPasskeyCeremonyGone
	}
	rp, err := s.relyingParty(origin)
	if err != nil {
		return "", err
	}
	parsed, err := protocol.ParseCredentialRequestResponseBytes(response)
	if err != nil {
		return "", fmt.Errorf("%w: the browser's passkey response can't be read", apperrs.ErrInvalid)
	}
	var owner passkeyUser
	_, cred, err := rp.ValidatePasskeyLogin(func(rawID, userHandle []byte) (webauthn.User, error) {
		p, err := s.cfg.Passkeys.GetPasskey(ctx, base64.RawURLEncoding.EncodeToString(rawID))
		if err != nil {
			return nil, err
		}
		if p.UserID != string(userHandle) {
			return nil, errPasskeyRefused
		}
		owner, err = s.loadPasskeyUser(ctx, p.UserID)
		return owner, err
	}, cer.session, parsed)
	if err != nil {
		logging.FromCtx(ctx).Info("passkey sign-in refused", "error", err)
		return "", errPasskeyRefused
	}
	if cred.Authenticator.CloneWarning {
		return "", fmt.Errorf("%w: its counter went backwards, so it may have been copied", errPasskeyRefused)
	}
	if owner.user.AccountStatus == AccountRemoved {
		return "", errAccountRemoved
	}
	if !accountIsActive(owner.user.AccountStatus) {
		return "", errAccountDisabled
	}
	blob, err := json.Marshal(cred)
	if err != nil {
		return "", fmt.Errorf("encode passkey: %w", err)
	}
	if err := s.cfg.Passkeys.UpdatePasskeyUse(ctx, base64.RawURLEncoding.EncodeToString(cred.ID), blob, s.cfg.Now()); err != nil {
		return "", fmt.Errorf("record passkey use: %w", err)
	}
	return s.CreateSession(ctx, owner.user.ID)
}

// RenamePasskey names one of the caller's passkeys.
func (s *Service) RenamePasskey(ctx context.Context, userID, id, name string) error {
	name = strings.TrimSpace(name)
	if name == "" || utf8.RuneCountInString(name) > passkeyNameMax {
		return fmt.Errorf("%w: a passkey name is 1 to %d characters", apperrs.ErrInvalid, passkeyNameMax)
	}
	if s.cfg.Passkeys == nil {
		return fmt.Errorf("%w: no such passkey", apperrs.ErrNotFound)
	}
	if err := s.cfg.Passkeys.RenamePasskey(ctx, userID, id, name); err != nil {
		return fmt.Errorf("rename passkey: %w", err)
	}
	return nil
}

// DeletePasskey removes one of the caller's passkeys; their linked accounts keep them able to sign in.
func (s *Service) DeletePasskey(ctx context.Context, userID, id string) error {
	if s.cfg.Passkeys == nil {
		return fmt.Errorf("%w: no such passkey", apperrs.ErrNotFound)
	}
	p, err := s.cfg.Passkeys.GetPasskey(ctx, id)
	if err != nil || p.UserID != userID {
		return fmt.Errorf("%w: no such passkey", apperrs.ErrNotFound)
	}
	if err := s.cfg.Passkeys.DeletePasskey(ctx, userID, id, passkeyEvent(TopicPasskeyRemoved, p)); err != nil {
		return fmt.Errorf("delete passkey: %w", err)
	}
	return nil
}

func passkeyEvent(topic string, p *Passkey) eventbus.OutboxEvent {
	return eventbus.OutboxEvent{ID: ids.New(), Topic: topic, Payload: PasskeyChangedEvent{UserID: p.UserID, PasskeyID: p.ID, Name: p.Name}}
}
