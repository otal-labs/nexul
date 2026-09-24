package pairing

import (
	"context"
	"time"

	"github.com/otal-labs/nexul/internal/platform/eventbus"
)

// Repo is the slice of SQLite the pairing domain needs; bearer tokens arrive already encrypted.
type Repo interface {
	SetupStore

	// SaveComputer upserts a computer and its events in one transaction; the tunnel is written on create only.
	SaveComputer(ctx context.Context, c Computer, evts ...eventbus.OutboxEvent) error
	// GetComputer returns one of userID's own computers, or ErrNotFound, never leaking a mismatched owner's row.
	GetComputer(ctx context.Context, userID, id string) (*Computer, error)
	// ListComputers returns userID's paired computers, newest first.
	ListComputers(ctx context.Context, userID string) ([]Computer, error)
	// DeleteComputer removes one of userID's own computers and writes its events in one transaction, or ErrNotFound.
	DeleteComputer(ctx context.Context, userID, id string, evts ...eventbus.OutboxEvent) error

	// GetDefaults returns userID's defaults, or a zero Defaults if never set (not an error, defaults are optional).
	GetDefaults(ctx context.Context, userID string) (Defaults, error)
	// SaveDefaults upserts userID's defaults row.
	SaveDefaults(ctx context.Context, d Defaults) error

	// GetComputerByID returns a computer by id regardless of owner; ResolveTarget uses it for project-linked computers.
	GetComputerByID(ctx context.Context, id string) (*Computer, error)

	// GetProjectLink returns projectID's link, or a zero ProjectLink if never linked (optional, like GetDefaults).
	GetProjectLink(ctx context.Context, projectID string) (ProjectLink, error)
	// SaveProjectLink upserts a project's link row.
	SaveProjectLink(ctx context.Context, link ProjectLink) error
	// DeleteProjectLink clears projectID's link. A no-op if the project was never linked.
	DeleteProjectLink(ctx context.Context, projectID string) error
}

// SetupStore persists setup confirmations; each write carries its event for the outbox in the same transaction.
type SetupStore interface {
	// SetSetupConfirmedAt sets or clears (nil) one of userID's own computers' overall confirmation, or ErrNotFound.
	SetSetupConfirmedAt(ctx context.Context, userID, computerID string, at *time.Time, evt eventbus.OutboxEvent) error
	// ListProviderSetups returns a computer's per-provider rows, ordered by provider.
	ListProviderSetups(ctx context.Context, computerID string) ([]ProviderSetup, error)
	// SaveProviderSetup upserts one provider's row on a computer; a nil ConfirmedAt records an un-confirmation.
	SaveProviderSetup(ctx context.Context, computerID string, p ProviderSetup, updatedAt time.Time, evt eventbus.OutboxEvent) error
	// SaveSetupTurn upserts one provider's setup turn and its events; the transcript arrives redacted.
	SaveSetupTurn(ctx context.Context, t SetupTurn, evts ...eventbus.OutboxEvent) error
	// ListLatestSetupTurns returns each provider's newest setup turn on a computer, ordered by provider.
	ListLatestSetupTurns(ctx context.Context, computerID string) ([]SetupTurnSummary, error)
	// SetSetupMCPToken stores one of userID's own computers' encrypted setup MCP token, or ErrNotFound.
	SetSetupMCPToken(ctx context.Context, userID, computerID, sealed string) error
}
