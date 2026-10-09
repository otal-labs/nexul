import { describe, expect, it } from "vitest";

import { GATEWAY_HEADER_H, GATEWAY_ROW_H } from "@/components/topology/GatewayNode";
import { RelationKind, ServiceStatus, type RelationEdge, type TopologyNode } from "@/models/Topology";
import { alignPillsToRows, footprintOf, layoutGraph, overlaps } from "@/utils/TopologyLayout";

const gateway: TopologyNode = {
  id: "svc-cf",
  type: "gateway",
  position: { x: 0, y: 0 },
  data: {
    service_id: "svc-cf",
    name: "cloudflared-local",
    status: ServiceStatus.Healthy,
    kind: "tunnel",
    networks: ["nexul_default"],
    routes: [
      { id: "e1", hostname: "llmtested.example.com", service: "hello-api", port: 3000 },
      { id: "e2", hostname: "example.com", service: "web", port: 80 },
    ],
  },
};
const pill = (id: string, hostname: string): TopologyNode => ({ id, type: "hostname", position: { x: 0, y: 0 }, data: { hostname } });

describe("alignPillsToRows", () => {
  it("places each pill level with its row and right-aligns them to one column", () => {
    const pills = [pill("host-e1", "llmtested.example.com"), pill("host-e2", "example.com")];
    const out = alignPillsToRows([gateway, ...pills], new Map([["svc-cf", { x: 500, y: 100 }]]));
    const first = out.get("host-e1")!;
    const second = out.get("host-e2")!;
    expect(first.y).toBe(100 + GATEWAY_HEADER_H + GATEWAY_ROW_H / 2 - 20);
    expect(second.y).toBe(first.y + GATEWAY_ROW_H);
    // Right edges line up: x + width is the same for both, regardless of hostname length.
    expect(first.x + footprintOf(pills[0]!).w).toBe(second.x + footprintOf(pills[1]!).w);
    expect(first.x + footprintOf(pills[0]!).w).toBeLessThan(500);
  });

  it("leaves nodes without a gateway untouched", () => {
    const positions = new Map([["svc-api", { x: 1, y: 2 }]]);
    expect(alignPillsToRows([pill("host-x", "x.dev")], positions)).toEqual(positions);
  });
});

describe("footprintOf", () => {
  it("prefers measured sizes and grows a gateway by its rows", () => {
    expect(footprintOf({ ...gateway, measured: { width: 300, height: 180 } }).h).toBe(180);
    expect(footprintOf(gateway).h).toBe(GATEWAY_HEADER_H + 2 * GATEWAY_ROW_H + 37);
    expect(footprintOf(pill("h", "a-much-longer-hostname.example.com")).w).toBeGreaterThan(footprintOf(pill("h", "a.dev")).w);
  });
});

const service = (id: string, x = 0, y = 0): TopologyNode => ({
  id,
  type: "service",
  position: { x, y },
  data: { service_id: id, name: id, status: ServiceStatus.Running },
});
const directWire = (id: string, source: string, target: string): RelationEdge => ({
  id,
  source,
  target,
  type: "relation",
  data: { kind: RelationKind.ConnectsTo, label: ":3000" },
});

describe("alignPillsToRows without a gateway on the canvas", () => {
  it("puts a pill wired straight to a service left of that service's network box, level with it", () => {
    const web = service("svc-web");
    const api = service("svc-api");
    const pill1 = pill("host-a", "atlas.example.com");
    const pill2 = pill("host-b", "api.atlas.example.com");
    const positions = new Map([
      ["svc-web", { x: 600, y: 100 }],
      ["svc-api", { x: 500, y: 300 }],
    ]);
    const out = alignPillsToRows(
      [web, api, pill1, pill2],
      positions,
      [directWire("w1", "host-a", "svc-web"), directWire("w2", "host-b", "svc-api")],
      [{ name: "atlas_default", memberIds: ["svc-web", "svc-api"] }],
    );
    const a = out.get("host-a")!;
    const b = out.get("host-b")!;
    // Both clear the box's left edge (the leftmost member less its padding), so neither sits inside it.
    expect(a.x + footprintOf(pill1).w).toBeLessThan(500 - 24);
    expect(b.x + footprintOf(pill2).w).toBeLessThan(500 - 24);
    expect(a.y + 20).toBeCloseTo(100 + footprintOf(web).h / 2);
    expect(b.y + 20).toBeCloseTo(300 + footprintOf(api).h / 2);
  });
});

describe("overlaps", () => {
  it("flags cards stored edge to edge, as positions saved before a card grew are", () => {
    const nodes = [service("site"), service("search")];
    expect(overlaps(nodes, new Map([["site", { x: 36, y: 162 }], ["search", { x: 36, y: 269 }]]))).toBe(true);
    expect(overlaps(nodes, new Map([["site", { x: 36, y: 162 }], ["search", { x: 36, y: 162 + 160 }]]))).toBe(false);
  });

  it("flags two network boxes that touch, but not boxes sharing a member", () => {
    const nodes = [service("a"), service("b"), service("c")];
    const positions = new Map([["a", { x: 0, y: 0 }], ["b", { x: 0, y: 190 }], ["c", { x: 400, y: 0 }]]);
    expect(overlaps(nodes, positions, [{ name: "one", memberIds: ["a"] }, { name: "two", memberIds: ["b"] }])).toBe(true);
    expect(overlaps(nodes, positions, [{ name: "one", memberIds: ["a", "b"] }, { name: "two", memberIds: ["b"] }])).toBe(false);
  });
});

describe("layoutGraph", () => {
  it("lays out a service on no network that has a wire leaving it", async () => {
    const nodes = [service("worker"), service("db")];
    const edges: RelationEdge[] = [{ id: "e", source: "worker", target: "db", type: "relation", data: { kind: RelationKind.DependsOn } }];
    const positions = await layoutGraph(nodes, edges);
    expect(positions.get("worker")!.x).toBeLessThan(positions.get("db")!.x);
  });
});
