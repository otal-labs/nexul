import { DayEntriesList } from "@/components/calendar/DayEntriesList";
import { DayNumber } from "@/components/calendar/DayNumber";
import type { DayEntries } from "@/hooks/useCalendarEntries";
import { formatLongDay, formatWeekday } from "@/utils/CalendarUtility";

interface WeekDaySectionProps {
  day: Date;
  entries: DayEntries;
  today: boolean;
}

// One day of the week: its name on the left, its items in time order beside it.
export const WeekDaySection = ({ day, entries, today }: WeekDaySectionProps) => (
  <section aria-label={formatLongDay(day.toISOString())} className="flex min-h-14 gap-4 px-2 py-2.5">
    <div className="flex w-14 shrink-0 flex-col items-start gap-1 sm:w-20">
      <span className="font-mono text-[11px] font-medium tracking-[0.12em] text-muted-foreground uppercase">{formatWeekday(day)}</span>
      <DayNumber day={day} today={today} />
    </div>
    <div className="min-w-0 flex-1">
      <DayEntriesList entries={entries} detailed />
    </div>
  </section>
);
