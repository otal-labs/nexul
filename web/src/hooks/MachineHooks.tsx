import { useMutation, useQuery } from "@tanstack/react-query";
import { toast } from "sonner";

import { api, errorMessage } from "@/api/client";
import { refetchRunnersAndQueue } from "@/hooks/RunnerHooks";
import type { LiveFollower } from "@/lib/live";
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

// A rule on a machine can add or take away where the viewer may deploy, and which machines, runners and jobs they see.
export const machineFollower: LiveFollower = {
  "access.grant.changed": ({ resource_type }: { resource_type: string }, { client }) =>
    resource_type === "machine" &&
    Promise.all([client.invalidateQueries({ queryKey: [getMachinesKey] }), refetchRunnersAndQueue(client)]),
};

export const useDiscoverMachine = () =>
  useMutation({
    mutationFn: async (machineId: string) =>
      (await api.post<GroupedDiscovery>(`/api/machines/${machineId}/discover`)).data,
    onError: (error) => toast.error(errorMessage(error)),
  });

export const useImportMachine = () =>
  useMutation({
    mutationFn: async ({ machineId, input }: { machineId: string; input: ImportRequest }) =>
      (await api.post<ImportResult>(`/api/machines/${machineId}/import`, input)).data,
    onSuccess: () => toast.success("Stacks imported"),
    onError: (error) => toast.error(errorMessage(error)),
  });
