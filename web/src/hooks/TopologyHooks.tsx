import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect } from "react";
import { toast } from "sonner";

import { api, errorMessage } from "@/api/client";
import type { Canvas, TopologyNode } from "@/models/Topology";
import { useFlowStore } from "@/stores/flowStore";
import { useWorkspaceStore } from "@/stores/workspaceStore";

const getTopologyKey = "getTopology";

// The raw canvas query of the selected workspace, no flow-store side effect; useFetchTopology feeds the store.
export const useFetchCanvas = () => {
  const workspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  return useQuery({
    queryKey: [getTopologyKey, workspaceId],
    queryFn: async () => (await api.get<Canvas>("/api/topology", { params: { workspace: workspaceId } })).data,
    enabled: !!workspaceId,
  });
};

// Pending until the store holds this workspace's canvas, so nothing lays out or saves the previous one's nodes.
export const useFetchTopology = () => {
  const workspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  const setCanvas = useFlowStore((s) => s.setCanvas);
  const loadedFor = useFlowStore((s) => s.workspaceId);
  const query = useFetchCanvas();

  // In an effect, not during render: the store is the editable canvas React Flow renders and patches live.
  useEffect(() => {
    if (query.data) setCanvas(query.data, workspaceId);
  }, [query.data, setCanvas, workspaceId]);

  return { ...query, isPending: query.isPending || (!!query.data && loadedFor !== workspaceId) };
};

// Replaces the store with the server's returned canvas, then invalidates the query too for other subscribers.
const useApplyCanvas = () => {
  const client = useQueryClient();
  const setCanvas = useFlowStore((s) => s.setCanvas);
  return async (canvas: Canvas, workspaceId: string) => {
    setCanvas(canvas, workspaceId);
    await client.invalidateQueries({ queryKey: [getTopologyKey, workspaceId] });
  };
};

export const useAddTopologyNode = () => {
  const workspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  const apply = useApplyCanvas();
  return useMutation({
    mutationFn: async (node: TopologyNode) =>
      (await api.post<Canvas>("/api/topology/nodes", node, { params: { workspace: workspaceId } })).data,
    onSuccess: (canvas) => void apply(canvas, workspaceId),
    onError: (error) => toast.error(errorMessage(error)),
  });
};

export const useRemoveTopologyNode = () => {
  const workspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  const apply = useApplyCanvas();
  return useMutation({
    mutationFn: async (nodeId: string) =>
      (await api.delete<Canvas>(`/api/topology/nodes/${nodeId}`, { params: { workspace: workspaceId } })).data,
    onSuccess: (canvas) => void apply(canvas, workspaceId),
    onError: (error) => toast.error(errorMessage(error)),
  });
};

// The server merges auto-managed service nodes back in, so a client save can never delete or rename them. It saves
// to the workspace the store's canvas came from, never one switched to since.
export const useSaveTopology = () => {
  const apply = useApplyCanvas();
  const toCanvas = useFlowStore((s) => s.toCanvas);
  return useMutation({
    mutationFn: async () => {
      const workspaceId = useFlowStore.getState().workspaceId;
      const canvas = (await api.put<Canvas>("/api/topology", toCanvas(), { params: { workspace: workspaceId } })).data;
      return { canvas, workspaceId };
    },
    onSuccess: ({ canvas, workspaceId }) => void apply(canvas, workspaceId),
    onError: (error) => toast.error(errorMessage(error)),
  });
};
