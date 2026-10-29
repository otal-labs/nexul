// @vitest-environment node
import { describe, expect, it } from "vitest";

import type { HistoryVersionSummary } from "@/models/History";
import { cappedCount, keptCounts, restoredLabel, versionSize } from "@/utils/HistoryVersionUtility";

const summary = (extra: Partial<HistoryVersionSummary> = {}): HistoryVersionSummary => ({
  version: 1,
  created_at: "2026-10-01T10:00:00Z",
  name: "",
  editor_ids: [],
  via: "",
  current: false,
  ...extra,
});

describe("cappedCount", () => {
  it.each([
    [0, "0"],
    [999, "999"],
    [1000, "1.0k"],
    [1234, "1.2k"],
    [12_499, "12k"],
  ])("writes %d as %s, never more than three digits", (n, want) => {
    expect(cappedCount(n)).toBe(want);
  });
});

describe("versionSize", () => {
  it("gives the exact figures the row's name and tooltip read from", () => {
    expect(versionSize(summary({ words_added: 38, words_removed: 21 }))).toEqual({ added: 38, removed: 21, exact: "38 words added, 21 removed" });
  });

  it.each([
    ["the oldest kept version, which has none", { words_added: null, words_removed: null }],
    ["a server that sends none", {}],
    ["a save that changed no words", { words_added: 0, words_removed: 0 }],
  ])("shows no size for %s", (_, extra) => {
    expect(versionSize(summary(extra))).toBeUndefined();
  });
});

describe("restoredLabel", () => {
  const kept = summary({ version: 2, created_at: "2026-10-02T10:00:00Z" });

  it("names the restored version by its time while it is kept, and by its number once a prune dropped it", () => {
    expect(restoredLabel(summary({ version: 4, restored_from: 2 }), [kept])).toMatch(/^Restored from \d{1,2} Oct \d{2}:00$/);
    expect(restoredLabel(summary({ version: 4, restored_from: 1 }), [kept])).toBe("Restored from version 1");
  });

  it("says nothing on a version that is not a restore", () => {
    expect(restoredLabel(summary({ restored_from: null }), [kept])).toBe("");
    expect(restoredLabel(summary(), [kept])).toBe("");
  });
});

describe("keptCounts", () => {
  it("counts everything kept and the named versions among it, and keeps the cap rule for the tooltip", () => {
    const versions = [summary(), summary(), summary({ name: "Signed off" })];

    expect(keptCounts(versions)).toEqual({ short: "3 versions · 1 named", long: "Only the last 10 unnamed versions are kept, plus named ones." });
    expect(keptCounts(versions.slice(0, 2)).short).toBe("2 versions");
    expect(keptCounts(versions, "all")).toEqual({ short: "3 versions", long: "Every version is kept." });
  });
});
