import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { api, errorMessage } from "@/api/client";
import type { Automation, AutomationTokenMint } from "@/models/Automation";
import type { PermissionInfo } from "@/models/Permission";
import { useWorkspaceStore } from "@/stores/workspaceStore";

export const getAutomationsKey = "getAutomations";

// Automations belong to a workspace: the page lists the selected one's.
export const useFetchAutomations = () => {
  const workspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  return useQuery({
    queryKey: [getAutomationsKey, workspaceId],
    queryFn: async () =>
      (await api.get<Automation[]>("/api/automations", { params: { workspace_id: workspaceId } })).data,
    enabled: workspaceId !== "",
  });
};

export const useFetchAutomation = (id: string | undefined) =>
  useQuery({
    queryKey: ["getAutomation", id],
    queryFn: async () => (await api.get<Automation>(`/api/automations/${id}`)).data,
    enabled: !!id,
  });

export const useFetchScopeCatalog = () =>
  useQuery({
    queryKey: ["getScopeCatalog"],
    queryFn: async () => (await api.get<{ scopes: PermissionInfo[] }>("/api/integrations/scopes")).data.scopes,
    staleTime: Infinity,
  });

export const useCreateAutomation = () => {
  const client = useQueryClient();
  const workspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  return useMutation({
    mutationFn: async (input: { name: string; scopes: string[] }) =>
      (await api.post<AutomationTokenMint>("/api/automations", { ...input, workspace_id: workspaceId })).data,
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: [getAutomationsKey] });
      toast.success("Automation created");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

export const useUpdateAutomationConfig = (id: string) => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async (configValues: Record<string, string>) =>
      (await api.patch<Automation>(`/api/automations/${id}/config`, { config_values: configValues })).data,
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: ["getAutomation", id] });
      toast.success("Configuration saved");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

export const useSetAutomationEnabled = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async ({ id, enabled }: { id: string; enabled: boolean }) =>
      (await api.patch<Automation>(`/api/automations/${id}/enabled`, { enabled })).data,
    onSuccess: async (automation) => {
      await client.invalidateQueries({ queryKey: [getAutomationsKey] });
      await client.invalidateQueries({ queryKey: ["getAutomation", automation.id] });
      toast.success(automation.enabled ? "Automation enabled" : "Automation disabled");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

// hostId null places the automation back on the bundled instance host.
export const useSetAutomationHost = (id: string) => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async (hostId: string | null) =>
      (await api.patch<Automation>(`/api/automations/${id}/host`, { host_id: hostId })).data,
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: ["getAutomation", id] });
      toast.success("Automation moved");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

export const useDeleteAutomation = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async (id: string) => (await api.delete(`/api/automations/${id}`)).data,
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: [getAutomationsKey] });
      toast.success("Automation deleted");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

export const useMintAutomationToken = (id: string) =>
  useMutation({
    mutationFn: async () => (await api.post<AutomationTokenMint>(`/api/automations/${id}/token`)).data,
    onError: (error) => toast.error(errorMessage(error)),
  });

export const useRevokeAutomationToken = (id: string) => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async () => (await api.delete<Automation>(`/api/automations/${id}/token`)).data,
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: ["getAutomation", id] });
      toast.success("Token revoked");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};
