import { useParticipants } from "@livekit/components-react";
import type { ReactNode } from "react";

import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { VoicePersonRow } from "@/components/voice/VoicePersonRow";

interface VoicePeoplePopoverProps {
  children: ReactNode;
}

// Everyone in the call, for when the stage or a strip cannot show them all.
export const VoicePeoplePopover = ({ children }: VoicePeoplePopoverProps) => {
  const participants = useParticipants();
  return (
    <Popover>
      <PopoverTrigger asChild>{children}</PopoverTrigger>
      <PopoverContent align="end" className="w-64 p-1.5">
        <p className="px-2 pt-1 pb-1.5 text-xs text-muted-foreground">
          <span className="font-mono tabular-nums">{participants.length}</span> in the call
        </p>
        <div className="flex max-h-80 flex-col overflow-y-auto">
          {participants.map((participant) => (
            <VoicePersonRow key={participant.identity} participant={participant} />
          ))}
        </div>
      </PopoverContent>
    </Popover>
  );
};
