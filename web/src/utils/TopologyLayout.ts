import {
  EXTERNAL_W,
  externalHeight,
  GATEWAY_W,
  gatewayHeight,
  LANE_GAP,
  networkBoxTop,
  pillHeight,
  pillWidth,
  routeRowHeights,
  routeRowOffsets,
  SERVICE_W,
  serviceHeight,
} from "@/components/topology/geometry";
import type { NetworkGroup, RelationEdge, TopologyNode } from "@/models/Topology";

// The layout the lanes below arrange. A stored canvas below it is laid out again once; bump it when the lanes change.
export const LANE_LAYOUT = 1;

export type Positions = Map<string, { x: number; y: number }>;

// The external card's "outside Nexul" microheader rides above the card, inside the node.
const EXTERNAL_HEADER = 22;

const designedFootprint = (n: TopologyNode): { w: number; h: number } => {
  if (n.type === "hostname") return { w: pillWidth(n.data), h: pillHeight(n.data) };
  if (n.type === "service") return { w: SERVICE_W, h: serviceHeight(n.data) };
  if (n.type === "gateway") return { w: GATEWAY_W, h: gatewayHeight(n.data) };
  if (n.type === "external") return { w: EXTERNAL_W, h: EXTERNAL_HEADER + externalHeight(n.data.name, n.data.url) };
  return { w: 192, h: 60 };
};

// A card's measurement lags its data (a service's machine row arrives with the stacks), so take the larger of the two.
export const footprintOf = (n: TopologyNode): { w: number; h: number } => {
  const designed = designedFootprint(n);
  return { w: Math.max(designed.w, n.measured?.width ?? 0), h: Math.max(designed.h, n.measured?.height ?? 0) };
};

const PADDING = 24;
const STACK_GAP = 24;
const BOX_GAP = 40;
const RELATION_GUTTER = 144;
// Room above the first card of a lane for its header.
export const LANE_TOP = 32;

const pillRows = (gateway: Extract<TopologyNode, { type: "gateway" }>, byId: Map<string, TopologyNode>) => {
  const pills = new Map<string, Extract<TopologyNode, { type: "hostname" }>["data"]>();
  for (const r of gateway.data.routes) {
    const pill = byId.get(`host-${r.id}`);
    if (pill?.type === "hostname") pills.set(r.id, pill.data);
  }
  const heights = gateway.data.rowHeights ?? routeRowHeights(gateway.data, pills);
  return { heights, offsets: routeRowOffsets(gateway.data, heights) };
};

// Pills are never stored: each sits level with its gateway row (or its service), right-aligned in one column.
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
    const { heights, offsets } = pillRows(n, byId);
    n.data.routes.forEach((r, i) => {
      const pill = byId.get(`host-${r.id}`);
      if (pill?.type !== "hostname") return;
      const { w, h } = designedFootprint(pill);
      out.set(pill.id, { x: at.x - LANE_GAP - w, y: at.y + offsets[i]! + heights[i]! / 2 - h / 2 });
    });
  }
  const direct = new Map<string, TopologyNode[]>();
  for (const e of edges) {
    const pill = byId.get(e.source);
    if (pill?.type !== "hostname" || e.type === "route") continue;
    direct.set(e.target, [...(direct.get(e.target) ?? []), pill]);
  }
  const boxMates = new Map<string, string[]>();
  for (const g of networks) for (const id of g.memberIds) boxMates.set(id, [...(boxMates.get(id) ?? []), ...g.memberIds]);
  // With gateways on the canvas these pills join the column the gateway pills form; without, they sit left of the box.
  const gatewayXs = nodes.filter((n) => n.type === "gateway").flatMap((n) => positions.get(n.id)?.x ?? []);
  for (const [targetId, pills] of direct) {
    const target = byId.get(targetId);
    const at = positions.get(targetId);
    if (!target || !at) continue;
    const boxed = boxMates.has(targetId);
    const boxLeft = Math.min(at.x, ...(boxMates.get(targetId) ?? []).map((id) => positions.get(id)?.x ?? at.x)) - (boxed ? PADDING : 0);
    const left = gatewayXs.length > 0 ? Math.min(boxLeft, ...gatewayXs) : boxLeft;
    let y = at.y + footprintOf(target).h / 2 - pills.reduce((sum, p) => sum + designedFootprint(p).h + 8, -8) / 2;
    for (const pill of pills) {
      const { w, h } = designedFootprint(pill);
      out.set(pill.id, { x: left - LANE_GAP - w, y });
      y += h + 8;
    }
  }
  return out;
};

