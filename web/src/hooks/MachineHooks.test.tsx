import { describe, expect, it } from "vitest";

import { machineFollower } from "@/hooks/MachineHooks";
import { stackFollower } from "@/hooks/StackHooks";
import { followFrame, isStale, seeded } from "@/test/followFrame";

describe("a rule on a machine", () => {
  const frame = { resource_type: "machine", resource_id: "m-1" };

  it("refetches the machines the picker offers", async () => {
    const client = seeded([[["machines", "deployable"], []]]);
    await followFrame(machineFollower, "access.grant.changed", frame, client);
    expect(isStale(client, ["machines", "deployable"])).toBe(true);
  });

  it("refetches the machine list, the runners and the queue, which show only the machines the viewer sees", async () => {
    const client = seeded([[["machines"], []], [["runners"], []], [["runnerQueue"], []]]);
    await followFrame(machineFollower, "access.grant.changed", frame, client);
    expect([isStale(client, ["machines"]), isStale(client, ["runners"]), isStale(client, ["runnerQueue"])]).toEqual([true, true, true]);
  });

  it("refetches the stack lists, whose instance stacks follow the machine", async () => {
    const client = seeded([[["getStacks"], []]]);
    await followFrame(stackFollower, "access.grant.changed", frame, client);
    expect(isStale(client, ["getStacks"])).toBe(true);
  });

  it("leaves the machines alone for a rule on anything else", async () => {
    const client = seeded([[["machines", "deployable"], []], [["runners"], []]]);
    await followFrame(machineFollower, "access.grant.changed", { resource_type: "stack", resource_id: "s-1" }, client);
    expect([isStale(client, ["machines", "deployable"]), isStale(client, ["runners"])]).toEqual([false, false]);
  });
});
