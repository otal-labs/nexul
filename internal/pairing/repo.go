package pairing

import (
	"context"
	"time"

	"github.com/otal-labs/nexul/internal/harness"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
)

// Repo is the slice of SQLite the pairing domain needs; bearer tokens arrive already encrypted.
type Repo interface {
	SetupStore

	// SaveComputer upserts a computer and its events in one transaction; the tunnel is written on create only.
	SaveComputer(ctx context.Context, c Computer, evts ...eventbus.OutboxEvent) error
	// GetComputer returns one of userID's own computers, or ErrNotFound, never leaking a mismatched owner's row.
	GetComputer(ctx context.Context, userID, id string) (*Computer, error)
	// ListComputers returns userID's paired computers in their order.
	ListComputers(ctx context.Context, userID string) ([]Computer, error)
	// ListComputersFor returns userID's own computers in their order, then those a live rule naming them shares with
	// them, each with Shared set, in one read (ADR 0148).
	ListComputersFor(ctx context.Context, userID string) ([]Computer, error)
	// DeleteComputer removes one of userID's own computers and writes its events in one transaction, or ErrNotFound.
	DeleteComputer(ctx context.Context, userID, id string, evts ...eventbus.OutboxEvent) error
	// SetFacts replaces one of userID's own computers' facts and their time, with evt in one transaction, or ErrNotFound.
	SetFacts(ctx context.Context, userID, computerID string, facts Facts, at time.Time, evt eventbus.OutboxEvent) error
	// ListTunnelComputers returns every computer still holding a tunnel, whoever owns it, oldest first.
	ListTunnelComputers(ctx context.Context) ([]Computer, error)
	// ClearTunnel forgets one of userID's own computers' tunnel, with evt in one transaction, or ErrNotFound.
	ClearTunnel(ctx context.Context, userID, computerID string, at time.Time, evt eventbus.OutboxEvent) error
	// ComputersToMove returns, for each owner never told to move, their oldest computer still on a tunnel or an address.
	ComputersToMove(ctx context.Context) ([]Computer, error)
	// RecordMoveNotice stores userID's move notice receipt at with evt in one transaction; a receipt already there
	// keeps its time and writes nothing.
	RecordMoveNotice(ctx context.Context, userID string, at time.Time, evt eventbus.OutboxEvent) error
	// ReorderComputers puts userID's own computers in ids' order, or ErrNotFound when one is not theirs.
	ReorderComputers(ctx context.Context, userID string, ids []string, at time.Time) error
	// SetComputerPlacement sets one of userID's own computers' runs at once and chat folder, or ErrNotFound.
	SetComputerPlacement(ctx context.Context, userID, computerID string, runsAtOnce int, chatProjectID string, at time.Time) error
	// CountRunning returns how many play runs are going on each of computerIDs, leaving out the idle ones.
	CountRunning(ctx context.Context, computerIDs []string) (map[string]int, error)
	// SwitchComputerKind moves computer id from kind from to to with evt's event for its owner; one not on from is untouched.
	SwitchComputerKind(ctx context.Context, id string, from, to harness.Kind, harnessVersion string, at time.Time, evt func(userID string) eventbus.OutboxEvent) error

	// GetDefaults returns userID's defaults, or a zero Defaults if never set (not an error, defaults are optional).
	GetDefaults(ctx context.Context, userID string) (Defaults, error)
	// SaveDefaults upserts userID's defaults row.
	SaveDefaults(ctx context.Context, d Defaults) error

	// GetProjectLink returns userID's own link for projectID with its computers and checkouts, zero-valued when unset.
	GetProjectLink(ctx context.Context, userID, projectID string) (ProjectLink, error)
	// ListProjectLinks returns every link userID has set or picked a checkout for, ordered by project.
	ListProjectLinks(ctx context.Context, userID string) ([]ProjectLink, error)
	// SaveProjectLink upserts link.UserID's own link for link.ProjectID, replacing its computers and checkouts.
	SaveProjectLink(ctx context.Context, link ProjectLink) error
	// DeleteProjectLink clears userID's link and computer order for projectID, keeping the checkouts they picked.
	DeleteProjectLink(ctx context.Context, userID, projectID string) error
	// SaveCheckout remembers the T3 project userID picked for projectID on computerID.
	SaveCheckout(ctx context.Context, userID, projectID, computerID, harnessProjectID string) error
}

// SetupStore persists setup confirmations; each write carries its event for the outbox in the same transaction.
type SetupStore interface {
	// SetSetupConfirmedAt sets or clears (nil) one of userID's own computers' overall confirmation, or ErrNotFound.
	SetSetupConfirmedAt(ctx context.Context, userID, computerID string, at *time.Time, evt eventbus.OutboxEvent) error
	// SaveSetupChoices records the Set up step's choices on one of userID's own computers, or ErrNotFound.
	SaveSetupChoices(ctx context.Context, userID, computerID string, choices SetupChoices) error
	// ListProviderSetups returns a computer's per-provider rows, ordered by provider.
	ListProviderSetups(ctx context.Context, computerID string) ([]ProviderSetup, error)
	// ListOutdatedSkillsComputers returns userID's own computers with a confirmed provider whose skills are not version.
	ListOutdatedSkillsComputers(ctx context.Context, userID, version string) ([]string, error)
	// SaveProviderSetup upserts one provider's row on a computer; a nil ConfirmedAt records an un-confirmation.
	SaveProviderSetup(ctx context.Context, computerID string, p ProviderSetup, updatedAt time.Time, evt eventbus.OutboxEvent) error
	// SaveSetupTurn upserts one provider's setup turn and its events; the transcript arrives redacted.
	SaveSetupTurn(ctx context.Context, t SetupTurn, evts ...eventbus.OutboxEvent) error
	// ListLatestSetupTurns returns each provider's newest setup turn on a computer, ordered by provider.
	ListLatestSetupTurns(ctx context.Context, computerID string) ([]SetupTurnSummary, error)
	// SetSetupMCPToken stores one of userID's own computers' encrypted setup MCP token, or ErrNotFound.
	SetSetupMCPToken(ctx context.Context, userID, computerID, sealed string) error
}
