import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { requires } from "@nexul/client-core/requires";

import { api, errorMessage } from "@/api/client";
import { getMachinesKey } from "@/hooks/RunnerHooks";
import { TERMINAL_OPEN_SIZE, type ComputerCommand, type ComputerCommandPage } from "@/models/ComputerCommand";
import type { Machine } from "@/models/Runner";
import type { LiveFollower } from "@/lib/live";

export const getMachineCommandsKey = "getMachineCommands";

// The newest commands the sheet shows; machine_list with commands reads the rest of the record.
const COMMANDS_SHOWN = 10;

// Read when the sheet opens; each command's start and end arrive as runner.shell_job_changed.
export const useFetchMachineCommands = (machineId: string, enabled: boolean) =>
  useQuery({
    queryKey: [getMachineCommandsKey, machineId],
    queryFn: async () =>
      (await api.get<ComputerCommandPage>(`/api/machines/${machineId}/commands`, { params: { limit: COMMANDS_SHOWN } })).data.items,
    enabled: !!machineId && enabled,
  });

export const useOpenMachineTerminal = (machineId: string) => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: requires(
      "machines:shell",
      async (runnerId: string) => (await api.post<ComputerCommand>(`/api/runners/${runnerId}/terminals`, TERMINAL_OPEN_SIZE)).data,
    ),
    onSuccess: () => client.invalidateQueries({ queryKey: [getMachineCommandsKey, machineId] }),
    onError: (error) => toast.error(errorMessage(error)),
  });
};

export const useCancelMachineCommand = () =>
  useMutation({
    mutationFn: requires(
      "machines:shell",
      async (commandId: string) => (await api.post<ComputerCommand>(`/api/runners/commands/${commandId}/cancel`)).data,
    ),
    onError: (error) => toast.error(errorMessage(error)),
  });

// The switch is the feedback, so it sends no success toast.
export const useSetMachineShell = (machineId: string) => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: requires(
      "machines:shell",
      async (enabled: boolean) =>
        (
          await api.put<Machine>(`/api/machines/${machineId}/shell`, {
            enabled,
          })
        ).data,
    ),
    onSuccess: () =>
      Promise.all([
        client.invalidateQueries({ queryKey: [getMachinesKey], exact: true }),
        client.invalidateQueries({
          queryKey: [getMachineCommandsKey, machineId],
        }),
      ]),
    onError: (error) => toast.error(errorMessage(error)),
  });
};

// The frame names the command, its runner and machine and its new status, never the command line or output.
interface ShellJobPayload {
  machine_id: string;
}

export const machineCommandFollower: LiveFollower = {
  // A machine's command list follows each command being queued, starting and ending.
  "runner.shell_job_changed": ({ machine_id }: ShellJobPayload, { client }) =>
    client.invalidateQueries({
      queryKey: [getMachineCommandsKey, machine_id],
      exact: true,
    }),
};
