package connectors

import "context"

// CredentialsStore is one encrypted credential per connector per instance; tokens are opaque, encrypted at rest.
type CredentialsStore interface {
	// GetCredentials returns the stored credential, or ErrNotFound if never connected.
	GetCredentials(ctx context.Context, connectorID string) (Credentials, error)
	// SaveCredentials upserts the row, for both first connection and reconnecting.
	SaveCredentials(ctx context.Context, c Credentials) error
	// DeleteCredentials removes the stored credential; already disconnected is a no-op, not an error.
	DeleteCredentials(ctx context.Context, connectorID string) error
}

// Status derives the safe, token-free view; Configured is true for an OAuth access token or manual verified fields.
func (c Credentials) Status() CredentialStatus {
	return CredentialStatus{
		Configured:  c.AccessToken != "" || len(c.ManualFields) > 0,
		ConnectedBy: c.ConnectedBy,
		ConnectedAt: c.ConnectedAt,
		ExpiresAt:   c.ExpiresAt,
	}
}
