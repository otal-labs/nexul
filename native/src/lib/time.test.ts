import { formatCalendarTime, formatDayLabel, formatRelativeTime } from "@/lib/time";

const now = Date.parse("2026-09-28T12:00:00Z");

test.each([
  ["2026-09-28T11:59:30Z", "now"],
  ["2026-09-28T11:55:00Z", "5m"],
  ["2026-09-28T09:00:00Z", "3h"],
  ["2026-09-26T12:00:00Z", "2d"],
])("%s reads as %s", (iso, expected) => {
  expect(formatRelativeTime(iso, now)).toBe(expected);
});

describe("formatCalendarTime", () => {
  // Zoneless inputs read in the test's own zone, so these hold wherever the suite runs.
  const lateEvening = Date.parse("2026-10-06T23:55:00");
  const justAfterMidnight = Date.parse("2026-10-07T00:05:00");

  test("the same calendar day is today", () => {
    expect(formatCalendarTime("2026-10-06T21:11:00", lateEvening)).toBe("Today at 21:11");
  });

  test("ten minutes earlier across midnight is yesterday", () => {
    expect(formatCalendarTime("2026-10-06T23:50:00", justAfterMidnight)).toBe("Yesterday at 23:50");
  });
});

describe("formatDayLabel", () => {
  const morning = Date.parse("2026-10-07T09:00:00");

  test.each([
    ["2026-10-07T08:00:00", "Today"],
    ["2026-10-06T23:59:00", "Yesterday"],
    ["2026-09-21T12:00:00", "Mon 21 Sept"],
    ["2025-12-31T12:00:00", "Wed 31 Dec 2025"],
  ])("%s reads as %s", (iso, label) => {
    expect(formatDayLabel(iso, morning)).toBe(label);
  });
});
