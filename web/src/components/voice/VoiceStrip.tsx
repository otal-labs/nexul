import type { Participant } from "livekit-client";

import { VoicePeoplePopover } from "@/components/voice/VoicePeoplePopover";
import { VoiceStripAvatar } from "@/components/voice/VoiceStripAvatar";
import { useElementSize } from "@/hooks/useElementSize";
import { cn } from "@/lib/utils";

interface VoiceStripProps {
  participants: Participant[];
  /** End when a caption sits at the strip's left; centre under a centred tile. */
  align?: "end" | "center";
}

const AVATAR = 28;
const MORE = 40;

// People as 24px avatars on one line; those who don't fit go behind a +N that opens the list of everyone.
export const VoiceStrip = ({ participants, align = "end" }: VoiceStripProps) => {
  const [ref, { width }] = useElementSize<HTMLDivElement>();
  const fits = Math.max(1, Math.floor((width - MORE) / AVATAR));
  const shown = participants.length > fits ? fits : participants.length;
  return (
    <div ref={ref} className={cn("flex min-w-0 flex-1 items-center gap-1", align === "end" ? "justify-end" : "justify-center")}>
      {participants.slice(0, shown).map((participant) => (
        <VoiceStripAvatar key={participant.identity} participant={participant} />
      ))}
      {shown < participants.length && (
        <VoicePeoplePopover>
          <button
            type="button"
            className="ml-1 flex h-6 items-center rounded-sm px-1.5 font-mono text-xs text-muted-foreground tabular-nums hover:bg-accent"
            aria-label={`${participants.length - shown} more in the call`}
          >
            +{participants.length - shown}
          </button>
        </VoicePeoplePopover>
      )}
    </div>
  );
};