// One column per kind; services follow the order of the rows that reach them, so a gateway's wires fan uncrossed.
export const laneLayout = (nodes: TopologyNode[], edges: RelationEdge[], networks: NetworkGroup[] = []): Positions => {
  const positions: Positions = new Map();
  const byId = new Map(nodes.map((n) => [n.id, n]));
  const pills = nodes.filter((n) => n.type === "hostname");
  const gateways = nodes.filter((n) => n.type === "gateway").sort((a, b) => a.data.kind.localeCompare(b.data.kind) || a.data.name.localeCompare(b.data.name));
  const services = nodes.filter((n) => n.type === "service");
  const drawn = nodes.filter((n) => n.type === "external" || n.type === "network");

  const pillLane = pills.length > 0 ? Math.max(...pills.map((p) => designedFootprint(p).w)) + LANE_GAP : 0;
  const gatewayX = pillLane;
  const boxX = gatewayX + (gateways.length > 0 ? GATEWAY_W + LANE_GAP : 0);
  const serviceX = boxX + PADDING;
  // A wider gutter before the drawn nodes: the relation labels that cross it sit in it, clear of both columns.
  const drawnX = boxX + PADDING * 2 + SERVICE_W + RELATION_GUTTER;

  // Where each service is first reached: by gateway, then row; a pill wired straight to it comes after every gateway.
  const rank = new Map<string, number>();
  let y = LANE_TOP;
  gateways.forEach((g, gi) => {
    positions.set(g.id, { x: gatewayX, y });
    g.data.routes.forEach((r, ri) => {
      const out = edges.find((e) => e.source === g.id && e.sourceHandle === r.id);
      if (out && !rank.has(out.target)) rank.set(out.target, gi * 1000 + ri);
    });
    y += footprintOf(g).h + STACK_GAP;
  });
  for (const e of edges) if (byId.get(e.source)?.type === "hostname" && !rank.has(e.target)) rank.set(e.target, 1e6 + rank.size);
  const order = (id: string) => rank.get(id) ?? Infinity;

  const home = new Map<string, string>();
  for (const g of networks) for (const id of g.memberIds) if (!home.has(id)) home.set(id, g.name);
  const byRank = (a: TopologyNode, b: TopologyNode) => order(a.id) - order(b.id) || a.id.localeCompare(b.id);
  // A box holds its network's services; a service on no network is a group of one, ranked among the boxes.
  const groups = [
    ...networks.map((g) => ({ boxed: true, top: networkBoxTop(g.name, g.machine), members: services.filter((s) => home.get(s.id) === g.name).sort(byRank) })),
    ...services.filter((s) => !home.has(s.id)).map((s) => ({ boxed: false, top: 0, members: [s] })),
  ]
    .filter((g) => g.members.length > 0)
    .sort((a, b) => byRank(a.members[0]!, b.members[0]!));
  y = LANE_TOP;
  for (const group of groups) {
    y += group.top;
    for (const s of group.members) {
      positions.set(s.id, { x: serviceX, y });
      y += footprintOf(s).h + STACK_GAP;
    }
    y += group.boxed ? PADDING - STACK_GAP + BOX_GAP : BOX_GAP - STACK_GAP;
  }

  const beside = (id: string) => {
    const linked = edges.filter((e) => e.source === id || e.target === id).map((e) => (e.source === id ? e.target : e.source));
    return Math.min(...linked.map((other) => positions.get(other)?.y ?? Infinity));
  };
  y = LANE_TOP;
  for (const n of [...drawn].sort((a, b) => beside(a.id) - beside(b.id))) {
    const want = beside(n.id);
    y = Math.max(y, Number.isFinite(want) ? want - EXTERNAL_HEADER : y);
    positions.set(n.id, { x: drawnX, y });
    y += footprintOf(n).h + STACK_GAP;
  }
  return alignPillsToRows(nodes, positions, edges, networks);
};

interface Rect {
  x: number;
  y: number;
  w: number;
  h: number;
}

// Closer than this reads as touching: two cards edge to edge look like one, two boxes like one double line.
const CLEARANCE = 12;

const hit = (a: Rect, b: Rect, clearance = CLEARANCE) =>
  a.x < b.x + b.w + clearance && b.x < a.x + a.w + clearance && a.y < b.y + b.h + clearance && b.y < a.y + a.h + clearance;

// Whether anything (pills included) draws on or touches anything else, so such an arrangement is laid out again.
// ponytail: pairwise O(n²), fine for a canvas of tens of nodes; a sweep line if canvases reach the hundreds.
export const overlaps = (nodes: TopologyNode[], positions: Positions, networks: NetworkGroup[] = [], edges: RelationEdge[] = []): boolean => {
  const placed = alignPillsToRows(nodes, positions, edges, networks);
  const cards = nodes.filter((n) => placed.has(n.id)).map((n) => ({ id: n.id, pill: n.type === "hostname", ...placed.get(n.id)!, ...footprintOf(n) }));
  // Pills stack at their rows' pitch, a few pixels apart by design, so two pills only clash when they truly overlap.
  for (let i = 0; i < cards.length; i++)
    for (let j = i + 1; j < cards.length; j++) if (hit(cards[i]!, cards[j]!, cards[i]!.pill && cards[j]!.pill ? 0 : CLEARANCE)) return true;
  const boxes = buildNetworkRects(networks, nodes.map((n) => ({ ...n, position: placed.get(n.id) ?? n.position }))).map((r) => ({
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

export interface NetworkRect {
  name: string;
  x: number;
  y: number;
  width: number;
  height: number;
  count: number;
  machine?: string;
  // Set while a hover traces routes that none of the box's services are on.
  dim?: boolean;
}

// One box per docker network, wrapped around wherever its members currently sit; nothing about it is stored.
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
        y: bounds.minY - networkBoxTop(group.name, group.machine),
        width: bounds.maxX - bounds.minX + PADDING * 2,
        height: bounds.maxY - bounds.minY + networkBoxTop(group.name, group.machine) + PADDING,
        count: members.length,
        ...(group.machine ? { machine: group.machine } : {}),
      },
    ];
  });

