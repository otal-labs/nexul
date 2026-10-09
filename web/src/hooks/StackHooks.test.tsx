import { describe, expect, it } from "vitest";

import { stackFollower } from "@/hooks/StackHooks";
import { followFrame, isStale, seeded } from "@/test/followFrame";

const histories = () =>
  seeded([
    [["getStackDeploys", "s-1"], [{ id: "d-1", stack_id: "s-1", status: "building" }]],
    [["getStackDeploys", "s-2"], [{ id: "d-2", stack_id: "s-2", status: "healthy" }]],
  ]);

describe("the stack follower", () => {
  it("refetches only the deploy history holding the deploy a frame names", async () => {
    const client = histories();
    await followFrame(stackFollower, "deploy.updated", { id: "d-1", status: "healthy" }, client);
    expect([isStale(client, ["getStackDeploys", "s-1"]), isStale(client, ["getStackDeploys", "s-2"])]).toEqual([true, false]);
  });

  it("refetches every deploy history for a deploy none holds, since the frame names no stack", async () => {
    const client = histories();
    await followFrame(stackFollower, "deploy.updated", { id: "d-9", status: "pending" }, client);
    expect([isStale(client, ["getStackDeploys", "s-1"]), isStale(client, ["getStackDeploys", "s-2"])]).toEqual([true, true]);
  });
});
