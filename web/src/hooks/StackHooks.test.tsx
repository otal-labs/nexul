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
});
