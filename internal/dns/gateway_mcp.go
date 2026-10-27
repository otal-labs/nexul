package dns

import (
	"context"
	"fmt"
	"slices"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/mcptool"
)

type gatewayListIn struct {
	mcptool.PageArgs
}

type gatewayCreateIn struct {
	Kind           GatewayKind `json:"kind" jsonschema:"tunnel (cloudflared dials out to a Cloudflare tunnel, no open ports) or proxy (Traefik listens on ports 80 and 443 of the machine and gets Let's Encrypt certificates)."`
	Machine        string      `json:"machine,omitempty" jsonschema:"The machine the backing stack deploys to, by name, from machine_list; needs Deploy to on it. Required unless instance_domain is set, then defaults to the machine of the runner named instance."`
	DockerNetwork  string      `json:"docker_network,omitempty" jsonschema:"The gateway's home docker network, for example shop_default; a network has at most one gateway. Required unless instance_domain is set, then defaults to nexul_proxy."`
	ZoneID         string      `json:"zone_id,omitempty" jsonschema:"The zone the gateway serves, from dns_zone_list. Required unless instance_domain is set."`
	Zone           string      `json:"zone,omitempty" jsonschema:"That zone's domain name. Required unless instance_domain is set."`
	TunnelID       string      `json:"tunnel_id,omitempty" jsonschema:"Required for kind tunnel: the tunnel cloudflared connects to, from dns_tunnel_list or dns_tunnel_create."`
	ServerAddress  string      `json:"server_address,omitempty" jsonschema:"Required for kind proxy without instance_domain: the machine's public IP address or hostname that exposure records point at."`
	StackName      string      `json:"stack_name,omitempty" jsonschema:"The backing stack's name. Defaults to cloudflared-<tunnel name> or traefik-<random>."`
	InstanceDomain string      `json:"instance_domain,omitempty" jsonschema:"Kind proxy only: a domain already resolving to the machine, for example nexul.example.com, that Traefik routes to this Nexul server over HTTPS."`
	Email          string      `json:"email,omitempty" jsonschema:"With instance_domain: the email for the Let's Encrypt account. Optional."`
}

type gatewayDeleteIn struct {
	ID string `json:"id" jsonschema:"The gateway's id, from gateway_list."`
}

// gatewayResult names the backing stack as a stack; the domain type calls its id a service id.
type gatewayResult struct {
	ID            string      `json:"id"`
	Kind          GatewayKind `json:"kind"`
	Machine       string      `json:"machine"`
	DockerNetwork string      `json:"docker_network"`
	Networks      []string    `json:"networks,omitempty"`
	StackID       string      `json:"stack_id,omitempty"`
	StackName     string      `json:"stack_name,omitempty"`
	TunnelID      string      `json:"tunnel_id,omitempty"`
	ZoneID        string      `json:"zone_id"`
	Zone          string      `json:"zone"`
	ServerAddress string      `json:"server_address,omitempty"`
}

func toGatewayResult(g *Gateway) gatewayResult {
	return gatewayResult{
		ID: g.ID, Kind: g.Kind, Machine: g.Machine, DockerNetwork: g.DockerNetwork, Networks: g.Networks,
		StackID: g.ServiceID, StackName: g.ServiceName, TunnelID: g.TunnelID,
		ZoneID: g.ZoneID, Zone: g.Zone, ServerAddress: g.ServerAddress,
	}
}

