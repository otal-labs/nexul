import { describe, expect, it } from "vitest";

import { BranchRowSchema, ruleFromRow, rowFromRule, sharesProduction, type BranchRow } from "@/models/BranchRow";

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
    expect(ruleFromRow(row({ overrides: "DATABASE_URL=postgres://qa/app" }))).toEqual({
      pattern: "feature/*",
      docker_network: "qa_default",
      hostname_template: "{branch}.example.com",
      port: 8080,
      overrides: { DATABASE_URL: "postgres://qa/app" },
    });
  });

  it("gives an exact row a name suffix so it deploys its own copy", () => {
    expect(ruleFromRow(row({ pattern: "release/v1.2", hostname: "" }))).toEqual({
      pattern: "release/v1.2",
      docker_network: "qa_default",
      name_suffix: "release-v1-2",
    });
  });

  it("round-trips through rowFromRule", () => {
    expect(rowFromRule(ruleFromRow(row()), "")).toEqual(row());
  });

  it("falls back to the service's port for a rule without one", () => {
    expect(rowFromRule({ pattern: "staging", docker_network: "web_default" }, "3000").port).toBe("3000");
  });
});

describe("BranchRowSchema", () => {
  it.each([
    [{ pattern: "" }, "Branch is required"],
    [{ pattern: "feature/*/x" }, "Use one * at the end, like feature/*"],
    [{ network: "" }, "Choose a network"],
    [{ hostname: "preview.example.com" }, "Put * where the branch goes, like *.example.com"],
    [{ port: "" }, "Set a port under Advanced options to serve a hostname"],
  ])("rejects %o", (change, message) => {
    const result = BranchRowSchema.safeParse(row(change));
    expect(result.success).toBe(false);
    expect(result.error?.issues[0]?.message).toBe(message);
  });

  it("accepts a row without a hostname or port", () => {
    expect(BranchRowSchema.safeParse(row({ hostname: "", port: "" })).success).toBe(true);
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
