import { DayEntriesList } from "@/components/calendar/DayEntriesList";
import { DayNumber } from "@/components/calendar/DayNumber";
import type { DayEntries } from "@/hooks/useCalendarEntries";
import { cn } from "@/lib/utils";
import { formatLongDay, formatWeekdayName } from "@/utils/CalendarUtility";

interface WeekDaySectionProps {
  day: Date;
  entries: DayEntries;
  today: boolean;
}

// One day of the week: its number and name as a heading, its items in time order under it on the shared tracks.
export const WeekDaySection = ({ day, entries, today }: WeekDaySectionProps) => {
  const empty = entries.due.length + entries.meetings.length === 0;
  return (
    <section aria-label={formatLongDay(day.toISOString())} className="py-2">
      <h3 className="flex h-6 items-center gap-2 px-1.5 text-[13px] font-medium">
        <DayNumber day={day} today={today} />
        <span className={cn(!today && "text-muted-foreground")}>{formatWeekdayName(day)}</span>
      </h3>
      {empty && <p className="flex h-6 items-center pr-1.5 pl-[1.125rem] text-sm text-muted-foreground">Nothing scheduled</p>}
      {!empty && <DayEntriesList entries={entries} detailed />}
    </section>
  );
};
