import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { requires } from "@nexul/client-core/requires";

import { api, errorMessage } from "@/api/client";
import { TERMINAL_OPEN_SIZE, type ComputerCommand, type ComputerCommandPage } from "@/models/ComputerCommand";
import type { LiveFollower } from "@/lib/live";

export const getComputerCommandsKey = "getComputerCommands";

// The newest commands the row shows; computer_list with commands reads the rest of the record.
const COMMANDS_SHOWN = 10;

// Read when the details open; each command's start and end arrive as computer.shell_job_changed.
export const useFetchComputerCommands = (computerId: string) =>
  useQuery({
    queryKey: [getComputerCommandsKey, computerId],
    queryFn: async () =>
      (await api.get<ComputerCommandPage>(`/api/pairing/computers/${computerId}/commands`, { params: { limit: COMMANDS_SHOWN } })).data
        .items,
    enabled: !!computerId,
  });

export const useOpenComputerTerminal = (computerId: string) => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: requires("none", async () =>
      (await api.post<ComputerCommand>(`/api/pairing/computers/${computerId}/terminals`, TERMINAL_OPEN_SIZE)).data),
    onSuccess: () => client.invalidateQueries({ queryKey: [getComputerCommandsKey, computerId] }),
    onError: (error) => toast.error(errorMessage(error)),
  });
};

export const getCommandKey = "getCommand";

// One command or terminal with what it printed, from the computer's routes or a machine runner's.
export const useFetchCommand = (command: ComputerCommand) =>
  useQuery({
    queryKey: [getCommandKey, command.id, command.machine_id, command.status],
    queryFn: async () =>
      (await api.get<ComputerCommand>(command.machine_id ? `/api/runners/commands/${command.id}` : `/api/pairing/commands/${command.id}`)).data,
  });

export const useCancelComputerCommand = () =>
  useMutation({
    mutationFn: requires("none", async (commandId: string) =>
      (await api.post<ComputerCommand>(`/api/pairing/commands/${commandId}/cancel`)).data),
    onError: (error) => toast.error(errorMessage(error)),
  });

// The frame names the command, its computer and its new status, never the command line or output.
interface ShellJobPayload {
  computer_id: string;
}

export const computerCommandFollower: LiveFollower = {
  // A computer's command list follows each command being queued, starting and ending.
  "computer.shell_job_changed": ({ computer_id }: ShellJobPayload, { client }) =>
    client.invalidateQueries({ queryKey: [getComputerCommandsKey, computer_id], exact: true }),
};
