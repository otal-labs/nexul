package attachments

import "context"

// Repo is the consumer-side persistence contract for attachments.
type Repo interface {
	Create(ctx context.Context, a *Attachment) error
	// GetByID returns the attachment with its bytes.
	GetByID(ctx context.Context, id string) (*Attachment, error)
	// ListByOwner returns an owner's attachments oldest first, without bytes.
	ListByOwner(ctx context.Context, owner Owner) ([]*Attachment, error)
	Delete(ctx context.Context, id string) error
}
