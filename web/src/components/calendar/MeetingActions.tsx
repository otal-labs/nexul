import { PencilIcon, RotateCcwIcon, XIcon } from "lucide-react";

import { isCancelled, type Meeting } from "@nexul/client-core/calendar";

import { MeetingReply } from "@/components/calendar/MeetingReply";
import { Button } from "@/components/ui/button";
import { useCanChangeMeeting } from "@/hooks/useCanChangeMeeting";
import { useMeetingActions } from "@/hooks/useMeetingActions";

interface MeetingActionsProps {
  meeting: Meeting;
  slot: string;
}

// An invitee answers; whoever may change the meeting edits, cancels or, once it is cancelled, restores it.
export const MeetingActions = ({ meeting, slot }: MeetingActionsProps) => {
  const canChange = useCanChangeMeeting(meeting);
  const { edit, cancel, restore, pending } = useMeetingActions(meeting, slot);
  const cancelled = isCancelled(meeting);
  return (
    <div className="flex flex-wrap items-center gap-2">
      <MeetingReply meeting={meeting} />
      {canChange && (
        <Button variant="outline" size="sm" onClick={() => void edit()}>
          <PencilIcon aria-hidden />
          Edit
        </Button>
      )}
      {canChange && !cancelled && (
        <Button variant="outline" size="sm" loading={pending} onClick={() => void cancel()}>
          <XIcon aria-hidden />
          Cancel meeting
        </Button>
      )}
      {canChange && cancelled && (
        <Button variant="outline" size="sm" loading={pending} onClick={() => void restore()}>
          <RotateCcwIcon aria-hidden />
          Restore
        </Button>
      )}
    </div>
  );
};
