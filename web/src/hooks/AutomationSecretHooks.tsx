import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { api, errorMessage } from "@/api/client";
import type { AutomationSecretMeta } from "@/models/AutomationSecret";
import { useWorkspaceStore } from "@/stores/workspaceStore";

export const getAutomationSecretsKey = "getAutomationSecrets";

// Secrets are a pool per workspace; every call names the selected one.
const useSecretsParams = () => ({ workspace_id: useWorkspaceStore((s) => s.selectedWorkspaceId) });

export const useFetchAutomationSecrets = () => {
  const params = useSecretsParams();
  return useQuery({
    queryKey: [getAutomationSecretsKey, params],
    queryFn: async () => (await api.get<AutomationSecretMeta[]>("/api/automation-secrets", { params })).data,
    enabled: params.workspace_id !== "",
  });
};

export const useSetAutomationSecret = () => {
  const client = useQueryClient();
  const params = useSecretsParams();
  return useMutation({
    mutationFn: async ({ name, value }: { name: string; value: string }) =>
      (await api.put(`/api/automation-secrets/${name}`, { value }, { params })).data,
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: [getAutomationSecretsKey] });
      toast.success("Secret saved");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

export const useDeleteAutomationSecret = () => {
  const client = useQueryClient();
  const params = useSecretsParams();
  return useMutation({
    mutationFn: async (name: string) => (await api.delete(`/api/automation-secrets/${name}`, { params })).data,
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: [getAutomationSecretsKey] });
      toast.success("Secret deleted");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};
