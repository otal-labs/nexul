import { isCancelled } from "@nexul/client-core/calendar";

import { useMeetingParams } from "@/hooks/useMeetingParams";
import type { MeetingEntry } from "@/hooks/useCalendarEntries";
import { cn } from "@/lib/utils";
import { formatTime, formatTimeRange } from "@/utils/CalendarUtility";

interface MeetingEntryRowProps {
  entry: MeetingEntry;
  // The week lists a meeting's whole span; a month cell has room for its start.
  detailed?: boolean;
}

// One time a meeting takes place. A cancelled one stays on the calendar, struck through, and opens like any other.
export const MeetingEntryRow = ({ entry: { meeting, occurrence }, detailed = false }: MeetingEntryRowProps) => {
  const { meetingId, slot, show } = useMeetingParams();
  const cancelled = isCancelled(meeting);
  const open = meetingId === meeting.id && (slot === "" || slot === occurrence.slot);
  return (
    <button
      type="button"
      onClick={() => show(meeting.id, occurrence.slot)}
      aria-current={open ? "true" : undefined}
      className={cn(
        "flex w-full min-w-0 flex-wrap items-baseline gap-x-2 rounded-md px-1.5 py-0.5 text-left text-xs transition-colors duration-150 ease-standard hover:bg-accent/40",
        open && "bg-accent",
        detailed && "py-1.5 text-sm",
        cancelled && "text-muted-foreground line-through",
      )}
    >
      <time dateTime={occurrence.starts_at} className={cn("shrink-0 font-mono text-muted-foreground tabular-nums", detailed && "min-w-32")}>
        {detailed ? formatTimeRange(occurrence.starts_at, occurrence.ends_at) : formatTime(occurrence.starts_at)}
      </time>
      <span className="min-w-0 flex-1 basis-20 truncate">{meeting.title}</span>
      {cancelled && <span className="sr-only">Cancelled</span>}
    </button>
  );
};
