import { ParticipantTile, isTrackReference, useIsSpeaking, useMaybeTrackRefContext } from "@livekit/components-react";

import { PersonAvatar } from "@/components/PersonAvatar";
import { cn } from "@/lib/utils";

interface VoiceTileProps {
  resolveLogin: (identity: string) => string;
}

// A real camera track renders ParticipantTile; audio-only gets a Discord-style avatar tile instead.
export const VoiceTile = ({ resolveLogin }: VoiceTileProps) => {
  const trackRef = useMaybeTrackRefContext();
  const speaking = useIsSpeaking(trackRef?.participant);
  if (!trackRef) return null;

  const isVideo = isTrackReference(trackRef);
  const identity = trackRef.participant.identity;
  const name = trackRef.participant.name || identity;
  return (
    <>
      {isVideo && <ParticipantTile />}
      {!isVideo && (
        <div className="flex h-full flex-col items-center justify-center gap-2 rounded-md bg-muted/40 p-2" title={name}>
          <PersonAvatar
            login={resolveLogin(identity)}
            className={cn("size-14 text-base ring-2 ring-transparent transition-shadow duration-150 ease-standard", speaking && "ring-primary")}
          />
          <span className="max-w-full truncate text-xs text-muted-foreground">{name}</span>
        </div>
      )}
    </>
  );
};
