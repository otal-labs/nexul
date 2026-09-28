import { describe, expect, it } from "vitest";

import type { PermissionInfo } from "@/models/Permission";
import {
  domainsOf,
  levelOf,
  summarize,
  uniformLevelOf,
  withExtra,
  withLevel,
  withLevelEverywhere,
} from "@/models/PermissionLevel";

const catalog: PermissionInfo[] = [
  { value: "docs:read", label: "Read docs", domain: "docs", action: "read" },
  { value: "docs:write", label: "Create and update docs", domain: "docs", action: "write" },
  { value: "docs:delete", label: "Delete docs", domain: "docs", action: "delete" },
  { value: "docs:thread", label: "See doc threads", domain: "docs", action: "thread" },
  { value: "audit:read", label: "Read audit log", domain: "audit", action: "read" },
];

const domains = domainsOf(catalog);
const docs = domains[0]!;
const audit = domains[1]!;

describe("PermissionLevel", () => {
  it("splits a domain into its read/write/delete ladder and its extra verbs", () => {
    expect(docs.name).toBe("Docs");
    expect(docs.levels.map((entry) => entry.action)).toEqual(["read", "write", "delete"]);
    expect(docs.extras.map((entry) => entry.action)).toEqual(["thread"]);
    expect(audit.name).toBe("Audit log");
  });

  it("reads a stored set that skips a rung at its highest action", () => {
    expect(levelOf(docs, ["docs:delete"])).toBe(3);
    expect(levelOf(docs, [])).toBe(0);
  });

  it("grants every rung up to the chosen level and leaves other domains alone", () => {
    expect(withLevel(["audit:read", "docs:delete"], docs, 2)).toEqual(["audit:read", "docs:read", "docs:write"]);
  });

  it("None clears the domain's extra verbs as well", () => {
    expect(withLevel(["docs:read", "docs:thread", "audit:read"], docs, 0)).toEqual(["audit:read"]);
  });

  it("setting every domain caps each at its own top rung, and reads back as that level", () => {
    const value = withLevelEverywhere(["docs:thread"], [docs, audit], 3);
    expect(value).toEqual(["docs:thread", "docs:read", "docs:write", "docs:delete", "audit:read"]);
    expect(uniformLevelOf([docs, audit], value)).toBe(3);
    expect(uniformLevelOf([docs, audit], ["docs:read"])).toBeUndefined();
  });

  it("toggles an extra verb without touching the level", () => {
    const thread = docs.extras[0]!;
    expect(withExtra(["docs:read"], thread, true)).toEqual(["docs:read", "docs:thread"]);
    expect(withExtra(["docs:read", "docs:thread"], thread, false)).toEqual(["docs:read"]);
  });

  it("summarises one line per touched domain when no level is shared by most", () => {
    const tickets = domainsOf([
      { value: "tickets:read", label: "Read tickets", domain: "tickets", action: "read" },
      { value: "tickets:write", label: "Create and update tickets", domain: "tickets", action: "write" },
    ])[0]!;
    expect(summarize([docs, audit, tickets], ["docs:read", "docs:write", "docs:delete", "docs:thread", "tickets:read"])).toEqual([
      "Docs · Delete + Thread",
      "Tickets · Read",
    ]);
    expect(summarize([docs, audit], [])).toEqual([]);
  });

  it("folds the level most domains share into one line and lists the exceptions", () => {
    expect(summarize([docs, audit], ["docs:read", "docs:thread", "audit:read"])).toEqual([
      "Docs · Read + Thread",
      "Everything else · Read",
    ]);
    expect(summarize([docs, audit], ["docs:read", "audit:read"])).toEqual(["Every domain · Read"]);
  });
});
