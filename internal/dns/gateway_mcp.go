package dns

import (
	"context"
	"slices"

	"github.com/otal-labs/nexul/internal/platform/mcptool"
)

type gatewayListIn struct {
	mcptool.PageArgs
}

type gatewayCreateIn struct {
	Kind          GatewayKind `json:"kind" jsonschema:"tunnel (cloudflared dials out to a Cloudflare tunnel, no open ports) or proxy (Traefik listens on ports 80 and 443 of the machine)."`
	Machine       string      `json:"machine" jsonschema:"The machine the gateway's backing stack deploys to, by name, from machine_list."`
	DockerNetwork string      `json:"docker_network" jsonschema:"The gateway's home docker network, for example shop_default; a network has at most one gateway."`
	ProjectID     string      `json:"project_id" jsonschema:"The project the backing stack belongs to, from project_list."`
	ZoneID        string      `json:"zone_id" jsonschema:"The zone the gateway serves, from dns_zone_list."`
	Zone          string      `json:"zone" jsonschema:"That zone's domain name, for example example.com."`
	TunnelID      string      `json:"tunnel_id,omitempty" jsonschema:"Required for kind tunnel: the tunnel cloudflared connects to, from dns_tunnel_list or dns_tunnel_create."`
	ServerAddress string      `json:"server_address,omitempty" jsonschema:"Required for kind proxy: the machine's public IP address or hostname that exposure records point at, for example 203.0.113.10."`
	StackName     string      `json:"stack_name,omitempty" jsonschema:"The backing stack's name. Defaults to cloudflared-<tunnel name> or traefik-<random>."`
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
			"Lists the gateways Nexul deployed, oldest first: each one's kind, home docker network and every network "+
				"it joined since, machine, backing stack, and the tunnel or server address behind it. Use it to pick a "+
				"gateway_id for exposure_create, or to find the gateway a network already has. It reads only Nexul's "+
				"own records; a tunnel's live connection state is on dns_tunnel_list with the tunnel's id.",
			mcptool.Hints{ReadOnly: true, Local: true},
			func(ctx context.Context, in gatewayListIn) (any, error) {
				gws, err := s.ListGateways(ctx)
				if err != nil {
					return nil, err
				}
				return mcptool.Paginate(shapeAll(gws, toGatewayResult), in.PageArgs), nil
			}),
		mcptool.New("gateway_create", "Create gateway",
			"Creates a gateway, the service giving one docker network's containers internet reachability, and "+
				"deploys its backing stack on the machine: cloudflared for kind tunnel, Traefik for kind proxy. "+
				"exposure_create already reuses or deploys a gateway on the container's machine, so use this to choose "+
				"the kind, network, or tunnel yourself, or to give the instance's own network a gateway. A tunnel gateway "+
				"needs a tunnel_id from dns_tunnel_create or dns_tunnel_list, and a proxy gateway needs the machine's "+
				"public server_address. A network that already has a gateway fails with a conflict; gateway_list shows it.",
			mcptool.Hints{},
			func(ctx context.Context, in gatewayCreateIn) (any, error) {
				g, err := s.CreateGateway(ctx, CreateGatewayInput{
					Kind: in.Kind, DockerNetwork: in.DockerNetwork, ZoneID: in.ZoneID, Zone: in.Zone,
					TunnelID: in.TunnelID, ServerAddress: in.ServerAddress,
					ProjectID: in.ProjectID, Target: in.Machine, Name: in.StackName,
				})
				if err != nil {
					return nil, listedBy(err, "dns_tunnel_list lists tunnels, project_list projects, and machine_list machines")
				}
				return toGatewayResult(g), nil
			}),
		mcptool.New("gateway_delete", "Delete gateway",
			"Deletes a gateway and tears down its backing stack, returning its id with deleted set. It refuses while "+
				"exposures still route through the gateway: exposure_list with gateway_id finds them and exposure_delete "+
				"removes them. The tunnel behind a tunnel gateway stays, since other gateways or the instance may use "+
				"it; dns_tunnel_delete removes it.",
			mcptool.Hints{Idempotent: true},
			func(ctx context.Context, in gatewayDeleteIn) (any, error) {
				if err := s.DeleteGateway(ctx, in.ID); err != nil {
					return nil, listedBy(err, "gateway_list lists gateways")
				}
				return mcptool.Gone(in.ID), nil
			}),
	}
}

type exposureListIn struct {
	ServiceID string `json:"service_id,omitempty" jsonschema:"Only exposures routing to this container, a service id from stack_get."`
	GatewayID string `json:"gateway_id,omitempty" jsonschema:"Only exposures routing through this gateway, from gateway_list."`
	mcptool.PageArgs
}

type exposureCreateIn struct {
	Hostname  string      `json:"hostname" jsonschema:"The public hostname, under zone, for example app.example.com."`
	ServiceID string      `json:"service_id" jsonschema:"The container to route to: a service id from stack_get."`
	Port      int         `json:"port" jsonschema:"The container port to route to, for example 8080."`
	ZoneID    string      `json:"zone_id" jsonschema:"The hostname's zone, from dns_zone_list."`
	Zone      string      `json:"zone" jsonschema:"That zone's domain name, for example example.com."`
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
				"narrowed to one container or one gateway. Each carries its hostname, gateway_id, the container's "+
				"service_id and service_name, the port, and the record_id of the DNS record behind it. Use it to find "+
				"where a service is reachable, or the exposure id exposure_delete needs.",
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
			"Makes a container reachable at a hostname: routes the hostname through a gateway to the container's "+
				"port and creates its DNS record, a proxied CNAME to the tunnel or an A or AAAA record at the proxy's "+
				"server address. Without gateway_id it reuses a gateway on the container's machine, preferring one "+
				"already on the container's network, or deploys a new one of the given kind. The gateway joins the "+
				"container's networks, right away when the container is running, else on its next deploy. Returns "+
				"the exposure; exposure_delete reverses it.",
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
			"Removes an exposure and returns its id with deleted set. It deletes the tunnel's ingress rule for the "+
				"hostname on a tunnel gateway, the hostname's DNS record, and Nexul's record of the exposure. The "+
				"container and the gateway keep running; gateway_delete removes a gateway once no exposure routes "+
				"through it.",
			mcptool.Hints{Idempotent: true},
			func(ctx context.Context, in exposureDeleteIn) (any, error) {
				if err := s.DeleteExposure(ctx, in.ID); err != nil {
					return nil, listedBy(err, "exposure_list lists exposures")
				}
				return mcptool.Gone(in.ID), nil
			}),
	}
}
