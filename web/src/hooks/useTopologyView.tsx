import { useEffect, useMemo, useRef, useState } from "react";
import { applyNodeChanges, useReactFlow, type OnNodesChange } from "@xyflow/react";

import { onNodesChange } from "@/components/topology/flowActions";
import { deriveWiring } from "@/components/topology/wiring";
import { useFetchExposures, useFetchGateways } from "@/hooks/DnsHooks";
import { useFetchAllContainers, useFetchStacks } from "@/hooks/StackHooks";
import { useFetchTopology, useSaveTopology } from "@/hooks/TopologyHooks";
import type { HostnameNode, TopologyNode } from "@/models/Topology";
import { useFlowStore } from "@/stores/flowStore";
import { alignPillsToRows, buildNetworkRects, layoutGraph, type Positions } from "@/utils/TopologyLayout";

// Two nodes should not fill the screen; cap at 1:1 so cards keep their designed size.
const FIT = { padding: 0.2, maxZoom: 1 };

// What React Flow tells us about a derived node that the wiring can't know: where it sits and how big it measured.
type HostState = Pick<HostnameNode, "position" | "measured">;

const hostStateOf = (n: HostnameNode): HostState => ({
  position: n.position,
  ...(n.measured ? { measured: n.measured } : {}),
});

// The canvas as rendered: stored nodes plus the derived hostname pills, gateway rows, and network boxes, laid out
// once and kept in step with React Flow's measurements. Needs the ReactFlowProvider mounted by TopologyCanvas.
export const useTopologyView = () => {
  const { fitView, setViewport } = useReactFlow();
  const nodes = useFlowStore((s) => s.nodes);
  const edges = useFlowStore((s) => s.edges);
  const save = useSaveTopology();
  const { isPending } = useFetchTopology();
  const { data: containers = [] } = useFetchAllContainers();
  const { data: stacks = [] } = useFetchStacks();
  const { data: gateways = [] } = useFetchGateways();
  const { data: exposures = [] } = useFetchExposures();

  const [hostState, setHostState] = useState<Map<string, HostState>>(() => new Map());

  const wiring = useMemo(
    () => deriveWiring(nodes, containers ?? [], stacks, gateways, exposures),
    [nodes, containers, stacks, gateways, exposures],
  );
  const hosts = useMemo(
    () => wiring.hosts.map((h): HostnameNode => ({ ...h, ...hostState.get(h.id) })),
    [wiring.hosts, hostState],
  );
  const viewNodes = useMemo(() => [...hosts, ...wiring.nodes], [hosts, wiring.nodes]);
  const viewEdges = useMemo(() => [...edges, ...wiring.edges], [edges, wiring.edges]);
  const networkRects = useMemo(() => buildNetworkRects(wiring.networks, viewNodes), [wiring.networks, viewNodes]);

  // Stored positions are the owner's: the automatic layout only runs while some stored node has never been placed
  // (the server adds a service node at 0,0), and its result is saved so it never runs again for those nodes. When
  // everything is placed, only the pills (never stored) are put level with their gateway rows. Runs again when the
  // set of nodes or the network boxes change, since containers load after the stored nodes do; a run that a newer
  // set superseded is dropped, not applied. The camera is set once: the saved viewport when there is one, else fit.
  const laidOut = useRef("");
  const cameraSet = useRef(false);
  useEffect(() => {
    if (isPending) return;
    const boxes = wiring.networks.map((g) => `${g.name}=${[...g.memberIds].sort().join(",")}`).join(";");
    const key = `${viewNodes.map((n) => n.id).sort().join("|")}#${boxes}`;
    if (!key || laidOut.current === key) return;
    laidOut.current = key;
    const settle = (positions: Positions, persist: boolean) => {
      if (laidOut.current !== key) return;
      const { nodes: current, setNodes } = useFlowStore.getState();
      setNodes(
        current.map((n) => {
          const placed = positions.get(n.id);
          return placed ? { ...n, position: placed } : n;
        }),
      );
      setHostState((prev) => {
        const next = new Map(prev);
        for (const [id, position] of positions) {
          if (id.startsWith("host-")) next.set(id, { ...prev.get(id), position });
        }
        return next;
      });
      if (persist) void save.mutateAsync();
      if (cameraSet.current) return;
      cameraSet.current = true;
      // Next frame, once the new positions have rendered.
      requestAnimationFrame(() => {
        const saved = useFlowStore.getState().viewport;
        if (saved && !persist) {
          void setViewport(saved);
          return;
        }
        void fitView(FIT);
      });
    };
    const unplaced = wiring.nodes.some((n) => n.position.x === 0 && n.position.y === 0);
    if (unplaced) {
      void layoutGraph(viewNodes, viewEdges, wiring.networks).then((positions) => settle(positions, true));
      return;
    }
    settle(alignPillsToRows(viewNodes, new Map(viewNodes.map((n) => [n.id, n.position]))), false);
  }, [isPending, viewNodes, viewEdges, wiring.nodes, wiring.networks, fitView, setViewport, save]);

  // Store nodes take their changes in the store; derived nodes keep theirs (measurements, mostly) here.
  const handleNodesChange: OnNodesChange<TopologyNode> = (changes) => {
    const hostIds = new Set(hosts.map((h) => h.id));
    const isHost = (c: (typeof changes)[number]) => "id" in c && hostIds.has(c.id);
    onNodesChange(changes.filter((c) => !isHost(c)));
    const mine = changes.filter(isHost);
    if (mine.length === 0) return;
    const next = applyNodeChanges(mine, hosts).filter((n): n is HostnameNode => n.type === "hostname");
    setHostState(new Map(next.map((h) => [h.id, hostStateOf(h)])));
  };

  return { viewNodes, viewEdges, networkRects, handleNodesChange };
};
