package dns

import (
	"context"
	"fmt"

	"github.com/otal-labs/nexul/internal/platform/mcptool"
)

type tunnelListIn struct {
	ID string `json:"id,omitempty" jsonschema:"Only this tunnel, with its live connection status read from Cloudflare."`
	mcptool.PageArgs
}

type tunnelCreateIn struct {
	Name string `json:"name" jsonschema:"The tunnel's name at Cloudflare, for example instance."`
}

type tunnelUpdateIn struct {
	ID                string  `json:"id" jsonschema:"The tunnel's id, from dns_tunnel_list."`
	Hostname          *string `json:"hostname,omitempty" jsonschema:"The public hostname to route into the tunnel, for example nexul.example.com. Omit to keep the current one."`
	ZoneID            *string `json:"zone_id,omitempty" jsonschema:"The hostname's zone, from dns_zone_list. Omit to keep the current one."`
	Zone              *string `json:"zone,omitempty" jsonschema:"That zone's domain name, for example example.com. Omit to keep the current one."`
	Service           *string `json:"service,omitempty" jsonschema:"The local URL cloudflared forwards the hostname to, for example http://web:80. Omit to keep the current one."`
	RotateCredentials bool    `json:"rotate_credentials,omitempty" jsonschema:"Issue the tunnel a new token and store it. Defaults to false."`
}

func (in tunnelUpdateIn) routes() bool {
	return in.Hostname != nil || in.ZoneID != nil || in.Zone != nil || in.Service != nil
}

type tunnelDeleteIn struct {
	ID string `json:"id" jsonschema:"The tunnel's id, from dns_tunnel_list."`
}

// tunnelUpdateResult names the changes that took effect, since routing and rotation commit separately.
type tunnelUpdateResult struct {
	Tunnel  *Tunnel  `json:"tunnel"`
	Applied []string `json:"applied"`
}

func tunnelTools(s *Service) []mcptool.Tool {
	return []mcptool.Tool{
		mcptool.New("dns_tunnel_list", "List tunnels",
			"Lists the Cloudflare tunnels Nexul manages, oldest first, with the hostname each routes, its zone, the "+
				"local service cloudflared forwards to, and the stack running cloudflared. With an id it returns only "+
				"that tunnel with its live status read from Cloudflare, healthy once cloudflared is connected, which is "+
				"how to wait for a new tunnel gateway to come up. Tunnel tokens are never returned.",
			mcptool.Hints{ReadOnly: true},
			func(ctx context.Context, in tunnelListIn) (any, error) {
				if in.ID != "" {
					t, err := s.TunnelStatus(ctx, in.ID)
					if err != nil {
						return nil, err
					}
					return mcptool.Paginate([]*Tunnel{t}, in.PageArgs), nil
				}
				tunnels, err := s.ListTunnels(ctx)
				if err != nil {
					return nil, err
				}
				return mcptool.Paginate(tunnels, in.PageArgs), nil
			}),
		mcptool.New("dns_tunnel_create", "Create tunnel",
			"Creates a Cloudflare tunnel and stores its token encrypted; the token is never returned. A name Nexul "+
				"already tracks returns that tunnel instead of a second one, so a retry is safe. Next, run cloudflared "+
				"for it with gateway_create (kind tunnel), then route a hostname into it with exposure_create or "+
				"dns_tunnel_update.",
			mcptool.Hints{Additive: true, Idempotent: true},
			func(ctx context.Context, in tunnelCreateIn) (any, error) {
				return s.CreateTunnel(ctx, CreateTunnelInput(in))
			}),
		mcptool.New("dns_tunnel_update", "Update tunnel",
			"Routes a public hostname into a tunnel, rotates its credentials, or both. Routing sets the ingress rule "+
				"sending hostname to service plus a proxied CNAME pointing hostname at the tunnel, and route fields you "+
				"omit keep their current values, so passing only service re-points the current hostname; for a "+
				"container Nexul deploys, use exposure_create instead. rotate_credentials issues and stores a new "+
				"token: a running cloudflared keeps its connection, but its stack still holds the old token, so its "+
				"next reconnect fails until the gateway is recreated. Routing runs before rotation, a failure stops "+
				"the call, and the result lists what was applied.",
			mcptool.Hints{},
			func(ctx context.Context, in tunnelUpdateIn) (any, error) {
				t, err := s.GetTunnel(ctx, in.ID)
				if err != nil {
					return nil, err
				}
				applied := []string{}
				if in.routes() {
					route := RouteTunnelInput{TunnelID: t.ID, Hostname: t.Hostname, ZoneID: t.ZoneID, Zone: t.Zone, Service: t.Service}
					overlay(&route.Hostname, in.Hostname)
					overlay(&route.ZoneID, in.ZoneID)
					overlay(&route.Zone, in.Zone)
					overlay(&route.Service, in.Service)
					if t, err = s.RouteTunnelHostname(ctx, route); err != nil {
						return nil, err
					}
					applied = append(applied, "route")
				}
				if in.RotateCredentials {
					if t, err = s.RotateTunnelCredentials(ctx, in.ID); err != nil {
						return nil, fmt.Errorf("applied %v, then rotate_credentials failed: %w", applied, err)
					}
					applied = append(applied, "rotate_credentials")
				}
				return tunnelUpdateResult{Tunnel: t, Applied: applied}, nil
			}),
		mcptool.New("dns_tunnel_delete", "Delete tunnel",
			"Deletes a Cloudflare tunnel and Nexul's record of it, returning its id with deleted set. Gateways and "+
				"exposures that route through the tunnel stop working, so delete them first; gateway_list shows each "+
				"gateway's tunnel_id. DNS records pointing at the tunnel stay; dns_record_delete removes them.",
			mcptool.Hints{Idempotent: true},
			func(ctx context.Context, in tunnelDeleteIn) (any, error) {
				if err := s.DeleteTunnel(ctx, in.ID); err != nil {
					return nil, err
				}
				return mcptool.Gone(in.ID), nil
			}),
	}
}
