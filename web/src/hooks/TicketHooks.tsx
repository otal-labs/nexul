import { queryOptions, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { requires } from "@nexul/client-core/requires";

import { api, errorMessage } from "@/api/client";
import {
  getTicketKey,
  getTicketsByDocKey,
  getTicketsByProjectKey,
  getTicketsKey,
  statusRemoved,
  ticketCategoryMoved,
  ticketChanged,
  ticketCreated,
  ticketEdited,
  ticketFetched,
  ticketProjectMoved,
  ticketRemoved,
  type TicketEdit,
} from "@/hooks/TicketCache";
import { getAllLabelsKey } from "@/hooks/LabelHooks";
import { getTicketLinkSetKey } from "@/hooks/TicketLinkHooks";
import {
  TicketRole,
  type BranchLink,
  type LinkBranchFormData,
  type LinkPRFormData,
  type PRLink,
  type SaveTicketFormData,
  type Ticket,
  type TicketLinks,
} from "@/models/Ticket";
import { isTrailActive, type RunFrame } from "@/models/Trail";
import { followEach, type LiveFollower } from "@/lib/live";

export const getTicketLinksKey = "getTicketLinks";
const getTicketIdByKeyKey = "getTicketIdByKey";

// The board and pickers show titles only, so the lists leave bodies out; a ticket's page reads its own.
export const useFetchTickets = () =>
  useQuery({
    queryKey: [getTicketsKey],
    queryFn: async () => (await api.get<Ticket[]>("/api/tickets", { params: { without: "body" } })).data,
  });

// Shared by the ticket page and the hover prefetch of whatever opens it.
export const ticketQuery = (id: string | undefined) =>
  queryOptions({
    queryKey: [getTicketKey, id],
    queryFn: async () => (await api.get<Ticket>(`/api/tickets/${id}`)).data,
    enabled: !!id,
  });

export const useFetchTicket = (id: string | undefined) => useQuery(ticketQuery(id));

// A PREFIX-NUMBER link reads its one ticket by key and seeds the ticket's own view, so the page reads it once.
export const ticketIdByKeyQuery = (key: string | undefined, workspaceId: string) =>
  queryOptions({
    queryKey: [getTicketIdByKeyKey, workspaceId, key],
    queryFn: async ({ client }) => {
      const ticket = (await api.get<Ticket>(`/api/tickets/${key}`, { params: { workspace: workspaceId } })).data;
      ticketFetched(client, ticket);
      return ticket.id;
    },
    enabled: !!key,
  });

export const useFetchTicketIdByKey = (key: string | undefined, workspaceId: string) => useQuery(ticketIdByKeyQuery(key, workspaceId));

export const useFetchTicketsByDoc = (docId: string | undefined) =>
  useQuery({
    queryKey: [getTicketsByDocKey, docId],
    queryFn: async () => (await api.get<Ticket[]>("/api/tickets", { params: { doc_id: docId } })).data,
    enabled: !!docId,
  });

export const useFetchTicketsByProject = (projectId: string | undefined) =>
  useQuery({
    queryKey: [getTicketsByProjectKey, projectId],
    queryFn: async () => (await api.get<Ticket[]>("/api/tickets", { params: { project_id: projectId, without: "body" } })).data,
    enabled: !!projectId,
  });

export const useCreateTicket = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: requires("tickets:write", async (input: SaveTicketFormData) => (await api.post<Ticket>("/api/tickets", input)).data),
    onSuccess: async (created, vars) => {
      await ticketCreated(client, created);
      if (vars.origin_id) await client.invalidateQueries({ queryKey: [getTicketLinkSetKey, vars.origin_id] });
      toast.success("Ticket created");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

export const useUpdateTicket = () => {
  const client = useQueryClient();
  return useMutation({
    // silent: the second half of a create (attaching its pasted files), which already said "Ticket created".
    mutationFn: requires("tickets:write", async ({ id, title, body }: { id: string; title: string; body: string; silent?: boolean }) =>
      (await api.patch<Ticket>(`/api/tickets/${id}`, { title, body })).data),
    onSuccess: async (updated, vars) => {
      await ticketChanged(client, updated);
      if (!vars.silent) toast.success("Ticket updated");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

export const useDeleteTicket = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: requires("tickets:delete", async (id: string) => {
      await api.delete(`/api/tickets/${id}`);
    }),
    // Not awaited: the page leaves first, so its refetch of the deleted ticket never shows a not-found screen.
    onSuccess: (_, id) => {
      void ticketRemoved(client, id);
      toast.success("Ticket deleted");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

// A field edit shows in every view on the click; a refusal puts the old value back and offers to send it again.
// silent: one step of a board drop, which shows the move itself and refetches once at the end (useBoardActions).
const useTicketEdit = <V extends { id: string; silent?: boolean }>(
  send: (vars: V) => Promise<Ticket>,
  fields: (vars: V, before: Ticket | undefined) => Partial<Ticket>,
  saved?: () => Promise<unknown>,
) => {
  const client = useQueryClient();
  const edit = useMutation({
    mutationFn: send,
    onMutate: async (vars): Promise<TicketEdit | undefined> =>
      vars.silent ? undefined : ticketEdited(client, vars.id, (before) => fields(vars, before)),
    onSuccess: async (ticket, vars, edited) => {
      if (vars.silent) return;
      await ticketChanged(client, ticket, edited?.before);
      await saved?.();
    },
    onError: (error, vars, edited) => {
      if (vars.silent) {
        toast.error(errorMessage(error));
        return;
      }
      edited?.undo();
      toast.error(errorMessage(error), { action: { label: "Retry", onClick: () => { edit.mutate(vars); } } });
    },
  });
  return edit;
};

export const useUpdateTicketStatus = () =>
  useTicketEdit(
    requires("tickets:write", async ({ id, status }: { id: string; status: string; silent?: boolean }) =>
      (await api.patch<Ticket>(`/api/tickets/${id}/status`, { status })).data),
    ({ status }) => ({ status }),
  );

export const useSetTicketPerson = () =>
  useTicketEdit(
    requires("tickets:write", async ({ id, role, login }: { id: string; role: TicketRole; login: string }) =>
      (await api.patch<Ticket>(`/api/tickets/${id}/${role}`, { login })).data),
    // The id goes with the login, or the old person's id would keep matching until the server answers.
    ({ role, login }) => (role === TicketRole.Tester ? { tester: login, tester_id: "" } : { developer: login, developer_id: "" }),
  );

export const useSetTicketDueOn = () =>
  useTicketEdit(
    requires("tickets:write", async ({ id, dueOn }: { id: string; dueOn: string }) =>
      (await api.patch<Ticket>(`/api/tickets/${id}/due-date`, { due_on: dueOn })).data),
    ({ dueOn }) => ({ due_on: dueOn }),
  );

export const useSetTicketSource = () =>
  useTicketEdit(
    requires("tickets:write", async ({ id, docId }: { id: string; docId: string }) =>
      (await api.patch<Ticket>(`/api/tickets/${id}/source`, { doc_id: docId })).data),
    ({ docId }) => ({ doc_id: docId }),
  );

export const useSetTicketType = () =>
  useTicketEdit(
    requires("tickets:write", async ({ id, typeId }: { id: string; typeId: string }) =>
      (await api.patch<Ticket>(`/api/tickets/${id}/type`, { type_id: typeId })).data),
    ({ typeId }) => ({ type_id: typeId }),
  );

const without = (ticket: Ticket | undefined, label: string) => (ticket?.labels ?? []).filter((l) => l !== label);

export const useAddLabel = () => {
  const client = useQueryClient();
  return useTicketEdit(
    requires("tickets:write", async ({ id, label }: { id: string; label: string }) =>
      (await api.post<Ticket>(`/api/tickets/${id}/labels`, { label })).data),
    ({ label }, before) => ({ labels: [...without(before, label), label] }),
    () => client.invalidateQueries({ queryKey: [getAllLabelsKey] }),
  );
};

export const useRemoveLabel = () =>
  useTicketEdit(
    requires("tickets:write", async ({ id, label }: { id: string; label: string }) =>
      (await api.delete<Ticket>(`/api/tickets/${id}/labels/${encodeURIComponent(label)}`)).data),
    ({ label }, before) => ({ labels: without(before, label) }),
  );

// No success toast: a reorder fans out into several calls at once, and the card animating in is feedback enough.
export const useUpdateTicketPosition = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: requires("tickets:write", async ({ id, position }: { id: string; position: number; silent?: boolean }) =>
      (await api.patch<Ticket>(`/api/tickets/${id}/position`, { position })).data),
    onSuccess: async (ticket, vars) => {
      if (vars.silent) return;
      await ticketChanged(client, ticket);
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

export const useFetchTicketLinks = (id: string | undefined) =>
  useQuery({
    queryKey: [getTicketLinksKey, id],
    queryFn: async () => (await api.get<TicketLinks>(`/api/tickets/${id}/links`)).data,
    enabled: !!id,
  });

export const useLinkBranch = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: requires("tickets:write", async ({ id, input }: { id: string; input: LinkBranchFormData }) =>
      (await api.post<TicketLinks>(`/api/tickets/${id}/branches`, input)).data),
    onSuccess: async (_, vars) => {
      await client.invalidateQueries({ queryKey: [getTicketLinksKey, vars.id] });
      toast.success("Branch linked");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

export const useLinkPR = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: requires("tickets:write", async ({ id, input }: { id: string; input: LinkPRFormData }) =>
      (await api.post<TicketLinks>(`/api/tickets/${id}/prs`, input)).data),
    onSuccess: async (_, vars) => {
      await client.invalidateQueries({ queryKey: [getTicketLinksKey, vars.id] });
      toast.success("Pull request linked");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

// The server answers with the ticket's links as they stand, so they replace the cached ones without a refetch.
const useUnlinkMutation = <TInput extends { id: string }>(request: (input: TInput) => Promise<TicketLinks>, success: string) => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: request,
    onSuccess: (links, vars) => {
      client.setQueryData([getTicketLinksKey, vars.id], links);
      toast.success(success);
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

export const useUnlinkBranch = () =>
  useUnlinkMutation(
    requires("tickets:write", async ({ id, branch }: { id: string; branch: BranchLink }) =>
      (await api.delete<TicketLinks>(`/api/tickets/${id}/branches`, { params: { owner: branch.owner, repo: branch.repo, branch: branch.branch } })).data),
    "Branch unlinked",
  );

export const useUnlinkPR = () =>
  useUnlinkMutation(
    requires("tickets:write", async ({ id, pr }: { id: string; pr: PRLink }) =>
      (await api.delete<TicketLinks>(`/api/tickets/${id}/prs`, { params: { owner: pr.owner, repo: pr.repo, number: pr.number } })).data),
    "Pull request unlinked",
  );

// Ticket frames carry the whole ticket after the change (internal/tickets/events.go).
interface TicketPayload {
  ticket: Ticket;
}

const changedTopics = ["ticket.updated", "ticket.status_changed", "ticket.developer_changed", "ticket.tester_changed", "ticket.due_date_changed", "ticket.finished"];

export const ticketFollower: LiveFollower = {
  "ticket.created": ({ ticket }: TicketPayload, { client }) => ticketCreated(client, ticket),
  ...followEach(changedTopics, ({ ticket }: TicketPayload, { client }) => ticketChanged(client, ticket)),
  "ticket.project_changed": ({ ticket_id, project_id, previous_project_id }: { ticket_id: string; project_id: string; previous_project_id: string }, { client }) =>
    ticketProjectMoved(client, ticket_id, project_id, previous_project_id),
  "ticket.category_changed": ({ ticket_id, category_id }: { ticket_id: string; category_id: string }, { client }) => ticketCategoryMoved(client, ticket_id, category_id),
  "ticket.deleted": ({ id }: { id: string }, { client }) => ticketRemoved(client, id),
  "status.deleted": ({ status }: { status: { id: string } }, { client }) => statusRemoved(client, status.id),
  // A finished run has moved its ticket to its new column and may have linked a branch or pull request.
  "play.run": (run: RunFrame, { client }) =>
    run.target_type === "ticket" &&
    !isTrailActive(run.state) &&
    Promise.all([ticketChanged(client, run.target_id), client.invalidateQueries({ queryKey: [getTicketLinksKey, run.target_id], exact: true })]),
};
