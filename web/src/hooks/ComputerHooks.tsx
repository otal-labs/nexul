import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { requires } from "@nexul/client-core/requires";

import { api, errorMessage } from "@/api/client";
import { computersQuery, getComputersKey, getHarnessProjectsKey } from "@/hooks/PairingHooks";
import { getRunnerEnrollmentsKey } from "@/hooks/RunnerHooks";
import { useConfirmationDialog } from "@/hooks/useConfirmationDialog";
import { notViewing } from "@/stores/viewAsStore";
import type { Computer, ComputerEnrollment, RenameComputerFormData } from "@/models/Pairing";
import { sharedComputerName, sharedComputers } from "@/models/SharedComputer";

// The computers other people share with the viewer by name, from the same request as their own.
export const useListSharedComputers = () => useQuery({ ...computersQuery, select: sharedComputers, enabled: notViewing });

// The names of these computers as the viewer reads them, in order; "" for one nobody shared with them, and none while unknown.
export const useComputerNames = (computerIds: string[]) =>
  useQuery({
    ...computersQuery,
    select: (computers) => computerIds.map((id) => {
      const computer = computers.find((c) => c.id === id);
      return computer ? sharedComputerName(computer) : "";
    }),
    enabled: computerIds.length > 0 && notViewing(),
  }).data ?? [];

// Adds a computer waiting for its runner, or with an id mints a fresh command for one that has none; the dialog shows failures.
export const useEnrollComputer = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: requires("none", async (computerId: string | undefined) =>
      (await api.post<ComputerEnrollment>("/api/pairing/computers/enrollments", computerId ? { id: computerId } : {})).data),
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
    mutationFn: requires("none", async () => (await api.post<Computer>(`/api/pairing/computers/${computer.id}/pair`)).data),
    onSuccess: async () => {
      void client.invalidateQueries({ queryKey: [getHarnessProjectsKey, computer.id] });
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
    mutationFn: requires("none", async (input: RenameComputerFormData) => (await api.patch<Computer>(`/api/pairing/computers/${computerId}`, input)).data),
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: [getComputersKey] });
    },
  });
};

const runsGoing = (running: number) => `${running} play ${running === 1 ? "run is" : "runs are"} going on it.`;

// Restarts the Nexul app on a computer, asking first while play runs are going on it; the row then reads Reconnecting
// until the app is back, and the facts it reports refresh the row.
export const useRestartApp = (computer: Computer) => {
  const client = useQueryClient();
  const { open: confirm } = useConfirmationDialog();
  const restart = useMutation({
    mutationFn: requires("none", async () => {
      await api.post(`/api/pairing/computers/${computer.id}/restart`);
    }),
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: [getComputersKey] });
      toast.success(`Restarting the Nexul app on ${computer.name}`);
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
  const ask = async () => {
    const ok =
      computer.running === 0 ||
      (await confirm({
        title: `Restart the Nexul app on ${computer.name}?`,
        message: `${runsGoing(computer.running)} Its connection to Nexul drops for a few seconds while the app restarts.`,
        confirmLabel: "Restart Nexul app",
        destructive: false,
      }));
    if (ok) restart.mutate();
  };
  return { restart: ask, isPending: restart.isPending };
};
