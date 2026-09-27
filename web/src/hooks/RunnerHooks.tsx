import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { api, errorMessage } from "@/api/client";
import type { Machine, QueuedJob, Runner, RunnerEnrollment, RunnerEnrollmentFormData } from "@/models/Runner";

export const getRunnersKey = "runners";
export const getRunnerQueueKey = "runnerQueue";
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

// The git token never leaves the browser: it only goes into the rendered command.
export const useCreateRunnerEnrollment = () =>
  useMutation({
    mutationFn: async ({ name, machine }: RunnerEnrollmentFormData) =>
      (await api.post<RunnerEnrollment>("/api/runners/enrollments", { name, machine: machine || undefined })).data,
    onError: (error) => toast.error(errorMessage(error)),
  });

export const useRemoveRunner = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async (id: string) => api.delete(`/api/runners/${id}`),
    onSuccess: async () => {
      await Promise.all([
        client.invalidateQueries({ queryKey: [getRunnersKey] }),
        client.invalidateQueries({ queryKey: [getRunnerQueueKey] }),
      ]);
      toast.success("Runner removed");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

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
