import { beforeEach, describe, expect, it } from "vitest";

import { useCalendarStore } from "@/stores/calendarStore";

beforeEach(() => useCalendarStore.setState({ view: "month", anchor: new Date(2026, 9, 24) }));

describe("the calendar store", () => {
  it("steps the anchor by the view's own unit", () => {
    useCalendarStore.getState().step(1);
    expect(useCalendarStore.getState().anchor).toEqual(new Date(2026, 10, 1));
    useCalendarStore.getState().setView("week");
    useCalendarStore.getState().step(-1);
    expect(useCalendarStore.getState().anchor).toEqual(new Date(2026, 9, 25));
  });

  it("returns to today, and opens a busy day as its week", () => {
    useCalendarStore.getState().showWeekOf(new Date(2026, 9, 27));
    expect(useCalendarStore.getState()).toMatchObject({ view: "week", anchor: new Date(2026, 9, 27) });
    useCalendarStore.getState().today();
    expect(useCalendarStore.getState().anchor.toDateString()).toBe(new Date().toDateString());
  });

  it("keeps the chosen view and not the day", () => {
    useCalendarStore.getState().setView("week");
    expect(Object.keys(JSON.parse(localStorage.getItem("calendar") ?? "{}").state ?? {})).toEqual(["view"]);
  });
});
