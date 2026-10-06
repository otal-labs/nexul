import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { api, errorMessage } from "@/api/client";
import { StatusKind, type BoardStatus } from "@/models/Status";
import { TicketStatus, type Ticket } from "@/models/Ticket";

export const getStatusesKey = "getStatuses";
export const getProjectStatusesKey = "getProjectStatuses";

export const useFetchProjectStatuses = (projectId: string | undefined) =>
  useQuery({
    queryKey: [getProjectStatusesKey, projectId],
    queryFn: async () =>
      (await api.get<BoardStatus[]>("/api/statuses", { params: { project_id: projectId } })).data,
    enabled: !!projectId,
  });

// Tickets filed before statuses became per-project columns can still hold one of these values.
const legacyStatuses: Record<string, Pick<BoardStatus, "name" | "kind" | "icon">> = {
  [TicketStatus.Open]: { name: "Open", kind: StatusKind.Backlog, icon: "" },
  [TicketStatus.InProgress]: { name: "In progress", kind: StatusKind.Progress, icon: "" },
  [TicketStatus.Done]: { name: "Done", kind: StatusKind.Done, icon: "" },
  [TicketStatus.Closed]: { name: "Closed", kind: StatusKind.Backlog, icon: "CircleX" },
};

export const useTicketStatus = (ticket: Pick<Ticket, "project_id" | "status">) => {
  const { data: statuses, isPending } = useFetchProjectStatuses(ticket.project_id);
  const status = statuses?.find((s) => s.id === ticket.status) ?? legacyStatuses[ticket.status];
  return { status, isPending };
};

export const useCreateStatus = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async ({
      project_id,
      name,
      kind,
      icon,
    }: {
      project_id: string;
      name: string;
      kind: StatusKind;
      icon: string;
    }) => (await api.post<BoardStatus>("/api/statuses", { project_id, name, kind, icon })).data,
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: [getStatusesKey] });
      await client.invalidateQueries({ queryKey: [getProjectStatusesKey] });
      toast.success("Status column added");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

export const useRenameStatus = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async ({
      id,
      name,
      kind,
      icon,
    }: {
      id: string;
      name: string;
      kind: StatusKind;
      icon: string;
    }) => (await api.patch<BoardStatus>(`/api/statuses/${id}`, { name, kind, icon })).data,
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: [getStatusesKey] });
      await client.invalidateQueries({ queryKey: [getProjectStatusesKey] });
      toast.success("Status column updated");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

export const useReorderStatuses = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async ({ project_id, ids }: { project_id: string; ids: string[] }) =>
      api.post("/api/statuses/reorder", { project_id, ids }),
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: [getStatusesKey] });
      await client.invalidateQueries({ queryKey: [getProjectStatusesKey] });
      toast.success("Status order updated");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

export const useDeleteStatus = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async (id: string) => api.delete(`/api/statuses/${id}`),
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: [getStatusesKey] });
      await client.invalidateQueries({ queryKey: [getProjectStatusesKey] });
      toast.success("Status column removed");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};
