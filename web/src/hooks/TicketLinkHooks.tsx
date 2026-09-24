import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { api, errorMessage } from "@/api/client";
import type { BlockersByTicket, TicketLinkSet } from "@/models/TicketLink";

export const getTicketLinkSetKey = "getTicketLinkSet";
export const getBlockersKey = "getBlockers";

export const useFetchTicketLinkSet = (id: string | undefined) =>
  useQuery({
    queryKey: [getTicketLinkSetKey, id],
    queryFn: async () => (await api.get<TicketLinkSet>(`/api/tickets/${id}/ticket-links`)).data,
    enabled: !!id,
  });

// One request per board: every TicketCard reads the same cached map.
export const useFetchBlockers = () =>
  useQuery({
    queryKey: [getBlockersKey],
    queryFn: async () => (await api.get<BlockersByTicket>("/api/tickets/blockers")).data,
  });

// A link changes both ends, so every cached link set refetches, not just this ticket's.
const useLinkMutation = <TInput,>(request: (input: TInput) => Promise<unknown>, success: string) => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: request,
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: [getTicketLinkSetKey] });
      await client.invalidateQueries({ queryKey: [getBlockersKey] });
      toast.success(success);
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

export const useAddBlocker = () =>
  useLinkMutation(
    ({ id, blockerId }: { id: string; blockerId: string }) =>
      api.post(`/api/tickets/${id}/blocked-by`, { blocker_id: blockerId }),
    "Blocker added",
  );

export const useRemoveBlocker = () =>
  useLinkMutation(
    ({ id, blockerId }: { id: string; blockerId: string }) =>
      api.delete(`/api/tickets/${id}/blocked-by/${encodeURIComponent(blockerId)}`),
    "Blocker removed",
  );

export const useSetFoundIn = () =>
  useLinkMutation(
    ({ id, originId }: { id: string; originId: string }) =>
      api.put(`/api/tickets/${id}/found-in`, { origin_id: originId }),
    "Found-in set",
  );

export const useRemoveFoundIn = () =>
  useLinkMutation(({ id }: { id: string }) => api.delete(`/api/tickets/${id}/found-in`), "Found-in removed");
