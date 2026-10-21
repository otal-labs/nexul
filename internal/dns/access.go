package dns

import (
	"context"
	"errors"
	"time"
)

// ErrCloudflareNotConnected marks an instance with no Cloudflare connection, so no DNS, tunnel, or Access call can work.
var ErrCloudflareNotConnected = errors.New("cloudflare is not connected")

// ErrZeroTrustDisabled marks a Cloudflare account that has no Zero Trust organization yet, so no Access call can work.
var ErrZeroTrustDisabled = errors.New("zero trust is not enabled on the cloudflare account")

// ServiceToken is the instance's one Cloudflare Access service token; ClientSecret is encrypted at rest and never serialized.
type ServiceToken struct {
	ID           string    `json:"id"`
	ClientID     string    `json:"client_id"`
	ClientSecret string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// AccessAppProvider removes the Access apps that closed computer tunnel hostnames (ADR 0062, retired by ADR 0146).
type AccessAppProvider interface {
	// DeleteAccessApp removes the app and its inline policy; deleting an already-absent one is a no-op success.
	DeleteAccessApp(ctx context.Context, appID string) error
}

// ServiceTokenProvider removes the instance's Access service token.
type ServiceTokenProvider interface {
	// DeleteServiceToken removes a token; deleting an already-absent one is a no-op success.
	DeleteServiceToken(ctx context.Context, tokenID string) error
}

// AccessProvider is the Cloudflare Access seam; errors map to the platform sentinels plus ErrZeroTrustDisabled.
type AccessProvider interface {
	AccessAppProvider
	ServiceTokenProvider
}
