import { useParticipants } from "@livekit/components-react";
import type { TrackReference } from "@livekit/components-core";
import type { ReactNode } from "react";

import { ScreenShareTile } from "@/components/chat/ScreenShareTile";
import { VoicePeoplePopover } from "@/components/voice/VoicePeoplePopover";
import { VoicePersonRow } from "@/components/voice/VoicePersonRow";
import { VoiceStripAvatar } from "@/components/voice/VoiceStripAvatar";
import { useElementSize } from "@/hooks/useElementSize";

interface VoiceShareLayoutProps {
  share: TrackReference;
}

const ROW = 32;
const AVATAR = 28;

const MoreButton = ({ count, className, children }: { count: number; className: string; children: ReactNode }) => (
  <VoicePeoplePopover>
    <button type="button" className={className} aria-label={`${count} more in the call`}>
      {children}
    </button>
  </VoicePeoplePopover>
);

// The shared screen takes the stage; who is in the call runs beside it when wide and under it as avatars when narrow.
export const VoiceShareLayout = ({ share }: VoiceShareLayoutProps) => {
  const participants = useParticipants();
  const [rosterRef, roster] = useElementSize<HTMLDivElement>();
  const [stripRef, strip] = useElementSize<HTMLDivElement>();
  const rows = Math.max(1, Math.floor(roster.height / ROW));
  const rowsShown = participants.length > rows ? rows - 1 : participants.length;
  const avatars = Math.max(1, Math.floor((strip.width - 40) / AVATAR));
  const avatarsShown = participants.length > avatars ? avatars : participants.length;

  return (
    <div className="flex min-h-0 flex-1 flex-col gap-2 @[1000px]/thread:flex-row">
      <div className="flex min-h-0 min-w-0 flex-1 overflow-hidden rounded-md bg-media dark:ring-1 dark:ring-border *:min-h-0 *:flex-1">
        <ScreenShareTile trackRef={share} />
      </div>
      <div ref={rosterRef} className="hidden w-50 shrink-0 flex-col overflow-hidden @[1000px]/thread:flex">
        {participants.slice(0, rowsShown).map((participant) => (
          <VoicePersonRow key={participant.identity} participant={participant} />
        ))}
        {rowsShown < participants.length && (
          <MoreButton
            count={participants.length - rowsShown}
            className="flex h-8 items-center gap-2 rounded-sm px-2 text-left text-[13px] text-muted-foreground hover:bg-accent"
          >
            <span aria-hidden className="flex size-5 shrink-0 items-center justify-center rounded-full bg-muted font-mono text-[10px] tabular-nums">
              +{participants.length - rowsShown}
            </span>
            {participants.length - rowsShown} more
          </MoreButton>
        )}
      </div>
      <div ref={stripRef} className="flex h-8 shrink-0 items-center gap-1 px-1 @[1000px]/thread:hidden">
        {participants.slice(0, avatarsShown).map((participant) => (
          <VoiceStripAvatar key={participant.identity} participant={participant} />
        ))}
        {avatarsShown < participants.length && (
          <MoreButton
            count={participants.length - avatarsShown}
            className="ml-1 flex h-6 items-center rounded-sm px-1.5 font-mono text-xs text-muted-foreground tabular-nums hover:bg-accent"
          >
            +{participants.length - avatarsShown}
          </MoreButton>
        )}
      </div>
    </div>
  );
};
