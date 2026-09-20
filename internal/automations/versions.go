package automations

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/ids"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// VersionStatus is a code version's place in an automation's history.
type VersionStatus string

const (
	VersionPending  VersionStatus = "pending"
	VersionActive   VersionStatus = "active"
	VersionInactive VersionStatus = "inactive"
)

// seedPusherID marks a version landed by the seeding hook rather than a person; never a real user id.
const seedPusherID = "nexul-seed"

// Version is immutable; merge/rollback repoints which version is active.
type Version struct {
	ID           string        `json:"id"`
	AutomationID string        `json:"automation_id"`
	Sequence     int           `json:"sequence"`
	Code         string        `json:"code"`
	PusherID     string        `json:"pusher_id"`
	Message      string        `json:"message,omitempty"`
	Status       VersionStatus `json:"status"`
	CreatedAt    time.Time     `json:"created_at"`
}

// VersionsRepo is the consumer-side persistence contract for automation code versions.
type VersionsRepo interface {
	// InsertActive inserts v as active — only valid for the very first version, with no history to repoint.
	InsertActive(ctx context.Context, v *Version) error
	// ReplacePending stores v as the one pending version, discarding whatever pending version preceded it.
	ReplacePending(ctx context.Context, v *Version) error
	// Activate repoints automationID's active version to versionID, moving the prior active version to inactive.
	Activate(ctx context.Context, automationID, versionID string) (*Version, error)
	Get(ctx context.Context, automationID, versionID string) (*Version, error)
	ListByAutomation(ctx context.Context, automationID string) ([]Version, error)
	// Pending and Active return apperrs.ErrNotFound when automationID has no version in that state.
	Pending(ctx context.Context, automationID string) (*Version, error)
	Active(ctx context.Context, automationID string) (*Version, error)
}

// VersionsService is the automations version-history use-case layer: push, merge, rollback, list, and pull.
type VersionsService struct {
	repo        VersionsRepo
	automations Repo
	perm        PermissionGate
	now         func() time.Time
}

// NewVersionsService wires the version use-cases over the given repos and the shared permission gate.
func NewVersionsService(repo VersionsRepo, automations Repo, perm PermissionGate) *VersionsService {
	return &VersionsService{repo: repo, automations: automations, perm: perm, now: time.Now}
}

// Push lands code as a pending version; it never activates itself, so the diff against active catches mistakes.
func (s *VersionsService) Push(ctx context.Context, actorID, automationID, code, message string) (*Version, error) {
	if err := requirePermission(ctx, s.perm, actorID, permissions.AutomationsWrite); err != nil {
		return nil, err
	}
	if err := s.mustExist(ctx, automationID); err != nil {
		return nil, err
	}
	if strings.TrimSpace(code) == "" {
		return nil, fmt.Errorf("%w: version code is required", apperrs.ErrInvalid)
	}
	v := &Version{
		ID:           ids.New(),
		AutomationID: automationID,
		Code:         code,
		PusherID:     actorID,
		Message:      strings.TrimSpace(message),
		Status:       VersionPending,
		CreatedAt:    s.now().UTC(),
	}
	if err := s.repo.ReplacePending(ctx, v); err != nil {
		return nil, fmt.Errorf("push version for automation %s: %w", automationID, err)
	}
	return v, nil
}

// Activate repoints the active version to versionID, the one operation behind both the merge and rollback endpoints.
func (s *VersionsService) Activate(ctx context.Context, actorID, automationID, versionID string) (*Version, error) {
	if err := requirePermission(ctx, s.perm, actorID, permissions.AutomationsWrite); err != nil {
		return nil, err
	}
	if err := s.mustExist(ctx, automationID); err != nil {
		return nil, err
	}
	if strings.TrimSpace(versionID) == "" {
		return nil, fmt.Errorf("%w: version id is required", apperrs.ErrInvalid)
	}
	v, err := s.repo.Activate(ctx, automationID, versionID)
	if err != nil {
		return nil, fmt.Errorf("activate version %s for automation %s: %w", versionID, automationID, err)
	}
	return v, nil
}

