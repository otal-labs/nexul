import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { requires } from "@nexul/client-core/requires";

import { api, errorMessage } from "@/api/client";
import { getAutomationsKey } from "@/hooks/AutomationHooks";
import { getHistoryKey } from "@/hooks/HistoryHooks";
import type { AutomationVersion, AutomationVersionDiff } from "@/models/AutomationVersion";

export const getAutomationVersionDiffKey = "getAutomationVersionDiff";

export const automationHistoryUrl = (automationId: string) => `/api/automations/${automationId}/history`;
export const automationVersionUrl = (automationId: string, sequence: number) => `/api/automations/${automationId}/versions/${sequence}`;

export const useFetchAutomationVersionDiff = (automationId: string | undefined) =>
  useQuery({
    queryKey: [getAutomationVersionDiffKey, automationId],
    queryFn: async () =>
      (await api.get<AutomationVersionDiff>(`/api/automations/${automationId}/versions/diff`)).data,
    enabled: !!automationId,
  });

const invalidateVersions = async (client: ReturnType<typeof useQueryClient>, automationId: string) => {
  await client.invalidateQueries({ queryKey: [getAutomationVersionDiffKey, automationId] });
  await client.invalidateQueries({ queryKey: [getHistoryKey, automationHistoryUrl(automationId)] });
  await client.invalidateQueries({ queryKey: ["getAutomation", automationId] });
  await client.invalidateQueries({ queryKey: [getAutomationsKey] });
};

export const useMergeAutomationVersion = (automationId: string) => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: requires("automations:write", async (versionId: string) =>
      (await api.post<AutomationVersion>(`/api/automations/${automationId}/versions/${versionId}/merge`)).data),
    onSuccess: async () => {
      await invalidateVersions(client, automationId);
      toast.success("Version merged");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

export const useRollbackAutomationVersion = (automationId: string) => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: requires("automations:write", async (versionId: string) =>
      (await api.post<AutomationVersion>(`/api/automations/${automationId}/versions/${versionId}/rollback`)).data),
    onSuccess: async () => {
      await invalidateVersions(client, automationId);
      toast.success("Rolled back");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};
