package dns

import (
	"context"
	"fmt"
	"slices"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/mcptool"
)

// MCPTools returns the dns tool definitions (named domain_action); credentials come from connectors, not this package.
func MCPTools(s *Service) []mcptool.Tool {
	return slices.Concat(
		zoneTools(s),
		recordTools(s),
		serviceHostnameTools(s),
		tunnelProvisioningTools(s),
		gatewayTools(s),
		exposureTools(s),
	)
}

func zoneTools(s *Service) []mcptool.Tool {
	return []mcptool.Tool{
		{
			Name:        "dns_verify_credentials",
			Description: "Verify the configured DNS provider credentials are valid.",
			InputSchema: objectSchema(nil),
			Call: func(ctx context.Context, _ map[string]any) (any, error) {
				if err := s.VerifyCredentials(ctx); err != nil {
					return nil, err
				}
				return map[string]string{"status": "ok"}, nil
			},
		},
		{
			Name:        "dns_list_zones",
			Description: "List the DNS zones the connected credentials can edit.",
			InputSchema: objectSchema(nil),
			Call: func(ctx context.Context, _ map[string]any) (any, error) {
				return s.ListZones(ctx)
			},
		},
	}
}

func recordTools(s *Service) []mcptool.Tool {
	return []mcptool.Tool{
		{
			Name:        "dns_list_records",
			Description: "List the DNS records in a zone.",
			InputSchema: objectSchema(map[string]any{
				"zone_id": map[string]any{"type": "string"},
			}, "zone_id"),
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				zoneID, err := mcptool.RequiredString(args, "zone_id")
				if err != nil {
					return nil, err
				}
				return s.ListRecords(ctx, zoneID)
			},
		},
		{
			Name:        "dns_create_record",
			Description: "Create a DNS record (A, AAAA, CNAME, TXT) in a zone.",
			InputSchema: objectSchema(map[string]any{
				"zone_id": map[string]any{"type": "string"},
				"type":    map[string]any{"type": "string", "enum": []string{"A", "AAAA", "CNAME", "TXT"}},
				"name":    map[string]any{"type": "string"},
				"content": map[string]any{"type": "string"},
				"ttl":     map[string]any{"type": "integer"},
			}, "zone_id", "type", "name", "content"),
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				vals, err := mcptool.RequiredStrings(args, "zone_id", "type", "name", "content")
				if err != nil {
					return nil, err
				}
				zoneID, recType, name, content := vals[0], vals[1], vals[2], vals[3]
				return s.CreateRecord(ctx, zoneID, RecordInput{
					Type: RecordType(recType), Name: name, Content: content, TTL: intArg(args["ttl"]),
				})
			},
		},
		{
			Name:        "dns_update_record",
			Description: "Update a DNS record's content in a zone.",
			InputSchema: objectSchema(map[string]any{
				"zone_id":   map[string]any{"type": "string"},
				"record_id": map[string]any{"type": "string"},
				"type":      map[string]any{"type": "string", "enum": []string{"A", "AAAA", "CNAME", "TXT"}},
				"name":      map[string]any{"type": "string"},
				"content":   map[string]any{"type": "string"},
				"ttl":       map[string]any{"type": "integer"},
			}, "zone_id", "record_id", "type", "name", "content"),
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				vals, err := mcptool.RequiredStrings(args, "zone_id", "record_id", "type", "name", "content")
				if err != nil {
					return nil, err
				}
				zoneID, recordID, recType, name, content := vals[0], vals[1], vals[2], vals[3], vals[4]
				return s.UpdateRecord(ctx, zoneID, recordID, RecordInput{
					Type: RecordType(recType), Name: name, Content: content, TTL: intArg(args["ttl"]),
				})
			},
		},
		{
			Name:        "dns_delete_record",
			Description: "Delete a DNS record from a zone (idempotent).",
			InputSchema: objectSchema(map[string]any{
				"zone_id":   map[string]any{"type": "string"},
				"record_id": map[string]any{"type": "string"},
			}, "zone_id", "record_id"),
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				vals, err := mcptool.RequiredStrings(args, "zone_id", "record_id")
				if err != nil {
					return nil, err
				}
				zoneID, recordID := vals[0], vals[1]
				if err := s.DeleteRecord(ctx, zoneID, recordID); err != nil {
					return nil, err
				}
				return map[string]string{"status": "deleted"}, nil
			},
		},
		{
			Name:        "dns_check_propagation",
			Description: "Verify a record has propagated to public DNS.",
			InputSchema: objectSchema(map[string]any{
				"zone_id":   map[string]any{"type": "string"},
				"record_id": map[string]any{"type": "string"},
			}, "zone_id", "record_id"),
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				vals, err := mcptool.RequiredStrings(args, "zone_id", "record_id")
				if err != nil {
					return nil, err
				}
				zoneID, recordID := vals[0], vals[1]
				records, err := s.ListRecords(ctx, zoneID)
				if err != nil {
					return nil, err
				}
				for i := range records {
					if records[i].ID == recordID {
						if err := s.CheckPropagation(ctx, zoneID, records[i]); err != nil {
							return nil, err
						}
						return map[string]string{"status": "propagated"}, nil
					}
				}
				return nil, fmt.Errorf("%w: record %s not found in zone", apperrs.ErrNotFound, recordID)
			},
		},
	}
}

