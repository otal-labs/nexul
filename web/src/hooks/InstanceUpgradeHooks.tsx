import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import type { AxiosError } from "axios";
import { toast } from "sonner";

import { api, errorMessage } from "@/api/client";
import { getServerVersionKey } from "@/hooks/VersionHooks";
import { isUpgradeInProgress, type InstanceUpgrade, type InstanceUpgradeRecord } from "@/models/InstanceUpgrade";

export const getInstanceUpgradeKey = "instanceUpgrade";

export const useInstanceUpgrade = () =>
  useQuery({
    queryKey: [getInstanceUpgradeKey],
    queryFn: async () => (await api.get<InstanceUpgrade>("/api/instance/upgrade")).data,
    refetchInterval: (query) => (isUpgradeInProgress(query.state.data?.upgrade ?? null) ? 5000 : false),
    staleTime: 5 * 60 * 1000,
  });

export const useRequestInstanceUpgrade = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async () => {
      try {
        return (await api.post<InstanceUpgradeRecord>("/api/instance/upgrade")).data;
      } catch (err) {
        const reason = (err as AxiosError<{ reason: string }>).response?.data?.reason;
        throw new Error(reason || (err as Error).message);
      }
    },
    onSuccess: () => toast.success("Upgrade started"),
    onError: (error) => toast.error(errorMessage(error)),
    onSettled: async () => {
      await client.invalidateQueries({ queryKey: [getInstanceUpgradeKey] });
      await client.invalidateQueries({ queryKey: [getServerVersionKey] });
    },
  });
};
