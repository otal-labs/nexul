package dns

import "context"

// DNSProvider is the provider-agnostic seam for DNS records; errors map to the platform sentinels.
type DNSProvider interface {
	// Verify checks the credentials are usable (e.g. token valid and active).
	Verify(ctx context.Context) error
	// ListZones returns the zones the credentials can edit.
	ListZones(ctx context.Context) ([]Zone, error)
	// ListRecords returns the records in a zone.
	ListRecords(ctx context.Context, zoneID string) ([]Record, error)
	// CreateRecord creates a record and returns it as stored at the provider.
	CreateRecord(ctx context.Context, zoneID string, in RecordInput) (*Record, error)
	// UpdateRecord replaces a record's content at the provider.
	UpdateRecord(ctx context.Context, zoneID, recordID string, in RecordInput) (*Record, error)
	// DeleteRecord removes a record; deleting an already-absent one is a no-op success.
	DeleteRecord(ctx context.Context, zoneID, recordID string) error
	// CheckPropagation verifies the record has propagated to public DNS.
	CheckPropagation(ctx context.Context, zoneID string, rec Record) error
}

// TunnelProvider errors map to the platform sentinels like DNSProvider.
type TunnelProvider interface {
	// CreateTunnel creates a remotely-managed tunnel and returns it with its token.
	CreateTunnel(ctx context.Context, name string) (*Tunnel, error)
	// ListTunnels returns the account's tunnels.
	ListTunnels(ctx context.Context) ([]Tunnel, error)
	// GetTunnel returns one tunnel, or ErrNotFound.
	GetTunnel(ctx context.Context, tunnelID string) (*Tunnel, error)
	// DeleteTunnel removes a tunnel; deleting an already-absent one is a no-op success.
	DeleteTunnel(ctx context.Context, tunnelID string) error
	// RouteTunnelHostname upserts the ingress rule sending the hostname to the local service (read-modify-write).
	RouteTunnelHostname(ctx context.Context, tunnelID, hostname, service string) error
	// RemoveTunnelHostname drops the ingress rule for this hostname (read-modify-write, other routes survive).
	RemoveTunnelHostname(ctx context.Context, tunnelID, hostname string) error
	// ListTunnelHostnames returns the ingress rules routing hostnames into the tunnel, catch-all excluded.
	ListTunnelHostnames(ctx context.Context, tunnelID string) ([]TunnelRoute, error)
	// RotateTunnelCredentials issues a fresh tunnel token; the old one can no longer connect.
	RotateTunnelCredentials(ctx context.Context, tunnelID string) (string, error)
	// TunnelToken returns the current tunnel token.
	TunnelToken(ctx context.Context, tunnelID string) (string, error)
}
