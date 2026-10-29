import { useEffect, useMemo, useRef, useState } from "react";
import { applyNodeChanges, MarkerType, useReactFlow, useStoreApi, type OnNodesChange } from "@xyflow/react";

import { onNodesChange } from "@/components/topology/flowActions";
import { deriveWiring, summarize, tracePath } from "@/components/topology/wiring";
import { useFetchExposures, useFetchGateways } from "@/hooks/DnsHooks";
import { useFetchAllContainers, useFetchStacks } from "@/hooks/StackHooks";
import { useFetchTopology, useSaveTopology } from "@/hooks/TopologyHooks";
import { useActionState } from "@/hooks/useActionState";
import { cn } from "@/lib/utils";
import { RelationKind, type HostnameNode, type RelationEdge, type TopologyNode } from "@/models/Topology";
import { useFlowStore } from "@/stores/flowStore";
import {
  alignPillsToRows,
  buildNetworkRects,
  LANE_LAYOUT,
  laneCamera,
  laneLayout,
  lanesOf,
  overlaps,
  type Positions,
} from "@/utils/TopologyLayout";

// What React Flow measures on a pill; where it sits is derived from its gateway row every render.
type Measured = HostnameNode["measured"];

const fade = "transition-opacity duration-150 ease-standard";

const arrow = { type: MarkerType.ArrowClosed, width: 14, height: 14, color: "var(--muted-foreground)" };

