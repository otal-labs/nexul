import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { api, errorMessage } from "@/api/client";
import type { TicketType } from "@/models/TicketType";

export const getTicketTypesKey = "getTicketTypes";
export const getProjectTicketTypesKey = "getProjectTicketTypes";

export const useFetchProjectTicketTypes = (projectId: string | undefined) =>
  useQuery({
    queryKey: [getProjectTicketTypesKey, projectId],
    queryFn: async () =>
      (await api.get<TicketType[]>("/api/ticket-types", { params: { project_id: projectId } })).data,
    enabled: !!projectId,
  });

export const useCreateTicketType = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async ({ project_id, name }: { project_id: string; name: string }) =>
      (await api.post<TicketType>("/api/ticket-types", { project_id, name })).data,
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: [getTicketTypesKey] });
      await client.invalidateQueries({ queryKey: [getProjectTicketTypesKey] });
      toast.success("Ticket type created");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

// Full-replacement PATCH: an omitted field clears server-side, so callers resend the other's current value.
export const useRenameTicketType = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async ({ id, name, color }: { id: string; name: string; color: string }) =>
      (await api.patch<TicketType>(`/api/ticket-types/${id}`, { name, color })).data,
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: [getTicketTypesKey] });
      await client.invalidateQueries({ queryKey: [getProjectTicketTypesKey] });
      toast.success("Ticket type updated");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

export const useSetTicketTypeTemplate = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async ({ id, body_template }: { id: string; body_template: string }) =>
      (await api.put<TicketType>(`/api/ticket-types/${id}/template`, { body_template })).data,
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: [getTicketTypesKey] });
      await client.invalidateQueries({ queryKey: [getProjectTicketTypesKey] });
      toast.success("Template saved");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

export const useDeleteTicketType = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async (id: string) => api.delete(`/api/ticket-types/${id}`),
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: [getTicketTypesKey] });
      await client.invalidateQueries({ queryKey: [getProjectTicketTypesKey] });
      toast.success("Ticket type removed");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};
