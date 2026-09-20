import { afterEach, describe, expect, it, vi } from "vitest";

import { daysAgo, formatDurationMs, formatRelativeTime } from "@/utils/TimeUtility";

describe("formatRelativeTime", () => {
  afterEach(() => {
    vi.useRealTimers();
  });

  it("returns the raw string when the timestamp is unparseable", () => {
    expect(formatRelativeTime("not-a-date")).toBe("not-a-date");
  });

  it("says just now for sub-minute and future timestamps", () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-08-14T12:00:00Z"));
    expect(formatRelativeTime("2026-08-14T11:59:30Z")).toBe("just now");
    expect(formatRelativeTime("2026-08-14T12:05:00Z")).toBe("just now");
  });

  it("formats minutes, hours, and days", () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-08-14T12:00:00Z"));
    expect(formatRelativeTime("2026-08-14T11:55:00Z")).toBe("5m ago");
    expect(formatRelativeTime("2026-08-14T09:00:00Z")).toBe("3h ago");
    expect(formatRelativeTime("2026-08-11T12:00:00Z")).toBe("3d ago");
  });
});

describe("daysAgo", () => {
  afterEach(() => {
    vi.useRealTimers();
  });

  it("returns NaN for an unparseable timestamp", () => {
    expect(daysAgo("not-a-date")).toBeNaN();
  });

  it("returns whole days elapsed", () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-08-14T12:00:00Z"));
    expect(daysAgo("2026-08-14T09:00:00Z")).toBe(0);
    expect(daysAgo("2026-08-06T12:00:00Z")).toBe(8);
  });
});

describe("formatDurationMs", () => {
  it("formats sub-second durations in ms", () => {
    expect(formatDurationMs(340)).toBe("340ms");
  });

  it("formats one second and above in seconds", () => {
    expect(formatDurationMs(2300)).toBe("2.3s");
    expect(formatDurationMs(1000)).toBe("1.0s");
  });
});