func gatewayTools(s *Service) []mcptool.Tool {
	return []mcptool.Tool{
		mcptool.New("gateway_list", "List gateways",
			"Lists the gateways Nexul deployed, oldest first, with kind, docker networks, machine, backing stack, and the "+
				"tunnel or server address behind each. Use it to pick a gateway_id for exposure_create, or to find the "+
				"gateway a network already has. A tunnel's live connection state is on dns_tunnel_list with the tunnel's id.",
			mcptool.Hints{ReadOnly: true, Local: true},
			func(ctx context.Context, in gatewayListIn) (any, error) {
				gws, err := s.ListGateways(ctx)
				if err != nil {
					return nil, err
				}
				return mcptool.Paginate(shapeAll(gws, toGatewayResult), in.PageArgs), nil
			}),
		mcptool.New("gateway_create", "Create gateway",
			"Creates a gateway and deploys its backing stack on the machine: cloudflared for kind tunnel, Traefik for "+
				"kind proxy. exposure_create already reuses or deploys a gateway, so use this to choose the kind, network, or "+
				"tunnel yourself, or to give the instance's own network one. With kind proxy and instance_domain it routes "+
				"that domain to this Nexul server instead, reusing or deploying the machine's proxy gateway; the certificate "+
				"is issued once the domain resolves to the machine and ports 80 and 443 are open, and calling again redeploys "+
				"the stack. Returns the gateway; stack_id is the stack whose deploys show the rollout.",
			mcptool.Hints{},
			func(ctx context.Context, in gatewayCreateIn) (any, error) {
				if in.InstanceDomain != "" {
					return s.createInstanceGateway(ctx, in)
				}
				g, err := s.CreateGateway(ctx, CreateGatewayInput{
					Kind: in.Kind, DockerNetwork: in.DockerNetwork, ZoneID: in.ZoneID, Zone: in.Zone,
					TunnelID: in.TunnelID, ServerAddress: in.ServerAddress,
					Target: in.Machine, Name: in.StackName,
				})
				if err != nil {
					return nil, listedBy(err, "dns_tunnel_list lists tunnels and machine_list machines")
				}
				return toGatewayResult(g), nil
			}),
		mcptool.New("gateway_delete", "Delete gateway",
			"Deletes a gateway and tears down its backing stack. It refuses while exposures still route through the "+
				"gateway: exposure_list with gateway_id finds them and exposure_delete removes them. The tunnel behind a "+
				"tunnel gateway stays, since other gateways or the instance may use it; dns_tunnel_delete removes it.",
			mcptool.Hints{Idempotent: true},
			func(ctx context.Context, in gatewayDeleteIn) (any, error) {
				if err := s.DeleteGateway(ctx, in.ID); err != nil {
					return nil, listedBy(err, "gateway_list lists gateways")
				}
				return mcptool.Gone(in.ID), nil
			}),
	}
}

// createInstanceGateway is gateway_create's instance_domain form, the same use-case as POST /api/dns/instance-proxy.
func (s *Service) createInstanceGateway(ctx context.Context, in gatewayCreateIn) (any, error) {
	if in.Kind != GatewayProxy {
		return nil, fmt.Errorf("%w: instance_domain needs kind proxy", apperrs.ErrInvalid)
	}
	g, err := s.ProvisionInstanceProxy(ctx, InstanceProxyInput{
		Domain: in.InstanceDomain, Email: in.Email, Target: in.Machine, DockerNetwork: in.DockerNetwork,
	})
	if err != nil {
		return nil, listedBy(err, "machine_list lists machines")
	}
	return toGatewayResult(g), nil
}

type exposureListIn struct {
	ServiceID string `json:"service_id,omitempty" jsonschema:"Only exposures routing to this container, a service id from stack_get."`
	GatewayID string `json:"gateway_id,omitempty" jsonschema:"Only exposures routing through this gateway, from gateway_list."`
	mcptool.PageArgs
}

type exposureCreateIn struct {
	Hostname  string      `json:"hostname" jsonschema:"The public hostname under zone, for example app.example.com."`
	ServiceID string      `json:"service_id" jsonschema:"The container to route to: a service id from stack_get."`
	Port      int         `json:"port" jsonschema:"The container port to route to."`
	ZoneID    string      `json:"zone_id" jsonschema:"The hostname's zone, from dns_zone_list."`
	Zone      string      `json:"zone" jsonschema:"That zone's domain name."`
	GatewayID string      `json:"gateway_id,omitempty" jsonschema:"The gateway to route through, from gateway_list. Omit to reuse or deploy one on the container's machine."`
	Kind      GatewayKind `json:"kind,omitempty" jsonschema:"tunnel or proxy: the kind of gateway to deploy when none is reused. Omit to pick tunnel when Cloudflare is connected and a tunnel exists, else proxy."`
}

