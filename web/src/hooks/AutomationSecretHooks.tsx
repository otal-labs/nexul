import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { api, errorMessage } from "@/api/client";
import type { AutomationSecretMeta } from "@/models/AutomationSecret";

export const getAutomationSecretsKey = "getAutomationSecrets";

export const useFetchAutomationSecrets = () =>
  useQuery({
    queryKey: [getAutomationSecretsKey],
    queryFn: async () => (await api.get<AutomationSecretMeta[]>("/api/automation-secrets")).data,
  });

export const useSetAutomationSecret = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async ({ name, value }: { name: string; value: string }) =>
      (await api.put(`/api/automation-secrets/${name}`, { value })).data,
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: [getAutomationSecretsKey] });
      toast.success("Secret saved");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

export const useDeleteAutomationSecret = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async (name: string) => (await api.delete(`/api/automation-secrets/${name}`)).data,
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: [getAutomationSecretsKey] });
      toast.success("Secret deleted");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};
