import type { QueryClient, QueryKey } from "@tanstack/react-query";

import { getBlockersKey, getTicketLinkSetKey, linkedTickets } from "@/hooks/TicketLinkHooks";
import type { Ticket } from "@/models/Ticket";
import type { BlockersByTicket, TicketLinkSet } from "@/models/TicketLink";

export const getTicketsKey = "getTickets";
export const getTicketKey = "getTicket";
export const getTicketsByDocKey = "getTicketsByDoc";
export const getTicketsByProjectKey = "getTicketsByProject";

// The three list views and which tickets each holds; the key's second element is the doc or project it filters on.
const listViews: { key: string; holds: (ticket: Ticket, param: unknown) => boolean }[] = [
  { key: getTicketsKey, holds: () => true },
  { key: getTicketsByDocKey, holds: (ticket, docId) => ticket.doc_id === docId },
  { key: getTicketsByProjectKey, holds: (ticket, projectId) => ticket.project_id === projectId },
];

const cachedLists = (client: QueryClient) =>
  listViews.flatMap((view) => client.getQueriesData<Ticket[]>({ queryKey: [view.key] }).map(([key, list]) => ({ view, key, list })));

const cachedTicket = (client: QueryClient, id: string): Ticket | undefined =>
  client.getQueryData<Ticket>([getTicketKey, id]) ??
  cachedLists(client)
    .flatMap(({ list }) => list ?? [])
    .find((t) => t.id === id);

const refetch = (client: QueryClient, queryKey: QueryKey) => client.invalidateQueries({ queryKey, exact: true });

const linkedIds = (set: TicketLinkSet): string[] => linkedTickets(set).map((t) => t.id);

// Other tickets' link sets and the board's blockers show a ticket's title and stage, so they follow those two fields.
const refetchMentions = (client: QueryClient, id: string, stageMayMove: boolean) => {
  const sets = client
    .getQueriesData<TicketLinkSet>({ queryKey: [getTicketLinkSetKey] })
    .filter(([, set]) => set && linkedIds(set).includes(id))
    .map(([key]) => refetch(client, key));
  const blockers = client.getQueryData<BlockersByTicket>([getBlockersKey]);
  const named = !!blockers && Object.values(blockers).some((list) => list.some((t) => t.id === id));
  return [...sets, ...(stageMayMove || named ? [refetch(client, [getBlockersKey])] : [])];
};

const settle = async (work: Promise<unknown>[]) => {
  await Promise.all(work);
};

// A bare id, from an endpoint that returns no ticket, refetches the views holding it instead of patching them.
export const ticketChanged = (client: QueryClient, ticket: Ticket | string): Promise<void> => {
  if (typeof ticket === "string") {
    const holders = cachedLists(client).filter(({ list }) => list?.some((t) => t.id === ticket));
    return settle([refetch(client, [getTicketKey, ticket]), ...holders.map(({ key }) => refetch(client, key)), ...refetchMentions(client, ticket, false)]);
  }
  const previous = cachedTicket(client, ticket.id);
  // A frame can trail a newer copy the viewer's own save already wrote.
  if (previous && Date.parse(ticket.updated_at) < Date.parse(previous.updated_at)) return settle([]);
  client.setQueryData<Ticket>([getTicketKey, ticket.id], (old) => old && ticket);
  const work: Promise<unknown>[] = [];
  for (const { view, key, list } of cachedLists(client)) {
    if (!list) continue;
    const belongs = view.holds(ticket, key[1]);
    const present = list.some((t) => t.id === ticket.id);
    if (present && belongs) client.setQueryData<Ticket[]>(key, list.map((t) => (t.id === ticket.id ? ticket : t)));
    if (present && !belongs) client.setQueryData<Ticket[]>(key, list.filter((t) => t.id !== ticket.id));
    if (!present && belongs) work.push(refetch(client, key));
  }
  const stageMoved = previous?.status !== ticket.status;
  if (stageMoved || previous?.title !== ticket.title) work.push(...refetchMentions(client, ticket.id, stageMoved));
  return settle(work);
};

// A new ticket: the lists that hold it refetch for the server's order, and its own view starts from the server's copy.
export const ticketCreated = (client: QueryClient, ticket: Ticket): Promise<void> => {
  client.setQueryData<Ticket>([getTicketKey, ticket.id], ticket);
  return settle(cachedLists(client).filter(({ view, key }) => view.holds(ticket, key[1])).map(({ key }) => refetch(client, key)));
};

// A deleted ticket leaves every list, and an open page of it refetches into its not-found state.
export const ticketRemoved = (client: QueryClient, id: string): Promise<void> => {
  for (const { key, list } of cachedLists(client)) {
    if (list?.some((t) => t.id === id)) client.setQueryData<Ticket[]>(key, list.filter((t) => t.id !== id));
  }
  return settle([refetch(client, [getTicketKey, id]), ...refetchMentions(client, id, true)]);
};

// A deleted column's tickets move to another one server-side, so every view holding one of them refetches.
export const statusRemoved = (client: QueryClient, statusId: string): Promise<void> => {
  const lists = cachedLists(client);
  const singles = client.getQueriesData<Ticket>({ queryKey: [getTicketKey] }).flatMap(([, t]) => (t ? [t] : []));
  const moved = new Set([...lists.flatMap(({ list }) => list ?? []), ...singles].filter((t) => t.status === statusId).map((t) => t.id));
  const holders = lists.filter(({ list }) => list?.some((t) => moved.has(t.id)));
  return settle([...holders.map(({ key }) => refetch(client, key)), ...[...moved].map((id) => refetch(client, [getTicketKey, id]))]);
};
