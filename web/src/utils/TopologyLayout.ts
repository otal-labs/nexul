import ELK, { type ElkNode } from "elkjs/lib/elk.bundled.js";

import { GATEWAY_FOOTER_H, GATEWAY_HEADER_H, GATEWAY_ROW_H } from "@/components/topology/GatewayNode";
import type { NetworkGroup, RelationEdge, TopologyNode } from "@/models/Topology";

// Fixed footprints: layout and network boxes run before React Flow measures nodes; must track the card classes.
const FOOTPRINTS: Record<Exclude<TopologyNode["type"], "gateway" | "hostname" | "service">, { w: number; h: number }> = {
  network: { w: 192, h: 60 },
  external: { w: 192, h: 84 },
};

// Mono glyph advance at 12px plus the pill's icons and padding; measured widths replace this once known.
const PILL_GLYPH = 7.2;
const PILL_CHROME = 76;
const PILL_H = 40;
// A service card: the 48px header (56 with a url line) and an 18px mono line per fact under a 16px-padded rule; the
// width grows from the name past the icon tile and the status.
const SERVICE_MIN_W = 240;
const NAME_GLYPH = 7.5;
const SERVICE_CHROME = 150;
// Card padding, glyph, arrow, and gaps around a "→ service:port (address:port)" row.
const ROW_GLYPH = 7.2;
const ROW_CHROME = 56;
const GATEWAY_MIN_W = 240;

// Where the pill column ends; every pill right-aligns here so wires in stay short and parallel.
const PILL_GAP = 72;

export const footprintOf = (n: TopologyNode): { w: number; h: number } => {
  if (n.measured?.width && n.measured.height) return { w: n.measured.width, h: n.measured.height };
  return designedFootprint(n);
};

