import { useQuery } from "@tanstack/react-query";

import { api } from "@/api/client";
import { defineQuery } from "@/lib/liveQuery";
import type { Runner } from "@/models/Runner";

export const getRunnersKey = "getRunners";

const runnersQuery = defineQuery({
  key: getRunnersKey,
  fetch: () => api.get<Runner[]>("/api/runners"),
  refreshes: { "runner.connected": "all", "runner.disconnected": "all" },
});

export const useFetchRunners = () => useQuery(runnersQuery.options());
