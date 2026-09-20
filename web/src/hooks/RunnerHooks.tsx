import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { api, errorMessage } from "@/api/client";
import type { Machine, QueuedJob, Runner, RunnerInstall } from "@/models/Runner";

export const getRunnersKey = "runners";
export const getRunnerQueueKey = "runnerQueue";
export const getRunnerInstallKey = "runnerInstall";
export const getMachinesKey = "machines";

export const useRunners = () =>
  useQuery({
    queryKey: [getRunnersKey],
    queryFn: async () => (await api.get<Runner[]>("/api/runners")).data,
  });

export const useRunnerQueue = () =>
  useQuery({
    queryKey: [getRunnerQueueKey],
    queryFn: async () => (await api.get<QueuedJob[]>("/api/runners/queue")).data,
  });

// Fetched only once the dialog is open, so the install secret isn't requested on every page view.
export const useRunnerInstall = (enabled: boolean) =>
  useQuery({
    queryKey: [getRunnerInstallKey],
    queryFn: async () => (await api.get<RunnerInstall>("/api/runners/install")).data,
    enabled,
  });

export const useFetchMachines = () =>
  useQuery({
    queryKey: [getMachinesKey],
    queryFn: async () => (await api.get<Machine[]>("/api/machines")).data,
  });

export const useUpdateMachine = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async ({ id, stackRoot }: { id: string; stackRoot: string }) =>
      (await api.patch<Machine>(`/api/machines/${id}`, { stack_root: stackRoot })).data,
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: [getMachinesKey] });
      toast.success("Stack root updated");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};