const designedFootprint = (n: TopologyNode): { w: number; h: number } => {
  if (n.type === "hostname") return { w: Math.round(n.data.hostname.length * PILL_GLYPH + PILL_CHROME), h: PILL_H };
  if (n.type === "service") {
    const facts = [n.data.target, n.data.replicas != null, n.data.volume].filter(Boolean).length;
    return {
      w: Math.max(SERVICE_MIN_W, Math.round(n.data.name.length * NAME_GLYPH + SERVICE_CHROME)),
      h: (n.data.url ? 56 : 48) + (facts > 0 ? 17 + facts * 18 : 0),
    };
  }
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
  const { w, h } = boundsOf(n);
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

// A service on no network (compose reads its own) has no story to tell here; it sits past everything else, unless
// a wire leaves it, which ELK cannot route out of the last layer.
const loose = (n: TopologyNode, sources: Set<string>): ElkNode => ({
  ...leaf(n),
  ...(n.type === "service" && !sources.has(n.id) ? { layoutOptions: { "elk.layered.layering.layerConstraint": "LAST" } } : {}),
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

// Clear space between a hostname pill and the box or card it points into.
const PILL_LEAD = 48;
const PILL_STEP = PILL_H + 8;

const membersByNode = (networks: NetworkGroup[]) => {
  const byNode = new Map<string, string[]>();
  for (const g of networks) for (const id of g.memberIds) byNode.set(id, [...(byNode.get(id) ?? []), ...g.memberIds]);
  return byNode;
};

// Each pill moves level with its gateway row and right-aligns to a common x, so the wires in read as one table. A
// pill wired straight to a service (no gateway on the canvas) sits left of that service's network box, centred on it.
export const alignPillsToRows = (
  nodes: TopologyNode[],
  positions: Positions,
  edges: RelationEdge[] = [],
  networks: NetworkGroup[] = [],
): Positions => {
  const out = new Map(positions);
  const byId = new Map(nodes.map((n) => [n.id, n]));
  for (const n of nodes) {
    if (n.type !== "gateway") continue;
    const at = positions.get(n.id);
    if (!at) continue;
    n.data.routes.forEach((r, i) => {
      const pill = byId.get(`host-${r.id}`);
      if (!pill) return;
      const { w } = footprintOf(pill);
      out.set(pill.id, { x: at.x - PILL_GAP - w, y: at.y + GATEWAY_HEADER_H + i * GATEWAY_ROW_H + GATEWAY_ROW_H / 2 - PILL_H / 2 });
    });
  }
  const direct = new Map<string, TopologyNode[]>();
  for (const e of edges) {
    const pill = byId.get(e.source);
    if (pill?.type !== "hostname" || e.type === "route") continue;
    direct.set(e.target, [...(direct.get(e.target) ?? []), pill]);
  }
  const boxMates = membersByNode(networks);
  for (const [targetId, pills] of direct) {
    const target = byId.get(targetId);
    const at = positions.get(targetId);
    if (!target || !at) continue;
    const left = Math.min(at.x, ...(boxMates.get(targetId) ?? []).map((id) => positions.get(id)?.x ?? at.x)) - PADDING;
    const middle = at.y + footprintOf(target).h / 2;
    pills.forEach((pill, i) => {
      const { w } = footprintOf(pill);
      out.set(pill.id, { x: left - PILL_LEAD - w, y: middle - PILL_H / 2 + (i - (pills.length - 1) / 2) * PILL_STEP });
    });
  }
  return out;
};

// A card's measurement lags its data (a service's machine row arrives with the stacks), so the check takes the larger of
// the measured and the designed size.
const boundsOf = (n: TopologyNode) => {
  const designed = designedFootprint(n);
  return { w: Math.max(designed.w, n.measured?.width ?? 0), h: Math.max(designed.h, n.measured?.height ?? 0) };
};

interface Rect {
  x: number;
  y: number;
  w: number;
  h: number;
}

// Closer than this reads as touching: two cards edge to edge look like one, two boxes like one double line.
const CLEARANCE = 12;

const hit = (a: Rect, b: Rect) =>
  a.x < b.x + b.w + CLEARANCE && b.x < a.x + a.w + CLEARANCE && a.y < b.y + b.h + CLEARANCE && b.y < a.y + a.h + CLEARANCE;

// Whether the stored arrangement draws anything on top of, or touching, anything else: two cards, a card and a network box it is not
// in, or two boxes with no member in common. Pills are placed afterwards, so they are not counted.
// ponytail: pairwise O(n²), fine for a canvas of tens of nodes; a sweep line if canvases reach the hundreds.
export const overlaps = (nodes: TopologyNode[], positions: Positions, networks: NetworkGroup[] = []): boolean => {
  const cards = nodes
    .filter((n) => n.type !== "hostname" && positions.has(n.id))
    .map((n) => ({ id: n.id, ...positions.get(n.id)!, ...boundsOf(n) }));
  for (let i = 0; i < cards.length; i++) for (let j = i + 1; j < cards.length; j++) if (hit(cards[i]!, cards[j]!)) return true;
  const placed = nodes.map((n) => ({ ...n, position: positions.get(n.id) ?? n.position }));
  const boxes = buildNetworkRects(networks, placed).map((r) => ({
    rect: { x: r.x, y: r.y, w: r.width, h: r.height },
    members: new Set(networks.find((g) => g.name === r.name)?.memberIds ?? []),
  }));
  for (let i = 0; i < boxes.length; i++) {
    const a = boxes[i]!;
    if (cards.some((c) => !a.members.has(c.id) && hit(a.rect, c))) return true;
    for (let j = i + 1; j < boxes.length; j++) {
      const b = boxes[j]!;
      if ([...a.members].some((id) => b.members.has(id))) continue;
      if (hit(a.rect, b.rect)) return true;
    }
  }
  return false;
};

export const layoutGraph = async (
  nodes: TopologyNode[],
  edges: RelationEdge[],
  networks: NetworkGroup[] = [],
): Promise<Positions> => {
  const sources = new Set(edges.map((e) => e.source));
  // ELK nests a node in one box only, so a node on two networks lays out in the first.
  const home = new Map<string, string>();
  for (const g of networks) for (const id of g.memberIds) if (!home.has(id)) home.set(id, g.name);
  const graph: ElkNode = {
    id: "root",
    children: [
      ...networks.map((g) => ({
        id: `net-${g.name}`,
        layoutOptions: { "elk.padding": GROUP_PADDING },
        children: nodes.filter((n) => home.get(n.id) === g.name).map(leaf),
      })),
      ...nodes.filter((n) => !home.has(n.id)).map((n) => loose(n, sources)),
    ],
    edges: edges.map((e) => ({
      id: e.id,
      sources: [e.sourceHandle ? `${e.source}:${e.sourceHandle}` : e.source],
      targets: [e.targetHandle ? `${e.target}:${e.targetHandle}` : e.target],
    })),
  };
  const positions: Positions = new Map();
  collect(await elk.layout(graph), 0, 0, positions);
  return alignPillsToRows(nodes, positions, edges, networks);
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
