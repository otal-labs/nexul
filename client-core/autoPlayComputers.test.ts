import { describe, expect, it } from "vitest";

import { autoPlayOwner, autoPlayPlaces, reaches, ruleMode, withMode, withValues, ruleValues, type AutoPlayWhose } from "@nexul/client-core/autoPlayComputers";

const own = [
  { id: "mini", labels: ["docker"], derived_labels: ["linux"] },
  { id: "desk", labels: [], derived_labels: ["windows"] },
  { id: "off", labels: [], derived_labels: ["linux"], auto_plays_off: true },
  { id: "stop", derived_labels: ["linux"], plays_off: true },
  { id: "new", unfinished: true },
];

const pinned = (field: "developer" | "tester", value: string, match: "all" | "any" = "all"): AutoPlayWhose["conditions"] => ({
  match: "all",
  groups: [{ match, rules: [{ field, op: "is", values: [value] }] }],
});
const base = { run_as: "", conditions: { match: "all" as const, groups: [] } };

describe("reaches", () => {
  it.each([
    ["no rule keeps every ready computer", undefined, ["mini", "desk"]],
    ["labelled keeps a computer with one of the labels, own or derived", { labels_in: ["docker", "windows"] }, ["mini", "desk"]],
    ["not labelled drops a computer carrying the label", { labels_out: ["windows"] }, ["mini"]],
    ["only these keeps the named ids", { ids: ["desk"] }, ["desk"]],
    ["a rule never brings back a computer that takes no auto plays", { ids: ["off", "stop", "new"] }, []],
  ])("%s", (_, rule, expected) => {
    expect(own.filter((c) => reaches(c, rule)).map((c) => c.id)).toEqual(expected);
  });
});

describe("autoPlayOwner", () => {
  it.each([
    ["a person auto play runs as its person", { ...base, run_on: "person" as const, run_as: "sam" }, "sam"],
    ["a draft run as the caller is the caller", { ...base, run_on: "person" as const, run_as_me: true }, "me"],
    ["a person nobody picked has no owner", { ...base, run_on: "person" as const }, undefined],
    ["a developer condition pins the developer", { ...base, run_on: "developer" as const, conditions: pinned("developer", "me") }, "me"],
    ["a tester pin does not pin the developer", { ...base, run_on: "developer" as const, conditions: pinned("tester", "me") }, undefined],
    ["a pin inside an any group of one rule still pins", { ...base, run_on: "developer" as const, conditions: pinned("developer", "me", "any") }, "me"],
    ["whoever caused it is known only at run time", { ...base, run_on: "causer" as const, conditions: pinned("developer", "me") }, undefined],
    [
      "an any match across two groups pins nothing",
      { ...base, run_on: "developer" as const, conditions: { match: "any" as const, groups: [pinned("developer", "me").groups[0]!, pinned("tester", "me").groups[0]!] } },
      undefined,
    ],
  ])("%s", (_, autoPlay, expected) => {
    expect(autoPlayOwner(autoPlay, "me")).toBe(expected);
  });
});

describe("autoPlayPlaces", () => {
  it("lists the viewer's computers, struck out where the rule drops them", () => {
    const places = autoPlayPlaces({ ...base, run_on: "person", run_as: "me" }, { labels_out: ["windows"] }, "me", own);
    expect(places).toMatchObject({ kind: "computers" });
    if (places.kind !== "computers") return;
    expect(places.computers.filter((c) => c.reached).map((c) => c.computer.id)).toEqual(["mini"]);
    expect(places.computers).toHaveLength(own.length);
  });

  it("names another person without listing computers that are not the viewer's", () => {
    expect(autoPlayPlaces({ ...base, run_on: "person", run_as: "sam" }, undefined, "me", own)).toEqual({ kind: "person", userId: "sam" });
  });

  it("names the role when the person is only known at run time", () => {
    expect(autoPlayPlaces({ ...base, run_on: "causer" }, undefined, "me", own)).toEqual({ kind: "role", role: "causer" });
    expect(autoPlayPlaces({ ...base, run_on: "tester" }, undefined, "me", own)).toEqual({ kind: "role", role: "tester" });
    expect(autoPlayPlaces({ ...base, run_on: "person" }, undefined, "me", own)).toEqual({ kind: "role", role: "nobody" });
  });
});

describe("the editor's choice over a rule", () => {
  it.each([
    [{}, "any"],
    [{ labels_in: [], labels_out: [], ids: [] }, "any"],
    [{ labels_in: ["gpu"] }, "labelled"],
    [{ labels_out: ["windows"] }, "not_labelled"],
    [{ ids: ["c1"], labels_in: ["gpu"] }, "only"],
  ])("reads %j as %s", (rule, mode) => {
    expect(ruleMode(rule)).toBe(mode);
  });

  it("changes only the picked part, so a rule with both label lists keeps the other", () => {
    expect(withValues({ labels_in: ["gpu"], labels_out: ["windows"] }, "labelled", ["docker"])).toEqual({ labels_in: ["docker"], labels_out: ["windows"] });
  });

  it("starts a fresh rule when the choice changes", () => {
    expect(withMode("not_labelled")).toEqual({ labels_out: [] });
    expect(ruleValues(withMode("only"), "only")).toEqual([]);
  });
});
