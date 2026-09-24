package dns

import (
	"context"
	"errors"
	"time"
)

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

// AccessAppProvider guards one hostname with an Access app admitting only one service token.
type AccessAppProvider interface {
	// CreateAccessApp creates a self-hosted app for hostname with an inline service-auth policy and returns its id.
	CreateAccessApp(ctx context.Context, hostname, serviceTokenID string) (string, error)
	// DeleteAccessApp removes the app and its inline policy; deleting an already-absent one is a no-op success.
	DeleteAccessApp(ctx context.Context, appID string) error
}

// ServiceTokenProvider manages Access service tokens; the secret is only ever returned on create and rotate.
type ServiceTokenProvider interface {
	CreateServiceToken(ctx context.Context, name string) (*ServiceToken, error)
	RotateServiceToken(ctx context.Context, tokenID string) (*ServiceToken, error)
	// DeleteServiceToken removes a token; deleting an already-absent one is a no-op success.
	DeleteServiceToken(ctx context.Context, tokenID string) error
}

// AccessProvider is the Cloudflare Access seam; errors map to the platform sentinels plus ErrZeroTrustDisabled.
type AccessProvider interface {
	AccessAppProvider
	ServiceTokenProvider
}
