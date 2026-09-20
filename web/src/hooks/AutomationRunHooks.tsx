import { useQuery } from "@tanstack/react-query";

import { api } from "@/api/client";
import type { AutomationRun } from "@/models/AutomationRun";

export const getAutomationRunsKey = "getAutomationRuns";

export const useFetchAutomationRuns = (automationId: string | undefined) =>
  useQuery({
    queryKey: [getAutomationRunsKey, automationId],
    queryFn: async () => (await api.get<AutomationRun[]>(`/api/automations/${automationId}/runs`)).data,
    enabled: !!automationId,
  });

export const useFetchAutomationRun = (automationId: string | undefined, runId: string | undefined) =>
  useQuery({
    queryKey: ["getAutomationRun", automationId, runId],
    queryFn: async () =>
      (await api.get<AutomationRun>(`/api/automations/${automationId}/runs/${runId}`)).data,
    enabled: !!automationId && !!runId,
  });
