package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/otal-labs/nexul/internal/dns"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/storage/sqlcgen"
)

var _ dns.Repo = (*DNSRepo)(nil)

type DNSRepo struct {
	db *sql.DB
	w  *Serializer
	q  *sqlcgen.Queries
}

func (r *DNSRepo) UpsertServiceHostname(ctx context.Context, sh dns.ServiceHostname, evts ...eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		err := r.q.WithTx(tx).UpsertServiceHostname(ctx, sqlcgen.UpsertServiceHostnameParams{
			Service: sh.Service, Hostname: sh.Hostname, ZoneID: sh.ZoneID, Zone: sh.Zone,
			RecordID: sh.RecordID, Type: string(sh.Type), Content: sh.Content, CreatedAt: sh.CreatedAt.Unix(),
		})
		if err != nil {
			return fmt.Errorf("upsert service hostname %s: %w", sh.Service, classifyWriteErr(err))
		}
		for _, evt := range evts {
			if err := insertOutboxRow(ctx, tx, evt.ID, evt.Topic, evt.Payload); err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *DNSRepo) GetServiceHostname(ctx context.Context, service string) (*dns.ServiceHostname, error) {
	row, err := r.q.GetServiceHostname(ctx, service)
	if err != nil {
		return nil, fmt.Errorf("get service hostname %s: %w", service, notFoundIfNoRows(err))
	}
	return toServiceHostname(row), nil
}

func (r *DNSRepo) ListServiceHostnames(ctx context.Context) ([]*dns.ServiceHostname, error) {
	rows, err := r.q.ListServiceHostnames(ctx)
	if err != nil {
		return nil, fmt.Errorf("list service hostnames: %w", err)
	}
	return toServiceHostnames(rows), nil
}

func (r *DNSRepo) DeleteServiceHostname(ctx context.Context, service string, evts ...eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		n, err := r.q.WithTx(tx).DeleteServiceHostname(ctx, service)
		if err != nil {
			return fmt.Errorf("delete service hostname %s: %w", service, err)
		}
		if n == 0 {
			return fmt.Errorf("delete service hostname %s: %w", service, apperrs.ErrNotFound)
		}
		for _, evt := range evts {
			if err := insertOutboxRow(ctx, tx, evt.ID, evt.Topic, evt.Payload); err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *DNSRepo) RecordChanged(ctx context.Context, evt eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		return insertOutboxRow(ctx, tx, evt.ID, evt.Topic, evt.Payload)
	})
}

