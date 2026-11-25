import { useMutation, useQuery, useQueryClient, type QueryClient } from "@tanstack/react-query";

import { requires } from "@nexul/client-core/requires";

import { api } from "@/api/client";
import { defineQuery } from "@/lib/liveQuery";
import { recordQueries } from "@/lib/queryClient";
import type { Ticket, TicketRole } from "@/models/Ticket";

export const getTicketsByProjectKey = "getTicketsByProject";
export const getTicketKey = "getTicket";

const isOlder = (incoming: Ticket, held: Ticket) => Date.parse(incoming.updated_at) < Date.parse(held.updated_at);

// A frame or a write's answer carries the whole ticket; a payload without one refetches the entries it names, else all.
const whole = (ticket: Partial<Ticket> | undefined): ticket is Ticket => !!ticket?.id && !!ticket.updated_at;

// The project's list takes the ticket in place and any other project's list drops it; a list that doesn't hold it yet
// refetches for the server's order. A copy older than the one shown (a frame trailing a save) is ignored.
const takeIntoLists = (client: QueryClient, ticket: Partial<Ticket> | undefined) => {
  if (!whole(ticket)) return void client.invalidateQueries({ queryKey: ticket?.project_id ? [getTicketsByProjectKey, ticket.project_id] : [getTicketsByProjectKey] });
  const own = client.getQueryData<Ticket[]>([getTicketsByProjectKey, ticket.project_id]);
  if (own && !own.some((t) => t.id === ticket.id)) void client.invalidateQueries({ queryKey: [getTicketsByProjectKey, ticket.project_id], exact: true });
  const listed = { ...ticket, body: "" };
  client.setQueriesData<Ticket[]>({ queryKey: [getTicketsByProjectKey, ticket.project_id], exact: true }, (list) => {
    const held = list?.find((t) => t.id === ticket.id);
    if (!list || !held || isOlder(ticket, held)) return list;
    return list.map((t) => (t.id === ticket.id ? listed : t));
  });
  client.setQueriesData<Ticket[]>({ queryKey: [getTicketsByProjectKey], predicate: (q) => q.queryKey[1] !== ticket.project_id }, (list) =>
    list?.some((t) => t.id === ticket.id) ? list.filter((t) => t.id !== ticket.id) : list,
  );
};

const takeIntoRecord = (client: QueryClient, ticket: Partial<Ticket> | undefined) => {
  if (!whole(ticket)) return void client.invalidateQueries(ticket?.id ? recordQueries(getTicketKey, ticket.id) : { queryKey: [getTicketKey] });
  client.setQueriesData<Ticket>(recordQueries(getTicketKey, ticket.id), (old) => (old && !isOlder(ticket, old) ? { ...old, ...ticket } : old));
};

// Every view holding the ticket takes the copy a write answered with, so a move or an assignment refetches nothing.
export const takeTicket = (client: QueryClient, ticket: Ticket) => {
  takeIntoLists(client, ticket);
  takeIntoRecord(client, ticket);
};

const intoLists = { patch: (client: QueryClient, p: { ticket?: unknown }) => takeIntoLists(client, p.ticket as Partial<Ticket> | undefined) };
const intoRecord = { patch: (client: QueryClient, p: { ticket?: unknown }) => takeIntoRecord(client, p.ticket as Partial<Ticket> | undefined) };

// The board shows titles only, so its list leaves bodies out; a ticket's screen reads its own.
// ticket.assignee_changed is left out: it is published beside ticket.developer_changed, so following both refetched twice.
const ticketsByProjectQuery = defineQuery({
  key: getTicketsByProjectKey,
  fetch: (projectId: string | undefined) => api.get<Ticket[]>(`/api/tickets?project_id=${encodeURIComponent(projectId ?? "")}&without=body`),
  refreshes: {
    "ticket.created": { key: (p) => p.ticket.project_id },
    "ticket.updated": intoLists,
    "ticket.status_changed": intoLists,
    "ticket.developer_changed": intoLists,
    "ticket.tester_changed": intoLists,
    "ticket.due_date_changed": intoLists,
    "ticket.finished": intoLists,
    "ticket.deleted": {
      patch: (client, p) => client.setQueriesData<Ticket[]>({ queryKey: [getTicketsByProjectKey] }, (list) => list?.filter((t) => t.id !== p.id)),
    },
    "ticket.project_changed": {
      patch: (client, p) => {
        void client.invalidateQueries({ queryKey: [getTicketsByProjectKey, p.project_id] });
        void client.invalidateQueries({ queryKey: [getTicketsByProjectKey, p.previous_project_id] });
      },
    },
  },
});

export const useFetchTicketsByProject = (projectId: string | undefined) =>
  useQuery({ ...ticketsByProjectQuery.options(projectId), enabled: !!projectId });

// id may be a key such as WEB-1, which is unique only within a workspace, so a key comes with the workspace's slug.
const ticketQuery = defineQuery({
  key: getTicketKey,
  fetch: (id: string | undefined, workspace: string) =>
    api.get<Ticket>(`/api/tickets/${encodeURIComponent(id ?? "")}${workspace ? `?workspace=${encodeURIComponent(workspace)}` : ""}`),
  // A deleted ticket refetches into its not-found state; a moved one, under its new project.
  refreshes: {
    "ticket.updated": intoRecord,
    "ticket.status_changed": intoRecord,
    "ticket.developer_changed": intoRecord,
    "ticket.tester_changed": intoRecord,
    "ticket.due_date_changed": intoRecord,
    "ticket.finished": intoRecord,
    "ticket.deleted": { record: (p) => p.id },
    "ticket.project_changed": { record: (p) => p.ticket_id },
  },
});

export const useFetchTicket = (id: string | undefined, workspace = "") =>
  useQuery({ ...ticketQuery.options(id, workspace), enabled: !!id });

// A move answers in milliseconds; one still open after this is stuck, so it fails and the board puts the ticket back.
const ticketMoveTimeoutMs = 15_000;

export const useUpdateTicketStatus = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: requires("tickets:write", ({ id, status }: { id: string; status: string }) =>
      api.patch<Ticket>(`/api/tickets/${id}/status`, { status }, ticketMoveTimeoutMs)),
    // The board moves the card at once, so a drop lands where it was let go; a refusal puts it back with the refetch.
    onMutate: async ({ id, status }) => {
      await client.cancelQueries({ queryKey: [getTicketsByProjectKey] });
      client.setQueriesData<Ticket[]>({ queryKey: [getTicketsByProjectKey] }, (old) =>
        old?.map((ticket) => (ticket.id === id ? { ...ticket, status } : ticket)),
      );
    },
    onError: () => client.invalidateQueries({ queryKey: [getTicketsByProjectKey] }),
    onSuccess: (ticket) => takeTicket(client, ticket),
  });
};

export const useSetTicketPerson = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: requires("tickets:write", ({ id, role, login }: { id: string; role: TicketRole; login: string }) =>
      api.patch<Ticket>(`/api/tickets/${id}/${role}`, { login })),
    onSuccess: (ticket) => takeTicket(client, ticket),
  });
};

// Gets or creates the ticket's chat thread; kept here (not ChatHooks) since it's a ticket-screen action, not a
// conversation-list concern — Chat only needs to render whatever conversation id this hands it.
export const useOpenTicketThread = () =>
  useMutation({
    mutationFn: requires("tickets:read", ({ id, workspaceId }: { id: string; workspaceId: string }) =>
      api.post<{ id: string }>(`/api/chat/tickets/${id}/thread`, { workspace_id: workspaceId })),
  });
