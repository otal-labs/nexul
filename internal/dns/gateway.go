package dns

import (
	"context"
	"encoding/json"
	"fmt"
	"net/mail"
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

// acmeVolumeMount keeps Let's Encrypt's account and certificates in a named volume, so a redeploy never re-issues.
const acmeVolumeMount = "nexul-traefik-acme:/letsencrypt"

const (
	certResolver = "letsencrypt"
	// dynamicConfigEnv carries the file provider's routes; the container's command writes it to dynamicConfigFile.
	dynamicConfigEnv  = "NEXUL_TRAEFIK_DYNAMIC"
	dynamicConfigFile = "/etc/traefik/nexul.yml"
	// instanceProxyStack/instanceProxyNetwork name a proxy gateway the instance route creates when its machine has none.
	instanceProxyStack   = "nexul-proxy"
	instanceProxyNetwork = "nexul_proxy"
)

// traefikCommand writes the dynamic config to the file provider's file, then starts Traefik on its TRAEFIK_* env
// (the image runs a non-Traefik command through sh; static config cannot mix env and flags).
func traefikCommand() []string {
	return []string{"sh", "-c", `mkdir -p /etc/traefik && printf '%s' "$` + dynamicConfigEnv + `" > ` + dynamicConfigFile + ` && exec traefik`}
}

// traefikSpec is the proxy gateway's container: docker provider for exposures, Let's Encrypt over HTTP-01 as every
// websecure router's default resolver, http redirected to https, and, when domain is set, a route to origin.
func traefikSpec(domain, origin, email string) AgentSpec {
	env := map[string]string{
		"TRAEFIK_PROVIDERS_DOCKER": "true",
		// exposedByDefault=false: Traefik sees every container on the machine's socket, so without this
		// one exposure would publish all of them; only a container an exposure labelled gets routed.
		"TRAEFIK_PROVIDERS_DOCKER_EXPOSEDBYDEFAULT":                               "false",
		"TRAEFIK_PROVIDERS_FILE_FILENAME":                                         dynamicConfigFile,
		"TRAEFIK_ENTRYPOINTS_WEB_ADDRESS":                                         ":80",
		"TRAEFIK_ENTRYPOINTS_WEB_HTTP_REDIRECTIONS_ENTRYPOINT_TO":                 "websecure",
		"TRAEFIK_ENTRYPOINTS_WEB_HTTP_REDIRECTIONS_ENTRYPOINT_SCHEME":             "https",
		"TRAEFIK_ENTRYPOINTS_WEBSECURE_ADDRESS":                                   ":443",
		"TRAEFIK_ENTRYPOINTS_WEBSECURE_HTTP_TLS_CERTRESOLVER":                     certResolver,
		"TRAEFIK_CERTIFICATESRESOLVERS_LETSENCRYPT_ACME_STORAGE":                  "/letsencrypt/acme.json",
		"TRAEFIK_CERTIFICATESRESOLVERS_LETSENCRYPT_ACME_HTTPCHALLENGE_ENTRYPOINT": "web",
		dynamicConfigEnv: dynamicConfig(domain, origin),
	}
	if email != "" {
		env["TRAEFIK_CERTIFICATESRESOLVERS_LETSENCRYPT_ACME_EMAIL"] = email
	}
	return AgentSpec{
		Strategy: "run",
		Image:    traefikImage,
		Ports:    []string{"80:80", "443:443"},
		Mounts:   []string{dockerSocketMount, acmeVolumeMount},
		Command:  traefikCommand(),
		Env:      env,
	}
}

// dynamicConfig is the file provider's config as JSON, which the YAML loader reads as-is: one router sending
// domain to origin, or no routes at all.
func dynamicConfig(domain, origin string) string {
	if domain == "" {
		return "{}"
	}
	type server struct {
		URL string `json:"url"`
	}
	cfg := map[string]any{"http": map[string]any{
		"routers": map[string]any{"nexul": map[string]any{
			"rule":        "Host(`" + domain + "`)",
			"entryPoints": []string{"websecure"},
			"service":     "nexul",
			"tls":         map[string]string{"certResolver": certResolver},
		}},
		"services": map[string]any{"nexul": map[string]any{
			"loadBalancer": map[string]any{"servers": []server{{URL: origin}}},
		}},
	}}
	// Maps of strings and string slices cannot fail to marshal.
	b, _ := json.Marshal(cfg)
	return string(b)
}

// InstanceProxyInput routes Domain through the machine's proxy gateway to this Nexul server.
type InstanceProxyInput struct {
	Domain string `json:"domain"`
	// Email registers the Let's Encrypt account; optional.
	Email string `json:"email,omitempty"`
	// Target, ProjectID, and DockerNetwork default to the bundled instance machine, the first project, and nexul_proxy.
	Target        string `json:"target,omitempty"`
	ProjectID     string `json:"project_id,omitempty"`
	DockerNetwork string `json:"docker_network,omitempty"`
}

// normalize validates the input and returns it with the domain lowercased and trimmed.
func (in InstanceProxyInput) normalize() (InstanceProxyInput, error) {
	domain, err := normalizeDomain(in.Domain)
	if err != nil {
		return in, err
	}
	in.Domain = domain
	in.Email = strings.TrimSpace(in.Email)
	if in.Email != "" {
		if _, err := mail.ParseAddress(in.Email); err != nil {
			return in, fmt.Errorf("%w: email %q is not an email address", apperrs.ErrInvalid, in.Email)
		}
	}
	return in, nil
}

// normalizeDomain accepts a bare DNS name with at least two labels, for example nexul.example.com.
func normalizeDomain(raw string) (string, error) {
	domain := strings.ToLower(strings.TrimSuffix(strings.TrimSpace(raw), "."))
	labels := strings.Split(domain, ".")
	if len(domain) > 253 || len(labels) < 2 {
		return "", fmt.Errorf("%w: domain %q is not a domain name like nexul.example.com", apperrs.ErrInvalid, raw)
	}
	for _, l := range labels {
		if !validLabel(l) {
			return "", fmt.Errorf("%w: domain %q is not a domain name like nexul.example.com", apperrs.ErrInvalid, raw)
		}
	}
	return domain, nil
}

func validLabel(l string) bool {
	if l == "" || len(l) > 63 || l[0] == '-' || l[len(l)-1] == '-' {
		return false
	}
	for _, r := range l {
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '-' {
			return false
		}
	}
	return true
}

// InstancePlacement says where the instance's own proxy deploys by default (ADR 0017): dns never imports runner or workspace.
type InstancePlacement interface {
	// InstanceMachine is the machine the bundled runner named "instance" runs on.
	InstanceMachine(ctx context.Context) (string, error)
	// DefaultProject is the project a new proxy stack belongs to.
	DefaultProject(ctx context.Context) (string, error)
}

// HostResolver looks a host name up; *net.Resolver satisfies it.
type HostResolver interface {
	LookupHost(ctx context.Context, host string) ([]string, error)
}
