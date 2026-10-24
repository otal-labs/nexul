import { create } from "zustand";
import { persist } from "zustand/middleware";

import { stepAnchor, type CalendarView } from "@nexul/client-core/calendar";

export type CalendarStore = {
  view: CalendarView;
  // The day the view is looking around; not kept, so the calendar opens on today.
  anchor: Date;
  setView: (view: CalendarView) => void;
  step: (by: -1 | 1) => void;
  today: () => void;
  // A busy day of the month opens as its week.
  showWeekOf: (day: Date) => void;
};

export const useCalendarStore = create<CalendarStore>()(
  persist(
    (set) => ({
      view: "month",
      anchor: new Date(),
      setView: (view) => set({ view }),
      step: (by) => set((s) => ({ anchor: stepAnchor(s.view, s.anchor, by) })),
      today: () => set({ anchor: new Date() }),
      showWeekOf: (day) => set({ view: "week", anchor: day }),
    }),
    { name: "calendar", partialize: (s) => ({ view: s.view }) },
  ),
);
