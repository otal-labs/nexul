import { dayKey } from "@nexul/client-core/calendar";

import { WeekDaySection } from "@/components/calendar/WeekDaySection";
import { entriesOn, type DayEntries } from "@/hooks/useCalendarEntries";
import { sameDay } from "@/utils/CalendarUtility";

interface CalendarWeekProps {
  days: Date[];
  byDay: Map<string, DayEntries>;
}

// The seven days of a week, one row each, with every item the day holds.
export const CalendarWeek = ({ days, byDay }: CalendarWeekProps) => {
  const today = new Date();
  return (
    <div className="divide-y divide-border border-y border-border">
      {days.map((day) => (
        <WeekDaySection key={dayKey(day)} day={day} entries={entriesOn(byDay, day)} today={sameDay(day, today)} />
      ))}
    </div>
  );
};
