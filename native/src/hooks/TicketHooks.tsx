import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { api } from "@/api/client";
import type { Ticket, TicketRole } from "@/models/Ticket";

export const getTicketsByProjectKey = "getTicketsByProject";
export const getTicketKey = "getTicket";

export const useFetchTicketsByProject = (projectId: string | undefined) =>
  useQuery({
    queryKey: [getTicketsByProjectKey, projectId],
    queryFn: () => api.get<Ticket[]>(`/api/tickets?project_id=${encodeURIComponent(projectId ?? "")}`),
    enabled: !!projectId,
  });

// id may be a key such as WEB-1, which is unique only within a workspace, so a key comes with the workspace's slug.
export const useFetchTicket = (id: string | undefined, workspace = "") =>
  useQuery({
    queryKey: [getTicketKey, id, workspace],
    queryFn: () =>
      api.get<Ticket>(`/api/tickets/${encodeURIComponent(id ?? "")}${workspace ? `?workspace=${encodeURIComponent(workspace)}` : ""}`),
    enabled: !!id,
  });

export const useUpdateTicketStatus = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: ({ id, status }: { id: string; status: string }) =>
      api.patch<Ticket>(`/api/tickets/${id}/status`, { status }),
    onSuccess: async (_, vars) => {
      await client.invalidateQueries({ queryKey: [getTicketsByProjectKey] });
      await client.invalidateQueries({ queryKey: [getTicketKey, vars.id] });
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
      await client.invalidateQueries({ queryKey: [getTicketKey, vars.id] });
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
