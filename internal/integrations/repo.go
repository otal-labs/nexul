package integrations

import "context"

// InstallStore persists installs; Revoke sets revoked_at, so every token of the install stops authenticating.
type InstallStore interface {
	Create(ctx context.Context, install *Install) error
	GetByID(ctx context.Context, id string) (*Install, error)
	List(ctx context.Context) ([]*Install, error)
	Revoke(ctx context.Context, id string) error
}

// TokenStore persists scoped tokens; only the SHA-256 hash is stored, the raw value is returned once at mint time.
type TokenStore interface {
	Create(ctx context.Context, token *IntegrationToken) error
	GetByHash(ctx context.Context, hash string) (*IntegrationToken, error)
	ListByInstall(ctx context.Context, installID string) ([]*IntegrationToken, error)
	RevokeByInstall(ctx context.Context, installID string) error
	TouchLastUsed(ctx context.Context, id string) error
}

// SubscriptionStore persists per-topic webhook subscriptions (ADR 0043).
type SubscriptionStore interface {
	Add(ctx context.Context, sub Subscription) error
	Remove(ctx context.Context, installID, topic string) error
	ListByInstall(ctx context.Context, installID string) ([]Subscription, error)
	ListByTopic(ctx context.Context, topic string) ([]Subscription, error)
}

// DeliveryStore persists deliveries: Create enqueues idempotently, Due returns rows to POST, Mark* updates state.
type DeliveryStore interface {
	Create(ctx context.Context, d *Delivery) error
	Due(ctx context.Context, limit int, now int64) ([]*Delivery, error)
	MarkDelivered(ctx context.Context, id string, at int64) error
	MarkFailed(ctx context.Context, id string, attempts int, nextAttemptAt int64) error
	MarkDead(ctx context.Context, id string) error
	ListByInstall(ctx context.Context, installID string) ([]*Delivery, error)
}

// SchemaStore persists the published event-schema catalog (ADR 0044).
type SchemaStore interface {
	Publish(ctx context.Context, entry SchemaEntry) error
	Catalog(ctx context.Context) ([]SchemaEntry, error)
}

// AuditStore persists audit-log rows (who did what with which token).
type AuditStore interface {
	Append(ctx context.Context, entry AuditEntry) error
	List(ctx context.Context, limit int) ([]AuditEntry, error)
}
