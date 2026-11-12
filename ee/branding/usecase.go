package branding

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/otal-labs/nexul/ee/license"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/ids"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// DefaultName is what the instance calls itself without branding.
const DefaultName = "Nexul"

// Gate checks the caller holds an instance permission in any workspace.
type Gate interface {
	RequireAnywhere(ctx context.Context, action permissions.Action) error
}

// Licenses answers whether the instance's key unlocks a feature; license.Service is the one implementation.
type Licenses interface {
	Enabled(ctx context.Context, f license.Feature) bool
}

// Service is the branding use-case layer the HTTP gateway, the MCP tool and the pages that show the look call.
type Service struct {
	repo     Repo
	licenses Licenses
	gate     Gate
	now      func() time.Time
}

// NewService reads and writes branding through repo, shown only while licenses unlocks branding.
func NewService(repo Repo, licenses Licenses) *Service {
	return &Service{repo: repo, licenses: licenses, now: time.Now}
}

// WithGate attaches the permission gate the Owner's reads and writes are checked against.
func (s *Service) WithGate(g Gate) *Service {
	s.gate = g
	return s
}

func (s *Service) require(ctx context.Context, action permissions.Action) error {
	if s.gate == nil {
		return permissions.Ungated(ctx)
	}
	return s.gate.RequireAnywhere(ctx, action)
}

func (s *Service) unlocked(ctx context.Context) bool {
	return s.licenses.Enabled(ctx, license.FeatureBranding)
}

var errLocked = fmt.Errorf("%w: branding is an enterprise feature; paste a license key that unlocks it first", apperrs.ErrForbidden)

// Get returns the saved branding with whether the key lets it show; it needs instance:read.
func (s *Service) Get(ctx context.Context) (Settings, error) {
	if err := s.require(ctx, permissions.InstanceRead); err != nil {
		return Settings{}, err
	}
	rec, _, err := s.repo.Load(ctx)
	if err != nil {
		return Settings{}, err
	}
	return rec.settings(s.unlocked(ctx)), nil
}

// Look is what every page paints: the saved branding while the key unlocks it, otherwise nothing at all. It is public,
// since the sign-in page wears the brand before anyone has signed in.
func (s *Service) Look(ctx context.Context) (Look, error) {
	if !s.unlocked(ctx) {
		return Look{}, nil
	}
	rec, found, err := s.repo.Load(ctx)
	if err != nil || !found {
		return Look{}, err
	}
	return rec.look(), nil
}

// ProductName is what the instance calls itself in what it sends: the saved name while branding shows, else Nexul.
func (s *Service) ProductName(ctx context.Context) string {
	l, err := s.Look(ctx)
	if err != nil || strings.TrimSpace(l.ProductName) == "" {
		return DefaultName
	}
	return l.ProductName
}

// Update applies a patch to the saved branding and returns it as Get shows it. It needs instance:write and a key
// unlocking branding; a theme file that holds no readable colours is refused with the reason.
func (s *Service) Update(ctx context.Context, p Patch) (Settings, error) {
	if err := s.require(ctx, permissions.InstanceWrite); err != nil {
		return Settings{}, err
	}
	if !s.unlocked(ctx) {
		return Settings{}, errLocked
	}
	rec, _, err := s.repo.Load(ctx)
	if err != nil {
		return Settings{}, err
	}
	if err := apply(&rec, p); err != nil {
		return Settings{}, err
	}
	now := s.now()
	rec.UpdatedBy, rec.UpdatedAt = actorID(ctx), now
	if err := s.repo.Save(ctx, rec, s.changed(ctx, false)); err != nil {
		return Settings{}, err
	}
	return s.Get(ctx)
}

func apply(rec *Record, p Patch) error {
	if p.ProductName != nil {
		name, err := validateName(*p.ProductName)
		if err != nil {
			return err
		}
		rec.ProductName = name
	}
	if p.Accent != nil {
		accent, err := validateAccent(*p.Accent)
		if err != nil {
			return err
		}
		rec.Accent = accent
	}
	switch p.RemoveTheme {
	case "":
	case "all":
		rec.Light, rec.Dark = nil, nil
	case string(Light):
		rec.Light = nil
	case string(Dark):
		rec.Dark = nil
	default:
		return fmt.Errorf("%w: remove_theme is light, dark or all", apperrs.ErrInvalid)
	}
	if p.Theme == nil {
		return nil
	}
	t, err := ImportTheme(*p.Theme)
	if err != nil {
		return err
	}
	if t.Appearance == Dark {
		rec.Dark = &t
		return nil
	}
	rec.Light = &t
	return nil
}

// SetLogo replaces the logo with an image. It needs instance:write and a key unlocking branding.
func (s *Service) SetLogo(ctx context.Context, data []byte) (Settings, error) {
	if err := s.require(ctx, permissions.InstanceWrite); err != nil {
		return Settings{}, err
	}
	if !s.unlocked(ctx) {
		return Settings{}, errLocked
	}
	contentType := sniffLogo(data)
	if err := validateLogo(contentType, data); err != nil {
		return Settings{}, err
	}
	sum := sha256.Sum256(data)
	version := hex.EncodeToString(sum[:6])
	if err := s.repo.SetLogo(ctx, contentType, data, version, actorID(ctx), s.now(), s.changed(ctx, false)); err != nil {
		return Settings{}, err
	}
	return s.Get(ctx)
}

// RemoveLogo removes the logo, so Nexul's mark shows again. It needs instance:write.
func (s *Service) RemoveLogo(ctx context.Context) (Settings, error) {
	if err := s.require(ctx, permissions.InstanceWrite); err != nil {
		return Settings{}, err
	}
	if err := s.repo.ClearLogo(ctx, actorID(ctx), s.now(), s.changed(ctx, false)); err != nil {
		return Settings{}, err
	}
	return s.Get(ctx)
}

// Clear removes all the saved branding. It needs instance:write and works without a key, so an expired instance can still
// wipe what it no longer shows.
func (s *Service) Clear(ctx context.Context) (Settings, error) {
	if err := s.require(ctx, permissions.InstanceWrite); err != nil {
		return Settings{}, err
	}
	if err := s.repo.Delete(ctx, s.changed(ctx, true)); err != nil {
		return Settings{}, err
	}
	return s.Get(ctx)
}

// Logo returns the image the instance shows as its mark; public like Look, and absent while branding is locked.
func (s *Service) Logo(ctx context.Context) (contentType string, data []byte, err error) {
	if !s.unlocked(ctx) {
		return "", nil, apperrs.ErrNotFound
	}
	return s.repo.Logo(ctx)
}

func (s *Service) changed(ctx context.Context, cleared bool) eventbus.OutboxEvent {
	return eventbus.OutboxEvent{ID: ids.New(), Topic: TopicChanged, Payload: ChangedEvent{Cleared: cleared, ChangedBy: actorID(ctx), ChangedAt: s.now().UTC()}}
}

func actorID(ctx context.Context) string {
	a, _ := identity.ActorFromCtx(ctx)
	return a.ID
}

// sniffLogo names the image type from the bytes, never from what the uploader claimed.
func sniffLogo(data []byte) string {
	head := bytes.TrimSpace(data[:min(len(data), 512)])
	if bytes.HasPrefix(head, []byte("<?xml")) || bytes.HasPrefix(head, []byte("<svg")) {
		if bytes.Contains(data[:min(len(data), 4096)], []byte("<svg")) {
			return "image/svg+xml"
		}
	}
	return http.DetectContentType(data)
}
