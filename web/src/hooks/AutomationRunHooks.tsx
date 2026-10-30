import { useQuery } from "@tanstack/react-query";

import { api } from "@/api/client";
import type { AutomationRun } from "@/models/AutomationRun";

export const getAutomationRunsKey = "getAutomationRuns";

// The newest 50 runs, or with failedOnly the newest 50 failures and crashes.
export const useFetchAutomationRuns = (automationId: string | undefined, failedOnly = false) =>
  useQuery({
    queryKey: [getAutomationRunsKey, automationId, failedOnly],
    queryFn: async () =>
      (await api.get<AutomationRun[]>(`/api/automations/${automationId}/runs`, { params: failedOnly ? { outcome: "failed" } : {} })).data,
    enabled: !!automationId,
  });

export const useFetchAutomationRun = (automationId: string | undefined, runId: string | undefined) =>
  useQuery({
    queryKey: ["getAutomationRun", automationId, runId],
    queryFn: async () =>
      (await api.get<AutomationRun>(`/api/automations/${automationId}/runs/${runId}`)).data,
    enabled: !!automationId && !!runId,
  });
