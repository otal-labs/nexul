import { dayKey } from "@nexul/client-core/calendar";

import { MonthDayCell } from "@/components/calendar/MonthDayCell";
import { entriesOn, type DayEntries } from "@/hooks/useCalendarEntries";
import { useCalendarStore } from "@/stores/calendarStore";
import { cn } from "@/lib/utils";
import { formatWeekday, sameDay } from "@/utils/CalendarUtility";

// Weekends take a narrower column, a work calendar's Saturday holding the odd due ticket, until the grid is too narrow to spare it.
const COLUMNS = "grid grid-cols-[repeat(5,minmax(0,1fr))_repeat(2,minmax(0,0.65fr))] @max-[55rem]:grid-cols-7";

// The line budget comes from the week count, never a measurement: a six-week month's shorter rows hold one line fewer.
const linesFor = (weeks: number) => (weeks > 5 ? 4 : 5);

interface CalendarMonthProps {
  days: Date[];
  byDay: Map<string, DayEntries>;
}

// Rows share the panel's height above a floor of a cell's 20px head, its 21px lines, 4px of padding and its border.
export const CalendarMonth = ({ days, byDay }: CalendarMonthProps) => {
  const anchor = useCalendarStore((s) => s.anchor);
  const today = new Date();
  const weeks = Math.ceil(days.length / 7);
  const lines = linesFor(weeks);
  return (
    <div className="@container flex min-h-0 flex-1 flex-col overflow-hidden rounded-lg border border-border">
      <div aria-hidden className={cn(COLUMNS, "border-b border-border")}>
        {days.slice(0, 7).map((day) => (
          <span key={dayKey(day)} className="px-1.5 py-1.5 font-mono @max-[55rem]:px-1 text-[11px] font-medium tracking-[0.12em] text-muted-foreground uppercase">
            {formatWeekday(day)}
          </span>
        ))}
      </div>
      <div className={cn(COLUMNS, "flex-1 [&>section:nth-child(7n)]:border-r-0 [&>section:nth-last-child(-n+7)]:border-b-0")} style={{ gridTemplateRows: `repeat(${weeks}, minmax(${24 + lines * 21}px, 1fr))` }}>
        {days.map((day) => (
          <MonthDayCell
            key={dayKey(day)}
            day={day}
            entries={entriesOn(byDay, day)}
            today={sameDay(day, today)}
            inMonth={day.getMonth() === anchor.getMonth()}
            lines={lines}
          />
        ))}
      </div>
    </div>
  );
};