func serviceHostnameTools(s *Service) []mcptool.Tool {
	return []mcptool.Tool{
		{
			Name:        "dns_list_service_hostnames",
			Description: "List the hostname associations for deployed services.",
			InputSchema: objectSchema(nil),
			Call: func(ctx context.Context, _ map[string]any) (any, error) {
				return s.ListServiceHostnames(ctx)
			},
		},
		{
			Name:        "dns_get_service_hostname",
			Description: "Get one service's hostname association.",
			InputSchema: objectSchema(map[string]any{
				"service": map[string]any{"type": "string"},
			}, "service"),
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				service, err := mcptool.RequiredString(args, "service")
				if err != nil {
					return nil, err
				}
				return s.GetServiceHostname(ctx, service)
			},
		},
		{
			Name:        "dns_set_service_hostname",
			Description: "Create a DNS record for a deployed service's hostname and associate it.",
			InputSchema: objectSchema(map[string]any{
				"service":  map[string]any{"type": "string"},
				"hostname": map[string]any{"type": "string"},
				"zone_id":  map[string]any{"type": "string"},
				"zone":     map[string]any{"type": "string"},
				"type":     map[string]any{"type": "string", "enum": []string{"A", "AAAA", "CNAME", "TXT"}},
				"target":   map[string]any{"type": "string"},
			}, "service", "hostname", "zone_id", "zone", "type", "target"),
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				vals, err := mcptool.RequiredStrings(args, "service", "hostname", "zone_id", "zone", "type", "target")
				if err != nil {
					return nil, err
				}
				service, hostname, zoneID, zone, recType, target := vals[0], vals[1], vals[2], vals[3], vals[4], vals[5]
				return s.SetServiceHostname(ctx, ServiceHostnameInput{
					Service: service, Hostname: hostname, ZoneID: zoneID, Zone: zone,
					Type: RecordType(recType), Target: target,
				})
			},
		},
		{
			Name:        "dns_remove_service_hostname",
			Description: "Delete a service's DNS record and drop its hostname association.",
			InputSchema: objectSchema(map[string]any{
				"service": map[string]any{"type": "string"},
			}, "service"),
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				service, err := mcptool.RequiredString(args, "service")
				if err != nil {
					return nil, err
				}
				if err := s.RemoveServiceHostname(ctx, service); err != nil {
					return nil, err
				}
				return map[string]string{"status": "deleted"}, nil
			},
		},
	}
}

