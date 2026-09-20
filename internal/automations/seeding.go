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

// Seeder upserts default automations' identity and code at startup.
type Seeder struct {
	automations Repo
	versions    *VersionsService
	defs        []DefaultDefinition
	log         *slog.Logger
	now         func() time.Time
}

// NewSeeder wires the seeding hook over the automations repo, version service, and the default definitions.
func NewSeeder(automations Repo, versions *VersionsService, defs []DefaultDefinition, log *slog.Logger) *Seeder {
	if log == nil {
		log = slog.Default()
	}
	return &Seeder{automations: automations, versions: versions, defs: defs, log: log, now: time.Now}
}

// Seed creates missing default automations and lands each definition's code into version storage.
func (s *Seeder) Seed(ctx context.Context) error {
	for _, def := range s.defs {
		scopes, err := normalizeScopes(def.Scopes)
		if err != nil {
			return fmt.Errorf("default automation %s scopes: %w", def.ID, err)
		}
		a, err := s.automations.Get(ctx, def.ID)
		if err != nil {
			if !errors.Is(err, apperrs.ErrNotFound) {
				return fmt.Errorf("get default automation %s: %w", def.ID, err)
			}
			a, err = s.create(ctx, def, scopes)
			if err != nil {
				return err
			}
		}
		// Scopes are code-owned: a default's token must carry whatever the shipped code needs after an upgrade.
		if !slices.Equal(a.Scopes, scopes) {
			a.Scopes = scopes
			a.UpdatedAt = s.now().UTC()
			if err := s.automations.Update(ctx, a); err != nil {
				return fmt.Errorf("update default automation %s scopes: %w", def.ID, err)
			}
		}
		v, err := s.versions.SeedVersion(ctx, a.ID, def.Code, "seed: "+def.Name)
		if err != nil {
			return fmt.Errorf("seed version for default automation %s: %w", def.ID, err)
		}
		if v != nil {
			s.log.Info("automations: landed default code", "automation_id", a.ID, "status", v.Status, "sequence", v.Sequence)
		}
	}
	return nil
}

func (s *Seeder) create(ctx context.Context, def DefaultDefinition, scopes []string) (*Automation, error) {
	// No owner to hand the raw token to (defaults dial in from the host), so it's written to the host token file instead.
	rawToken, hash, prefix, err := mintToken()
	if err != nil {
		return nil, fmt.Errorf("mint token for default automation %s: %w", def.ID, err)
	}
	now := s.now().UTC()
	a := &Automation{
		ID:          def.ID,
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
		return nil, fmt.Errorf("default automation %s: %w", def.ID, err)
	}
	if err := s.automations.Create(ctx, a); err != nil {
		return nil, fmt.Errorf("create default automation %s: %w", def.ID, err)
	}
	s.log.Info("automations: created default automation", "automation_id", a.ID, "name", a.Name)
	if err := writeHostToken(a.ID, a.Name, rawToken); err != nil {
		s.log.Warn("automations: failed to write host token, the bundled host won't pick this default up on its own", "automation_id", a.ID, "error", err)
	}
	return a, nil
}
