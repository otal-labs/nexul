import { DueTicketRow } from "@/components/calendar/DueTicketRow";
import { MeetingEntryRow } from "@/components/calendar/MeetingEntryRow";
import type { DayEntries } from "@/hooks/useCalendarEntries";
import { cn } from "@/lib/utils";
import { overlapsEarlier } from "@/utils/CalendarUtility";

interface DayEntriesListProps {
  entries: DayEntries;
  detailed?: boolean;
}

// Due tickets lead, being all-day; meetings follow in start order, the week marking one that starts inside another.
export const DayEntriesList = ({ entries, detailed = false }: DayEntriesListProps) => {
  const live = entries.meetings.map(({ meeting, occurrence }) => ({ ...occurrence, ...(meeting.cancelled_at ? { ends_at: occurrence.starts_at } : {}) }));
  const overlaps = detailed ? overlapsEarlier(live) : [];
  return (
    <ul className={cn(!detailed && "space-y-px")}>
      {entries.due.map((ticket) => (
        // A box gives a pixel of its line to the gap, so two stacked boxes never read as one doubled hairline.
        <li key={ticket.id} className={cn(!detailed && "pb-px")}>
          <DueTicketRow ticket={ticket} detailed={detailed} />
        </li>
      ))}
      {entries.meetings.map((entry, i) => (
        <li key={`${entry.meeting.id}@${entry.occurrence.slot}`}>
          <MeetingEntryRow entry={entry} detailed={detailed} overlaps={overlaps[i] === true && !entry.meeting.cancelled_at} />
        </li>
      ))}
    </ul>
  );
};
