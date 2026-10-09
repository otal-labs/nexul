import { useMutation, useQuery, useQueryClient, type QueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { api, errorMessage } from "@/api/client";
import type {
  CreateGatewayFormData,
  DnsRecord,
  Exposure,
  ExposeServiceFormData,
  Gateway,
  RecordType,
  Tunnel,
  Zone,
} from "@/models/DNS";
import { getServicesKey } from "@/hooks/ServiceHooks";
import { resolvesHere, type PublicAddress } from "@/models/Setup";
import type { LiveFollower } from "@/lib/live";

export const getDnsZonesKey = "dnsZones";

export const useFetchDnsZones = (enabled = true) =>
  useQuery({
    queryKey: [getDnsZonesKey],
    queryFn: async () => (await api.get<Zone[]>("/api/dns/zones")).data,
    enabled,
  });

export const useCreateInstanceRecord = () =>
  useMutation({
    mutationFn: async (input: {
      zone_id: string;
      zone: string;
      type: RecordType;
      target: string;
    }) => (await api.post<DnsRecord>("/api/dns/instance-record", input)).data,
    onSuccess: () => toast.success("Instance record created"),
    onError: (error) => toast.error(errorMessage(error)),
  });

export const getDnsTunnelsKey = "dnsTunnels";

export const useFetchTunnels = (enabled = true) =>
  useQuery({
    queryKey: [getDnsTunnelsKey],
    queryFn: async () => (await api.get<Tunnel[]>("/api/dns/tunnels")).data,
    enabled,
  });

const getDnsTunnelStatusKey = "dnsTunnelStatus";

// Polls Cloudflare's connector state every few seconds until cloudflared reports in, then stops.
export const useFetchTunnelStatus = (tunnelId: string | undefined) =>
  useQuery({
    queryKey: [getDnsTunnelStatusKey, tunnelId],
    queryFn: async () => (await api.get<Tunnel>(`/api/dns/tunnels/${tunnelId}/status`)).data,
    enabled: !!tunnelId,
    refetchInterval: (query) => (query.state.data?.status === "healthy" ? false : 3000),
  });

export const useCreateTunnel = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async (input: { name: string; account_id: string }) =>
      (await api.post<Tunnel>("/api/dns/tunnels", input)).data,
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: [getDnsTunnelsKey] });
      toast.success("Tunnel created");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

export const useRouteTunnelHostname = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async (input: {
      tunnel_id: string;
      hostname: string;
      zone_id: string;
      zone: string;
      service: string;
    }) => (await api.post<Tunnel>(`/api/dns/tunnels/${encodeURIComponent(input.tunnel_id)}/route`, input)).data,
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: [getDnsTunnelsKey] });
      toast.success("Hostname routed to tunnel");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

export const useProvisionTunnelAgent = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async (input: { tunnel_id: string; target: string; docker_network: string }) =>
      (
        await api.post<{ service_id: string }>(
          `/api/dns/tunnels/${encodeURIComponent(input.tunnel_id)}/agent`,
          input,
        )
      ).data,
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: [getDnsTunnelsKey] });
      toast.success("cloudflared deployed");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

export const useProvisionReverseProxy = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async (input: {
      target: string;
      docker_network: string;
      image?: string;
    }) => (await api.post<{ service_id: string }>("/api/dns/reverse-proxy", input)).data,
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: [getServicesKey] });
      toast.success("Reverse proxy deployed");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

export const getDnsGatewaysKey = "dnsGateways";

export const useFetchGateways = (enabled = true) =>
  useQuery({
    queryKey: [getDnsGatewaysKey],
    queryFn: async () => (await api.get<Gateway[]>("/api/dns/gateways")).data,
    enabled,
  });

export const useCreateGateway = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async (input: CreateGatewayFormData) => (await api.post<Gateway>("/api/dns/gateways", input)).data,
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: [getDnsGatewaysKey] });
      await client.invalidateQueries({ queryKey: [getServicesKey] });
      toast.success("Gateway created");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

export const useDeleteGateway = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async (gatewayId: string) =>
      (await api.delete(`/api/dns/gateways/${encodeURIComponent(gatewayId)}`)).data,
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: [getDnsGatewaysKey] });
      toast.success("Gateway deleted");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

export const getDnsExposuresKey = "dnsExposures";

export const useFetchExposures = (enabled = true) =>
  useQuery({
    queryKey: [getDnsExposuresKey],
    queryFn: async () => (await api.get<Exposure[]>("/api/dns/exposures")).data,
    enabled,
  });

export const useCreateExposure = () => {
  const client = useQueryClient();
  return useMutation({
    // gateway_id is optional: the backend reuses or provisions one for the target's machine when omitted (spec §7).
    mutationFn: async (input: ExposeServiceFormData & { gateway_id?: string }) =>
      (await api.post<Exposure>("/api/dns/exposures", input)).data,
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: [getDnsExposuresKey] });
      toast.success("Service exposed");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

// The wizard's reach step targets a container directly (service_id) and lets the gateway be chosen for it
// (spec §7: reuse a gateway on the target's machine, or provision one) — unlike useCreateExposure above, which
// still carries the legacy gateway_id/service pair the settings-page dialog fills in explicitly.
export const useCreateServiceExposure = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async (input: {
      hostname: string;
      service_id: string;
      port: number;
      zone_id: string;
      zone: string;
    }) => (await api.post<Exposure>("/api/dns/exposures", input)).data,
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: [getDnsExposuresKey] });
      toast.success("Service exposed");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

export const useDeleteExposure = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async (exposureId: string) =>
      (await api.delete(`/api/dns/exposures/${encodeURIComponent(exposureId)}`)).data,
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: [getDnsExposuresKey] });
      toast.success("Service unexposed");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

// One tunnel ticker check (ingress, record or reachable); resolves with what the server found.
export const verifyTunnelCheck = async (tunnelId: string, check: string) =>
  (
    await api.post<{ detail: string }>(`/api/dns/tunnels/${encodeURIComponent(tunnelId)}/verify`, null, {
      params: { check },
    })
  ).data.detail;

const getDnsResolveKey = "dnsResolve";

// Polls until every answer for host is this server, so the proxy's certificate request can succeed.
export const useResolveHost = (host: string, address: PublicAddress) =>
  useQuery({
    queryKey: [getDnsResolveKey, host],
    queryFn: async () => (await api.get<{ addresses: string[] }>("/api/dns/resolve", { params: { host } })).data,
    enabled: !!host,
    refetchInterval: (query) => (resolvesHere(query.state.data?.addresses, address) ? false : 5000),
  });

// Retry-safe on the server: a second call for the same domain redeploys the same gateway. Errors render inline.
export const useDeployInstanceProxy = () =>
  useMutation({
    mutationFn: async ({ domain, email }: { domain: string; email: string }) =>
      (await api.post<{ service_id: string }>("/api/dns/instance-proxy", email ? { domain, email } : { domain })).data,
  });

// Each DNS list is one instance-wide read; a change refetches the list of its kind.
const refetchList = (key: string) => (_payload: unknown, { client }: { client: QueryClient }) => client.invalidateQueries({ queryKey: [key] });

export const dnsFollower: LiveFollower = {
  "dns.record_changed": refetchList(getDnsZonesKey),
  "dns.tunnel_changed": refetchList(getDnsTunnelsKey),
  "dns.gateway_changed": refetchList(getDnsGatewaysKey),
  "dns.exposure_changed": refetchList(getDnsExposuresKey),
};