// The canvas as rendered, stored and derived nodes in lanes. Needs the ReactFlowProvider mounted by TopologyCanvas.
export const useTopologyView = () => {
  const { setViewport } = useReactFlow();
  const flow = useStoreApi();
  const nodes = useFlowStore((s) => s.nodes);
  const edges = useFlowStore((s) => s.edges);
  const save = useSaveTopology();
  const canWrite = useActionState({ permission: "topology:write" }).state === "live";
  const { isPending } = useFetchTopology();
  const { data: containers = [], isPending: containersPending } = useFetchAllContainers();
  const { data: stacks = [] } = useFetchStacks();
  const { data: gateways = [], isPending: gatewaysPending } = useFetchGateways();
  const { data: exposures = [], isPending: exposuresPending } = useFetchExposures();
  // The lanes need every card and wire, so nothing is laid out until the containers, gateways and exposures are in.
  const loading = isPending || containersPending || gatewaysPending || exposuresPending;

  const [measured, setMeasured] = useState<Map<string, Measured>>(() => new Map());

  const wiring = useMemo(
    () => deriveWiring(nodes, containers ?? [], stacks, gateways, exposures),
    [nodes, containers, stacks, gateways, exposures],
  );
  // Pills are never dragged or stored: each follows its gateway row (or its service) wherever that card sits.
  const hosts = useMemo(() => {
    const at = alignPillsToRows([...wiring.hosts, ...wiring.nodes], new Map(wiring.nodes.map((n) => [n.id, n.position])), wiring.edges, wiring.networks);
    return wiring.hosts.map((h): HostnameNode => {
      const size = measured.get(h.id);
      return { ...h, draggable: false, position: at.get(h.id) ?? h.position, ...(size ? { measured: size } : {}) };
    });
  }, [wiring, measured]);
  const viewNodes = useMemo(() => [...hosts, ...wiring.nodes], [hosts, wiring.nodes]);
  const viewEdges = useMemo(
    () => [...edges.map((e): RelationEdge => (e.data?.kind === RelationKind.DependsOn ? { ...e, markerEnd: arrow } : e)), ...wiring.edges],
    [edges, wiring.edges],
  );

  // Hovering a card traces every route it is on, end to end, and dims the rest of the map.
  const [hovered, setHovered] = useState<string | null>(null);
  const lit = useMemo(() => (hovered ? tracePath(hovered, viewEdges) : null), [hovered, viewEdges]);
  const shownNodes = useMemo(
    () =>
      viewNodes.map((n): TopologyNode => {
        const node = { ...n, className: cn(fade, lit && !lit.nodes.has(n.id) && "opacity-35") } as TopologyNode;
        if (!lit || node.type !== "gateway") return node;
        const litRoutes = node.data.routes.filter((r) => lit.edges.has(`route-${r.id}-in`)).map((r) => r.id);
        return { ...node, data: { ...node.data, litRoutes } };
      }),
    [viewNodes, lit],
  );
  const shownEdges = useMemo(
    () =>
      viewEdges.map((e): RelationEdge => {
        const dim = !!lit && !lit.edges.has(e.id);
        return { ...e, className: cn(fade, dim && "opacity-35"), ...(e.data ? { data: { ...e.data, dim } } : {}) };
      }),
    [viewEdges, lit],
  );
  const networkRects = useMemo(() => buildNetworkRects(wiring.networks, viewNodes), [wiring.networks, viewNodes]);
  const shownRects = useMemo(
    () =>
      networkRects.map((r) => {
        const members = wiring.networks.find((g) => g.name === r.name)?.memberIds ?? [];
        return { ...r, dim: !!lit && !members.some((id) => lit.nodes.has(id)) };
      }),
    [networkRects, wiring.networks, lit],
  );
  const lanes = useMemo(() => lanesOf(viewNodes, networkRects), [viewNodes, networkRects]);
  const summary = useMemo(() => summarize(wiring), [wiring]);

  const place = (positions: Positions) => {
    const { nodes: current, setNodes } = useFlowStore.getState();
    setNodes(current.map((n) => ({ ...n, position: positions.get(n.id) ?? n.position })));
  };

  const cameraFor = (positions: Positions, width: number) => {
    const placed = viewNodes.map((n) => ({ ...n, position: positions.get(n.id) ?? n.position }));
    return laneCamera(lanesOf(placed, buildNetworkRects(wiring.networks, placed)), width);
  };

  // The pane is measured a frame or two after the canvas mounts; a camera set before that fits a 0px pane at 1:1.
  const whenSized = (then: (width: number) => void, frames = 30) =>
    requestAnimationFrame(() => {
      const width = flow.getState().width;
      if (width > 0 || frames === 0) return then(width);
      whenSized(then, frames - 1);
    });

  // Lays every card out in lanes again, saves it with the layout's version, and frames the result.
  const arrange = (positions = laneLayout(viewNodes, viewEdges, wiring.networks)) => {
    place(positions);
    useFlowStore.getState().setLayout(LANE_LAYOUT);
    if (canWrite) void save.mutateAsync();
    whenSized((width) => void setViewport(cameraFor(positions, width)));
  };

  // Stored positions win: the lanes run once for an older layout, else only for an unplaced node or an overlap.
  const laidOut = useRef("");
  const cameraSet = useRef(false);
  useEffect(() => {
    if (loading) return;
    const boxes = wiring.networks.map((g) => `${g.name}=${[...g.memberIds].sort().join(",")}`).join(";");
    const key = `${viewNodes.map((n) => n.id).sort().join("|")}#${boxes}`;
    if (!key || laidOut.current === key) return;
    laidOut.current = key;
    const stored = new Map(viewNodes.map((n) => [n.id, n.position]));
    const unplaced = wiring.nodes.some((n) => n.position.x === 0 && n.position.y === 0);
    const stale = useFlowStore.getState().layout < LANE_LAYOUT;
    const relayout = stale || unplaced || overlaps(viewNodes, stored, wiring.networks, viewEdges);
    if (relayout && wiring.nodes.length > 0) {
      cameraSet.current = true;
      arrange();
      return;
    }
    if (cameraSet.current) return;
    cameraSet.current = true;
    whenSized((width) => {
      const saved = useFlowStore.getState().viewport;
      if (saved && Math.round(saved.width ?? 0) === Math.round(width)) {
        void setViewport(saved);
        return;
      }
      void setViewport(cameraFor(stored, width));
    });
    // arrange, place and cameraFor read the latest render's values; the key above decides when this runs.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [loading, viewNodes, viewEdges, wiring.nodes, wiring.networks]);

  // Store nodes take their changes in the store; derived nodes keep theirs (measurements, mostly) here.
  const handleNodesChange: OnNodesChange<TopologyNode> = (changes) => {
    const hostIds = new Set(hosts.map((h) => h.id));
    const isHost = (c: (typeof changes)[number]) => "id" in c && hostIds.has(c.id);
    onNodesChange(changes.filter((c) => !isHost(c)));
    const mine = changes.filter(isHost);
    if (mine.length === 0) return;
    const next = applyNodeChanges(mine, hosts).filter((n): n is HostnameNode => n.type === "hostname");
    setMeasured(new Map(next.map((h) => [h.id, h.measured])));
  };

  const fit = () => void setViewport(cameraFor(new Map(viewNodes.map((n) => [n.id, n.position])), flow.getState().width));

  return {
    viewNodes: shownNodes,
    viewEdges: shownEdges,
    networkRects: shownRects,
    lanes,
    handleNodesChange,
    summary,
    arrange: () => arrange(),
    fit,
    trace: { set: setHovered },
  };
};
