// @vitest-environment node
import { describe, expect, it } from "vitest";

import type { PermissionInfo, PermissionSource } from "@/models/Permission";
import { changesOf, groupActions, parseTarget, sentenceParts, targetKey } from "@/models/PermissionPanel";

const source = (over: Partial<PermissionSource>): PermissionSource => ({
  kind: "rule", entity_type: "project", entity_id: "p-1", target_type: "role", target_id: "r-1", state: "deny",
  entity_name: "Storefront", target_name: "Client", sentence: "Denied on project Storefront for role Client", ...over,
});

describe("sentenceParts", () => {
  it("marks the entity and the role it names, each for its own link", () => {
    expect(sentenceParts(source({}))).toEqual([
      { text: "Denied on project " },
      { text: "Storefront", names: "entity" },
      { text: " for role " },
      { text: "Client", names: "role" },
    ]);
  });

  it("leaves a person's name and the workspace plain", () => {
    const parts = sentenceParts(source({ entity_type: "doc", entity_name: "Pricing", target_type: "person", target_name: "bob", sentence: "Allowed on doc Pricing for bob" }));
    expect(parts.filter((p) => p.names).map((p) => p.text)).toEqual(["Pricing"]);
    expect(sentenceParts(source({ kind: "owner", entity_type: "workspace", entity_name: "Acme", target_type: "person", sentence: "Owner of Acme" }))).toEqual([
      { text: "Owner of Acme" },
    ]);
  });
});

describe("changesOf", () => {
  it("sends only what differs from the saved rules, one change per target and state", () => {
    const grants = [
      { resource_type: "doc", resource_id: "d", target: "person" as const, user_id: "bob", allow: ["docs:read"], deny: [], created_at: "", updated_at: "" },
      { resource_type: "doc", resource_id: "d", target: "role" as const, user_id: "", role_id: "r-dev", allow: [], deny: ["docs:write"], created_at: "", updated_at: "" },
    ];
    const held = {
      "person:bob docs:read": "allow", "person:bob docs:write": "deny", "person:bob docs:delete": "deny", "person:lena docs:read": "fallback",
      "role:r-dev docs:write": "deny", "role:r-dev docs:read": "allow",
    } as const;
    expect(changesOf(held, grants)).toEqual([
      { target: "person:bob", state: "deny", actions: ["docs:write", "docs:delete"] },
      { target: "role:r-dev", state: "allow", actions: ["docs:read"] },
    ]);
  });
});

describe("groupActions", () => {
  it("leads a project with its own group and names the rest by their domain", () => {
    const catalog: PermissionInfo[] = [{ value: "tickets:read", label: "Read tickets", domain: "tickets", action: "read", area: "project" }];
    expect(groupActions(["projects:open", "tickets:read", "projects:read", "permissions:write"], catalog)).toEqual([
      { name: "Project", actions: ["projects:open", "projects:read"] },
      { name: "Tickets", actions: ["tickets:read"] },
      { name: "Access", actions: ["permissions:write"] },
    ]);
  });
});

it("round-trips a target key", () => {
  expect(parseTarget(targetKey("role", "r:1"))).toEqual({ kind: "role", id: "r:1" });
  expect(parseTarget("person:bob")).toEqual({ kind: "person", id: "bob" });
});
