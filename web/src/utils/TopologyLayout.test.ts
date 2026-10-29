// @vitest-environment node
import { describe, expect, it } from "vitest";

import { deriveWiring } from "@/components/topology/wiring";
import { pillHeight, routeRowOffsets } from "@/components/topology/geometry";
import type { Exposure, Gateway } from "@/models/DNS";
import { ContainerStatus, DeployStrategy, type Container, type Stack } from "@/models/Stack";
import { RelationKind, ServiceStatus, type RelationEdge, type TopologyNode } from "@/models/Topology";
import {
  alignPillsToRows,
  buildNetworkRects,
  footprintOf,
  laneBefore,
  laneBeyond,
  laneCamera,
  laneLayout,
  panBackTo,
  panTo,
  lanesOf,
  MIN_FIT_ZOOM,
  overlaps,
} from "@/utils/TopologyLayout";

const service = (id: string, x = 0, y = 0): TopologyNode => ({
  id,
  type: "service",
  position: { x, y },
  data: { service_id: id, name: id, status: ServiceStatus.Running },
});
const container = (id: string, network?: string): Container => ({
  id,
  stack_id: "s",
  name: id,
  declared: {},
  status: ContainerStatus.Healthy,
  ...(network ? { networks: [{ name: network }] } : {}),
});
const stack: Stack = { id: "s", project_id: "p", name: "s", slug: "s", machine: "box", strategy: DeployStrategy.Run, managed: true, created_at: "", updated_at: "" };
const gateway: Gateway = { id: "gw", kind: "proxy", docker_network: "edge", service_id: "traefik", zone_id: "z", zone: "example.com", created_at: "", updated_at: "" };
const expose = (id: string, hostname: string, target: string): Exposure => ({
  id,
  gateway_id: "gw",
  hostname,
  service_id: target,
  service: target,
  port: 3000,
  zone_id: "z",
  zone: "example.com",
  created_at: "",
  updated_at: "",
});

// A proxy whose first row reaches the "previews" box and whose second reaches "docs", plus a hand-drawn database.
const scene = (hostnames = ["lanes.example.com", "docs.example.com", "hotfix.example.com"]) => {
  const stored = ["traefik", "lanes", "site", "hotfix", "search"].map((id) => service(id));
  const db: TopologyNode = { id: "db", type: "external", position: { x: 0, y: 0 }, data: { name: "db", label: "database" } };
  const w = deriveWiring(
    [...stored, db],
    [container("traefik"), container("lanes", "previews"), container("site", "docs"), container("hotfix", "previews"), container("search", "docs")],
    [stack],
    [gateway],
    [expose("e1", hostnames[0]!, "lanes"), expose("e2", hostnames[1]!, "site"), expose("e3", hostnames[2]!, "hotfix")],
  );
  const relation: RelationEdge = { id: "r", source: "site", target: "db", type: "relation", data: { kind: RelationKind.ConnectsTo } };
  const nodes = [...w.hosts, ...w.nodes];
  const edges = [...w.edges, relation];
  return { w, nodes, edges, positions: laneLayout(nodes, edges, w.networks) };
};

