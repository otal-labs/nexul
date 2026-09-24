import { describe, expect, it } from "vitest";

import {
  branchRowsFormSchema,
  duplicateHostnameMessage,
  ruleFromRow,
  rowFromRule,
  sharesProduction,
  type BranchRow,
} from "@/models/BranchRow";

const row = (overrides: Partial<BranchRow> = {}): BranchRow => ({
  pattern: "feature/*",
  hostname: "*.example.com",
  network: "qa_default",
  port: "8080",
  overrides: "",
  ...overrides,
});

describe("ruleFromRow", () => {
  it("turns a wildcard row into a wildcard rule with a {branch} hostname template", () => {
    expect(ruleFromRow(row({ overrides: "DATABASE_URL=postgres://qa/app" }), true)).toEqual({
      pattern: "feature/*",
      docker_network: "qa_default",
      hostname_template: "{branch}.example.com",
      port: 8080,
      overrides: { DATABASE_URL: "postgres://qa/app" },
    });
  });

  it("gives an exact row a name suffix so it deploys its own copy", () => {
    expect(ruleFromRow(row({ pattern: "release/v1.2", hostname: "" }), true)).toEqual({
      pattern: "release/v1.2",
      docker_network: "qa_default",
      name_suffix: "release-v1-2",
    });
  });

  it("round-trips through rowFromRule", () => {
    expect(rowFromRule(ruleFromRow(row(), true), "")).toEqual(row());
  });

  it("drops the hostname on a network no gateway serves", () => {
    expect(ruleFromRow(row(), false)).toEqual({ pattern: "feature/*", docker_network: "qa_default" });
  });

  it("falls back to the service's port for a rule without one", () => {
    expect(rowFromRule({ pattern: "staging", docker_network: "web_default" }, "3000").port).toBe("3000");
  });
});

describe("branchRowsFormSchema", () => {
  const schema = branchRowsFormSchema(new Set(["qa_default"]));
  const firstMessage = (rows: BranchRow[]) =>
    schema.safeParse({ deployDefault: true, rows }).error?.issues[0]?.message;

  it.each([
    [{ pattern: "" }, "Branch is required"],
    [{ pattern: "feature/*/x" }, "Use one * at the end, like feature/*"],
    [{ network: "" }, "Choose a network"],
    [{ hostname: "preview.example.com" }, "Put * where the branch goes, like *.example.com"],
    [{ port: "" }, "Set a port under Advanced options to serve a hostname"],
  ])("rejects %o", (change, message) => {
    expect(firstMessage([row(change)])).toBe(message);
  });

  it("accepts a row without a hostname or port", () => {
    expect(firstMessage([row({ hostname: "", port: "" })])).toBeUndefined();
  });

  it("refuses a second row with the same hostname pattern, on the second row", () => {
    const result = schema.safeParse({ deployDefault: true, rows: [row(), row({ pattern: "bugfix/*" })] });
    expect(result.error?.issues).toEqual([
      expect.objectContaining({ path: ["rows", 1, "hostname"], message: duplicateHostnameMessage }),
    ]);
  });

  it("ignores the hostname of a row whose network has no gateway", () => {
    expect(firstMessage([row({ network: "web_default", hostname: "preview.example.com", port: "" })])).toBeUndefined();
    expect(firstMessage([row(), row({ pattern: "bugfix/*", network: "web_default" })])).toBeUndefined();
  });
});

describe("sharesProduction", () => {
  it("is true on production's network with no overrides", () => {
    expect(sharesProduction(row({ network: "web_default" }), "web_default")).toBe(true);
  });

  it("is false once an override replaces a value", () => {
    expect(sharesProduction(row({ network: "web_default", overrides: "DATABASE_URL=x" }), "web_default")).toBe(false);
  });

  it("is false on another network", () => {
    expect(sharesProduction(row(), "web_default")).toBe(false);
  });
});
