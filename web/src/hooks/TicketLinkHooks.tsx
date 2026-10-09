import { useMutation, useQuery, useQueryClient, type QueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { api, errorMessage } from "@/api/client";
import type { BlockersByTicket, LinkedTicket, TicketLinkSet } from "@/models/TicketLink";
import { followEach, refetchHolding, type LiveFollower } from "@/lib/live";

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
    "Found in linked",
  );

export const useRemoveFoundIn = () =>
  useLinkMutation(({ id }: { id: string }) => api.delete(`/api/tickets/${id}/found-in`), "Found in unlinked");

export const linkedTickets = (set: TicketLinkSet): LinkedTicket[] =>
  [set.found_in, ...set.bugs_found, ...set.blocked_by, ...set.blocks].filter((t): t is LinkedTicket => !!t);

// A column's stage moved or the column went: the link sets naming a ticket in it, and the board's blockers, refetch.
export const stageMoved = (client: QueryClient, statusId: string) =>
  Promise.all([
    refetchHolding(client, [getTicketLinkSetKey], (set: TicketLinkSet) => linkedTickets(set).some((t) => t.status === statusId)),
    client.invalidateQueries({ queryKey: [getBlockersKey], exact: true }),
  ]);

// A link frame names both ends (internal/tickets/model.go TicketLink).
interface LinkPayload {
  link: { ticket_id: string; kind: string; target_id: string };
}

export const ticketLinkFollower: LiveFollower = followEach(["ticket.link_created", "ticket.link_deleted"], ({ link }: LinkPayload, { client }) =>
  Promise.all([
    client.invalidateQueries({ queryKey: [getTicketLinkSetKey, link.ticket_id], exact: true }),
    client.invalidateQueries({ queryKey: [getTicketLinkSetKey, link.target_id], exact: true }),
    link.kind === "blocked_by" && client.invalidateQueries({ queryKey: [getBlockersKey], exact: true }),
  ]),
);
