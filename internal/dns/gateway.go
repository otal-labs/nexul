package dns

import (
	"context"
	"fmt"
	"strings"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// GatewayKind is how a gateway makes its docker network's services reachable from the internet.
type GatewayKind string

const (
	GatewayTunnel GatewayKind = "tunnel"
	GatewayProxy  GatewayKind = "proxy"
)

// Gateway is a service giving a machine's containers internet reachability via hostnames (ADR 0037). DockerNetwork
// is its home network (at most one gateway per network as a home); Networks is every network it has since
// joined via docker network connect, home network included — a gateway spans networks rather than pinning to
// just the one it was created on.
type Gateway struct {
	ID            string      `json:"id"`
	Kind          GatewayKind `json:"kind"`
	DockerNetwork string      `json:"docker_network"`
	// Networks is the set of docker networks this gateway container has joined (home network plus every
	// network an exposure later added).
	Networks []string `json:"networks,omitempty"`
	// Machine is the runner name the gateway's backing service deploys to; an exposure's target must share it.
	Machine string `json:"machine"`
	// ServiceID/ServiceName are the backing service this gateway provisioned: cloudflared or Traefik.
	ServiceID   string `json:"service_id,omitempty"`
	ServiceName string `json:"service_name,omitempty"`
	// TunnelID is the dns tunnel this gateway routes into (kind tunnel only).
	TunnelID string `json:"tunnel_id,omitempty"`
	// ZoneID/Zone is the zone the gateway was created for, kept for display; an exposure's own ZoneID/Zone
	// decide where its record is created, since one gateway can serve hostnames in several zones.
	ZoneID string `json:"zone_id"`
	Zone   string `json:"zone"`
	// ServerAddress is the A/AAAA record content exposures create (kind proxy only).
	ServerAddress string    `json:"server_address,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// hasNetwork reports whether the gateway is already connected to network (home or joined).
func (g *Gateway) hasNetwork(network string) bool {
	for _, n := range g.Networks {
		if n == network {
			return true
		}
	}
	return false
}

// CreateGatewayInput is the validated input to Service.CreateGateway.
type CreateGatewayInput struct {
	Kind          GatewayKind `json:"kind"`
	DockerNetwork string      `json:"docker_network"`
	ZoneID        string      `json:"zone_id"`
	Zone          string      `json:"zone"`
	// TunnelID references an existing dns tunnel (kind tunnel).
	TunnelID string `json:"tunnel_id,omitempty"`
	// ServerAddress is the server's public IP/hostname the proxy listens on (kind proxy).
	ServerAddress string `json:"server_address,omitempty"`
	// ProjectID/Target/Name feed the backing service provisioning; Name defaults per kind when empty.
	// Target also becomes the gateway's Machine — the backing service always deploys where the gateway runs.
	ProjectID string `json:"project_id"`
	Target    string `json:"target"`
	Name      string `json:"name,omitempty"`
}

// Validate rejects gateway creations that cannot proceed.
func (in CreateGatewayInput) Validate() error {
	switch in.Kind {
	case GatewayTunnel, GatewayProxy:
	default:
		return fmt.Errorf("%w: unsupported gateway kind %q", apperrs.ErrInvalid, in.Kind)
	}
	if strings.TrimSpace(in.DockerNetwork) == "" {
		return fmt.Errorf("%w: docker network is required", apperrs.ErrInvalid)
	}
	if strings.TrimSpace(in.ZoneID) == "" || strings.TrimSpace(in.Zone) == "" {
		return fmt.Errorf("%w: zone is required", apperrs.ErrInvalid)
	}
	if strings.TrimSpace(in.ProjectID) == "" {
		return fmt.Errorf("%w: project is required", apperrs.ErrInvalid)
	}
	if strings.TrimSpace(in.Target) == "" {
		return fmt.Errorf("%w: target host is required", apperrs.ErrInvalid)
	}
	if in.Kind == GatewayTunnel && strings.TrimSpace(in.TunnelID) == "" {
		return fmt.Errorf("%w: tunnel id is required for a tunnel gateway", apperrs.ErrInvalid)
	}
	if in.Kind == GatewayProxy && strings.TrimSpace(in.ServerAddress) == "" {
		return fmt.Errorf("%w: server address is required for a proxy gateway", apperrs.ErrInvalid)
	}
	return nil
}

// ExposureTarget is the deploy-side view of a container an exposure routes to, or a gateway's own backing
// container (ADR 0017: dns never imports deploy directly). Networks falls back to the stack's own default network
// when the container hasn't reported in yet, so a not-yet-deployed stack can still be exposed/provisioned for.
type ExposureTarget struct {
	ContainerID string
	// Name is the container's runtime name, used for the tunnel origin and proxy label (Container.ContainerName,
	// falling back to the stack slug when not yet observed).
	Name      string
	StackID   string
	ProjectID string
	Machine   string
	Networks  []string
	// Running reports whether the container is currently up, so CreateExposure knows whether to ask the
	// runner to join networks immediately or leave it for the stack's next deploy.
	Running bool
}

// ContainerLookup is the deploy seam resolving the container an exposure targets or a gateway's own backing
// container (ADR 0017); dns never imports deploy directly.
type ContainerLookup interface {
	// ContainerByID resolves an exposure's service_id directly to the container it targets.
	ContainerByID(ctx context.Context, containerID string) (*ExposureTarget, error)
	// ContainerByStackName resolves the legacy "service" name — a stack's name — to its single container,
	// the fallback CreateExposure accepts for the deprecated Service field.
	ContainerByStackName(ctx context.Context, stackName string) (*ExposureTarget, error)
	// ContainerByStackID resolves a stack id to its single container, for a gateway's own backing stack
	// (Gateway.ServiceID) when a redeploy needs to rejoin it to its network.
	ContainerByStackID(ctx context.Context, stackID string) (*ExposureTarget, error)
}

// RunnerJoiner asks a connected runner to join a gateway container onto networks immediately, when an
// exposure is created for an already-running stack; best-effort, never blocks CreateExposure — a
// runner that isn't connected right now still gets the same join as part of its next deploy.
type RunnerJoiner interface {
	JoinNetworks(ctx context.Context, machine, gatewayContainer string, networks []string) error
}

// traefikImage is the pinned proxy-gateway backing image.
const traefikImage = "traefik:v3"

// dockerSocketMount is the read-only Docker socket bind mount Traefik needs
// to watch running containers for its label-driven routes.
const dockerSocketMount = "/var/run/docker.sock:/var/run/docker.sock:ro"