func (r *DNSRepo) SaveTunnel(ctx context.Context, t dns.Tunnel, evts ...eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		err := r.q.WithTx(tx).SaveTunnel(ctx, sqlcgen.SaveTunnelParams{
			ID: t.ID, Name: t.Name, AccountID: t.AccountID, Status: t.Status, Hostname: t.Hostname,
			ZoneID: t.ZoneID, Zone: t.Zone, RecordID: t.RecordID, Service: t.Service,
			AgentServiceID: t.AgentServiceID, Token: t.Token, CreatedAt: t.CreatedAt.Unix(), UpdatedAt: t.UpdatedAt.Unix(),
		})
		if err != nil {
			return fmt.Errorf("save tunnel %s: %w", t.ID, classifyWriteErr(err))
		}
		for _, evt := range evts {
			if err := insertOutboxRow(ctx, tx, evt.ID, evt.Topic, evt.Payload); err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *DNSRepo) GetTunnel(ctx context.Context, tunnelID string) (*dns.Tunnel, error) {
	row, err := r.q.GetTunnel(ctx, tunnelID)
	if err != nil {
		return nil, fmt.Errorf("get tunnel %s: %w", tunnelID, notFoundIfNoRows(err))
	}
	return toTunnel(row), nil
}

func (r *DNSRepo) ListTunnels(ctx context.Context) ([]*dns.Tunnel, error) {
	rows, err := r.q.ListTunnels(ctx)
	if err != nil {
		return nil, fmt.Errorf("list tunnels: %w", err)
	}
	return toTunnels(rows), nil
}

func (r *DNSRepo) DeleteTunnel(ctx context.Context, tunnelID string, evts ...eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		n, err := r.q.WithTx(tx).DeleteTunnel(ctx, tunnelID)
		if err != nil {
			return fmt.Errorf("delete tunnel %s: %w", tunnelID, err)
		}
		if n == 0 {
			return fmt.Errorf("delete tunnel %s: %w", tunnelID, apperrs.ErrNotFound)
		}
		for _, evt := range evts {
			if err := insertOutboxRow(ctx, tx, evt.ID, evt.Topic, evt.Payload); err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *DNSRepo) SaveGateway(ctx context.Context, g dns.Gateway, evts ...eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		err := r.q.WithTx(tx).SaveGateway(ctx, sqlcgen.SaveGatewayParams{
			ID: g.ID, Kind: string(g.Kind), DockerNetwork: g.DockerNetwork, Machine: g.Machine, Networks: networksListJSON(g.Networks),
			ServiceID: g.ServiceID, ServiceName: g.ServiceName, TunnelID: g.TunnelID, ZoneID: g.ZoneID, Zone: g.Zone,
			ServerAddress: g.ServerAddress, CreatedAt: g.CreatedAt.Unix(), UpdatedAt: g.UpdatedAt.Unix(),
		})
		if err != nil {
			return fmt.Errorf("save gateway %s: %w", g.ID, classifyWriteErr(err))
		}
		for _, evt := range evts {
			if err := insertOutboxRow(ctx, tx, evt.ID, evt.Topic, evt.Payload); err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *DNSRepo) GetGateway(ctx context.Context, gatewayID string) (*dns.Gateway, error) {
	row, err := r.q.GetGateway(ctx, gatewayID)
	if err != nil {
		return nil, fmt.Errorf("get gateway %s: %w", gatewayID, notFoundIfNoRows(err))
	}
	return toGateway(row), nil
}

func (r *DNSRepo) GetGatewayByNetwork(ctx context.Context, dockerNetwork string) (*dns.Gateway, error) {
	row, err := r.q.GetGatewayByNetwork(ctx, dockerNetwork)
	if err != nil {
		return nil, fmt.Errorf("get gateway for network %s: %w", dockerNetwork, notFoundIfNoRows(err))
	}
	return toGateway(row), nil
}

func (r *DNSRepo) ListGateways(ctx context.Context) ([]*dns.Gateway, error) {
	rows, err := r.q.ListGateways(ctx)
	if err != nil {
		return nil, fmt.Errorf("list gateways: %w", err)
	}
	return toGateways(rows), nil
}

func (r *DNSRepo) ListGatewaysByMachine(ctx context.Context, machine string) ([]*dns.Gateway, error) {
	rows, err := r.q.ListGatewaysByMachine(ctx, machine)
	if err != nil {
		return nil, fmt.Errorf("list gateways for machine %s: %w", machine, err)
	}
	return toGateways(rows), nil
}

func (r *DNSRepo) DeleteGateway(ctx context.Context, gatewayID string, evts ...eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		n, err := r.q.WithTx(tx).DeleteGateway(ctx, gatewayID)
		if err != nil {
			return fmt.Errorf("delete gateway %s: %w", gatewayID, err)
		}
		if n == 0 {
			return fmt.Errorf("delete gateway %s: %w", gatewayID, apperrs.ErrNotFound)
		}
		for _, evt := range evts {
			if err := insertOutboxRow(ctx, tx, evt.ID, evt.Topic, evt.Payload); err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *DNSRepo) SaveExposure(ctx context.Context, e dns.Exposure, evts ...eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		err := r.q.WithTx(tx).SaveExposure(ctx, sqlcgen.SaveExposureParams{
			ID: e.ID, GatewayID: e.GatewayID, Hostname: e.Hostname, Service: e.Service,
			ServiceID: sql.NullString{String: e.ServiceID, Valid: e.ServiceID != ""}, Port: int64(e.Port),
			ZoneID: e.ZoneID, Zone: e.Zone, RecordID: e.RecordID, CreatedAt: e.CreatedAt.Unix(), UpdatedAt: e.UpdatedAt.Unix(),
		})
		if err != nil {
			return fmt.Errorf("save exposure %s: %w", e.ID, classifyWriteErr(err))
		}
		for _, evt := range evts {
			if err := insertOutboxRow(ctx, tx, evt.ID, evt.Topic, evt.Payload); err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *DNSRepo) GetExposure(ctx context.Context, exposureID string) (*dns.Exposure, error) {
	row, err := r.q.GetExposure(ctx, exposureID)
	if err != nil {
		return nil, fmt.Errorf("get exposure %s: %w", exposureID, notFoundIfNoRows(err))
	}
	return toExposure(row), nil
}

func (r *DNSRepo) ListExposures(ctx context.Context) ([]*dns.Exposure, error) {
	rows, err := r.q.ListExposures(ctx)
	if err != nil {
		return nil, fmt.Errorf("list exposures: %w", err)
	}
	return toExposures(rows), nil
}

func (r *DNSRepo) ListExposuresByGateway(ctx context.Context, gatewayID string) ([]*dns.Exposure, error) {
	rows, err := r.q.ListExposuresByGateway(ctx, gatewayID)
	if err != nil {
		return nil, fmt.Errorf("list exposures: %w", err)
	}
	return toExposures(rows), nil
}

func (r *DNSRepo) ListExposuresByService(ctx context.Context, serviceID string) ([]*dns.Exposure, error) {
	rows, err := r.q.ListExposuresByService(ctx, sql.NullString{String: serviceID, Valid: serviceID != ""})
	if err != nil {
		return nil, fmt.Errorf("list exposures: %w", err)
	}
	return toExposures(rows), nil
}

func (r *DNSRepo) ListExposuresByServiceName(ctx context.Context, serviceName string) ([]*dns.Exposure, error) {
	rows, err := r.q.ListExposuresByServiceName(ctx, serviceName)
	if err != nil {
		return nil, fmt.Errorf("list exposures: %w", err)
	}
	return toExposures(rows), nil
}

func (r *DNSRepo) DeleteExposure(ctx context.Context, exposureID string, evts ...eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		n, err := r.q.WithTx(tx).DeleteExposure(ctx, exposureID)
		if err != nil {
			return fmt.Errorf("delete exposure %s: %w", exposureID, err)
		}
		if n == 0 {
			return fmt.Errorf("delete exposure %s: %w", exposureID, apperrs.ErrNotFound)
		}
		for _, evt := range evts {
			if err := insertOutboxRow(ctx, tx, evt.ID, evt.Topic, evt.Payload); err != nil {
				return err
			}
		}
		return nil
	})
}

func toGateway(row sqlcgen.DnsGateway) *dns.Gateway {
	return &dns.Gateway{
		ID: row.ID, Kind: dns.GatewayKind(row.Kind), DockerNetwork: row.DockerNetwork, Machine: row.Machine,
		Networks:  networksListFromJSON(row.Networks),
		ServiceID: row.ServiceID, ServiceName: row.ServiceName, TunnelID: row.TunnelID,
		ZoneID: row.ZoneID, Zone: row.Zone, ServerAddress: row.ServerAddress,
		CreatedAt: time.Unix(row.CreatedAt, 0).UTC(), UpdatedAt: time.Unix(row.UpdatedAt, 0).UTC(),
	}
}

func toGateways(rows []sqlcgen.DnsGateway) []*dns.Gateway {
	var out []*dns.Gateway
	for _, row := range rows {
		out = append(out, toGateway(row))
	}
	return out
}

// networksListJSON/networksListFromJSON persist Gateway.Networks the way stacks_repo.go persists other
// string-list columns: best-effort marshal, empty-safe unmarshal.
func networksListJSON(networks []string) string {
	if len(networks) == 0 {
		return "[]"
	}
	b, err := json.Marshal(networks)
	if err != nil {
		return "[]"
	}
	return string(b)
}

func networksListFromJSON(raw string) []string {
	var out []string
	_ = json.Unmarshal([]byte(raw), &out)
	return out
}

func toExposure(row sqlcgen.DnsExposure) *dns.Exposure {
	return &dns.Exposure{
		ID: row.ID, GatewayID: row.GatewayID, Hostname: row.Hostname, Service: row.Service,
		ServiceID: row.ServiceID.String, Port: int(row.Port),
		ZoneID: row.ZoneID, Zone: row.Zone, RecordID: row.RecordID,
		CreatedAt: time.Unix(row.CreatedAt, 0).UTC(), UpdatedAt: time.Unix(row.UpdatedAt, 0).UTC(),
	}
}

func toExposures(rows []sqlcgen.DnsExposure) []*dns.Exposure {
	var out []*dns.Exposure
	for _, row := range rows {
		out = append(out, toExposure(row))
	}
	return out
}

func toTunnel(row sqlcgen.DnsTunnel) *dns.Tunnel {
	return &dns.Tunnel{
		ID: row.ID, Name: row.Name, AccountID: row.AccountID, Status: row.Status, Hostname: row.Hostname,
		ZoneID: row.ZoneID, Zone: row.Zone, RecordID: row.RecordID, Service: row.Service,
		AgentServiceID: row.AgentServiceID, Token: row.Token,
		CreatedAt: time.Unix(row.CreatedAt, 0).UTC(), UpdatedAt: time.Unix(row.UpdatedAt, 0).UTC(),
	}
}

func toTunnels(rows []sqlcgen.DnsTunnel) []*dns.Tunnel {
	var out []*dns.Tunnel
	for _, row := range rows {
		out = append(out, toTunnel(row))
	}
	return out
}

func toServiceHostname(row sqlcgen.DnsServiceHostname) *dns.ServiceHostname {
	return &dns.ServiceHostname{
		Service: row.Service, Hostname: row.Hostname, ZoneID: row.ZoneID, Zone: row.Zone,
		RecordID: row.RecordID, Type: dns.RecordType(row.Type), Content: row.Content,
		CreatedAt: time.Unix(row.CreatedAt, 0).UTC(),
	}
}

func toServiceHostnames(rows []sqlcgen.DnsServiceHostname) []*dns.ServiceHostname {
	var out []*dns.ServiceHostname
	for _, row := range rows {
		out = append(out, toServiceHostname(row))
	}
	return out
}
