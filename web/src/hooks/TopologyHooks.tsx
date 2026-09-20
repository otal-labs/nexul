import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect } from "react";
import { toast } from "sonner";

import { api, errorMessage } from "@/api/client";
import type { Canvas, TopologyNode } from "@/models/Topology";
import { useFlowStore } from "@/stores/flowStore";

const getTopologyKey = "getTopology";

// The raw canvas query, no flow-store side effect; useFetchTopology feeds the store, others read this directly.
export const useFetchCanvas = (environment = "default") =>
  useQuery({
    queryKey: [getTopologyKey, environment],
    queryFn: async () => (await api.get<Canvas>("/api/topology", { params: { environment } })).data,
  });

export const useFetchTopology = (environment = "default") => {
  const setCanvas = useFlowStore((s) => s.setCanvas);
  const query = useFetchCanvas(environment);

  // In an effect, not during render: the store is the editable canvas React Flow renders and patches live.
  useEffect(() => {
    if (query.data) setCanvas(query.data);
  }, [query.data, setCanvas]);

  return query;
};

// Replaces the store with the server's returned canvas, then invalidates the query too for other subscribers.
const useApplyCanvas = () => {
  const client = useQueryClient();
  const setCanvas = useFlowStore((s) => s.setCanvas);
  return async (canvas: Canvas, environment: string) => {
    setCanvas(canvas);
    await client.invalidateQueries({ queryKey: [getTopologyKey, environment] });
  };
};

export const useAddTopologyNode = (environment = "default") => {
  const apply = useApplyCanvas();
  return useMutation({
    mutationFn: async (node: TopologyNode) =>
      (await api.post<Canvas>("/api/topology/nodes", node, { params: { environment } })).data,
    onSuccess: (canvas) => void apply(canvas, environment),
    onError: (error) => toast.error(errorMessage(error)),
  });
};

export const useRemoveTopologyNode = (environment = "default") => {
  const apply = useApplyCanvas();
  return useMutation({
    mutationFn: async (nodeId: string) =>
      (await api.delete<Canvas>(`/api/topology/nodes/${nodeId}`, { params: { environment } })).data,
    onSuccess: (canvas) => void apply(canvas, environment),
    onError: (error) => toast.error(errorMessage(error)),
  });
};

// The server merges auto-managed service nodes back in, so a client save can never delete or rename them.
export const useSaveTopology = (environment = "default") => {
  const apply = useApplyCanvas();
  const toCanvas = useFlowStore((s) => s.toCanvas);
  return useMutation({
    mutationFn: async () =>
      (await api.put<Canvas>("/api/topology", toCanvas(), { params: { environment } })).data,
    onSuccess: (canvas) => void apply(canvas, environment),
    onError: (error) => toast.error(errorMessage(error)),
  });
};