// List returns every version of automationID, newest first.
func (s *VersionsService) List(ctx context.Context, actorID, automationID string) ([]Version, error) {
	if err := requirePermission(ctx, s.perm, actorID, permissions.AutomationsRead); err != nil {
		return nil, err
	}
	if err := s.mustExist(ctx, automationID); err != nil {
		return nil, err
	}
	list, err := s.repo.ListByAutomation(ctx, automationID)
	if err != nil {
		return nil, fmt.Errorf("list versions for automation %s: %w", automationID, err)
	}
	return list, nil
}

// Get pulls one version's full code, backing the SDK's pull command.
func (s *VersionsService) Get(ctx context.Context, actorID, automationID, versionID string) (*Version, error) {
	if err := requirePermission(ctx, s.perm, actorID, permissions.AutomationsRead); err != nil {
		return nil, err
	}
	if err := s.mustExist(ctx, automationID); err != nil {
		return nil, err
	}
	v, err := s.repo.Get(ctx, automationID, versionID)
	if err != nil {
		return nil, fmt.Errorf("get version %s for automation %s: %w", versionID, automationID, err)
	}
	return v, nil
}

// Diff returns the active and pending versions for the UI to diff client-side; pending is nil when there is none.
func (s *VersionsService) Diff(ctx context.Context, actorID, automationID string) (active, pending *Version, err error) {
	if err := requirePermission(ctx, s.perm, actorID, permissions.AutomationsRead); err != nil {
		return nil, nil, err
	}
	if err := s.mustExist(ctx, automationID); err != nil {
		return nil, nil, err
	}
	active, err = s.repo.Active(ctx, automationID)
	if err != nil {
		if !errors.Is(err, apperrs.ErrNotFound) {
			return nil, nil, fmt.Errorf("get active version for automation %s: %w", automationID, err)
		}
		active = nil
	}
	pending, perr := s.repo.Pending(ctx, automationID)
	if perr != nil {
		if !errors.Is(perr, apperrs.ErrNotFound) {
			return nil, nil, fmt.Errorf("get pending version for automation %s: %w", automationID, perr)
		}
		pending = nil
	}
	return active, pending, nil
}

// SeedVersion lands shipped code at startup: the first version activates, later ones land pending so an upgrade never silently changes a default's behaviour; ungated.
func (s *VersionsService) SeedVersion(ctx context.Context, automationID, code, message string) (*Version, error) {
	active, err := s.repo.Active(ctx, automationID)
	if err != nil {
		if !errors.Is(err, apperrs.ErrNotFound) {
			return nil, fmt.Errorf("get active version for automation %s: %w", automationID, err)
		}
		v := &Version{
			ID:           ids.New(),
			AutomationID: automationID,
			Code:         code,
			PusherID:     seedPusherID,
			Message:      message,
			Status:       VersionActive,
			CreatedAt:    s.now().UTC(),
		}
		if err := s.repo.InsertActive(ctx, v); err != nil {
			return nil, fmt.Errorf("seed initial version for automation %s: %w", automationID, err)
		}
		return v, nil
	}
	if active.Code == code {
		return nil, nil
	}
	v := &Version{
		ID:           ids.New(),
		AutomationID: automationID,
		Code:         code,
		PusherID:     seedPusherID,
		Message:      message,
		Status:       VersionPending,
		CreatedAt:    s.now().UTC(),
	}
	if err := s.repo.ReplacePending(ctx, v); err != nil {
		return nil, fmt.Errorf("land upgrade version for automation %s: %w", automationID, err)
	}
	return v, nil
}

func (s *VersionsService) mustExist(ctx context.Context, automationID string) error {
	if strings.TrimSpace(automationID) == "" {
		return fmt.Errorf("%w: automation id is required", apperrs.ErrInvalid)
	}
	if _, err := s.automations.Get(ctx, automationID); err != nil {
		return fmt.Errorf("get automation %s: %w", automationID, err)
	}
	return nil
}
