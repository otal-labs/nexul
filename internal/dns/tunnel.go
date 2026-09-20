package dns

import (
	"context"
	"fmt"
	"strings"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// Tunnel is a Cloudflare tunnel Nexul manages; the association, hostname, and token are stored locally.
// TunnelRoute is one ingress rule at the provider: a public hostname and the local service it reaches.
type TunnelRoute struct {
	Hostname string `json:"hostname"`
	Service  string `json:"service"`
}

// TunnelInfo is what the provider knows about a tunnel found running on a machine: its identity and
// status, whether this instance already tracks it, the hostnames its ingress routes, and the DNS records
// pointing at it across the account's zones. Read-only; adoption stays a separate step.
type TunnelInfo struct {
	ID      string        `json:"id"`
	Name    string        `json:"name"`
	Status  string        `json:"status"`
	Tracked bool          `json:"tracked"`
	Routes  []TunnelRoute `json:"routes"`
	Records []Record      `json:"records"`
}

type Tunnel struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	AccountID string `json:"account_id"`
	Status    string `json:"status"`
	// Hostname is the public hostname routed into the tunnel; records live at the provider.
	Hostname string `json:"hostname,omitempty"`
	ZoneID   string `json:"zone_id,omitempty"`
	Zone     string `json:"zone,omitempty"`
	RecordID string `json:"record_id,omitempty"`
	Service  string `json:"service,omitempty"`
	// AgentServiceID is the service definition running cloudflared for this tunnel, if provisioned.
	AgentServiceID string `json:"agent_service_id,omitempty"`
	// Token is the cloudflared tunnel token, encrypted at rest; never serialized, logged, or put on events.
	Token     string
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CreateTunnelInput is the validated input to Service.CreateTunnel.
type CreateTunnelInput struct {
	Name string `json:"name"`
}

// Validate rejects tunnel creations that cannot proceed.
func (in CreateTunnelInput) Validate() error {
	if strings.TrimSpace(in.Name) == "" {
		return fmt.Errorf("%w: tunnel name is required", apperrs.ErrInvalid)
	}
	return nil
}

// RouteTunnelInput routes a hostname into a tunnel: an ingress rule plus a CNAME pointing at the tunnel.
type RouteTunnelInput struct {
	TunnelID string `json:"tunnel_id"`
	Hostname string `json:"hostname"`
	ZoneID   string `json:"zone_id"`
	Zone     string `json:"zone"`
	// Service is the local service URL cloudflared forwards to.
	Service string `json:"service"`
}

// Validate rejects tunnel routes that cannot be created.
func (in RouteTunnelInput) Validate() error {
	if strings.TrimSpace(in.TunnelID) == "" {
		return fmt.Errorf("%w: tunnel id is required", apperrs.ErrInvalid)
	}
	if strings.TrimSpace(in.Hostname) == "" {
		return fmt.Errorf("%w: hostname is required", apperrs.ErrInvalid)
	}
	if strings.TrimSpace(in.ZoneID) == "" {
		return fmt.Errorf("%w: zone is required", apperrs.ErrInvalid)
	}
	if strings.TrimSpace(in.Zone) == "" {
		return fmt.Errorf("%w: zone is required", apperrs.ErrInvalid)
	}
	if strings.TrimSpace(in.Service) == "" {
		return fmt.Errorf("%w: local service is required", apperrs.ErrInvalid)
	}
	return nil
}

// AgentSpec is the dns-side view of a service for an entry-path agent; the root adapts it to a deploy service (ADR 0017).
type AgentSpec struct {
	ProjectID     string   `json:"project_id"`
	Target        string   `json:"target"`
	Name          string   `json:"name,omitempty"`
	Strategy      string   `json:"strategy,omitempty"`
	ComposeDir    string   `json:"compose_dir,omitempty"`
	DockerNetwork string   `json:"docker_network,omitempty"`
	Ports         []string `json:"ports,omitempty"`
	// Mounts are host→container bind mounts, e.g. a proxy gateway's Traefik watching the Docker socket.
	Mounts []string `json:"mounts,omitempty"`
	// Command overrides the image's default command; cloudflared needs `tunnel run` to connect.
	Command   []string          `json:"command,omitempty"`
	Image     string            `json:"image,omitempty"`
	Env       map[string]string `json:"env,omitempty"`
	HealthURL string            `json:"health_url,omitempty"`
}

// AgentProvisioned is the result of provisioning an entry-path agent.
type AgentProvisioned struct {
	ServiceID string `json:"service_id"`
}

// ServiceProvisioner is the deploy seam for entry-path agents (ADR 0037): creates and deploys a service, or tears one down.
type ServiceProvisioner interface {
	Provision(ctx context.Context, in AgentSpec) (*AgentProvisioned, error)
	// Deprovision removes the backing service definition; already-absent is a no-op success.
	Deprovision(ctx context.Context, serviceID string) error
}
