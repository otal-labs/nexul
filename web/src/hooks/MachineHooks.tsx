import { useMutation, useQuery, useQueryClient, type QueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { requires } from "@nexul/client-core/requires";

import { api, errorMessage } from "@/api/client";
import { getRunnerEnrollmentsKey, refetchRunnersAndQueue } from "@/hooks/RunnerHooks";
import { dropRow, type LiveFollower } from "@/lib/live";
import type { DeployTarget, GroupedDiscovery, ImportRequest, ImportResult, Machine } from "@/models/Machine";

const getMachinesKey = "machines";

export const useFetchMachines = () =>
  useQuery({
    queryKey: [getMachinesKey],
    queryFn: async () => (await api.get<Machine[]>("/api/machines")).data,
  });

// The machines a stack can be aimed at: the ones the viewer holds Deploy to on, not every machine they see. Keyed under
// the machines list so anything refreshing it refreshes the picker.
export const useFetchDeployTargets = () =>
  useQuery({
    queryKey: [getMachinesKey, "deployable"],
    queryFn: async () => (await api.get<DeployTarget[]>("/api/machines/deployable")).data,
  });

// A deleted machine takes its offline runners, its waiting codes and its place in the deploy picker with it.
const machineGone = (client: QueryClient, machineId: string) => {
  dropRow(client, [getMachinesKey], machineId);
  return Promise.all([
    client.invalidateQueries({ queryKey: [getMachinesKey, "deployable"] }),
    client.invalidateQueries({ queryKey: [getRunnerEnrollmentsKey], exact: true }),
    refetchRunnersAndQueue(client),
  ]);
};

// A rule on a machine can add or take away where the viewer may deploy, and which machines, runners and jobs they see.
export const machineFollower: LiveFollower = {
  "machine.deleted": ({ machine_id }: { machine_id: string }, { client }) => machineGone(client, machine_id),
  "access.grant.changed": ({ resource_type }: { resource_type: string }, { client }) =>
    resource_type === "machine" &&
    Promise.all([client.invalidateQueries({ queryKey: [getMachinesKey] }), refetchRunnersAndQueue(client)]),
};

// A refusal names each stack, gateway or connected runner still on the machine, and the toast shows it.
export const useDeleteMachine = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: requires("machines:delete", async (machineId: string) => (await api.delete(`/api/machines/${machineId}`)).data),
    onSuccess: async (_data, machineId) => {
      await machineGone(client, machineId);
      toast.success("Machine deleted");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

export const useDiscoverMachine = () =>
  useMutation({
    mutationFn: requires("machines:write", async (machineId: string) =>
      (await api.post<GroupedDiscovery>(`/api/machines/${machineId}/discover`)).data),
    onError: (error) => toast.error(errorMessage(error)),
  });

export const useImportMachine = () =>
  useMutation({
    mutationFn: requires("stacks:write", async ({ machineId, input }: { machineId: string; input: ImportRequest }) =>
      (await api.post<ImportResult>(`/api/machines/${machineId}/import`, input)).data),
    onSuccess: () => toast.success("Stacks imported"),
    onError: (error) => toast.error(errorMessage(error)),
  });