func tunnelProvisioningTools(s *Service) []mcptool.Tool {
	return []mcptool.Tool{
		{
			Name:        "dns_tunnel_create",
			Description: "Create a Cloudflare tunnel and store its credentials.",
			InputSchema: objectSchema(map[string]any{
				"name": map[string]any{"type": "string"},
			}, "name"),
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				name, err := mcptool.RequiredString(args, "name")
				if err != nil {
					return nil, err
				}
				return s.CreateTunnel(ctx, CreateTunnelInput{Name: name})
			},
		},
		{
			Name:        "dns_tunnel_list",
			Description: "List the locally-tracked Cloudflare tunnels.",
			InputSchema: objectSchema(nil),
			Call: func(ctx context.Context, _ map[string]any) (any, error) {
				return s.ListTunnels(ctx)
			},
		},
		{
			Name:        "dns_tunnel_get",
			Description: "Get one locally-tracked Cloudflare tunnel.",
			InputSchema: objectSchema(map[string]any{
				"tunnel_id": map[string]any{"type": "string"},
			}, "tunnel_id"),
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				id, err := mcptool.RequiredString(args, "tunnel_id")
				if err != nil {
					return nil, err
				}
				return s.GetTunnel(ctx, id)
			},
		},
		{
			Name:        "dns_tunnel_route",
			Description: "Route a public hostname into a Cloudflare tunnel (ingress + CNAME record).",
			InputSchema: objectSchema(map[string]any{
				"tunnel_id": map[string]any{"type": "string"},
				"hostname":  map[string]any{"type": "string"},
				"zone_id":   map[string]any{"type": "string"},
				"zone":      map[string]any{"type": "string"},
				"service":   map[string]any{"type": "string"},
			}, "tunnel_id", "hostname", "zone_id", "zone", "service"),
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				vals, err := mcptool.RequiredStrings(args, "tunnel_id", "hostname", "zone_id", "zone", "service")
				if err != nil {
					return nil, err
				}
				tunnelID, hostname, zoneID, zone, service := vals[0], vals[1], vals[2], vals[3], vals[4]
				return s.RouteTunnelHostname(ctx, RouteTunnelInput{
					TunnelID: tunnelID, Hostname: hostname, ZoneID: zoneID, Zone: zone, Service: service,
				})
			},
		},
		{
			Name:        "dns_tunnel_rotate",
			Description: "Rotate a tunnel's credentials and store the new token.",
			InputSchema: objectSchema(map[string]any{
				"tunnel_id": map[string]any{"type": "string"},
			}, "tunnel_id"),
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				id, err := mcptool.RequiredString(args, "tunnel_id")
				if err != nil {
					return nil, err
				}
				return s.RotateTunnelCredentials(ctx, id)
			},
		},
		{
			Name:        "dns_tunnel_delete",
			Description: "Delete a Cloudflare tunnel and its local association (idempotent).",
			InputSchema: objectSchema(map[string]any{
				"tunnel_id": map[string]any{"type": "string"},
			}, "tunnel_id"),
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				id, err := mcptool.RequiredString(args, "tunnel_id")
				if err != nil {
					return nil, err
				}
				if err := s.DeleteTunnel(ctx, id); err != nil {
					return nil, err
				}
				return map[string]string{"status": "deleted"}, nil
			},
		},
		{
			Name:        "dns_tunnel_provision_agent",
			Description: "Provision the cloudflared service definition for a tunnel, feeding it the tunnel token.",
			InputSchema: objectSchema(map[string]any{
				"tunnel_id":      map[string]any{"type": "string"},
				"project_id":     map[string]any{"type": "string"},
				"target":         map[string]any{"type": "string"},
				"name":           map[string]any{"type": "string"},
				"strategy":       map[string]any{"type": "string"},
				"docker_network": map[string]any{"type": "string"},
				"ports":          map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
				"health_url":     map[string]any{"type": "string"},
			}, "tunnel_id", "project_id", "target"),
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				vals, err := mcptool.RequiredStrings(args, "tunnel_id", "project_id", "target")
				if err != nil {
					return nil, err
				}
				tunnelID, projectID, target := vals[0], vals[1], vals[2]
				spec := AgentSpec{
					ProjectID:     projectID,
					Target:        target,
					Name:          strArg(args["name"]),
					Strategy:      strArg(args["strategy"]),
					DockerNetwork: strArg(args["docker_network"]),
					HealthURL:     strArg(args["health_url"]),
				}
				return s.ProvisionTunnelAgent(ctx, tunnelID, spec)
			},
		},
		{
			Name:        "dns_provision_reverse_proxy",
			Description: "Provision a reverse-proxy service definition (Traefik) as a Nexul service.",
			InputSchema: objectSchema(map[string]any{
				"project_id":     map[string]any{"type": "string"},
				"target":         map[string]any{"type": "string"},
				"name":           map[string]any{"type": "string"},
				"strategy":       map[string]any{"type": "string"},
				"compose_dir":    map[string]any{"type": "string"},
				"docker_network": map[string]any{"type": "string"},
				"ports":          map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
				"image":          map[string]any{"type": "string"},
				"health_url":     map[string]any{"type": "string"},
			}, "project_id", "target"),
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				vals, err := mcptool.RequiredStrings(args, "project_id", "target")
				if err != nil {
					return nil, err
				}
				projectID, target := vals[0], vals[1]
				spec := AgentSpec{
					ProjectID:     projectID,
					Target:        target,
					Name:          strArg(args["name"]),
					Strategy:      strArg(args["strategy"]),
					ComposeDir:    strArg(args["compose_dir"]),
					DockerNetwork: strArg(args["docker_network"]),
					Ports:         stringSliceArg(args["ports"]),
					Image:         strArg(args["image"]),
					HealthURL:     strArg(args["health_url"]),
				}
				return s.ProvisionReverseProxy(ctx, spec)
			},
		},
	}
}

