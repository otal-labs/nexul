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

describe("a machine that was deleted", () => {
  const machine = (id: string) => ({ id, name: id });

  it("leaves the machine list at once and refetches what hung off it", async () => {
    const client = seeded([
      [["machines"], [machine("m-1"), machine("m-2")]],
      [["machines", "deployable"], [machine("m-1")]],
      [["runners"], []],
      [["runnerQueue"], []],
      [["runnerEnrollments"], []],
    ]);
    await followFrame(machineFollower, "machine.deleted", { machine_id: "m-1", name: "m-1", user_ids: [] }, client);
    expect(client.getQueryData(["machines"])).toEqual([machine("m-2")]);
    expect(isStale(client, ["machines", "deployable"])).toBe(true);
    expect([isStale(client, ["runners"]), isStale(client, ["runnerQueue"]), isStale(client, ["runnerEnrollments"])]).toEqual([true, true, true]);
  });
});
