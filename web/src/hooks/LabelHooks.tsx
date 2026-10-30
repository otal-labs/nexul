import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { requires } from "@nexul/client-core/requires";

import { api, errorMessage } from "@/api/client";

export const getAllLabelsKey = "getAllLabels";
const getLabelColorsKey = "getLabelColors";

interface LabelInput {
  label: string;
  project_id: string;
}

interface LabelColorInput extends LabelInput {
  color: string;
}

const putLabelColor = requires("tickets:write", async ({ label, color, project_id }: LabelColorInput) =>
  (await api.put<{ label: string; color: string }>(`/api/tickets/labels/${encodeURIComponent(label)}/color`, { project_id, color })).data);

// Without a project, the labels on every ticket the caller may read; with one, also the labels made in its settings.
export const useFetchAllLabels = (projectId?: string) =>
  useQuery({
    queryKey: [getAllLabelsKey, projectId],
    queryFn: async () => (await api.get<string[]>("/api/tickets/labels", { params: { project_id: projectId } })).data,
  });

// TanStack dedupes identical queryKeys, so every TicketCard sharing this hook still fires one HTTP call.
export const useFetchLabelColors = (projectId: string | undefined) => {
  const { data: labels } = useFetchAllLabels(projectId);
  return useQuery({
    queryKey: [getLabelColorsKey, projectId, labels],
    queryFn: requires("tickets:read", async () =>
      (await api.post<Record<string, string>>("/api/tickets/labels/colors", { project_id: projectId, labels })).data),
    enabled: !!projectId && !!labels && labels.length > 0,
  });
};

export const useSetLabelColor = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: putLabelColor,
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: [getLabelColorsKey] });
      toast.success("Label color updated");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

// A label is made by giving it a color, so it exists before any ticket carries it.
export const useCreateLabel = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: putLabelColor,
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: [getAllLabelsKey] });
      toast.success("Label added");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

export const useDeleteLabel = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: requires("tickets:write", async ({ label, project_id }: LabelInput) => {
      await api.delete(`/api/tickets/labels/colors/${encodeURIComponent(label)}`, { params: { project_id } });
    }),
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: [getAllLabelsKey] });
      toast.success("Label deleted");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};
