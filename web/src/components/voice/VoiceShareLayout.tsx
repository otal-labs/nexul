import { useParticipants } from "@livekit/components-react";
import type { TrackReference } from "@livekit/components-core";

import { ScreenShareTile } from "@/components/chat/ScreenShareTile";
import { VoiceOwnShare } from "@/components/voice/VoiceOwnShare";
import { VoicePeoplePopover } from "@/components/voice/VoicePeoplePopover";
import { VoiceStripAvatar } from "@/components/voice/VoiceStripAvatar";
import { useElementSize } from "@/hooks/useElementSize";

interface VoiceShareLayoutProps {
  share: TrackReference;
}

const AVATAR = 28;
const MORE = 40;

// The shared screen takes the stage's full width; under it, whose screen it is and who is in the call as avatars.
export const VoiceShareLayout = ({ share }: VoiceShareLayoutProps) => {
  const participants = useParticipants();
  const [avatarsRef, avatars] = useElementSize<HTMLDivElement>();
  const fits = Math.max(1, Math.floor((avatars.width - MORE) / AVATAR));
  const shown = participants.length > fits ? fits : participants.length;
  const own = share.participant.isLocal;
  const sharer = share.participant.name || share.participant.identity;

  return (
    <div className="flex min-h-0 flex-1 flex-col gap-2">
      <div className="flex min-h-0 min-w-0 flex-1 overflow-hidden rounded-md bg-media dark:ring-1 dark:ring-border *:min-h-0 *:flex-1">
        {own && <VoiceOwnShare />}
        {!own && <ScreenShareTile trackRef={share} />}
      </div>
      <div className="flex h-8 shrink-0 items-center gap-3 px-1">
        <span dir="auto" className="max-w-1/2 min-w-0 truncate text-xs text-muted-foreground">
          {own ? "Your screen" : `${sharer}'s screen`}
        </span>
        <div ref={avatarsRef} className="flex min-w-0 flex-1 items-center justify-end gap-1">
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
      </div>
    </div>
  );
};
