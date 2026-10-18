// @vitest-environment node
import { describe, expect, it } from "vitest";

import { getExplanationKey, getGrantsKey, permissionRulesFollower } from "@/hooks/PermissionHooks";
import { followFrame, isStale, seeded } from "@/test/followFrame";

describe("permissionRulesFollower", () => {
  it("rereads the rules and answers of the doc a rule changed on, and no other", async () => {
    const client = seeded([
      [[getGrantsKey, "doc", "doc-1"], []],
      [[getExplanationKey, "doc", "doc-1", ""], {}],
      [[getExplanationKey, "doc", "doc-1", "bob"], {}],
      [[getGrantsKey, "doc", "doc-2"], []],
      [[getGrantsKey, "play", "doc-1"], []],
    ]);

    await followFrame(permissionRulesFollower, "access.grant.changed", { resource_type: "doc", resource_id: "doc-1", user_id: "bob" }, client);

    expect(isStale(client, [getGrantsKey, "doc", "doc-1"])).toBe(true);
    expect(isStale(client, [getExplanationKey, "doc", "doc-1", ""])).toBe(true);
    expect(isStale(client, [getExplanationKey, "doc", "doc-1", "bob"])).toBe(true);
    expect(isStale(client, [getGrantsKey, "doc", "doc-2"])).toBe(false);
    expect(isStale(client, [getGrantsKey, "play", "doc-1"])).toBe(false);
  });

  it("rereads every open stack's answers when a machine's rule changed, and no doc's", async () => {
    const client = seeded([
      [[getExplanationKey, "stack", "stack-1", ""], {}],
      [[getExplanationKey, "stack", "stack-2", ""], {}],
      [[getExplanationKey, "doc", "doc-1", ""], {}],
    ]);

    await followFrame(permissionRulesFollower, "access.grant.changed", { resource_type: "machine", resource_id: "machine-1", role_id: "role-1" }, client);

    expect(isStale(client, [getExplanationKey, "stack", "stack-1", ""])).toBe(true);
    expect(isStale(client, [getExplanationKey, "stack", "stack-2", ""])).toBe(true);
    expect(isStale(client, [getExplanationKey, "doc", "doc-1", ""])).toBe(false);
  });
});
