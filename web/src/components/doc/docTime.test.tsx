import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { formatUpdatedAgo } from "@/components/doc/docTime";

const NOW = new Date("2026-08-14T12:00:00Z");

beforeEach(() => {
  vi.useFakeTimers();
  vi.setSystemTime(NOW);
});

afterEach(() => {
  vi.useRealTimers();
});

describe("formatUpdatedAgo", () => {
  it.each([
    ["under a minute", 30_000, "just now"],
    ["minutes", 5 * 60_000, "5m ago"],
    ["hours", 3 * 3_600_000, "3h ago"],
    ["days", 2 * 86_400_000, "2d ago"],
    ["an hour boundary", 60 * 60_000, "1h ago"],
    ["a day boundary", 24 * 3_600_000, "1d ago"],
    ["future timestamps", -60_000, "just now"],
  ] as const)("renders %s", (_label, elapsedMs, expected) => {
    expect(formatUpdatedAgo(new Date(NOW.getTime() - elapsedMs).toISOString())).toBe(expected);
  });

  it("falls back to a short absolute date past a week", () => {
    const iso = new Date(NOW.getTime() - 8 * 86_400_000).toISOString();
    const expected = new Date(iso).toLocaleDateString(undefined, { month: "short", day: "numeric" });
    expect(formatUpdatedAgo(iso)).toBe(expected);
  });

  it("returns an empty string for an unparseable date", () => {
    expect(formatUpdatedAgo("not-a-date")).toBe("");
  });
});
