import { DueTicketRow } from "@/components/calendar/DueTicketRow";
import { MeetingEntryRow } from "@/components/calendar/MeetingEntryRow";
import type { DayEntries } from "@/hooks/useCalendarEntries";

interface DayEntriesListProps {
  entries: DayEntries;
  detailed?: boolean;
  // A month cell shows this many and leaves the rest to its week.
  limit?: number;
}

// Due tickets lead, being all-day; meetings follow in start order.
export const DayEntriesList = ({ entries, detailed = false, limit = Infinity }: DayEntriesListProps) => {
  const shownDue = entries.due.slice(0, limit);
  const shownMeetings = entries.meetings.slice(0, limit - shownDue.length);
  return (
    <ul className="space-y-0.5">
      {shownDue.map((ticket) => (
        <li key={ticket.id}>
          <DueTicketRow ticket={ticket} detailed={detailed} />
        </li>
      ))}
      {shownMeetings.map((entry) => (
        <li key={`${entry.meeting.id}@${entry.occurrence.slot}`}>
          <MeetingEntryRow entry={entry} detailed={detailed} />
        </li>
      ))}
    </ul>
  );
};
