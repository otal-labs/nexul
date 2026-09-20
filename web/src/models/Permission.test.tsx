import { describe, expect, it } from "vitest";

import { hasPermission, SetPermissionsSchema } from "@/models/Permission";

describe("SetPermissionsSchema", () => {
  it("accepts a bulk set", () => {
    const data = SetPermissionsSchema.parse({
      doc_ids: ["doc-1", "doc-2"],
      user_ids: ["u1"],
      actions: ["docs:read", "docs:write"],
      grant: true,
    });
    expect("doc_ids" in data).toBe(true);
    if ("doc_ids" in data) expect(data.doc_ids).toHaveLength(2);
    expect(data.grant).toBe(true);
  });

  it("rejects empty doc/user/action lists", () => {
    expect(
      SetPermissionsSchema.safeParse({ doc_ids: [], user_ids: ["u1"], actions: ["docs:read"], grant: true }).success,
    ).toBe(false);
    expect(
      SetPermissionsSchema.safeParse({ doc_ids: ["doc-1"], user_ids: [], actions: ["docs:read"], grant: true })
        .success,
    ).toBe(false);
    expect(
      SetPermissionsSchema.safeParse({ doc_ids: ["doc-1"], user_ids: ["u1"], actions: [], grant: true }).success,
    ).toBe(false);
  });

  it("accepts a play exclusion (ticket 21)", () => {
    const data = SetPermissionsSchema.parse({
      resource_type: "play",
      resource_ids: ["play-1"],
      user_ids: ["u1"],
      actions: ["plays:run"],
      grant: false,
    });
    expect("resource_ids" in data).toBe(true);
    if ("resource_ids" in data) expect(data.resource_ids).toEqual(["play-1"]);
    expect(data.grant).toBe(false);
  });

  it("rejects an empty resource_ids list on a play exclusion", () => {
    expect(
      SetPermissionsSchema.safeParse({
        resource_type: "play",
        resource_ids: [],
        user_ids: ["u1"],
        actions: ["plays:run"],
        grant: false,
      }).success,
    ).toBe(false);
  });
});

describe("hasPermission", () => {
  it("checks the raw string value against the permissions list", () => {
    expect(hasPermission(["docs:read", "projects:write"], "projects:write")).toBe(true);
    expect(hasPermission(["docs:read"], "projects:write")).toBe(false);
    expect(hasPermission(undefined, "projects:write")).toBe(false);
  });
});
