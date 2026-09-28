import { formatRelativeTime } from "@/lib/time";

const now = Date.parse("2026-09-28T12:00:00Z");

test.each([
  ["2026-09-28T11:59:30Z", "now"],
  ["2026-09-28T11:55:00Z", "5m"],
  ["2026-09-28T09:00:00Z", "3h"],
  ["2026-09-26T12:00:00Z", "2d"],
])("%s reads as %s", (iso, expected) => {
  expect(formatRelativeTime(iso, now)).toBe(expected);
});
