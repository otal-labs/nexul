import { TrackRefContext, useParticipants } from "@livekit/components-react";
import type { TrackReferenceOrPlaceholder } from "@livekit/components-core";

import { VoiceStrip } from "@/components/voice/VoiceStrip";
import { VoiceTile } from "@/components/voice/VoiceTile";
import { useElementSize } from "@/hooks/useElementSize";
import { useLargeTrack } from "@/hooks/useLargeTrack";

interface VoiceSpeakerLayoutProps {
  tracks: TrackReferenceOrPlaceholder[];
}

const VIDEO = 16 / 9;
// The speaking ring sits 4px outside the tile.
const RING = 8;

// One 16:9 tile for whoever is speaking, as large as the well allows; everyone else as the avatar strip under it.
export const VoiceSpeakerLayout = ({ tracks }: VoiceSpeakerLayoutProps) => {
  const [ref, { width, height }] = useElementSize<HTMLDivElement>();
  const participants = useParticipants();
  const large = useLargeTrack(tracks);
  const tileWidth = Math.floor(Math.min(width - RING, (height - RING) * VIDEO));
  const others = participants.filter((p) => p.identity !== large?.participant.identity);

  return (
    <div className="flex min-h-0 flex-1 flex-col gap-2">
      <div ref={ref} className="flex min-h-0 flex-1 items-center justify-center">
        {large && tileWidth > 0 && (
          <TrackRefContext.Provider value={large}>
            <VoiceTile style={{ width: tileWidth, height: Math.floor(tileWidth / VIDEO) }} />
          </TrackRefContext.Provider>
        )}
      </div>
      <div className="flex h-8 shrink-0 items-center px-1">
        <VoiceStrip participants={others} align="center" />
      </div>
    </div>
  );
};
