import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { api, errorMessage } from "@/api/client";
import type { Deploy, DeployLogLine } from "@/models/Stack";

export const getDeployKey = "getDeploy";
export const getDeployLogKey = "getDeployLog";

export const useFetchDeploy = (id: string | undefined) =>
  useQuery({
    queryKey: [getDeployKey, id],
    queryFn: async () => (await api.get<Deploy>(`/api/deploys/${id}`)).data,
    enabled: !!id,
  });

export const useFetchDeployLog = (id: string | undefined) =>
  useQuery({
    queryKey: [getDeployLogKey, id],
    queryFn: async () => (await api.get<DeployLogLine[]>(`/api/deploys/${id}/log`)).data,
    enabled: !!id,
  });

export const useCancelDeploy = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async (id: string) => api.post(`/api/deploys/${id}/cancel`),
    onSuccess: async (_data, id) => {
      await client.invalidateQueries({ queryKey: [getDeployKey, id] });
      toast.success("Cancel requested");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};
