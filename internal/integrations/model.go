// Package integrations implements scoped API tokens, signed webhooks-out, the schema catalog, and audit log (ADR 0043).
package integrations

import "time"

// TrustTier is the trust level shown at install (ADR 0043): verified is reviewed, community is at-your-own-risk.
type TrustTier string

const (
	TrustVerified  TrustTier = "verified"
	TrustCommunity TrustTier = "community"
)

// Install is the consent record minted at install (ADR 0043); revoked stops auth and webhooks immediately.
type Install struct {
	ID            string     `json:"id"`
	Name          string     `json:"name"`
	TrustTier     TrustTier  `json:"trust_tier"`
	WebhookURL    string     `json:"webhook_url"`
	WebhookSecret string     `json:"-"`
	Scopes        []Scope    `json:"scopes"`
	CreatedBy     string     `json:"created_by"`
	CreatedAt     time.Time  `json:"created_at"`
	RevokedAt     *time.Time `json:"revoked_at,omitempty"`
}

// IntegrationToken is a scoped credential minted at install (ADR 0043); only its hash is stored, the raw value shown once.
type IntegrationToken struct {
	ID         string     `json:"id"`
	InstallID  string     `json:"install_id"`
	TokenHash  string     `json:"-"`
	Prefix     string     `json:"prefix"`
	CreatedAt  time.Time  `json:"created_at"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	RevokedAt  *time.Time `json:"revoked_at,omitempty"`
}

// Subscription binds an install's webhook URL to one topic (ADR 0043); fan-out enqueues a delivery for every event on it.
type Subscription struct {
	InstallID string    `json:"install_id"`
	Topic     string    `json:"topic"`
	CreatedAt time.Time `json:"created_at"`
}

// DeliveryStatus is one delivery's lifecycle (ADR 0043): pending -> delivered, or pending -> failed (backoff) -> dead.
type DeliveryStatus string

const (
	DeliveryPending   DeliveryStatus = "pending"
	DeliveryDelivered DeliveryStatus = "delivered"
	DeliveryFailed    DeliveryStatus = "failed"
	DeliveryDead      DeliveryStatus = "dead"
)

// Delivery is one signed webhook POST; URL is the install's webhook URL resolved at enqueue time, not read live.
type Delivery struct {
	ID            string         `json:"id"`
	InstallID     string         `json:"install_id"`
	Topic         string         `json:"topic"`
	EventID       string         `json:"event_id"`
	Payload       []byte         `json:"payload"`
	Signature     string         `json:"signature"`
	URL           string         `json:"url"`
	Status        DeliveryStatus `json:"status"`
	Attempts      int            `json:"attempts"`
	NextAttemptAt time.Time      `json:"next_attempt_at"`
	CreatedAt     time.Time      `json:"created_at"`
	DeliveredAt   *time.Time     `json:"delivered_at,omitempty"`
}

// AuditEntry is one audit-log row: who did what with which token, so a misbehaving integration is diagnosable.
type AuditEntry struct {
	ID        string    `json:"id"`
	ActorType string    `json:"actor_type"`
	ActorID   string    `json:"actor_id"`
	TokenID   string    `json:"token_id,omitempty"`
	Action    string    `json:"action"`
	CreatedAt time.Time `json:"created_at"`
}

// SchemaEntry is one published version of an event's JSON Schema (ADR 0044); breaking changes ship as a new version.
type SchemaEntry struct {
	Topic     string    `json:"topic"`
	Version   int       `json:"version"`
	Schema    string    `json:"schema"`
	CreatedAt time.Time `json:"created_at"`
}
