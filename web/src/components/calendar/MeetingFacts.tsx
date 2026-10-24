import { REMIND_OPTIONS, REPEAT_LABELS, type Meeting } from "@nexul/client-core/calendar";

import { MeetingWhereFact } from "@/components/calendar/MeetingWhereFact";
import { Fact } from "@/components/Fact";
import { formatShortDay } from "@/utils/CalendarUtility";

const remindLabel = (minutes: number): string =>
  REMIND_OPTIONS.find((o) => o.minutes === minutes)?.label ?? `${minutes} minutes before`;

// What a meeting is besides its time: how it repeats, where it happens and when its reminder goes.
export const MeetingFacts = ({ meeting }: { meeting: Meeting }) => (
  <dl className="grid grid-cols-2 gap-x-4 gap-y-4">
    {meeting.repeats !== "none" && (
      <Fact label="Repeats">
        {REPEAT_LABELS[meeting.repeats]}
        {meeting.repeat_until && <span className="text-muted-foreground">until {formatShortDay(new Date(meeting.repeat_until))}</span>}
      </Fact>
    )}
    <MeetingWhereFact meeting={meeting} />
    <Fact label="Reminder">{remindLabel(meeting.remind_minutes)}</Fact>
  </dl>
);
