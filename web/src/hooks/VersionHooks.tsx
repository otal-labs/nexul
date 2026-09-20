import { useQuery, type QueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { api } from "@/api/client";
import type { Version } from "@/models/Version";

export const getServerVersionKey = "serverVersion";

// The version this tab first loaded; compared against later reconnect fetches to catch an upgrade
// that happened underneath the open tab.
let firstVersion: string | null = null;
let reloadPromptShown = false;

export const useServerVersion = (enabled = true) =>
  useQuery({
    queryKey: [getServerVersionKey],
    queryFn: async () => {
      const data = (await api.get<Version>("/api/version")).data;
      if (firstVersion === null) firstVersion = data.version;
      return data;
    },
    staleTime: 5 * 60 * 1000,
    retry: false,
    enabled,
  });

export const shouldPromptReload = (initial: string | null, current: string): boolean =>
  initial !== null && initial !== current;

// Called on every live-events reconnect: refetches /api/version and, the first time it differs from
// the version this tab started with, prompts a reload instead of leaving stale code running against
// an upgraded server.
export const notifyIfServerUpdated = async (queryClient: QueryClient) => {
  const data = await queryClient.fetchQuery({
    queryKey: [getServerVersionKey],
    queryFn: async () => (await api.get<Version>("/api/version")).data,
  });
  if (firstVersion === null) firstVersion = data.version;
  if (reloadPromptShown || !shouldPromptReload(firstVersion, data.version)) return;
  reloadPromptShown = true;
  toast("Nexul was updated", {
    action: { label: "Reload", onClick: () => window.location.reload() },
    duration: Infinity,
  });
};
