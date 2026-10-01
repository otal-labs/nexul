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

	// GetProjectLink returns userID's own link for projectID, or a zero ProjectLink if they never set one.
	GetProjectLink(ctx context.Context, userID, projectID string) (ProjectLink, error)
	// ListProjectLinks returns every link userID has set, ordered by project.
	ListProjectLinks(ctx context.Context, userID string) ([]ProjectLink, error)
	// SaveProjectLink upserts link.UserID's own link for link.ProjectID.
	SaveProjectLink(ctx context.Context, link ProjectLink) error
	// DeleteProjectLink clears userID's link for projectID. A no-op if they never set one.
	DeleteProjectLink(ctx context.Context, userID, projectID string) error
}

// SetupStore persists setup confirmations; each write carries its event for the outbox in the same transaction.
type SetupStore interface {
	// SetSetupConfirmedAt sets or clears (nil) one of userID's own computers' overall confirmation, or ErrNotFound.
	SetSetupConfirmedAt(ctx context.Context, userID, computerID string, at *time.Time, evt eventbus.OutboxEvent) error
	// SaveSetupChoices records the Set up step's choices on one of userID's own computers, or ErrNotFound.
	SaveSetupChoices(ctx context.Context, userID, computerID string, choices SetupChoices) error
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
