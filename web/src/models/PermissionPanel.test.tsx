import { describe, expect, it } from "vitest";

import type { PermissionSource } from "@/models/Permission";
import { changesOf, sentenceParts } from "@/models/PermissionPanel";

const source = (over: Partial<PermissionSource>): PermissionSource => ({
  kind: "rule", entity_type: "project", entity_id: "p-1", target_type: "role", target_id: "r-1", state: "deny",
  entity_name: "Storefront", target_name: "Client", sentence: "Denied on project Storefront for role Client", ...over,
});

describe("sentenceParts", () => {
  it("sets the entity and the role it names in medium weight", () => {
    expect(sentenceParts(source({}))).toEqual([
      { text: "Denied on project ", strong: false },
      { text: "Storefront", strong: true },
      { text: " for role ", strong: false },
      { text: "Client", strong: true },
    ]);
  });

  it("leaves a person's name and the workspace plain", () => {
    const parts = sentenceParts(source({ entity_type: "doc", entity_name: "Pricing", target_type: "person", target_name: "bob", sentence: "Allowed on doc Pricing for bob" }));
    expect(parts.filter((p) => p.strong).map((p) => p.text)).toEqual(["Pricing"]);
    expect(sentenceParts(source({ kind: "owner", entity_type: "workspace", entity_name: "Acme", target_type: "person", sentence: "Owner of Acme" }))).toEqual([
      { text: "Owner of Acme", strong: false },
    ]);
  });
});

describe("changesOf", () => {
  it("sends only what differs from the saved rules, one change per person and state", () => {
    const grants = [{ resource_type: "doc", resource_id: "d", user_id: "bob", allow: ["docs:read"], deny: [], created_at: "", updated_at: "" }];
    const held = { "bob docs:read": "allow", "bob docs:write": "deny", "bob docs:delete": "deny", "lena docs:read": "fallback" } as const;
    expect(changesOf(held, grants)).toEqual([{ userId: "bob", state: "deny", actions: ["docs:write", "docs:delete"] }]);
  });
});
