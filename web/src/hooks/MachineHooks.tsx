import { useMutation, useQuery } from "@tanstack/react-query";
import { toast } from "sonner";

import { api, errorMessage } from "@/api/client";
import type { GroupedDiscovery, ImportRequest, ImportResult, Machine } from "@/models/Machine";

const getMachinesKey = "machines";

export const useFetchMachines = () =>
  useQuery({
    queryKey: [getMachinesKey],
    queryFn: async () => (await api.get<Machine[]>("/api/machines")).data,
  });

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
    onSuccess: () => toast.success("Imported"),
    onError: (error) => toast.error(errorMessage(error)),
  });
