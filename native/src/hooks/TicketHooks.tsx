import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { api } from "@/api/client";
import { defineQuery } from "@/lib/liveQuery";
import { recordQueries } from "@/lib/queryClient";
import type { Ticket, TicketRole } from "@/models/Ticket";

export const getTicketsByProjectKey = "getTicketsByProject";
export const getTicketKey = "getTicket";

// ticket.assignee_changed is left out: it is published beside ticket.developer_changed, so following both refetched twice.
const ticketsByProjectQuery = defineQuery({
  key: getTicketsByProjectKey,
  fetch: (projectId: string | undefined) => api.get<Ticket[]>(`/api/tickets?project_id=${encodeURIComponent(projectId ?? "")}`),
  refreshes: {
    "ticket.created": { key: (p) => p.ticket.project_id },
    "ticket.updated": { key: (p) => p.ticket.project_id },
    "ticket.status_changed": { key: (p) => p.ticket.project_id },
    "ticket.developer_changed": { key: (p) => p.ticket.project_id },
    "ticket.tester_changed": { key: (p) => p.ticket.project_id },
    "ticket.finished": { key: (p) => p.ticket.project_id },
    "ticket.deleted": { key: (p) => p.project_id },
  },
});

export const useFetchTicketsByProject = (projectId: string | undefined) =>
  useQuery({ ...ticketsByProjectQuery.options(projectId), enabled: !!projectId });

// id may be a key such as WEB-1, which is unique only within a workspace, so a key comes with the workspace's slug.
const ticketQuery = defineQuery({
  key: getTicketKey,
  fetch: (id: string | undefined, workspace: string) =>
    api.get<Ticket>(`/api/tickets/${encodeURIComponent(id ?? "")}${workspace ? `?workspace=${encodeURIComponent(workspace)}` : ""}`),
  refreshes: {
    "ticket.updated": { record: (p) => p.ticket.id },
    "ticket.status_changed": { record: (p) => p.ticket.id },
    "ticket.developer_changed": { record: (p) => p.ticket.id },
    "ticket.tester_changed": { record: (p) => p.ticket.id },
    "ticket.finished": { record: (p) => p.ticket.id },
    "ticket.deleted": { record: (p) => p.id },
  },
});

export const useFetchTicket = (id: string | undefined, workspace = "") =>
  useQuery({ ...ticketQuery.options(id, workspace), enabled: !!id });

export const useUpdateTicketStatus = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: ({ id, status }: { id: string; status: string }) =>
      api.patch<Ticket>(`/api/tickets/${id}/status`, { status }),
    // The board moves the card at once, so a drop lands where it was let go; a refusal puts it back with the refetch.
    onMutate: async ({ id, status }) => {
      await client.cancelQueries({ queryKey: [getTicketsByProjectKey] });
      client.setQueriesData<Ticket[]>({ queryKey: [getTicketsByProjectKey] }, (old) =>
        old?.map((ticket) => (ticket.id === id ? { ...ticket, status } : ticket)),
      );
    },
    onError: () => client.invalidateQueries({ queryKey: [getTicketsByProjectKey] }),
    onSuccess: async (_, vars) => {
      await client.invalidateQueries({ queryKey: [getTicketsByProjectKey] });
      await client.invalidateQueries(recordQueries(getTicketKey, vars.id));
    },
  });
};

export const useSetTicketPerson = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: ({ id, role, login }: { id: string; role: TicketRole; login: string }) =>
      api.patch<Ticket>(`/api/tickets/${id}/${role}`, { login }),
    onSuccess: async (_, vars) => {
      await client.invalidateQueries({ queryKey: [getTicketsByProjectKey] });
      await client.invalidateQueries(recordQueries(getTicketKey, vars.id));
    },
  });
};

// Gets or creates the ticket's chat thread; kept here (not ChatHooks) since it's a ticket-screen action, not a
// conversation-list concern — Chat only needs to render whatever conversation id this hands it.
export const useOpenTicketThread = () =>
  useMutation({
    mutationFn: ({ id, workspaceId }: { id: string; workspaceId: string }) =>
      api.post<{ id: string }>(`/api/chat/tickets/${id}/thread`, { workspace_id: workspaceId }),
  });
