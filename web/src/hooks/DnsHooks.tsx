import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
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

const getDnsZonesKey = "dnsZones";

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

const getDnsTunnelsKey = "dnsTunnels";

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
    mutationFn: async (input: { name: string }) => (await api.post<Tunnel>("/api/dns/tunnels", input)).data,
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
    mutationFn: async (input: { tunnel_id: string; project_id: string; target: string; docker_network: string }) =>
      (
        await api.post<{ service_id: string }>(
          `/api/dns/tunnels/${encodeURIComponent(input.tunnel_id)}/agent`,
          input,
        )
      ).data,
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: [getDnsTunnelsKey] });
      toast.success("cloudflared provisioned");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

export const useProvisionReverseProxy = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async (input: {
      project_id: string;
      target: string;
      docker_network: string;
      image?: string;
    }) => (await api.post<{ service_id: string }>("/api/dns/reverse-proxy", input)).data,
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: [getServicesKey] });
      toast.success("Reverse proxy provisioned");
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

export const useFetchExposures = () =>
  useQuery({
    queryKey: [getDnsExposuresKey],
    queryFn: async () => (await api.get<Exposure[]>("/api/dns/exposures")).data,
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
