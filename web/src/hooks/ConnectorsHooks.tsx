import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { api, errorMessage } from "@/api/client";
import type { AppConfigStatus, ConnectorStatus, CredentialStatus } from "@/models/Connectors";

const getConnectorsKey = "connectors";

export const useFetchConnectors = () =>
  useQuery({
    queryKey: [getConnectorsKey],
    queryFn: async () => (await api.get<ConnectorStatus[]>("/api/connectors")).data,
  });

// Selects one connector's status from the shared list query, so callers never own credential state themselves.
export const useFetchConnectorStatus = (id: string) => {
  const query = useFetchConnectors();
  return { ...query, data: query.data?.find((entry) => entry.connector.id === id) };
};

export const useStartConnectorOAuth = () =>
  useMutation({
    mutationFn: async (id: string) =>
      (await api.get<{ url: string }>(`/api/connectors/${id}/oauth/start`)).data,
    onError: (error) => toast.error(errorMessage(error)),
  });

// Read live by the connector's OAuthClient next use; the list refetch only flips app_configured so Connect appears.
export const useSetConnectorAppConfig = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (payload: {
      id: string;
      client_id: string;
      client_secret: string;
      base_url: string;
      app_slug: string;
    }) =>
      (
        await api.put<AppConfigStatus>(`/api/connectors/${payload.id}/app-config`, {
          client_id: payload.client_id,
          client_secret: payload.client_secret,
          base_url: payload.base_url,
          app_slug: payload.app_slug,
        })
      ).data,
    onSuccess: () => {
      toast.success("Connector app config saved");
      void queryClient.invalidateQueries({ queryKey: [getConnectorsKey] });
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

// The verify error comes back as the request rejection, not a toast, so ConnectorCard renders it inline.
// Live provider check only; the dialog turns the result into a green light, so no toast here.
export const useVerifyManualCredentials = () =>
  useMutation({
    mutationFn: async (payload: { id: string; fields: Record<string, string> }) => {
      await api.post(`/api/connectors/${payload.id}/manual/verify`, payload.fields);
    },
  });

export const useSaveManualCredentials = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async (payload: { id: string; fields: Record<string, string> }) =>
      (await api.post<CredentialStatus>(`/api/connectors/${payload.id}/manual`, payload.fields)).data,
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: [getConnectorsKey] });
      toast.success("Connector connected");
    },
  });
};

export const useDisconnectConnector = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async (id: string) => (await api.post(`/api/connectors/${id}/disconnect`)).data,
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: [getConnectorsKey] });
      toast.success("Connector disconnected");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};
