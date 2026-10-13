import { describe, expect, it } from "vitest";

import { domainsOf } from "@/models/PermissionLevel";
import type { Role } from "@/models/Role";
import { OWNER_REASON, roleChoices } from "@/models/RoleChoice";

const entry = (domain: string, action: string, label: string) => ({ value: `${domain}:${action}`, label, domain, action, area: "project" as const });

const domains = domainsOf([
  entry("deploys", "read", "Read deploys"),
  entry("deploys", "write", "Start deploys"),
  entry("deploys", "delete", "Delete deploys"),
  entry("runners", "read", "Read runners"),
  entry("runners", "write", "Enroll runners"),
  entry("plays", "read", "Read plays"),
  entry("plays", "run", "Run plays"),
  entry("docs", "read", "Read docs"),
]);

const role = (id: string, permissions: string[], owner = false): Role => ({ id, workspace_id: "ws-1", name: id, permissions, welcome_steps: [], is_owner_role: owner, created_at: "", updated_at: "" });

const owner = role("Owner", [], true);
const dev = role("Developer", ["deploys:read", "plays:read"]);
const lead = role("Lead", ["deploys:read", "deploys:write", "deploys:delete", "runners:read", "runners:write"]);
const wide = role("Wide", ["deploys:delete", "runners:write", "plays:run", "docs:read"]);

describe("roleChoices", () => {
  it("lists the Owner first and never givable, and names at its level each area a role reaches past the viewer", () => {
    const choices = roleChoices([dev, lead, owner, wide], [{ id: "Developer", name: "Developer", is_owner: false }], domains);

    expect(choices.map((choice) => [choice.role.id, choice.givable, choice.reason])).toEqual([
      ["Owner", false, OWNER_REASON],
      ["Developer", true, ""],
      ["Lead", false, "Holds Deploys: Delete and Runners: Write, which you don't hold."],
      ["Wide", false, "Holds Deploys: Delete, Runners: Write and 2 more, which you don't hold."],
    ]);
  });

  it("lets an Owner give every role but the Owner role", () => {
    const choices = roleChoices([owner, lead, wide], [{ id: "Owner", name: "Owner", is_owner: true }], domains);

    expect(choices.map((choice) => choice.givable)).toEqual([false, true, true]);
  });
});
