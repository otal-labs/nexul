package automations

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// DefaultDefinition is one shipped default automation, supplied by the composition root.
type DefaultDefinition struct {
	// ID is stable across restarts and upgrades, how Seed tells a fresh install from an existing one apart.
	ID          string
	Name        string
	Description string
	// Code is the shipped bundle text; seeding compares it against the active version to decide whether to land it.
	Code string
	// Scopes are the least-privilege token scopes this default needs.
	Scopes []string
	// Enabled sets whether a brand-new install starts on or off; ignored once the automation already exists.
	Enabled bool
}

// WorkspaceLister lists every workspace, so a startup seed reaches each one.
type WorkspaceLister interface {
	ListWorkspaceIDs(ctx context.Context) ([]string, error)
}

// Seeder upserts every workspace's default automations' identity and code.
type Seeder struct {
	automations Repo
	versions    *VersionsService
	defs        []DefaultDefinition
	workspaces  WorkspaceLister
	home        string
	log         *slog.Logger
	now         func() time.Time
}

// NewSeeder wires seeding; the home workspace's defaults keep the definitions' ids, others' get the workspace suffix.
func NewSeeder(automations Repo, versions *VersionsService, defs []DefaultDefinition, workspaces WorkspaceLister, homeWorkspaceID string, log *slog.Logger) *Seeder {
	if log == nil {
		log = slog.Default()
	}
	return &Seeder{automations: automations, versions: versions, defs: defs, workspaces: workspaces, home: homeWorkspaceID, log: log, now: time.Now}
}

// Seed runs SeedWorkspace for every workspace, so a default or code change shipped in an upgrade lands everywhere.
func (s *Seeder) Seed(ctx context.Context) error {
	workspaceIDs, err := s.workspaces.ListWorkspaceIDs(ctx)
	if err != nil {
		return fmt.Errorf("list workspaces: %w", err)
	}
	for _, workspaceID := range workspaceIDs {
		if err := s.SeedWorkspace(ctx, workspaceID); err != nil {
			return err
		}
	}
	return nil
}

// SeedWorkspace creates workspaceID's missing defaults and lands each definition's code into version storage.
func (s *Seeder) SeedWorkspace(ctx context.Context, workspaceID string) error {
	for _, def := range s.defs {
		scopes, err := normalizeScopes(def.Scopes)
		if err != nil {
			return fmt.Errorf("default automation %s scopes: %w", def.ID, err)
		}
		id := s.defaultID(def.ID, workspaceID)
		a, err := s.automations.Get(ctx, id)
		if err != nil {
			if !errors.Is(err, apperrs.ErrNotFound) {
				return fmt.Errorf("get default automation %s: %w", id, err)
			}
			a, err = s.create(ctx, def, id, workspaceID, scopes)
			if err != nil {
				return err
			}
		}
		// Scopes are code-owned: a default's token must carry whatever the shipped code needs after an upgrade.
		if !slices.Equal(a.Scopes, scopes) {
			a.Scopes = scopes
			a.UpdatedAt = s.now().UTC()
			if err := s.automations.Update(ctx, a); err != nil {
				return fmt.Errorf("update default automation %s scopes: %w", id, err)
			}
		}
		v, err := s.versions.SeedVersion(ctx, a.ID, def.Code, "seed: "+def.Name)
		if err != nil {
			return fmt.Errorf("seed version for default automation %s: %w", id, err)
		}
		if v != nil {
			s.log.Info("automations: landed default code", "automation_id", a.ID, "status", v.Status, "sequence", v.Sequence)
		}
	}
	return nil
}

// defaultID keeps the home workspace's original ids, so its version history, cursors, and runs carry over.
func (s *Seeder) defaultID(id, workspaceID string) string {
	if workspaceID == s.home {
		return id
	}
	return id + "-" + workspaceID
}

func (s *Seeder) create(ctx context.Context, def DefaultDefinition, id, workspaceID string, scopes []string) (*Automation, error) {
	// Nobody is handed the raw token: the instance host's workers dial in with host-scoped tokens derived from its hash.
	_, hash, prefix, err := mintToken()
	if err != nil {
		return nil, fmt.Errorf("mint token for default automation %s: %w", id, err)
	}
	now := s.now().UTC()
	a := &Automation{
		ID:          id,
		WorkspaceID: workspaceID,
		Name:        def.Name,
		Description: def.Description,
		Kind:        KindDefault,
		Enabled:     def.Enabled,
		Scopes:      scopes,
		TokenHash:   hash,
		TokenPrefix: prefix,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := a.Validate(); err != nil {
		return nil, fmt.Errorf("default automation %s: %w", id, err)
	}
	if err := s.automations.Create(ctx, a); err != nil {
		return nil, fmt.Errorf("create default automation %s: %w", id, err)
	}
	s.log.Info("automations: created default automation", "automation_id", a.ID, "name", a.Name)
	return a, nil
}
