import type { Invitee } from "@nexul/client-core/calendar";

import { MeetingPersonRow } from "@/components/calendar/MeetingPersonRow";
import { Microheader } from "@/components/Microheader";

interface MeetingReplyGroupProps {
  label: string;
  invitees: Invitee[];
}

export const MeetingReplyGroup = ({ label, invitees }: MeetingReplyGroupProps) => (
  <section aria-label={label}>
    <Microheader>
      {label} <span className="tabular-nums">{invitees.length}</span>
    </Microheader>
    <ul>
      {invitees.map((invitee) => (
        <MeetingPersonRow key={invitee.user_id} userId={invitee.user_id} />
      ))}
    </ul>
  </section>
);
