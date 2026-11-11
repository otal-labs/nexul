package license

import (
	"context"
	"crypto/ed25519"
	"fmt"
	"log/slog"
	"strings"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/ids"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// Gate checks the caller holds an instance permission in any workspace.
type Gate interface {
	RequireAnywhere(ctx context.Context, action permissions.Action) error
}

// Service is the license use-case layer the HTTP gateway, the MCP tools and every enterprise feature call.
type Service struct {
	repo        Repo
	instanceURL func(context.Context) (string, error)
	gate        Gate
	pub         ed25519.PublicKey
	now         func() time.Time
}

// NewService checks keys against the built-in issuer key and the instance URL instanceURL reads.
func NewService(repo Repo, instanceURL func(context.Context) (string, error)) *Service {
	return &Service{repo: repo, instanceURL: instanceURL, pub: PublicKey(), now: time.Now}
}

// WithGate attaches the permission gate the reads and writes are checked against.
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

// Enabled is the one check an enterprise feature asks before it runs: a failed read counts as no license.
func (s *Service) Enabled(ctx context.Context, f Feature) bool {
	st, err := s.status(ctx)
	if err != nil {
		slog.WarnContext(ctx, "read license key", "feature", f, "error", err)
		return false
	}
	return st.Unlocks(f)
}

// Status returns the license as it stands now; it needs instance:read.
func (s *Service) Status(ctx context.Context) (Status, error) {
	if err := s.require(ctx, permissions.InstanceRead); err != nil {
		return Status{}, err
	}
	return s.status(ctx)
}

func (s *Service) status(ctx context.Context) (Status, error) {
	key, err := s.repo.Key(ctx)
	if err != nil || key == "" {
		return Status{State: StateNone}, err
	}
	claims, err := Parse(key, s.pub)
	if err != nil {
		return Status{State: StateInvalid, Problem: problemOf(err)}, nil
	}
	instanceURL, err := s.instanceURL(ctx)
	if err != nil {
		return Status{}, fmt.Errorf("read instance url: %w", err)
	}
	return evaluate(claims, instanceURL, s.now()), nil
}

// Set replaces the instance's key with key; a key for another instance, with a bad signature or already expired is
// refused with the reason. It needs instance:write.
func (s *Service) Set(ctx context.Context, key string) (Status, error) {
	if err := s.require(ctx, permissions.InstanceWrite); err != nil {
		return Status{}, err
	}
	key = strings.TrimSpace(key)
	claims, err := Parse(key, s.pub)
	if err != nil {
		return Status{}, err
	}
	instanceURL, err := s.instanceURL(ctx)
	if err != nil {
		return Status{}, fmt.Errorf("read instance url: %w", err)
	}
	now := s.now()
	st := evaluate(claims, instanceURL, now)
	if st.State == StateInvalid {
		return Status{}, fmt.Errorf("%w: %s", apperrs.ErrInvalid, st.Problem)
	}
	if st.State == StateExpired {
		return Status{}, fmt.Errorf("%w: this key expired on %s; paste a renewed one", apperrs.ErrInvalid, st.ExpiresAt.Format("2 Jan 2006"))
	}
	actor := actorID(ctx)
	evt := s.changed(ChangedEvent{Licensee: claims.Licensee, Features: claims.Features, ExpiresAt: claims.ExpiresAt.UTC(), ChangedBy: actor}, now)
	if err := s.repo.SetKey(ctx, key, actor, now, evt); err != nil {
		return Status{}, err
	}
	return st, nil
}

// Remove deletes the instance's key: the enterprise features stop and everything they made stays. It needs
// instance:write, and removing no key changes nothing.
func (s *Service) Remove(ctx context.Context) (Status, error) {
	if err := s.require(ctx, permissions.InstanceWrite); err != nil {
		return Status{}, err
	}
	key, err := s.repo.Key(ctx)
	if err != nil || key == "" {
		return Status{State: StateNone}, err
	}
	evt := s.changed(ChangedEvent{Removed: true, ChangedBy: actorID(ctx)}, s.now())
	if err := s.repo.DeleteKey(ctx, evt); err != nil {
		return Status{}, err
	}
	return Status{State: StateNone}, nil
}

func (s *Service) changed(e ChangedEvent, at time.Time) eventbus.OutboxEvent {
	e.ChangedAt = at.UTC()
	return eventbus.OutboxEvent{ID: ids.New(), Topic: TopicChanged, Payload: e}
}

func actorID(ctx context.Context) string {
	a, _ := identity.ActorFromCtx(ctx)
	return a.ID
}
