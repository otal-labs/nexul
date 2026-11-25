import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { requires } from "@nexul/client-core/requires";

import { api, errorMessage } from "@/api/client";
import type { ComputerCommand, ComputerCommandPage } from "@/models/ComputerCommand";
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

export const useRunComputerCommand = (computerId: string) => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: requires("none", async (command: string) =>
      (await api.post<ComputerCommand>(`/api/pairing/computers/${computerId}/commands`, { command })).data),
    onSuccess: () => client.invalidateQueries({ queryKey: [getComputerCommandsKey, computerId] }),
    onError: (error) => toast.error(errorMessage(error)),
  });
};

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
