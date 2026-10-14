import { useMutation, useQuery, useQueryClient, type QueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { machineAnswer, type EntityAnswer } from "@nexul/client-core/actionState";
import { requires } from "@nexul/client-core/requires";

import { api, errorMessage } from "@/api/client";
import type { Machine, OpenEnrollment, QueuedJob, Runner, RunnerEnrollment, RunnerEnrollmentInput } from "@/models/Runner";
import { followEach, type LiveFollower } from "@/lib/live";

export const getRunnersKey = "runners";
export const getRunnerQueueKey = "runnerQueue";
export const getMachinesKey = "machines";
export const getRunnerEnrollmentsKey = "runnerEnrollments";

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

// The unused codes the viewer may revoke, runner and computer codes alike; a caller shows the kind it wants.
export const useOpenEnrollments = (computers: boolean) =>
  useQuery({
    queryKey: [getRunnerEnrollmentsKey],
    queryFn: async () => (await api.get<OpenEnrollment[]>("/api/runners/enrollments")).data,
    select: (codes) => codes.filter((c) => !!c.computer_id === computers),
  });

export const useCreateRunnerEnrollment = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: requires("machines:create", async ({ name, machine }: RunnerEnrollmentInput) =>
      (await api.post<RunnerEnrollment>("/api/runners/enrollments", { name, machine })).data),
    onSuccess: () => client.invalidateQueries({ queryKey: [getRunnerEnrollmentsKey] }),
    onError: (error) => toast.error(errorMessage(error)),
  });
};

export const useRevokeEnrollment = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: requires("machines:write", async (id: string) => api.delete(`/api/runners/enrollments/${id}`)),
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: [getRunnerEnrollmentsKey] });
      toast.success("Code revoked");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

export const useRemoveRunner = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: requires("machines:write", async (id: string) => api.delete(`/api/runners/${id}`)),
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

// A machine missing from the list is one the viewer can't see, so it reads as refused; undefined until the list loads.
export const useDeployToAnswer = (name: string | undefined): EntityAnswer | undefined => {
  const { data: machines } = useFetchMachines();
  if (!machines || !name) return undefined;
  const machine = machines.find((m) => m.name === name);
  return { allowed: !!machine && machineAnswer(machine, "machines:deploy").allowed, source: { sentence: `No Deploy to on machine ${name}` } };
};

export const useUpdateMachine = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: requires("machines:write", async ({ id, stackRoot }: { id: string; stackRoot: string }) =>
      (await api.patch<Machine>(`/api/machines/${id}`, { stack_root: stackRoot })).data),
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: [getMachinesKey] });
      toast.success("Stack root updated");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

const refetch = (client: QueryClient, key: string) => client.invalidateQueries({ queryKey: [key], exact: true });

// Who sees a machine moves its runners and the queue with it.
export const refetchRunnersAndQueue = (client: QueryClient) =>
  Promise.all([refetch(client, getRunnersKey), refetch(client, getRunnerQueueKey)]);

// A job frame names its deploy: the runners list refetches until a runner shows it, and the queue while it holds it.
const followJob = ({ id }: { id: string }, { client }: { client: QueryClient }) =>
  Promise.all([
    !client.getQueryData<Runner[]>([getRunnersKey])?.some((r) => r.running_job?.id === id) && refetch(client, getRunnersKey),
    !!client.getQueryData<QueuedJob[]>([getRunnerQueueKey])?.some((j) => j.id === id) && refetch(client, getRunnerQueueKey),
  ]);

export const runnerFollower: LiveFollower = {
  // A runner on a machine the viewer just got to see brings that machine with it.
  ...followEach(["runner.connected", "runner.disconnected"], (_payload: unknown, { client }) =>
    Promise.all([refetch(client, getRunnersKey), refetch(client, getMachinesKey), refetch(client, getRunnerEnrollmentsKey)]),
  ),
  // A personal runner enrolling spends its computer's code.
  "runner.personal_changed": (_payload: unknown, { client }) => refetch(client, getRunnerEnrollmentsKey),
  ...followEach(["deploy.build_started", "deploy.build_progress", "deploy.build_completed", "deploy.deploy_progress"], followJob),
  // A finished job frees its runner, which takes the next queued one.
  "deploy.status_changed": (_payload: unknown, { client }) => Promise.all([refetch(client, getRunnersKey), refetch(client, getRunnerQueueKey)]),
};
