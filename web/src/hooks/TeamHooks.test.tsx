// @vitest-environment node
import { describe, expect, it } from "vitest";

import { teamFollower } from "@/hooks/TeamHooks";
import { followFrame, isStale, seeded } from "@/test/followFrame";

describe("the team follower", () => {
  it.each(["account.presence_changed", "workspace.member.updated", "access.grant.changed"])("refetches the Team on %s", async (topic) => {
    const client = seeded([[["getTeam"], { people: [] }]]);
    await followFrame(teamFollower, topic, { user_id: "u-2", workspace_id: "ws-1" }, client);
    expect(isStale(client, ["getTeam"])).toBe(true);
  });

  it("refetches the Team when the Welcome wizard progress of someone on it changes, and only then", async () => {
    const holdingBob = seeded([[["getTeam"], { people: [{ id: "u-bob" }] }]]);
    await followFrame(teamFollower, "welcome.progress_changed", { user_id: "u-bob" }, holdingBob);
    expect(isStale(holdingBob, ["getTeam"])).toBe(true);

    const withoutBob = seeded([[["getTeam"], { people: [{ id: "u-lena" }] }]]);
    await followFrame(teamFollower, "welcome.progress_changed", { user_id: "u-bob" }, withoutBob);
    expect(isStale(withoutBob, ["getTeam"])).toBe(false);
  });
});
