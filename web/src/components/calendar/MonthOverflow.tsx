import type { ShownEntries } from "@/hooks/useCalendarEntries";
import { useCalendarStore } from "@/stores/calendarStore";

interface MonthOverflowProps {
  day: Date;
  shown: ShownEntries;
}

// What a month cell had no line for, counted by band beside the day's number (one count where the cell is narrow); it opens the day's week.
export const MonthOverflow = ({ day, shown: { hiddenDue, hiddenMeetings } }: MonthOverflowProps) => {
  const showWeekOf = useCalendarStore((s) => s.showWeekOf);
  const hidden = hiddenDue + hiddenMeetings;
  if (hidden === 0) return null;
  const label = [hiddenDue > 0 && `+${hiddenDue} due`, hiddenMeetings > 0 && `+${hiddenMeetings}`].filter(Boolean).join(" · ");
  return (
    <button
      type="button"
      onClick={() => showWeekOf(day)}
      aria-label={`${label}, show the week`}
      title={label}
      className="h-5 min-w-0 truncate rounded-sm px-1 font-mono leading-5 text-[11px] text-muted-foreground tabular-nums transition-colors duration-150 ease-standard hover:bg-accent/40 hover:text-foreground"
    >
      <span className="@max-[55rem]:hidden">{label}</span>
      <span className="hidden @max-[55rem]:inline">+{hidden}</span>
    </button>
  );
};
