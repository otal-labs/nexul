package imports

import (
	"context"
	"time"

	"github.com/otal-labs/nexul/internal/platform/eventbus"
)

// ItemState is how far one record got: created is the Nexul record written, filled is its comments and files too.
type ItemState string

const (
	ItemCreated ItemState = "created"
	ItemFilled  ItemState = "filled"
	ItemFailed  ItemState = "failed"
)

// Item is one import's record of one source record.
type Item struct {
	ImportID string
	Kind     string
	SourceID string
	NexulID  string
	State    ItemState
	Error    string
}

// Repo persists imports, sealing the credential at rest; writes carry outbox events in the same transaction.
type Repo interface {
	// Create is ErrConflict when the workspace already has an import that has not ended.
	Create(ctx context.Context, imp *Import, evts ...eventbus.OutboxEvent) error
	Get(ctx context.Context, id string) (*Import, error)
	// ListByWorkspace is the workspace's imports, newest first.
	ListByWorkspace(ctx context.Context, workspaceID string, limit int) ([]*Import, error)
	// Open is every import that has not ended, oldest first.
	Open(ctx context.Context) ([]*Import, error)
	// Update writes imp's state, counts and report only while the stored state is still from, and reports whether it
	// did; a final state also clears the credential.
	Update(ctx context.Context, imp *Import, from State, evts ...eventbus.OutboxEvent) (bool, error)
	// Move changes only the state, and on a reason the report's error, while the stored state is still from; a final
	// state clears the credential. It returns the import as it now stands and the event built from it, in one write.
	Move(ctx context.Context, id string, from, to State, reason string, at time.Time, event func(*Import) eventbus.OutboxEvent) (*Import, bool, error)
	ItemRepo
}

// ItemRepo records what each import did with each record.
type ItemRepo interface {
	// Item is this import's record of a source record; false when it has none.
	Item(ctx context.Context, importID, kind, sourceID string) (Item, bool, error)
	// Imported is the Nexul id an earlier import into the workspace gave a source record; false when none did.
	Imported(ctx context.Context, workspaceID, source, kind, sourceID string) (string, bool, error)
	// Record writes the item, and on its first write past failed the workspace's source id for it, in one transaction.
	Record(ctx context.Context, workspaceID, source string, item Item) error
}
