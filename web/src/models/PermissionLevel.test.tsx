import { describe, expect, it } from "vitest";

import type { PermissionInfo } from "@/models/Permission";
import {
  accessSummary,
  changeSummary,
  domainsIn,
  domainsOf,
  projectLevelLabel,
  levelOf,
  summarize,
  uniformLevelOf,
  withExtra,
  withLevel,
  withLevelEverywhere,
} from "@/models/PermissionLevel";

const catalog: PermissionInfo[] = [
  { value: "docs:read", label: "Read docs", domain: "docs", action: "read", area: "workspace" },
  { value: "docs:write", label: "Create and update docs", domain: "docs", action: "write", area: "workspace" },
  { value: "docs:delete", label: "Delete docs", domain: "docs", action: "delete", area: "workspace" },
  { value: "docs:thread", label: "See doc threads", domain: "docs", action: "thread", area: "workspace" },
  { value: "audit:read", label: "Read audit log", domain: "audit", action: "read", area: "workspace" },
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
      { value: "tickets:read", label: "Read tickets", domain: "tickets", action: "read", area: "workspace" },
      { value: "tickets:write", label: "Create and update tickets", domain: "tickets", action: "write", area: "workspace" },
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

  describe("project areas", () => {
    const areaCatalog: PermissionInfo[] = [
      { value: "tickets:read", label: "Read tickets", domain: "tickets", action: "read", area: "project" },
      { value: "tickets:write", label: "Create and update tickets", domain: "tickets", action: "write", area: "project" },
      { value: "chat:read", label: "Read chat", domain: "chat", action: "read", area: "workspace" },
      { value: "repos:read", label: "Read pull requests", domain: "repos", action: "read", area: "project" },
    ];
    const areas = domainsIn(areaCatalog, ["project"]);

    it("takes only the domains of the given areas", () => {
      expect(areas.map((area) => area.domain)).toEqual(["tickets", "repos"]);
    });

    it("reads a project as its shared level, or Custom with a summary of each area once they differ", () => {
      expect(projectLevelLabel(areas, ["tickets:read", "repos:read"])).toBe("Read");
      expect(accessSummary(areas, ["tickets:read", "repos:read"])).toBe("every area Read");
      expect(projectLevelLabel(areas, ["tickets:read", "tickets:write"])).toBe("Custom");
      expect(accessSummary(areas, ["tickets:read", "tickets:write"])).toBe("tickets Write");
      expect(accessSummary(areas, [])).toBe("None");
    });

    it("names what one change moved, for the toast", () => {
      expect(changeSummary(areas, [], ["tickets:read", "tickets:write"])).toBe("tickets → Write");
      expect(changeSummary(areas, [], ["tickets:read", "repos:read"])).toBe("every area → Read");
    });
  });
});
