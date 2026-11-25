import { useQuery } from "@tanstack/react-query";

import { api } from "@/api/client";
import type { SharedRun } from "@/models/SharedRun";

export const getSharedRunsKey = "getSharedRuns";

// Read when the details open; a run starting or ending on the computer refreshes the computers it counts against.
export const useFetchSharedRuns = (computerId: string) =>
  useQuery({
    queryKey: [getSharedRunsKey, computerId],
    queryFn: async () => (await api.get<{ runs: SharedRun[] }>(`/api/pairing/computers/${computerId}/shared-runs`)).data.runs,
    enabled: !!computerId,
  });