type exposureDeleteIn struct {
	ID string `json:"id" jsonschema:"The exposure's id, from exposure_list."`
}

type exposureResult struct {
	ID          string `json:"id"`
	Hostname    string `json:"hostname"`
	GatewayID   string `json:"gateway_id"`
	ServiceID   string `json:"service_id,omitempty"`
	ServiceName string `json:"service_name,omitempty"`
	Port        int    `json:"port"`
	ZoneID      string `json:"zone_id"`
	Zone        string `json:"zone"`
	RecordID    string `json:"record_id,omitempty"`
}

func toExposureResult(e *Exposure) exposureResult {
	return exposureResult{
		ID: e.ID, Hostname: e.Hostname, GatewayID: e.GatewayID, ServiceID: e.ServiceID, ServiceName: e.Service,
		Port: e.Port, ZoneID: e.ZoneID, Zone: e.Zone, RecordID: e.RecordID,
	}
}

func exposureTools(s *Service) []mcptool.Tool {
	return []mcptool.Tool{
		mcptool.New("exposure_list", "List exposures",
			"Lists exposures, the hostnames Nexul routes through a gateway to a container, oldest first, optionally "+
				"narrowed to one container or one gateway. Each carries the record_id of the DNS record behind it. Use it to "+
				"find where a service is reachable, or the exposure id exposure_delete needs.",
			mcptool.Hints{ReadOnly: true, Local: true},
			func(ctx context.Context, in exposureListIn) (any, error) {
				exps, err := s.ListExposures(ctx)
				if err != nil {
					return nil, err
				}
				exps = slices.DeleteFunc(exps, func(e *Exposure) bool {
					return (in.ServiceID != "" && e.ServiceID != in.ServiceID) || (in.GatewayID != "" && e.GatewayID != in.GatewayID)
				})
				return mcptool.Paginate(shapeAll(exps, toExposureResult), in.PageArgs), nil
			}),
		mcptool.New("exposure_create", "Expose container",
			"Makes a container reachable at a hostname: routes it through a gateway to the container's port and creates "+
				"its DNS record. Without gateway_id it reuses a gateway on the container's machine, preferring one on its "+
				"network, or deploys one of the given kind; the gateway joins the container's networks when it is running, "+
				"else on its next deploy. Returns the exposure; exposure_delete reverses it.",
			mcptool.Hints{},
			func(ctx context.Context, in exposureCreateIn) (any, error) {
				e, err := s.CreateExposure(ctx, CreateExposureInput{
					GatewayID: in.GatewayID, Hostname: in.Hostname, ServiceID: in.ServiceID, Port: in.Port,
					ZoneID: in.ZoneID, Zone: in.Zone, Kind: in.Kind,
				})
				if err != nil {
					return nil, listedBy(err, "stack_get lists a stack's services and gateway_list the gateways")
				}
				return toExposureResult(e), nil
			}),
		mcptool.New("exposure_delete", "Delete exposure",
			"Removes an exposure: the tunnel's ingress rule for the hostname on a tunnel gateway, the hostname's DNS "+
				"record, and Nexul's record of the exposure. The container and the gateway keep running. gateway_delete "+
				"removes a gateway once no exposure routes through it.",
			mcptool.Hints{Idempotent: true},
			func(ctx context.Context, in exposureDeleteIn) (any, error) {
				if err := s.DeleteExposure(ctx, in.ID); err != nil {
					return nil, listedBy(err, "exposure_list lists exposures")
				}
				return mcptool.Gone(in.ID), nil
			}),
	}
}
