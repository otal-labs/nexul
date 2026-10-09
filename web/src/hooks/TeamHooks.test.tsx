import { describe, expect, it } from "vitest";

import { teamFollower } from "@/hooks/TeamHooks";
import { followFrame, isStale, seeded } from "@/test/followFrame";

describe("the team follower", () => {
  it.each(["account.presence_changed", "workspace.member.updated", "access.grant.changed"])("refetches the Team on %s", async (topic) => {
    const client = seeded([[["getTeam"], { people: [] }]]);
    await followFrame(teamFollower, topic, { user_id: "u-2", workspace_id: "ws-1" }, client);
    expect(isStale(client, ["getTeam"])).toBe(true);
  });
});
