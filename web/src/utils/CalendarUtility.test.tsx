// @vitest-environment node
import { describe, expect, it } from "vitest";

import { dateInput, daysLater, endOfDayInstant, instantOf, overlapsEarlier, periodTitle, sameDay, timeInput } from "@/utils/CalendarUtility";

describe("the form's day and time", () => {
  it("round-trips a local day and time through the instant they name", () => {
    const instant = instantOf("2026-11-02", "09:05");
    expect(dateInput(instant)).toBe("2026-11-02");
    expect(timeInput(instant)).toBe("09:05");
    expect(new Date(instant).getTime()).toBe(new Date(2026, 10, 2, 9, 5).getTime());
  });

  it("ends a series on the last second of its day", () => {
    const end = new Date(endOfDayInstant("2026-12-31"));
    expect([end.getFullYear(), end.getMonth(), end.getDate(), end.getHours(), end.getMinutes(), end.getSeconds()]).toEqual([2026, 11, 31, 23, 59, 59]);
  });
});

describe("periodTitle", () => {
  it("names a month by the anchor and a week by its first and last day", () => {
    const anchor = new Date(2026, 9, 24);
    expect(periodTitle("month", anchor, [])).toMatch(/2026/);
    const days = [new Date(2026, 9, 19), new Date(2026, 9, 25)];
    expect(periodTitle("week", anchor, days)).toBe(`${new Intl.DateTimeFormat(undefined, { day: "numeric", month: "short" }).format(days[0])} – ${new Intl.DateTimeFormat(undefined, { day: "numeric", month: "short" }).format(days[1])}`);
  });

  it("names a week with no days as nothing", () => {
    expect(periodTitle("week", new Date(), [])).toBe("");
  });
});

describe("sameDay", () => {
  it("compares the calendar day, not the instant", () => {
    expect(sameDay(new Date(2026, 9, 24, 1), new Date(2026, 9, 24, 23))).toBe(true);
    expect(sameDay(new Date(2026, 9, 24, 23), new Date(2026, 9, 25, 0))).toBe(false);
  });
});

describe("overlapsEarlier", () => {
  const span = (from: number, to: number) => ({ starts_at: new Date(2026, 9, 27, from).toISOString(), ends_at: new Date(2026, 9, 27, to).toISOString() });

  it("marks a meeting that starts inside any earlier one, not only the one before it", () => {
    expect(overlapsEarlier([span(10, 12), { ...span(10, 11), ends_at: new Date(2026, 9, 27, 10, 15).toISOString() }, span(11, 13), span(13, 14)])).toEqual([false, true, true, false]);
  });
});

describe("daysLater", () => {
  it("counts the midnights a meeting runs past, and none for one that ends at midnight sharp", () => {
    expect(daysLater(new Date(2026, 9, 27, 23).toISOString(), new Date(2026, 9, 28, 1).toISOString())).toBe(1);
    expect(daysLater(new Date(2026, 9, 27, 23).toISOString(), new Date(2026, 9, 28, 0).toISOString())).toBe(0);
    expect(daysLater(new Date(2026, 9, 27, 9).toISOString(), new Date(2026, 9, 27, 10).toISOString())).toBe(0);
  });
});
