package dns

import (
	"context"

	"github.com/otal-labs/nexul/internal/platform/eventbus"
)

// Repo is the slice of SQLite the dns domain needs; credentials live in connectors, reached via TokenProvider.
type Repo interface {
	// UpsertServiceHostname writes the association and its outbox events in one transaction.
	UpsertServiceHostname(ctx context.Context, sh ServiceHostname, evts ...eventbus.OutboxEvent) error
	// GetServiceHostname returns one service's association, or ErrNotFound.
	GetServiceHostname(ctx context.Context, service string) (*ServiceHostname, error)
	// ListServiceHostnames returns every association, oldest first.
	ListServiceHostnames(ctx context.Context) ([]*ServiceHostname, error)
	// DeleteServiceHostname removes the association and enqueues its outbox events in one transaction.
	DeleteServiceHostname(ctx context.Context, service string, evts ...eventbus.OutboxEvent) error

	// RecordChanged enqueues a record_changed row for a provider mutation with no local row; best-effort by nature.
	RecordChanged(ctx context.Context, evt eventbus.OutboxEvent) error

	// SaveTunnel upserts a tunnel row and its events in one transaction; the token arrives already encrypted.
	SaveTunnel(ctx context.Context, t Tunnel, evts ...eventbus.OutboxEvent) error
	// GetTunnel returns one tunnel row, or ErrNotFound.
	GetTunnel(ctx context.Context, tunnelID string) (*Tunnel, error)
	// ListTunnels returns every tunnel row, oldest first.
	ListTunnels(ctx context.Context) ([]*Tunnel, error)
	// DeleteTunnel removes a tunnel row and its outbox events in one transaction.
	DeleteTunnel(ctx context.Context, tunnelID string, evts ...eventbus.OutboxEvent) error

	// SaveAccessServiceToken upserts the instance's one Access service token; the secret arrives already encrypted.
	SaveAccessServiceToken(ctx context.Context, t ServiceToken) error
	// GetAccessServiceToken returns the stored service token, or ErrNotFound.
	GetAccessServiceToken(ctx context.Context) (*ServiceToken, error)
	// DeleteAccessServiceToken removes the stored service token, or returns ErrNotFound.
	DeleteAccessServiceToken(ctx context.Context) error

	// SaveGateway upserts a gateway row and its outbox events in one transaction.
	SaveGateway(ctx context.Context, g Gateway, evts ...eventbus.OutboxEvent) error
	// GetGateway returns one gateway row, or ErrNotFound.
	GetGateway(ctx context.Context, gatewayID string) (*Gateway, error)
	// GetGatewayByNetwork returns the gateway whose home network is dockerNetwork, or ErrNotFound.
	GetGatewayByNetwork(ctx context.Context, dockerNetwork string) (*Gateway, error)
	// ListGateways returns every gateway row, oldest first.
	ListGateways(ctx context.Context) ([]*Gateway, error)
	// ListGatewaysByMachine returns a machine's gateways, oldest first, for the wizard's reuse-or-provision step.
	ListGatewaysByMachine(ctx context.Context, machine string) ([]*Gateway, error)
	// DeleteGateway removes a gateway row and its outbox events in one transaction.
	DeleteGateway(ctx context.Context, gatewayID string, evts ...eventbus.OutboxEvent) error

	// SaveExposure upserts an exposure row and its outbox events in one transaction.
	SaveExposure(ctx context.Context, e Exposure, evts ...eventbus.OutboxEvent) error
	// GetExposure returns one exposure row, or ErrNotFound.
	GetExposure(ctx context.Context, exposureID string) (*Exposure, error)
	// ListExposures returns every exposure row, oldest first.
	ListExposures(ctx context.Context) ([]*Exposure, error)
	// ListExposuresByGateway returns a gateway's exposures, used to refuse a gateway delete while exposures exist.
	ListExposuresByGateway(ctx context.Context, gatewayID string) ([]*Exposure, error)
	// ListExposuresByService returns a container's exposures, keyed by service_id.
	ListExposuresByService(ctx context.Context, serviceID string) ([]*Exposure, error)
	// ListExposuresByServiceName returns a service's exposures by the legacy name column, for the by-name
	// alias route and the branch-deploy flow, which both still key by name.
	ListExposuresByServiceName(ctx context.Context, serviceName string) ([]*Exposure, error)
	// DeleteExposure removes an exposure row and its outbox events in one transaction.
	DeleteExposure(ctx context.Context, exposureID string, evts ...eventbus.OutboxEvent) error
}
