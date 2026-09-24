import { describe, expect, it } from "vitest";

import type { Container, Stack } from "@/models/Stack";
import { machineNetworks, networkLabel } from "@/utils/MachineNetworkUtility";

const stack = (id: string, machine: string): Stack => ({
  id,
  project_id: "p-1",
  name: id,
  slug: id,
  machine,
  strategy: "compose",
  managed: true,
  created_at: "",
  updated_at: "",
});

const container = (stackId: string, name: string, networks: string[]): Container => ({
  id: `${stackId}-${name}`,
  stack_id: stackId,
  name,
  declared: {},
  status: "running",
  networks: networks.map((n) => ({ name: n })),
});

describe("machineNetworks", () => {
  const stacks = [stack("qa", "prod"), stack("web", "prod"), stack("elsewhere", "laptop")];
  const containers = [
    container("qa", "redis", ["qa_default"]),
    container("qa", "postgres", ["qa_default", "bridge"]),
    container("elsewhere", "mysql", ["other_default"]),
  ];

  it("lists each network on the machine with what runs on it, skipping docker's built-in ones", () => {
    expect(machineNetworks(stacks, containers, "prod", "web_default", new Set(["web_default"]))).toEqual([
      { name: "qa_default", services: ["postgres", "redis"], hasGateway: false },
      { name: "web_default", services: [], hasGateway: true },
    ]);
  });

  it("labels a network with its services, or just its name when nothing was seen on it", () => {
    expect(networkLabel({ name: "qa_default", services: ["postgres", "redis"], hasGateway: true })).toBe(
      "qa_default — postgres, redis",
    );
    expect(networkLabel({ name: "web_default", services: [], hasGateway: true })).toBe("web_default");
  });

  it("labels a network without a gateway as unable to serve hostnames", () => {
    expect(networkLabel({ name: "qa_default", services: ["postgres", "redis"], hasGateway: false })).toBe(
      "qa_default — postgres, redis · no gateway, hostnames unavailable",
    );
  });
});
