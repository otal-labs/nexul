import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { api, errorMessage } from "@/api/client";
import { ticketChanged } from "@/hooks/TicketCache";
import type { Category } from "@/models/Category";

export const getCategoriesKey = "getCategories";
export const getProjectCategoriesKey = "getProjectCategories";

export const useFetchCategories = () =>
  useQuery({
    queryKey: [getCategoriesKey],
    queryFn: async () => (await api.get<Category[]>("/api/categories")).data,
  });

export const useFetchProjectCategories = (projectId: string | undefined) =>
  useQuery({
    queryKey: [getProjectCategoriesKey, projectId],
    queryFn: async () =>
      (await api.get<Category[]>("/api/categories", { params: { project_id: projectId } })).data,
    enabled: !!projectId,
  });

export const useCreateCategory = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async ({ project_id, name, color }: { project_id: string; name: string; color?: string }) =>
      (await api.post<Category>("/api/categories", { project_id, name, color })).data,
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: [getCategoriesKey] });
      await client.invalidateQueries({ queryKey: [getProjectCategoriesKey] });
      toast.success("Category created");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

// Full-replacement PATCH: omitting color silently clears it, so a future color-picker caller must resend it.
export const useRenameCategory = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async ({ id, name, color }: { id: string; name: string; color?: string }) =>
      (await api.patch<Category>(`/api/categories/${id}`, { name, color })).data,
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: [getCategoriesKey] });
      await client.invalidateQueries({ queryKey: [getProjectCategoriesKey] });
      toast.success("Category renamed");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

export const useReorderCategories = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async ({ project_id, ids }: { project_id: string; ids: string[] }) =>
      api.post("/api/categories/reorder", { project_id, ids }),
    // Applied before the request so a dropped row stays where it landed instead of snapping back until the refetch.
    onMutate: async ({ project_id, ids }) => {
      const key = [getProjectCategoriesKey, project_id];
      await client.cancelQueries({ queryKey: key });
      client.setQueryData<Category[]>(key, (current) =>
        current && [...current].sort((a, b) => ids.indexOf(a.id) - ids.indexOf(b.id)),
      );
    },
    onSuccess: () => toast.success("Category order updated"),
    onError: (error) => toast.error(errorMessage(error)),
    onSettled: async () => {
      await client.invalidateQueries({ queryKey: [getCategoriesKey] });
      await client.invalidateQueries({ queryKey: [getProjectCategoriesKey] });
    },
  });
};

export const useDeleteCategory = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async (id: string) => api.delete(`/api/categories/${id}`),
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: [getCategoriesKey] });
      await client.invalidateQueries({ queryKey: [getProjectCategoriesKey] });
      toast.success("Category deleted. Its tickets are uncategorized now.");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

export const useMoveTicketToCategory = () => {
  const client = useQueryClient();
  return useMutation({
    // silent: part of a board drop, which refetches once at the end (useBoardActions) instead of per step.
    mutationFn: async ({ ticketId, categoryId }: { ticketId: string; categoryId: string; silent?: boolean }) =>
      api.post(`/api/categories/${categoryId}/tickets/${ticketId}`),
    onSuccess: async (_, vars) => {
      if (vars.silent) return;
      await client.invalidateQueries({ queryKey: [getCategoriesKey] });
      await ticketChanged(client, vars.ticketId);
      toast.success("Ticket moved");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

export const useClearTicketCategory = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async ({ ticketId }: { ticketId: string; silent?: boolean }) => api.delete(`/api/categories/tickets/${ticketId}`),
    onSuccess: async (_, vars) => {
      if (vars.silent) return;
      await client.invalidateQueries({ queryKey: [getCategoriesKey] });
      await ticketChanged(client, vars.ticketId);
      toast.success("Ticket uncategorized");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};
