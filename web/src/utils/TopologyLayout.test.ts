import { describe, expect, it } from "vitest";

import { GATEWAY_HEADER_H, GATEWAY_ROW_H } from "@/components/topology/GatewayNode";
import { ServiceStatus, type TopologyNode } from "@/models/Topology";
import { alignPillsToRows, footprintOf } from "@/utils/TopologyLayout";

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
