// @vitest-environment node
import { afterEach, describe, expect, it, vi } from "vitest";

import { daysAgo, formatCalendarTime, formatDurationMs, formatRelativeTime, formatVersionTime } from "@/utils/TimeUtility";

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

describe("formatCalendarTime", () => {
  afterEach(() => {
    vi.useRealTimers();
  });

  it("names today and yesterday by the reader's calendar day, else the date", () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date(2026, 9, 6, 0, 5));
    expect(formatCalendarTime(new Date(2026, 9, 6, 0, 1).toISOString())).toBe("Today at 00:01");
    expect(formatCalendarTime(new Date(2026, 9, 5, 21, 11).toISOString())).toBe("Yesterday at 21:11");
    expect(formatCalendarTime(new Date(2026, 9, 4, 21, 11).toISOString())).toBe(`${new Date(2026, 9, 4).toLocaleDateString()} 21:11`);
  });

  it("returns the raw string when the timestamp is unparseable", () => {
    expect(formatCalendarTime("soon")).toBe("soon");
  });
});

describe("formatVersionTime", () => {
  afterEach(() => {
    vi.useRealTimers();
  });

  it.each([
    ["today", new Date(2026, 9, 6, 0, 1), "Today 00:01"],
    ["yesterday", new Date(2026, 9, 5, 21, 11), "Yesterday 21:11"],
  ])("reads %s as the calendar word", (_, when, want) => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date(2026, 9, 6, 0, 5));
    expect(formatVersionTime(when.toISOString())).toBe(want);
  });

  it("gives an older day as the short date and the clock, with the year only when it is not this one", () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date(2026, 9, 6, 0, 5));
    const thisYear = formatVersionTime(new Date(2026, 8, 25, 14, 23).toISOString());
    expect(thisYear).toMatch(/25 .*14:23$|.* 25 14:23$/);
    expect(thisYear).not.toContain("2026");
    expect(formatVersionTime(new Date(2025, 9, 25, 14, 23).toISOString())).toContain("2025");
  });
});
