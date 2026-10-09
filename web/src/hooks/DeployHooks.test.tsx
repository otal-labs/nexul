import { describe, expect, it } from "vitest";

import { deployFollower } from "@/hooks/DeployHooks";
import { followFrame, isStale, seeded } from "@/test/followFrame";

describe("the deploy follower", () => {
  it("refetches the deploy a frame names and its log, and leaves another open deploy alone", async () => {
    const client = seeded([
      [["getDeploy", "d-1"], {}],
      [["getDeployLog", "d-1"], []],
      [["getDeploy", "d-2"], {}],
      [["getDeployLog", "d-2"], []],
    ]);
    await followFrame(deployFollower, "deploy.updated", { id: "d-2", status: "building" }, client);
    const keys = [["getDeploy", "d-1"], ["getDeployLog", "d-1"], ["getDeploy", "d-2"], ["getDeployLog", "d-2"]];
    expect(keys.map((key) => isStale(client, key))).toEqual([false, false, true, true]);
  });
});
