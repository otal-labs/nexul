import { useParticipants } from "@livekit/components-react";
import type { TrackReference } from "@livekit/components-core";

import { ScreenShareTile } from "@/components/chat/ScreenShareTile";
import { VoiceOwnShare } from "@/components/voice/VoiceOwnShare";
import { VoiceStrip } from "@/components/voice/VoiceStrip";

interface VoiceShareLayoutProps {
  share: TrackReference;
}

// The shared screen takes the stage's full width; under it, whose screen it is and who is in the call as avatars.
export const VoiceShareLayout = ({ share }: VoiceShareLayoutProps) => {
  const participants = useParticipants();
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
        <VoiceStrip participants={participants} />
      </div>
    </div>
  );
};
