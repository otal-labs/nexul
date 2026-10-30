import { isCancelled } from "@nexul/client-core/calendar";

import { ENTRY_ROW, MONTH_ROW, WEEK_ROW } from "@/components/calendar/entryLayout";
import { MeetingMeta } from "@/components/calendar/MeetingMeta";
import { useMeetingParams } from "@/hooks/useMeetingParams";
import type { MeetingEntry } from "@/hooks/useCalendarEntries";
import { cn } from "@/lib/utils";
import { daysLater, formatTime, formatTimeRange } from "@/utils/CalendarUtility";

interface MeetingEntryRowProps {
  entry: MeetingEntry;
  // The week lists a meeting's whole span; a month cell has room for its start.
  detailed?: boolean;
  // Starts before an earlier meeting that day has ended.
  overlaps?: boolean;
}

// One time a meeting takes place. A cancelled one stays on the calendar, its title struck through, and opens like any other.
export const MeetingEntryRow = ({ entry: { meeting, occurrence }, detailed = false, overlaps = false }: MeetingEntryRowProps) => {
  const { meetingId, slot, show } = useMeetingParams();
  const cancelled = isCancelled(meeting);
  const open = meetingId === meeting.id && (slot === "" || slot === occurrence.slot);
  const range = formatTimeRange(occurrence.starts_at, occurrence.ends_at);
  const later = daysLater(occurrence.starts_at, occurrence.ends_at);
  return (
    <button
      type="button"
      onClick={() => show(meeting.id, occurrence.slot)}
      aria-current={open ? "true" : undefined}
      title={`${range} ${meeting.title}`}
      className={cn(
        ENTRY_ROW,
        detailed ? cn(WEEK_ROW, "text-sm") : cn(MONTH_ROW, "text-[13px]"),
        open && "bg-accent",
        cancelled && "text-muted-foreground",
      )}
    >
      {!detailed && (
        <span className="flex items-center">
          <time dateTime={occurrence.starts_at} className="font-mono text-[11px] text-muted-foreground tabular-nums @max-[55rem]:sr-only">
            {formatTime(occurrence.starts_at)}
          </time>
          <span aria-hidden className="hidden size-1.5 rounded-full bg-muted-foreground/60 @max-[55rem]:block" />
        </span>
      )}
      {detailed && (
        <span className="flex items-center gap-1.5 font-mono text-xs text-muted-foreground tabular-nums">
          <span className="flex size-1.5 shrink-0">
            {overlaps && <span role="img" aria-label="Overlaps another meeting" title="Overlaps another meeting" className="size-1.5 rounded-full bg-warning-mark" />}
          </span>
          <time dateTime={occurrence.starts_at} className="relative top-px">
            {range}
          </time>
          {later > 0 && (
            <span aria-label={`ends ${later} day later`} className="relative top-px">
              +{later}
            </span>
          )}
        </span>
      )}
      <span className={cn("truncate", cancelled && "line-through")}>{meeting.title}</span>
      {cancelled && <span className="sr-only">Cancelled</span>}
      {detailed && <MeetingMeta meeting={meeting} />}
    </button>
  );
};
