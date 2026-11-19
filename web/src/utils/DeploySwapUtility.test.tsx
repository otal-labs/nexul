// @vitest-environment node
import { describe, expect, it } from "vitest";

import { ContainerStatus, type Container } from "@/models/Stack";
import { deployModes, SwapBlocker } from "@/utils/DeploySwapUtility";

const service = (name: string, extra: Partial<Container> = {}): Container => ({
  id: name,
  stack_id: "s1",
  name,
  declared: {},
  status: ContainerStatus.Running,
  ...extra,
});

const routed = (id: string) => (id === "web" || id === "api" || id === "proxy" || id === "fresh" ? [`${id}.example.com`] : []);

describe("deployModes", () => {
  it("swaps a routed service with a healthcheck and says why each other one restarts", () => {
    const modes = deployModes(
      [
        service("web", { healthcheck: true, ports: ["80/tcp"] }),
        service("api", { healthcheck: false }),
        service("proxy", { healthcheck: true, ports: ["8080:80/tcp"] }),
        service("db", { healthcheck: true }),
        service("worker"),
      ],
      routed,
    );

    expect(modes.swapped).toEqual(["web"]);
    expect(modes.restarted).toEqual([
      { reason: SwapBlocker.NoHealthcheck, names: ["api"] },
      { reason: SwapBlocker.HostPort, names: ["proxy"] },
      { reason: SwapBlocker.NoHostname, names: ["db", "worker"] },
    ]);
  });

  it("leaves out a routed service no runner has reported a healthcheck for yet", () => {
    expect(deployModes([service("fresh")], routed)).toEqual({ swapped: [], restarted: [] });
  });
});
