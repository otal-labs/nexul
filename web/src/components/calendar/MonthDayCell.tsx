import { DayEntriesList } from "@/components/calendar/DayEntriesList";
import { DayNumber } from "@/components/calendar/DayNumber";
import type { DayEntries } from "@/hooks/useCalendarEntries";
import { useCalendarStore } from "@/stores/calendarStore";
import { cn } from "@/lib/utils";
import { formatLongDay } from "@/utils/CalendarUtility";

const SHOWN = 3;

interface MonthDayCellProps {
  day: Date;
  entries: DayEntries;
  today: boolean;
  // Days of the weeks that spill into the neighbouring months are muted.
  inMonth: boolean;
}

export const MonthDayCell = ({ day, entries, today, inMonth }: MonthDayCellProps) => {
  const showWeekOf = useCalendarStore((s) => s.showWeekOf);
  const more = entries.due.length + entries.meetings.length - SHOWN;
  return (
    <section
      aria-label={formatLongDay(day.toISOString())}
      className={cn("flex min-h-28 min-w-0 flex-col gap-1 border-r border-b border-border p-1.5", !inMonth && "bg-surface-2/60 text-muted-foreground")}
    >
      <DayNumber day={day} today={today} />
      <DayEntriesList entries={entries} limit={SHOWN} />
      {more > 0 && (
        <button
          type="button"
          onClick={() => showWeekOf(day)}
          className="rounded-md px-1.5 py-0.5 text-left text-xs text-muted-foreground transition-colors duration-150 ease-standard hover:bg-accent/40 hover:text-foreground"
        >
          {more} more
        </button>
      )}
    </section>
  );
};
