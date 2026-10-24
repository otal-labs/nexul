import { groupReplies, type Meeting } from "@nexul/client-core/calendar";

import { MeetingPersonRow } from "@/components/calendar/MeetingPersonRow";
import { MeetingReplyGroup } from "@/components/calendar/MeetingReplyGroup";
import { Microheader } from "@/components/Microheader";

// The organiser, then the people invited by how they answered.
export const MeetingPeople = ({ meeting }: { meeting: Meeting }) => {
  const { accepted, declined, pending } = groupReplies(meeting);
  return (
    <div className="space-y-4">
      <section aria-label="Organiser">
        <Microheader>Organiser</Microheader>
        <ul>
          <MeetingPersonRow userId={meeting.organiser_id} />
        </ul>
      </section>
      {accepted.length > 0 && <MeetingReplyGroup label="Coming" invitees={accepted} />}
      {declined.length > 0 && <MeetingReplyGroup label="Declined" invitees={declined} />}
      {pending.length > 0 && <MeetingReplyGroup label="No reply" invitees={pending} />}
    </div>
  );
};