describe("laneLayout", () => {
  it("puts hostnames, gateways, services and drawn nodes in columns left to right, each kind at one x", () => {
    const { positions } = scene();
    const x = (id: string) => positions.get(id)!.x;
    expect(new Set(["lanes", "site", "hotfix", "search"].map(x)).size).toBe(1);
    expect(x("host-e1") + footprintOf(scene().nodes.find((n) => n.id === "host-e1")!).w).toBeLessThan(x("traefik"));
    expect(x("traefik")).toBeLessThan(x("lanes"));
    expect(x("lanes")).toBeLessThan(x("db"));
  });

  it("orders the services lane by the gateway rows that reach it, so the wires out fan without crossing", () => {
    const { positions } = scene();
    const y = (id: string) => positions.get(id)!.y;
    // Row one reaches lanes (previews), row two site (docs): the previews box comes first even though docs sorts first.
    expect(y("lanes")).toBeLessThan(y("hotfix"));
    expect(y("hotfix")).toBeLessThan(y("site"));
    expect(y("site")).toBeLessThan(y("search"));
  });

  it("ranks a service on no network among the boxes, so the row that reaches it points across, not down", () => {
    const stored = ["traefik", "loose", "site"].map((id) => service(id));
    const w = deriveWiring(stored, [container("traefik"), container("loose"), container("site", "docs")], [stack], [gateway], [
      expose("e1", "a.example.com", "loose"),
      expose("e2", "b.example.com", "site"),
    ]);
    const positions = laneLayout([...w.hosts, ...w.nodes], w.edges, w.networks);
    expect(positions.get("loose")!.y).toBeLessThan(positions.get("site")!.y);
  });

  it("draws nothing over anything else", () => {
    const { nodes, positions, w, edges } = scene(["a-very-long-hostname-for-the-first-route.preview.example.com", "docs.example.com", "hotfix.example.com"]);
    expect(overlaps(nodes, positions, w.networks, edges)).toBe(false);
  });
});

describe("alignPillsToRows", () => {
  it("keeps each pill level with its row, also when a long hostname wraps and its row grows to match", () => {
    const { nodes, positions } = scene(["shipment-tracking-webhooks.ap-southeast.warehouse-sync.globex.example.com", "docs.example.com", "hotfix.example.com"]);
    const proxy = nodes.find((n) => n.type === "gateway")!;
    if (proxy.type !== "gateway") throw new Error("no gateway");
    const heights = proxy.data.rowHeights!;
    const offsets = routeRowOffsets(proxy.data, heights);
    proxy.data.routes.forEach((r, i) => {
      const pill = nodes.find((n) => n.id === `host-${r.id}`)!;
      if (pill.type !== "hostname") throw new Error("no pill");
      const centre = positions.get(pill.id)!.y + pillHeight(pill.data) / 2;
      expect(centre).toBe(positions.get(proxy.id)!.y + offsets[i]! + heights[i]! / 2);
    });
    expect(heights[0]).toBeGreaterThan(heights[1]!);
    // The card's footprint counts the grown rows, so the next card down starts below it.
    const tunnelFree = footprintOf(proxy).h;
    expect(tunnelFree).toBeGreaterThanOrEqual(heights.reduce((a, b) => a + b, 0) + 56);
  });

  it("puts a pill wired straight to a service (no gateway on the canvas) left of that service's box, level with it", () => {
    const web = service("svc-web");
    const pill: TopologyNode = { id: "host-a", type: "hostname", position: { x: 0, y: 0 }, data: { hostname: "atlas.example.com" } };
    const wire: RelationEdge = { id: "w", source: "host-a", target: "svc-web", type: "relation", data: { kind: RelationKind.ConnectsTo, label: ":3000" } };
    const out = alignPillsToRows([web, pill], new Map([["svc-web", { x: 600, y: 100 }]]), [wire], [{ name: "atlas_default", memberIds: ["svc-web"] }]);
    const at = out.get("host-a")!;
    expect(at.x + footprintOf(pill).w).toBeLessThan(600 - 24);
    expect(at.y + footprintOf(pill).h / 2).toBe(100 + footprintOf(web).h / 2);
  });
});

