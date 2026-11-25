package auditlog

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/otal-labs/nexul/ee/license"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/ids"
	"github.com/otal-labs/nexul/internal/platform/permissions"
	"github.com/otal-labs/nexul/internal/platform/secretref"
)

// Gate checks the caller holds an instance permission in any workspace.
type Gate interface {
	RequireAnywhere(ctx context.Context, action permissions.Action) error
}

// Licenses is the check every enterprise feature asks before it runs.
type Licenses interface {
	Enabled(ctx context.Context, f license.Feature) bool
}

// Secrets resolves a stream secret that is a secret reference; a nil Secrets takes each stored value as it is.
type Secrets interface {
	Read(ctx context.Context, stored string) (string, error)
}

// Config wires the audit log use-cases.
type Config struct {
	Repo     Repo
	Licenses Licenses
	Gate     Gate
	Secrets  Secrets
	Now      func() time.Time
}

// Service is the audit log use-case layer the HTTP gateway, the MCP tool and the retention loop call.
type Service struct {
	cfg Config
}

// NewService wires the use-cases over cfg.
func NewService(cfg Config) *Service {
	if cfg.Now == nil {
		cfg.Now = time.Now
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
	return s.cfg.Licenses.Enabled(ctx, license.FeatureAudit)
}

// RetentionDays is how many days of audit rows the retention purge keeps: the saved days whatever the license says,
// since a lapse keeps what the feature made (ADR 0184), else MinDays. Unchecked, since the purge loop calls it.
func (s *Service) RetentionDays(ctx context.Context) (int, error) {
	st, err := s.cfg.Repo.Get(ctx)
	if err != nil {
		return 0, err
	}
	return st.Days(), nil
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
	return s.view(ctx, st), nil
}

func (s *Service) view(ctx context.Context, st Settings) View {
	unlocked := s.unlocked(ctx)
	return View{
		Unlocked: unlocked, RetentionDays: st.Days(), StreamURL: st.StreamURL, SecretSet: st.StreamSecret != "",
		StreamEnabled: st.StreamEnabled, Streaming: unlocked && st.StreamEnabled,
	}
}

// Update saves the settings; it needs instance:write and a key that unlocks audit, and refuses days outside MinDays
// to MaxDays, a destination that is not https, and a secret reference that does not resolve. A destination saved
// without a secret gets one the server makes, returned once.
func (s *Service) Update(ctx context.Context, p Patch) (Saved, error) {
	if err := s.require(ctx, permissions.InstanceWrite); err != nil {
		return Saved{}, err
	}
	if !s.unlocked(ctx) {
		return Saved{}, errNeedsLicense
	}
	cur, err := s.cfg.Repo.Get(ctx)
	if err != nil {
		return Saved{}, err
	}
	next, made, err := s.apply(ctx, cur, p)
	if err != nil {
		return Saved{}, err
	}
	next.UpdatedBy, next.UpdatedAt = actorID(ctx), s.cfg.Now()
	evt := s.changed(SettingsChangedEvent{
		RetentionDays: next.Days(), StreamURL: next.StreamURL, StreamEnabled: next.StreamEnabled, ChangedBy: next.UpdatedBy,
	}, next.UpdatedAt)
	if err := s.cfg.Repo.Set(ctx, next, evt); err != nil {
		return Saved{}, err
	}
	return Saved{View: s.view(ctx, next), NewSecret: made}, nil
}

// apply returns cur with p's changes, checked, and the secret it made, if any.
func (s *Service) apply(ctx context.Context, cur Settings, p Patch) (Settings, string, error) {
	next := cur
	if p.RetentionDays != nil {
		if err := checkDays(*p.RetentionDays); err != nil {
			return Settings{}, "", err
		}
		next.RetentionDays = *p.RetentionDays
	}
	if p.StreamURL != nil {
		next.StreamURL = strings.TrimSpace(*p.StreamURL)
		if next.StreamURL == "" {
			next.StreamSecret, next.StreamEnabled = "", false
		}
	}
	if next.StreamURL != "" {
		if err := checkStreamURL(next.StreamURL); err != nil {
			return Settings{}, "", err
		}
	}
	if secret := trim(p.StreamSecret); secret != "" {
		if next.StreamURL == "" {
			return Settings{}, "", fmt.Errorf("%w: stream_secret signs deliveries to stream_url, so set stream_url too", apperrs.ErrInvalid)
		}
		if err := s.checkSecret(ctx, secret); err != nil {
			return Settings{}, "", err
		}
		next.StreamSecret = secret
	}
	if p.StreamEnabled != nil {
		next.StreamEnabled = *p.StreamEnabled
	}
	if next.StreamEnabled && next.StreamURL == "" {
		return Settings{}, "", fmt.Errorf("%w: the stream needs a stream_url before it can be switched on", apperrs.ErrInvalid)
	}
	return withSecret(next)
}

// withSecret gives a destination saved without a secret one the server makes, and returns it.
func withSecret(st Settings) (Settings, string, error) {
	if st.StreamURL == "" || st.StreamSecret != "" {
		return st, "", nil
	}
	made, err := newSecret()
	if err != nil {
		return Settings{}, "", err
	}
	st.StreamSecret = made
	return st, made, nil
}

// checkSecret refuses a secret reference the manager cannot read, naming only why.
func (s *Service) checkSecret(ctx context.Context, stored string) error {
	if err := secretref.Check("stream secret", stored); err != nil {
		return err
	}
	kind, ref := secretref.Parse(stored)
	if s.cfg.Secrets == nil || kind != secretref.Reference {
		return nil
	}
	if _, err := s.cfg.Secrets.Read(ctx, stored); err != nil {
		return fmt.Errorf("%w: stream secret: %s", apperrs.ErrInvalid, secretref.Reason(secretref.ManagerName(ref.Scheme), err))
	}
	return nil
}

// Remove deletes the settings: the log goes back to 45 days, so the next purge deletes the older rows, and the stream
// stops. It needs instance:write, and works without a key so a lapsed license never traps a setting in place.
func (s *Service) Remove(ctx context.Context) (View, error) {
	if err := s.require(ctx, permissions.InstanceWrite); err != nil {
		return View{}, err
	}
	cur, err := s.cfg.Repo.Get(ctx)
	if err != nil {
		return View{}, err
	}
	if cur == (Settings{}) {
		return s.view(ctx, Settings{}), nil
	}
	evt := s.changed(SettingsChangedEvent{RetentionDays: MinDays, Removed: true, ChangedBy: actorID(ctx)}, s.cfg.Now())
	if err := s.cfg.Repo.Delete(ctx, evt); err != nil {
		return View{}, err
	}
	return s.view(ctx, Settings{}), nil
}

func (s *Service) changed(e SettingsChangedEvent, at time.Time) eventbus.OutboxEvent {
	e.ChangedAt = at.UTC()
	return eventbus.OutboxEvent{ID: ids.New(), Topic: TopicSettingsChanged, Payload: e}
}

// newSecret is 32 random bytes as hex, the key a receiver checks X-Nexul-Signature with.
func newSecret() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate stream secret: %w", err)
	}
	return hex.EncodeToString(b), nil
}

func trim(p *string) string {
	if p == nil {
		return ""
	}
	return strings.TrimSpace(*p)
}

func actorID(ctx context.Context) string {
	a, _ := identity.ActorFromCtx(ctx)
	return a.ID
}
