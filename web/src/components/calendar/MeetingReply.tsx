import type { Meeting, Reply } from "@nexul/client-core/calendar";
import { myReply } from "@nexul/client-core/calendar";

import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group";
import { useFetchMe } from "@/hooks/AuthHooks";
import { useReplyToMeeting } from "@/hooks/CalendarHooks";

// Accept or Decline for an invitee, the answer they gave selected. It covers every occurrence of a repeating meeting.
export const MeetingReply = ({ meeting }: { meeting: Meeting }) => {
  const { data: me } = useFetchMe();
  const reply = useReplyToMeeting();
  const current: Reply | undefined = me && myReply(meeting, me.user.id);
  if (!current) return null;
  return (
    <ToggleGroup
      type="single"
      variant="segmented"
      size="xs"
      value={current === "pending" ? "" : current}
      // Radix reports "" when the chosen answer is pressed again; the answer stands.
      onValueChange={(value) => value && reply.mutate({ id: meeting.id, reply: value as "accepted" | "declined" })}
      disabled={reply.isPending}
      aria-label="Your reply"
    >
      <ToggleGroupItem value="accepted">Accept</ToggleGroupItem>
      <ToggleGroupItem value="declined">Decline</ToggleGroupItem>
    </ToggleGroup>
  );
};
