package botwebhook

import (
	"context"

	"github.com/otal-labs/nexul/internal/platform/eventbus"
)

// Repo persists bots, their token sealed at rest; a duplicate live name in a conversation is ErrConflict.
type Repo interface {
	Create(ctx context.Context, b *Bot, evts ...eventbus.OutboxEvent) error
	// Get returns a bot deleted or not, with its token; ErrNotFound for none.
	Get(ctx context.Context, id string) (*Bot, error)
	// List returns a conversation's live bots, or its deleted ones, oldest first.
	List(ctx context.Context, conversationID string, deleted bool) ([]*Bot, error)
	// Update writes b's name, avatar, token, and deletion; ErrNotFound for a bot that is gone.
	Update(ctx context.Context, b *Bot, evts ...eventbus.OutboxEvent) error
}
