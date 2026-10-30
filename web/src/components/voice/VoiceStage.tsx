import { isTrackReference, useTracks } from "@livekit/components-react";
import { Track } from "livekit-client";

import { VoiceShareLayout } from "@/components/voice/VoiceShareLayout";
import { VoiceStageGrid } from "@/components/voice/VoiceStageGrid";

// Placeholders give audio-only people a tile; a shared screen takes the stage instead of the grid.
export const VoiceStage = () => {
  const tracks = useTracks(
    [
      { source: Track.Source.Camera, withPlaceholder: true },
      { source: Track.Source.ScreenShare, withPlaceholder: false },
    ],
    { onlySubscribed: false },
  );
  const share = tracks.find((t) => t.source === Track.Source.ScreenShare && isTrackReference(t));
  const cameras = tracks.filter((t) => t.source === Track.Source.Camera);
  return (
    <>
      {share && isTrackReference(share) && <VoiceShareLayout share={share} />}
      {!share && <VoiceStageGrid tracks={cameras} />}
    </>
  );
};
