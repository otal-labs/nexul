import { useQuery } from "@tanstack/react-query";

import { hasPermission } from "@nexul/client-core/permissions";

import { api } from "@/api/client";
import { useFetchMe } from "@/hooks/AuthHooks";
import { defineQuery } from "@/lib/liveQuery";
import type { Runner } from "@/models/Runner";

export const getRunnersKey = "getRunners";

const runnersQuery = defineQuery({
  key: getRunnersKey,
  fetch: () => api.get<Runner[]>("/api/runners"),
  refreshes: { "runner.connected": "all", "runner.disconnected": "all", "machine.deleted": "all" },
});

export const useFetchRunners = () => useQuery(runnersQuery.options());

// The Runners row is for whoever sees a runner or can add a machine; the route itself still opens for every member.
export const useShowRunnersRow = (): boolean => {
  const { data: runners } = useFetchRunners();
  const { data: me } = useFetchMe(true);
  return (runners?.length ?? 0) > 0 || me?.instance_owner === true || hasPermission(me?.instance_permissions, "machines:create");
};
