import { describe, expect, it } from "vitest";

import { stackFollower } from "@/hooks/StackHooks";
import { followFrame, isStale, seeded } from "@/test/followFrame";

const histories = () =>
  seeded([
    [["getStackDeploys", "s-1"], [{ id: "d-1", stack_id: "s-1", status: "building" }]],
    [["getStackDeploys", "s-2"], [{ id: "d-2", stack_id: "s-2", status: "healthy" }]],
  ]);

describe("the stack follower", () => {
  const stale = (client: ReturnType<typeof histories>) => [isStale(client, ["getStackDeploys", "s-1"]), isStale(client, ["getStackDeploys", "s-2"])];

  it("refetches only the history of the stack a frame names when its deploy's status moves", async () => {
    const client = histories();
    await followFrame(stackFollower, "deploy.updated", { id: "d-1", status: "healthy", stack_id: "s-1" }, client);
    expect(stale(client)).toEqual([true, false]);
  });

  it("refetches the named stack's history for a deploy it does not hold yet", async () => {
    const client = histories();
    await followFrame(stackFollower, "deploy.updated", { id: "d-9", status: "pending", stack_id: "s-2" }, client);
    expect(stale(client)).toEqual([false, true]);
  });

  it("leaves the history alone for a log batch, which keeps the deploy's status", async () => {
    const client = histories();
    await followFrame(stackFollower, "deploy.updated", { id: "d-1", status: "building", stack_id: "s-1" }, client);
    expect(stale(client)).toEqual([false, false]);
  });

  it("refetches the read of the stack an update frame names, so another editor's environment shows, and no other stack's", async () => {
    const client = seeded([
      [["getStack", "s-1"], { id: "s-1" }],
      [["getStack", "s-2"], { id: "s-2" }],
    ]);
    await followFrame(stackFollower, "service.updated", { stack: { id: "s-1" } }, client);
    expect([isStale(client, ["getStack", "s-1"]), isStale(client, ["getStack", "s-2"])]).toEqual([true, false]);
  });

  it.each(["service.container_stopped", "service.container_recovered", "stack.error_rate_crossed", "stack.error_rate_recovered"])(
    "refetches only the read of the stack %s names, since that read carries its open outages and error-rate episode",
    async (topic) => {
      const client = seeded([
        [["getStack", "s-1"], { id: "s-1" }],
        [["getStack", "s-2"], { id: "s-2" }],
        [["getStackServices", "s-1"], []],
      ]);
      await followFrame(stackFollower, topic, { stack_id: "s-1", service: "web", outage_id: "o-1" }, client);
      expect([isStale(client, ["getStack", "s-1"]), isStale(client, ["getStack", "s-2"]), isStale(client, ["getStackServices", "s-1"])]).toEqual([true, false, false]);
    },
  );

  it("refetches the stack lists and the named stack's reads when its rule changes, and nothing for another entity's", async () => {
    const client = seeded([
      [["getStacks", "all"], []],
      [["getStack", "s-1"], { id: "s-1" }],
      [["getStack", "s-2"], { id: "s-2" }],
      [["getStackDeploys", "s-1"], []],
    ]);
    const marks = () => [["getStacks", "all"], ["getStack", "s-1"], ["getStack", "s-2"], ["getStackDeploys", "s-1"]].map((key) => isStale(client, key));
    await followFrame(stackFollower, "access.grant.changed", { resource_type: "doc", resource_id: "s-1" }, client);
    expect(marks()).toEqual([false, false, false, false]);
    await followFrame(stackFollower, "access.grant.changed", { resource_type: "stack", resource_id: "s-1" }, client);
    expect(marks()).toEqual([true, true, false, true]);
  });
});
