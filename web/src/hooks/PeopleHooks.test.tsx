import { describe, expect, it } from "vitest";

import { peopleFollower } from "@/hooks/PeopleHooks";
import { followFrame, isStale, seeded } from "@/test/followFrame";

const person = (id: string) => ({ user_id: id, login: id, display_name: id, avatar_url: "" });

const directories = () =>
  seeded([
    [["getWorkspacePeople", "ws-1"], [person("u-2")]],
    [["getWorkspacePeople", "ws-2"], [person("u-3")]],
    [["getProjectPeople", "p-1"], [person("u-2")]],
    [["getProjectPeople", "p-2"], [person("u-3")]],
  ]);

const keys = [["getWorkspacePeople", "ws-1"], ["getWorkspacePeople", "ws-2"], ["getProjectPeople", "p-1"], ["getProjectPeople", "p-2"]];

describe("the people follower", () => {
  it("refetches the lists that show someone who changed their name or picture", async () => {
    const client = directories();
    await followFrame(peopleFollower, "account.profile_updated", { account_id: "u-2" }, client);
    expect(keys.map((key) => isStale(client, key))).toEqual([true, false, true, false]);
  });

  it("refetches the directory of the workspace someone joined, and the project pickers a grant opens", async () => {
    const client = directories();
    await followFrame(peopleFollower, "workspace.member.added", { user_id: "u-9", workspace_id: "ws-2" }, client);
    await followFrame(peopleFollower, "access.grant.changed", { user_id: "u-9", resource_type: "project", resource_id: "p-1" }, client);
    expect(keys.map((key) => isStale(client, key))).toEqual([false, true, true, false]);
  });
});