export interface Lane {
  label: string;
  left: number;
  right: number;
  top: number;
  bottom: number;
}

const kindLabel: Partial<Record<TopologyNode["type"], string>> = { hostname: "Hostnames", gateway: "Gateways", service: "Services" };

// Each kind's column as it sits; two kinds sharing one (a dragged card) merge unlabelled, so no header lies.
export const lanesOf = (nodes: TopologyNode[], rects: NetworkRect[]): Lane[] => {
  const spans = new Map<string, Rect & { label: string }>();
  const grow = (key: string, label: string, r: Rect) => {
    const s = spans.get(key);
    if (!s) return spans.set(key, { ...r, label });
    const x = Math.min(s.x, r.x);
    const y = Math.min(s.y, r.y);
    return spans.set(key, { label, x, y, w: Math.max(s.x + s.w, r.x + r.w) - x, h: Math.max(s.y + s.h, r.y + r.h) - y });
  };
  for (const n of nodes) {
    const kind = n.type === "network" ? "external" : n.type;
    grow(kind, kindLabel[n.type] ?? "", { ...n.position, ...footprintOf(n) });
  }
  for (const r of rects) grow("service", "Services", { x: r.x, y: r.y, w: r.width, h: r.height });
  const sorted = [...spans.values()].sort((a, b) => a.x - b.x);
  const merged: Lane[] = [];
  for (const s of sorted) {
    const last = merged.at(-1);
    if (last && s.x < last.right) {
      merged[merged.length - 1] = {
        label: "",
        left: last.left,
        right: Math.max(last.right, s.x + s.w),
        top: Math.min(last.top, s.y),
        bottom: Math.max(last.bottom, s.y + s.h),
      };
      continue;
    }
    merged.push({ label: s.label, left: s.x, right: s.x + s.w, top: s.y, bottom: s.y + s.h });
  }
  return merged;
};

// The camera never fits below this: under it the mono rows drop below 10px on screen and stop reading.
export const MIN_FIT_ZOOM = 0.8;
const CAMERA_PAD = 32;
// Room above a lane's top card for its header.
const HEADER_ROOM = 28;

// The widest run of whole lanes from the left that reads at 0.8 to 1, top-aligned; the rest is a pan away.
export const laneCamera = (lanes: Lane[], paneWidth: number): { x: number; y: number; zoom: number } => {
  if (lanes.length === 0 || paneWidth <= 0) return { x: CAMERA_PAD, y: CAMERA_PAD, zoom: 1 };
  const left = lanes[0]!.left;
  const top = Math.min(...lanes.map((l) => l.top)) - HEADER_ROOM;
  let zoom = MIN_FIT_ZOOM;
  for (let k = lanes.length; k > 0; k--) {
    const fits = (paneWidth - CAMERA_PAD * 2) / (lanes[k - 1]!.right - left);
    if (fits < MIN_FIT_ZOOM) continue;
    zoom = Math.min(1, fits);
    break;
  }
  return { x: CAMERA_PAD - left * zoom, y: CAMERA_PAD - top * zoom, zoom };
};

// The last lane that starts off the pane's left edge (after panning to later lanes), or none.
export const laneBefore = (lanes: Lane[], viewport: { x: number; zoom: number }): Lane | undefined =>
  [...lanes].reverse().find((l) => viewport.x + l.left * viewport.zoom < CAMERA_PAD / 2);

// Pans so that lane's left edge comes into view, keeping the zoom.
export const panBackTo = (lane: Lane, viewport: { y: number; zoom: number }) => ({
  x: CAMERA_PAD - lane.left * viewport.zoom,
  y: viewport.y,
  zoom: viewport.zoom,
});

// Whether the drawing runs on below the pane's bottom edge.
export const drawingBelow = (lanes: Lane[], viewport: { y: number; zoom: number }, paneHeight: number): boolean =>
  lanes.some((l) => viewport.y + l.bottom * viewport.zoom > paneHeight - CAMERA_PAD / 2);

// The first lane that runs past the pane's right edge, or none when every lane is in view.
export const laneBeyond = (lanes: Lane[], viewport: { x: number; zoom: number }, paneWidth: number): Lane | undefined =>
  lanes.find((l) => viewport.x + l.right * viewport.zoom > paneWidth - CAMERA_PAD / 2);

// Pans so that lane's right edge comes into view, keeping the zoom.
export const panTo = (lane: Lane, viewport: { y: number; zoom: number }, paneWidth: number) => ({
  x: paneWidth - CAMERA_PAD - lane.right * viewport.zoom,
  y: viewport.y,
  zoom: viewport.zoom,
});