func gatewayTools(s *Service) []mcptool.Tool {
	return []mcptool.Tool{
		{
			Name:        "dns_gateway_create",
			Description: "Create a gateway (tunnel or proxy) giving a docker network internet reachability; provisions the backing service.",
			InputSchema: objectSchema(map[string]any{
				"kind":           map[string]any{"type": "string", "enum": []string{"tunnel", "proxy"}},
				"docker_network": map[string]any{"type": "string"},
				"zone_id":        map[string]any{"type": "string"},
				"zone":           map[string]any{"type": "string"},
				"tunnel_id":      map[string]any{"type": "string"},
				"server_address": map[string]any{"type": "string"},
				"project_id":     map[string]any{"type": "string"},
				"target":         map[string]any{"type": "string"},
				"name":           map[string]any{"type": "string"},
			}, "kind", "docker_network", "zone_id", "zone", "project_id", "target"),
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				vals, err := mcptool.RequiredStrings(args, "kind", "docker_network", "zone_id", "zone", "project_id", "target")
				if err != nil {
					return nil, err
				}
				kind, network, zoneID, zone, projectID, target := vals[0], vals[1], vals[2], vals[3], vals[4], vals[5]
				return s.CreateGateway(ctx, CreateGatewayInput{
					Kind: GatewayKind(kind), DockerNetwork: network, ZoneID: zoneID, Zone: zone,
					TunnelID: strArg(args["tunnel_id"]), ServerAddress: strArg(args["server_address"]),
					ProjectID: projectID, Target: target, Name: strArg(args["name"]),
				})
			},
		},
		{
			Name:        "dns_gateway_list",
			Description: "List the locally-tracked gateways.",
			InputSchema: objectSchema(nil),
			Call: func(ctx context.Context, _ map[string]any) (any, error) {
				return s.ListGateways(ctx)
			},
		},
		{
			Name:        "dns_gateway_delete",
			Description: "Delete a gateway (deprovisions its backing service). Fails if it still has exposures.",
			InputSchema: objectSchema(map[string]any{
				"gateway_id": map[string]any{"type": "string"},
			}, "gateway_id"),
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				id, err := mcptool.RequiredString(args, "gateway_id")
				if err != nil {
					return nil, err
				}
				if err := s.DeleteGateway(ctx, id); err != nil {
					return nil, err
				}
				return map[string]string{"status": "deleted"}, nil
			},
		},
	}
}

// exposureTools' create/delete are named after the entity, not the dns package (spec §9: exposure_create,
// exposure_delete), matching stack_* in the deploy domain; dns_exposure_list keeps the package prefix since
// the spec's MCP tool list doesn't name it.
func exposureTools(s *Service) []mcptool.Tool {
	return []mcptool.Tool{
		{
			Name:        "exposure_create",
			Description: "Route a hostname through a gateway to a container (ingress rule + CNAME for a tunnel gateway, A/AAAA record for a proxy gateway). Without gateway_id, reuses or provisions one on the container's machine.",
			InputSchema: objectSchema(map[string]any{
				"gateway_id": map[string]any{"type": "string"},
				"hostname":   map[string]any{"type": "string"},
				"service_id": map[string]any{"type": "string"},
				"service":    map[string]any{"type": "string"},
				"port":       map[string]any{"type": "integer"},
				"zone_id":    map[string]any{"type": "string"},
				"zone":       map[string]any{"type": "string"},
				"kind":       map[string]any{"type": "string", "enum": []string{"tunnel", "proxy"}},
			}, "hostname", "port", "zone_id", "zone"),
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				vals, err := mcptool.RequiredStrings(args, "hostname", "zone_id", "zone")
				if err != nil {
					return nil, err
				}
				hostname, zoneID, zone := vals[0], vals[1], vals[2]
				return s.CreateExposure(ctx, CreateExposureInput{
					GatewayID: strArg(args["gateway_id"]), Hostname: hostname,
					ServiceID: strArg(args["service_id"]), Service: strArg(args["service"]),
					Port: intArg(args["port"]), ZoneID: zoneID, Zone: zone, Kind: GatewayKind(strArg(args["kind"])),
				})
			},
		},
		{
			Name:        "dns_exposure_list",
			Description: "List the locally-tracked exposures.",
			InputSchema: objectSchema(nil),
			Call: func(ctx context.Context, _ map[string]any) (any, error) {
				return s.ListExposures(ctx)
			},
		},
		{
			Name:        "exposure_delete",
			Description: "Remove an exposure (ingress rule / DNS record) and its local association.",
			InputSchema: objectSchema(map[string]any{
				"exposure_id": map[string]any{"type": "string"},
			}, "exposure_id"),
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				id, err := mcptool.RequiredString(args, "exposure_id")
				if err != nil {
					return nil, err
				}
				if err := s.DeleteExposure(ctx, id); err != nil {
					return nil, err
				}
				return map[string]string{"status": "deleted"}, nil
			},
		},
	}
}

func strArg(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func stringSliceArg(v any) []string {
	raw, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		if s, ok := item.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

func objectSchema(properties map[string]any, required ...string) map[string]any {
	schema := map[string]any{"type": "object", "properties": properties}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}

func intArg(v any) int {
	if f, ok := v.(float64); ok && f > 0 {
		return int(f)
	}
	return 0
}
