import { dayKey } from "@nexul/client-core/calendar";

import { MonthDayCell } from "@/components/calendar/MonthDayCell";
import { entriesOn, type DayEntries } from "@/hooks/useCalendarEntries";
import { useCalendarStore } from "@/stores/calendarStore";
import { formatWeekday, sameDay } from "@/utils/CalendarUtility";

interface CalendarMonthProps {
  days: Date[];
  byDay: Map<string, DayEntries>;
}

// The weeks of the anchor's month as a grid, a week to a row.
export const CalendarMonth = ({ days, byDay }: CalendarMonthProps) => {
  const anchor = useCalendarStore((s) => s.anchor);
  const today = new Date();
  return (
    <div className="border-t border-l border-border">
      <div aria-hidden className="grid grid-cols-7 border-r border-b border-border">
        {days.slice(0, 7).map((day) => (
          <span key={dayKey(day)} className="px-2 py-1.5 font-mono text-[11px] font-medium tracking-[0.12em] text-muted-foreground uppercase">
            {formatWeekday(day)}
          </span>
        ))}
      </div>
      <div className="grid grid-cols-7">
        {days.map((day) => (
          <MonthDayCell
            key={dayKey(day)}
            day={day}
            entries={entriesOn(byDay, day)}
            today={sameDay(day, today)}
            inMonth={day.getMonth() === anchor.getMonth()}
          />
        ))}
      </div>
    </div>
  );
};
