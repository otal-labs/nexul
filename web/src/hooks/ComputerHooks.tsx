import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { api, errorMessage } from "@/api/client";
import { computersQuery, getComputersKey } from "@/hooks/PairingHooks";
import { getRunnerEnrollmentsKey } from "@/hooks/RunnerHooks";
import type { Computer, ComputerEnrollment, RenameComputerFormData } from "@/models/Pairing";
import { sharedComputers } from "@/models/SharedComputer";

// The computers other people share with the viewer by name, from the same request as their own.
export const useListSharedComputers = () => useQuery({ ...computersQuery, select: sharedComputers });

// Adds a computer waiting for its runner, or with an id mints a fresh command for one that has none; the dialog shows failures.
export const useEnrollComputer = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async (computerId: string | undefined) =>
      (await api.post<ComputerEnrollment>("/api/pairing/computers/enrollments", computerId ? { id: computerId } : {})).data,
    onSuccess: async () => {
      await Promise.all([
        client.invalidateQueries({ queryKey: [getComputersKey] }),
        client.invalidateQueries({ queryKey: [getRunnerEnrollmentsKey] }),
      ]);
    },
  });
};

// Pairs a computer again through its runner, now, instead of waiting for the next report.
export const usePairNow = (computer: Computer) => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async () => (await api.post<Computer>(`/api/pairing/computers/${computer.id}/pair`)).data,
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: [getComputersKey] });
      toast.success(`${computer.name} re-paired`);
    },
    onError: async (error) => {
      await client.invalidateQueries({ queryKey: [getComputersKey] });
      toast.error(errorMessage(error));
    },
  });
};

// No toast: the rename dialog shows a refused name on its field and closes on success.
export const useRenameComputer = (computerId: string) => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async (input: RenameComputerFormData) => (await api.patch<Computer>(`/api/pairing/computers/${computerId}`, input)).data,
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: [getComputersKey] });
    },
  });
};
