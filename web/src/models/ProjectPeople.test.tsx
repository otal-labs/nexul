import { describe, expect, it } from "vitest";

import type { PermissionGrant } from "@/models/Permission";
import { projectPeopleRules, rulesHere } from "@/models/ProjectPeople";

const rule = (over: Partial<PermissionGrant>): PermissionGrant => ({
  resource_type: "project", resource_id: "p-1", target: "person", user_id: "", allow: [], deny: [], created_at: "2026-10-01", updated_at: "", ...over,
});

describe("projectPeopleRules", () => {
  it("lists the roles it lets in, the people named, and the people kept out, oldest first", () => {
    const grants = [
      rule({ target: "role", role_id: "r-client", allow: ["projects:open"], created_at: "2026-10-03" }),
      rule({ target: "role", role_id: "r-dev", allow: ["projects:open"], created_at: "2026-10-02" }),
      rule({ target: "role", role_id: "r-ops", deny: ["tickets:write"] }),
      rule({ user_id: "lena", allow: ["tickets:write"] }),
      rule({ user_id: "bob", deny: ["projects:open"] }),
    ];
    const got = projectPeopleRules(grants);
    expect(got.roles.map((g) => g.role_id)).toEqual(["r-dev", "r-client"]);
    expect(got.people.map((g) => g.user_id)).toEqual(["lena"]);
    expect(got.keptOut.map((g) => g.user_id)).toEqual(["bob"]);
  });

  it("counts a target's rules beyond being in the project", () => {
    expect(rulesHere(rule({ allow: ["projects:open", "tickets:write"], deny: ["docs:delete"] }))).toBe(2);
    expect(rulesHere(rule({ allow: ["projects:open"] }))).toBe(0);
  });
});
