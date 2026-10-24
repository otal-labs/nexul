import { isCancelled, occurrenceTimes } from "@nexul/client-core/calendar";

import { MeetingActions } from "@/components/calendar/MeetingActions";
import { MeetingFacts } from "@/components/calendar/MeetingFacts";
import { MeetingPeople } from "@/components/calendar/MeetingPeople";
import { DiscordMarkdown } from "@/components/chat/DiscordMarkdown";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { SheetDescription, SheetHeader, SheetTitle } from "@/components/ui/sheet";
import { useFetchMeeting } from "@/hooks/CalendarHooks";
import { formatLongDay, formatTimeRange } from "@/utils/CalendarUtility";

interface MeetingPanelProps {
  meetingId: string;
  slot: string;
}

// The open meeting: its title and time, then where, the notes and the people, with what the viewer can do at the foot.
export const MeetingPanel = ({ meetingId, slot }: MeetingPanelProps) => {
  const { data: meeting, error, isPending } = useFetchMeeting(meetingId);
  const times = meeting && occurrenceTimes(meeting, slot);
  return (
    <>
      <SheetHeader>
        <SheetTitle className={meeting && isCancelled(meeting) ? "text-muted-foreground line-through" : undefined}>{meeting?.title ?? "Meeting"}</SheetTitle>
        <SheetDescription>
          {times && `${formatLongDay(times.starts_at)}, ${formatTimeRange(times.starts_at, times.ends_at)}`}
          {!times && "Loading…"}
        </SheetDescription>
        {meeting && isCancelled(meeting) && (
          <p className="flex items-center gap-1.5 text-sm">
            <span aria-hidden className="size-2 rounded-full bg-destructive" />
            Cancelled
          </p>
        )}
      </SheetHeader>
      <div className="min-h-0 flex-1 space-y-5 overflow-y-auto px-4 pb-4">
        {isPending && <LoadingDisplay />}
        {error && <ErrorDisplay error={error} title="Couldn't load the meeting." />}
        {meeting && <MeetingFacts meeting={meeting} />}
        {meeting && meeting.notes !== "" && <DiscordMarkdown text={meeting.notes} className="space-y-0.5 text-sm" />}
        {meeting && <MeetingPeople meeting={meeting} />}
      </div>
      {meeting && (
        <div className="border-t border-border p-4">
          <MeetingActions meeting={meeting} slot={slot} />
        </div>
      )}
    </>
  );
};