describe("overlaps", () => {
  it("flags cards stored edge to edge, as positions saved before a card grew are", () => {
    const nodes = [service("site"), service("search")];
    const { h } = footprintOf(nodes[0]!);
    expect(overlaps(nodes, new Map([["site", { x: 36, y: 162 }], ["search", { x: 36, y: 162 + h }]]))).toBe(true);
    expect(overlaps(nodes, new Map([["site", { x: 36, y: 162 }], ["search", { x: 36, y: 162 + h + 40 }]]))).toBe(false);
  });

  it("flags two network boxes that touch, but not boxes sharing a member", () => {
    const nodes = [service("a"), service("b"), service("c")];
    const { h } = footprintOf(nodes[0]!);
    const positions = new Map([["a", { x: 0, y: 0 }], ["b", { x: 0, y: h + 80 }], ["c", { x: 400, y: 0 }]]);
    expect(overlaps(nodes, positions, [{ name: "one", memberIds: ["a"] }, { name: "two", memberIds: ["b"] }])).toBe(true);
    expect(overlaps(nodes, positions, [{ name: "one", memberIds: ["a", "b"] }, { name: "two", memberIds: ["b"] }])).toBe(false);
  });

  it("counts the pills, so a stored gateway that puts its pills on a network box is laid out again", () => {
    const { nodes, positions, w, edges } = scene();
    const moved = new Map(positions);
    // The gateway moves right of the services: its pills, always a gap left of it, now land on the boxes.
    moved.set("traefik", { x: positions.get("lanes")!.x + 400, y: positions.get("lanes")!.y });
    expect(overlaps(nodes, moved, w.networks, edges)).toBe(true);
  });
});

describe("lanesOf and laneCamera", () => {
  const lanes = () => {
    const { nodes, positions, w } = scene();
    const placed = nodes.map((n) => ({ ...n, position: positions.get(n.id)! }));
    return lanesOf(placed, buildNetworkRects(w.networks, placed));
  };

  it("names each column once, over its leftmost card, and leaves the drawn nodes' column to their own labels", () => {
    expect(lanes().map((l) => l.label)).toEqual(["Hostnames", "Gateways", "Services", ""]);
  });

  it("drops the header where two kinds share a column, so none labels the wrong one", () => {
    const proxy: TopologyNode = {
      id: "b",
      type: "gateway",
      position: { x: 0, y: 0 },
      data: { service_id: "b", name: "b", status: ServiceStatus.Running, kind: "proxy", routes: [], networks: [] },
    };
    const placed = [service("a", 0, 0), proxy];
    expect(lanesOf(placed, []).map((l) => l.label)).toEqual([""]);
  });

  it("fits every whole lane when they read at 0.8 or more, else only the lanes from the left that do", () => {
    const all = lanes();
    const span = all.at(-1)!.right - all[0]!.left;
    expect(laneCamera(all, span + 64).zoom).toBe(1);
    const narrow = laneCamera(all, 700);
    expect(narrow.zoom).toBeGreaterThanOrEqual(MIN_FIT_ZOOM);
    const shownRight = (700 - narrow.x) / narrow.zoom;
    // Some lane ends inside the view and the next one starts past it: no lane is cut by the fit itself.
    expect(all.some((l) => l.right <= shownRight)).toBe(true);
    expect(laneCamera(all, 200).zoom).toBe(MIN_FIT_ZOOM);
  });
});

describe("laneBeyond and laneBefore", () => {
  const lanes = [
    { label: "Hostnames", left: 0, right: 300, top: 0, bottom: 100 },
    { label: "Gateways", left: 356, right: 716, top: 0, bottom: 100 },
    { label: "Services", left: 772, right: 1108, top: 0, bottom: 100 },
  ];

  it("pans More to the first lane past the right edge and Back to the last one off the left, and back again", () => {
    const start = { x: 32, y: 0, zoom: 1 };
    expect(laneBefore(lanes, start)).toBeUndefined();
    const next = laneBeyond(lanes, start, 800)!;
    expect(next.label).toBe("Services");
    const panned = panTo(next, start, 800);
    expect(laneBefore(lanes, panned)!.label).toBe("Hostnames");
    const back = panBackTo(laneBefore(lanes, panned)!, panned);
    expect(back.x).toBe(start.x);
    expect(laneBefore(lanes, back)).toBeUndefined();
  });
});
