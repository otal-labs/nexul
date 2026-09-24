import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { api, errorMessage } from "@/api/client";
import {
  TicketRole,
  type LinkBranchFormData,
  type LinkPRFormData,
  type SaveTicketFormData,
  type Ticket,
  type TicketLinks,
  type TicketStatus,
} from "@/models/Ticket";

export const getTicketsKey = "getTickets";
export const getTicketKey = "getTicket";
const getTicketsByDocKey = "getTicketsByDoc";
const getTicketsByProjectKey = "getTicketsByProject";
export const getTicketLinksKey = "getTicketLinks";
const getLabelColorsKey = "getLabelColors";

export const useFetchTickets = () =>
  useQuery({
    queryKey: [getTicketsKey],
    queryFn: async () => (await api.get<Ticket[]>("/api/tickets")).data,
  });

export const useFetchTicket = (id: string | undefined) =>
  useQuery({
    queryKey: [getTicketKey, id],
    queryFn: async () => (await api.get<Ticket>(`/api/tickets/${id}`)).data,
    enabled: !!id,
  });

export const useFetchTicketsByDoc = (docId: string | undefined) =>
  useQuery({
    queryKey: [getTicketsByDocKey, docId],
    queryFn: async () => (await api.get<Ticket[]>("/api/tickets", { params: { doc_id: docId } })).data,
    enabled: !!docId,
  });

export const useFetchTicketsByProject = (projectId: string | undefined) =>
  useQuery({
    queryKey: [getTicketsByProjectKey, projectId],
    queryFn: async () => (await api.get<Ticket[]>("/api/tickets", { params: { project_id: projectId } })).data,
    enabled: !!projectId,
  });

export const useCreateTicket = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async (input: SaveTicketFormData) => (await api.post<Ticket>("/api/tickets", input)).data,
    onSuccess: async (_, vars) => {
      await client.invalidateQueries({ queryKey: [getTicketsKey] });
      if (vars.doc_id) await client.invalidateQueries({ queryKey: [getTicketsByDocKey, vars.doc_id] });
      if (vars.project_id) await client.invalidateQueries({ queryKey: [getTicketsByProjectKey, vars.project_id] });
      toast.success("Ticket created");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

export const useUpdateTicket = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async ({ id, title, body }: { id: string; title: string; body: string }) =>
      (await api.patch<Ticket>(`/api/tickets/${id}`, { title, body })).data,
    onSuccess: async (_, vars) => {
      await client.invalidateQueries({ queryKey: [getTicketsKey] });
      await client.invalidateQueries({ queryKey: [getTicketKey, vars.id] });
      await client.invalidateQueries({ queryKey: [getTicketsByDocKey] });
      await client.invalidateQueries({ queryKey: [getTicketsByProjectKey] });
      toast.success("Ticket updated");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

export const useUpdateTicketStatus = () => {
  const client = useQueryClient();
  return useMutation({
    // silent: part of a board drop, which refetches once at the end (useBoardActions) instead of per step.
    mutationFn: async ({ id, status }: { id: string; status: TicketStatus; silent?: boolean }) =>
      (await api.patch<Ticket>(`/api/tickets/${id}/status`, { status })).data,
    onSuccess: async (_, vars) => {
      if (vars.silent) return;
      await client.invalidateQueries({ queryKey: [getTicketsKey] });
      await client.invalidateQueries({ queryKey: [getTicketKey, vars.id] });
      await client.invalidateQueries({ queryKey: [getTicketsByDocKey] });
      await client.invalidateQueries({ queryKey: [getTicketsByProjectKey] });
      toast.success("Status updated");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

export const useSetTicketPerson = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async ({ id, role, login }: { id: string; role: TicketRole; login: string }) =>
      (await api.patch<Ticket>(`/api/tickets/${id}/${role}`, { login })).data,
    onSuccess: async (_, vars) => {
      await client.invalidateQueries({ queryKey: [getTicketsKey] });
      await client.invalidateQueries({ queryKey: [getTicketKey, vars.id] });
      toast.success(vars.role === TicketRole.Tester ? "Tester updated" : "Developer updated");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

export const useSetTicketType = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async ({ id, typeId }: { id: string; typeId: string }) =>
      (await api.patch<Ticket>(`/api/tickets/${id}/type`, { type_id: typeId })).data,
    onSuccess: async (_, vars) => {
      await client.invalidateQueries({ queryKey: [getTicketsKey] });
      await client.invalidateQueries({ queryKey: [getTicketKey, vars.id] });
      toast.success("Type updated");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

export const useAddLabel = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async ({ id, label }: { id: string; label: string }) =>
      (await api.post<Ticket>(`/api/tickets/${id}/labels`, { label })).data,
    onSuccess: async (_, vars) => {
      await client.invalidateQueries({ queryKey: [getTicketsKey] });
      await client.invalidateQueries({ queryKey: [getTicketKey, vars.id] });
      await client.invalidateQueries({ queryKey: ["getAllLabels"] });
      toast.success("Label added");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

export const useRemoveLabel = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async ({ id, label }: { id: string; label: string }) =>
      (await api.delete<Ticket>(`/api/tickets/${id}/labels/${encodeURIComponent(label)}`)).data,
    onSuccess: async (_, vars) => {
      await client.invalidateQueries({ queryKey: [getTicketsKey] });
      await client.invalidateQueries({ queryKey: [getTicketKey, vars.id] });
      toast.success("Label removed");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

// No success toast: a reorder fans out into several calls at once, and the card animating in is feedback enough.
export const useUpdateTicketPosition = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async ({ id, position }: { id: string; position: number; silent?: boolean }) =>
      (await api.patch<Ticket>(`/api/tickets/${id}/position`, { position })).data,
    onSuccess: async (_, vars) => {
      if (vars.silent) return;
      await client.invalidateQueries({ queryKey: [getTicketsKey] });
      await client.invalidateQueries({ queryKey: [getTicketKey, vars.id] });
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
    mutationFn: async ({ id, input }: { id: string; input: LinkBranchFormData }) =>
      (await api.post<TicketLinks>(`/api/tickets/${id}/branches`, input)).data,
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
    mutationFn: async ({ id, input }: { id: string; input: LinkPRFormData }) =>
      (await api.post<TicketLinks>(`/api/tickets/${id}/prs`, input)).data,
    onSuccess: async (_, vars) => {
      await client.invalidateQueries({ queryKey: [getTicketLinksKey, vars.id] });
      toast.success("Pull request linked");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

export const useFetchAllLabels = () =>
  useQuery({
    queryKey: ["getAllLabels"],
    queryFn: async () => (await api.get<string[]>("/api/tickets/labels")).data,
  });

// TanStack dedupes identical queryKeys, so every TicketCard sharing this hook still fires one HTTP call.
export const useFetchLabelColors = (projectId: string | undefined) => {
  const { data: labels } = useFetchAllLabels();
  return useQuery({
    queryKey: [getLabelColorsKey, projectId, labels],
    queryFn: async () =>
      (await api.post<Record<string, string>>("/api/tickets/labels/colors", { project_id: projectId, labels })).data,
    enabled: !!projectId && !!labels && labels.length > 0,
  });
};

export const useSetLabelColor = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async ({ label, color, project_id }: { label: string; color: string; project_id: string }) =>
      (
        await api.put<{ label: string; color: string }>(`/api/tickets/labels/${encodeURIComponent(label)}/color`, {
          project_id,
          color,
        })
      ).data,
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: [getLabelColorsKey] });
      toast.success("Label color updated");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};
