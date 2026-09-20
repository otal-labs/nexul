import ELK, { type ElkNode } from "elkjs/lib/elk.bundled.js";

import { GATEWAY_FOOTER_H, GATEWAY_HEADER_H, GATEWAY_ROW_H } from "@/components/topology/GatewayNode";
import type { NetworkGroup, RelationEdge, TopologyNode } from "@/models/Topology";

// Fixed footprints: layout and network boxes run before React Flow measures nodes; must track the card classes.
const FOOTPRINTS: Record<Exclude<TopologyNode["type"], "gateway" | "hostname">, { w: number; h: number }> = {
  service: { w: 256, h: 112 },
  network: { w: 192, h: 60 },
  external: { w: 192, h: 84 },
};

// Mono glyph advance at 12px plus the pill's icons and padding; measured widths replace this once known.
const PILL_GLYPH = 7.2;
const PILL_CHROME = 76;
const PILL_H = 40;
// Card padding, glyph, arrow, and gaps around a "→ service:port (address:port)" row.
const ROW_GLYPH = 7.2;
const ROW_CHROME = 56;
const GATEWAY_MIN_W = 240;

// Where the pill column ends; every pill right-aligns here so wires in stay short and parallel.
const PILL_GAP = 72;

export const footprintOf = (n: TopologyNode): { w: number; h: number } => {
  if (n.measured?.width && n.measured.height) return { w: n.measured.width, h: n.measured.height };
  if (n.type === "hostname") return { w: Math.round(n.data.hostname.length * PILL_GLYPH + PILL_CHROME), h: PILL_H };
  if (n.type === "gateway") {
    const longest = Math.max(0, ...n.data.routes.map((r) => `${r.service}:${r.port} (${r.address ?? ""}:${r.port})`.length));
    return {
      w: Math.max(GATEWAY_MIN_W, Math.round(longest * ROW_GLYPH + ROW_CHROME)),
      h: GATEWAY_HEADER_H + n.data.routes.length * GATEWAY_ROW_H + GATEWAY_FOOTER_H,
    };
  }
  return FOOTPRINTS[n.type];
};

// Traffic enters from the left: hostname → gateway → service, with each docker network laid out as one compound box.
const elk = new ELK({
  defaultLayoutOptions: {
    "elk.algorithm": "layered",
    "elk.direction": "RIGHT",
    "elk.hierarchyHandling": "INCLUDE_CHILDREN",
    "elk.spacing.nodeNode": "32",
    "elk.layered.spacing.nodeNodeBetweenLayers": "120",
  },
});

const GROUP_PADDING = "[top=48,left=24,bottom=24,right=24]";

export type Positions = Map<string, { x: number; y: number }>;

// A gateway exposes one west and one east port per route row, in row order, so ELK keeps the wires from crossing.
const leaf = (n: TopologyNode): ElkNode => {
  const { w, h } = footprintOf(n);
  const base: ElkNode = { id: n.id, width: w, height: h };
  if (n.type !== "gateway") return base;
  return {
    ...base,
    layoutOptions: { "elk.portConstraints": "FIXED_ORDER" },
    ports: n.data.routes.flatMap((r) => [
      { id: `${n.id}:in-${r.id}`, layoutOptions: { "elk.port.side": "WEST" } },
      { id: `${n.id}:${r.id}`, layoutOptions: { "elk.port.side": "EAST" } },
    ]),
  };
};

// A service on no network (compose reads its own) has no story to tell here; it sits past everything else.
const loose = (n: TopologyNode): ElkNode => ({
  ...leaf(n),
  ...(n.type === "service" ? { layoutOptions: { "elk.layered.layering.layerConstraint": "LAST" } } : {}),
});

// Child coordinates come back relative to their parent, so the walk flattens them to canvas space.
const collect = (parent: ElkNode, ox: number, oy: number, into: Positions) => {
  for (const child of parent.children ?? []) {
    const x = (child.x ?? 0) + ox;
    const y = (child.y ?? 0) + oy;
    if (child.children?.length) {
      collect(child, x, y, into);
      continue;
    }
    into.set(child.id, { x, y });
  }
};

// Each pill moves level with its gateway row and right-aligns to a common x, so the wires in read as one table.
export const alignPillsToRows = (nodes: TopologyNode[], positions: Positions): Positions => {
  const out = new Map(positions);
  for (const n of nodes) {
    if (n.type !== "gateway") continue;
    const at = positions.get(n.id);
    if (!at) continue;
    n.data.routes.forEach((r, i) => {
      const pill = nodes.find((p) => p.id === `host-${r.id}`);
      if (!pill) return;
      const { w } = footprintOf(pill);
      out.set(pill.id, { x: at.x - PILL_GAP - w, y: at.y + GATEWAY_HEADER_H + i * GATEWAY_ROW_H + GATEWAY_ROW_H / 2 - PILL_H / 2 });
    });
  }
  return out;
};

export const layoutGraph = async (
  nodes: TopologyNode[],
  edges: RelationEdge[],
  networks: NetworkGroup[] = [],
): Promise<Positions> => {
  const grouped = new Set(networks.flatMap((g) => g.memberIds));
  const graph: ElkNode = {
    id: "root",
    children: [
      ...networks.map((g) => ({
        id: `net-${g.name}`,
        layoutOptions: { "elk.padding": GROUP_PADDING },
        children: nodes.filter((n) => g.memberIds.includes(n.id)).map(leaf),
      })),
      ...nodes.filter((n) => !grouped.has(n.id)).map(loose),
    ],
    edges: edges.map((e) => ({
      id: e.id,
      sources: [e.sourceHandle ? `${e.source}:${e.sourceHandle}` : e.source],
      targets: [e.targetHandle ? `${e.target}:${e.targetHandle}` : e.target],
    })),
  };
  const positions: Positions = new Map();
  collect(await elk.layout(graph), 0, 0, positions);
  return alignPillsToRows(nodes, positions);
};

export interface NetworkRect {
  name: string;
  x: number;
  y: number;
  width: number;
  height: number;
}

const PADDING = 24;
const LABEL_ROOM = 24;

// One dashed box per docker network, wrapped around wherever its members currently sit; nothing about it is stored.
export const buildNetworkRects = (networks: NetworkGroup[], nodes: TopologyNode[]): NetworkRect[] =>
  networks.flatMap((group) => {
    const members = nodes.filter((n) => group.memberIds.includes(n.id));
    if (members.length === 0) return [];
    const bounds = members.reduce(
      (acc, n) => {
        const { w, h } = footprintOf(n);
        return {
          minX: Math.min(acc.minX, n.position.x),
          minY: Math.min(acc.minY, n.position.y),
          maxX: Math.max(acc.maxX, n.position.x + w),
          maxY: Math.max(acc.maxY, n.position.y + h),
        };
      },
      { minX: Infinity, minY: Infinity, maxX: -Infinity, maxY: -Infinity },
    );
    return [
      {
        name: group.name,
        x: bounds.minX - PADDING,
        y: bounds.minY - PADDING - LABEL_ROOM,
        width: bounds.maxX - bounds.minX + PADDING * 2,
        height: bounds.maxY - bounds.minY + PADDING * 2 + LABEL_ROOM,
      },
    ];
  });
