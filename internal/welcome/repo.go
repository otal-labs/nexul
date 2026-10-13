package welcome

import (
	"context"
	"time"

	"github.com/otal-labs/nexul/internal/platform/eventbus"
)

// Repo persists the per-person step records; each write carries its events for the outbox in the same transaction.
type Repo interface {
	// Records returns every record kept for userIDs, in one statement whatever their number.
	Records(ctx context.Context, userIDs []string) ([]Record, error)
	// SetSkipped sets (at) or clears (nil) userID's skip of step.
	SetSkipped(ctx context.Context, userID, step string, at *time.Time, events ...eventbus.OutboxEvent) error
	// Give clears each gift's skip and sets its given time when it has one.
	Give(ctx context.Context, gifts []Gift, events ...eventbus.OutboxEvent) error
	// MarkDone sets userID's step done at, keeping an earlier time.
	MarkDone(ctx context.Context, userID, step string, at time.Time, events ...eventbus.OutboxEvent) error
	// Announce writes events alone, for a change another domain already committed.
	Announce(ctx context.Context, events ...eventbus.OutboxEvent) error
}

// Gift is a step given to a person again; GivenAt is nil before their first run, which gives nothing.
type Gift struct {
	UserID  string
	Step    string
	GivenAt *time.Time
}

// PeopleWith answers which of userIDs hold something another domain owns, in one statement whatever their number.
type PeopleWith func(ctx context.Context, userIDs []string) (map[string]bool, error)

// Seams are the reads of other domains' data, wired in the composition root (ADR 0017); welcome never writes their
// tables.
type Seams struct {
	// TickedSteps is the union of welcome steps over each person's roles in every workspace, the Owner's included.
	TickedSteps func(ctx context.Context, userIDs []string) (map[string][]string, error)
	// RoleSteps is the union of the steps roleIDs tick.
	RoleSteps func(ctx context.Context, roleIDs []string) ([]string, error)
	// RoleHolders lists who holds roleID.
	RoleHolders func(ctx context.Context, roleID string) ([]string, error)
	// WorkspaceMembers lists who is in workspaceID.
	WorkspaceMembers func(ctx context.Context, workspaceID string) ([]string, error)
	// GitHub is a working GitHub link; Computer a paired computer whose setup is confirmed; InAProject at least one
	// project; DeployServer a connected deploy runner they enrolled; FirstRunDone the Welcome wizard's first run.
	GitHub, Computer, InAProject, DeployServer, FirstRunDone PeopleWith
	// MarkFirstRun records userID's first run, which the auth domain keeps.
	MarkFirstRun func(ctx context.Context, userID string) error
}
