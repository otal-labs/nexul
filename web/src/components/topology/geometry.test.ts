// @vitest-environment node
import { describe, expect, it } from "vitest";

import { footerOnOneLine, hostnameLines, routeLines } from "@/components/topology/geometry";
import { ServiceStatus } from "@/models/Topology";

describe("hostnameLines", () => {
  it("keeps a hostname that fits on one line, and wraps a long one only after its dots", () => {
    expect(hostnameLines({ hostname: "sync.example.com" })).toEqual(["sync.example.com"]);
    const lines = hostnameLines({ hostname: "shipment-tracking-webhooks.ap-southeast.warehouse-sync.globex.example.com" });
    expect(lines.length).toBeGreaterThan(1);
    expect(lines.join("")).toBe("shipment-tracking-webhooks.ap-southeast.warehouse-sync.globex.example.com");
    expect(lines.slice(0, -1).every((l) => l.endsWith("."))).toBe(true);
  });

  it("never leaves one label alone on the last line", () => {
    for (const hostname of ["metrics.internal.globex.example.com", "labels.carrier.globex.example.com"]) {
      const lines = hostnameLines({ hostname, tone: ServiceStatus.Failed });
      expect(lines.at(-1)!.split(".").length).toBeGreaterThan(1);
    }
  });
});

describe("routeLines", () => {
  const route = (service: string, address?: string) => ({ id: "r", hostname: "h", service, port: 3000, ...(address ? { address } : {}) });

  it("keeps the name and address on one line when both fit", () => {
    expect(routeLines(route("site", "172.20.2.6"))).toEqual({ name: 1, address: false });
  });

  it("drops the address to its own line before it splits a name that fits alone", () => {
    expect(routeLines(route("atlas-web-feature-lanes", "172.20.2.9"))).toEqual({ name: 1, address: true });
  });

  it("breaks only a name wider than the whole row", () => {
    const lines = routeLines(route("sync-worker-reconciliation-and-backfill-for-every-carrier-region", "10.42.0.11"));
    expect(lines.name).toBeGreaterThan(1);
    expect(lines.address).toBe(true);
  });
});

describe("footerOnOneLine", () => {
  const gateway = (networks: string[]) => ({
    service_id: "g",
    name: "g",
    status: "healthy" as const,
    kind: "proxy" as const,
    routes: [],
    target: "wh-ap-southeast-02",
    networks,
  });

  it("joins the machine and networks on one line when they fit, else gives each its own line", () => {
    expect(footerOnOneLine(gateway(["previews"]))).toBe(true);
    expect(footerOnOneLine(gateway(["warehouse-sync_default", "warehouse-sync_observability-and-metrics-internal"]))).toBe(false);
  });
});
