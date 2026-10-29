import { DayEntriesList } from "@/components/calendar/DayEntriesList";
import { DayNumber } from "@/components/calendar/DayNumber";
import { MonthOverflow } from "@/components/calendar/MonthOverflow";
import { shownIn, type DayEntries } from "@/hooks/useCalendarEntries";
import { cn } from "@/lib/utils";
import { formatLongDay } from "@/utils/CalendarUtility";

interface MonthDayCellProps {
  day: Date;
  entries: DayEntries;
  today: boolean;
  // Days of the weeks that spill into the neighbouring months sit in the board's well shade.
  inMonth: boolean;
  // How many entries the month's row height holds.
  lines: number;
}

export const MonthDayCell = ({ day, entries, today, inMonth, lines }: MonthDayCellProps) => {
  const shown = shownIn(entries, lines);
  return (
    <section
      aria-label={formatLongDay(day.toISOString())}
      className={cn("flex min-w-0 flex-col overflow-hidden border-r border-b border-border px-1.5 py-0.5 @max-[55rem]:px-1", !inMonth && "bg-well-shade text-muted-foreground")}
    >
      <div className="flex h-5 shrink-0 items-center justify-between gap-1">
        <DayNumber day={day} today={today} />
        <MonthOverflow day={day} shown={shown} />
      </div>
      <DayEntriesList entries={shown} />
    </section>
  );
};
