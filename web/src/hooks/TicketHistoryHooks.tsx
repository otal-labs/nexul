import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { requires } from "@nexul/client-core/requires";

import { api, errorMessage } from "@/api/client";
import { getHistoryKey } from "@/hooks/HistoryHooks";
import { ticketChanged } from "@/hooks/TicketCache";
import type { Ticket } from "@/models/Ticket";

export const ticketHistoryUrl = (ticketId: string) => `/api/tickets/${ticketId}/history`;
export const ticketVersionUrl = (ticketId: string, version: number) => `/api/tickets/${ticketId}/versions/${version}`;

// The old text lands as a new version on top; an open editing room follows it from the server.
export const useRestoreTicketVersion = (ticketId: string) => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: requires("tickets:write", async (version: number) => (await api.post<Ticket>(`${ticketVersionUrl(ticketId, version)}/restore`)).data),
    onSuccess: async (ticket) => {
      await ticketChanged(client, ticket);
      await client.invalidateQueries({ queryKey: [getHistoryKey, ticketHistoryUrl(ticketId)] });
      toast.success("Version restored");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};
