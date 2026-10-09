import { describe, expect, it } from "vitest";

import { roleFollower } from "@/hooks/RoleHooks";
import { followFrame, isStale, seeded } from "@/test/followFrame";

describe("the role follower", () => {
  it("refetches the roles of the workspace whose role changed, and no other", async () => {
    const client = seeded([
      [["getWorkspaceRoles", "ws-1"], []],
      [["getWorkspaceRoles", "ws-2"], []],
    ]);
    await followFrame(roleFollower, "role.updated", { role_id: "r-1", workspace_id: "ws-2" }, client);
    expect([isStale(client, ["getWorkspaceRoles", "ws-1"]), isStale(client, ["getWorkspaceRoles", "ws-2"])]).toEqual([false, true]);
  });
});
