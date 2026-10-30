import { isTrackReference, useTracks } from "@livekit/components-react";
import { Track } from "livekit-client";

import { VoiceShareLayout } from "@/components/voice/VoiceShareLayout";
import { VoiceSpeakerLayout } from "@/components/voice/VoiceSpeakerLayout";
import { VoiceStageGrid } from "@/components/voice/VoiceStageGrid";
import { useVoiceStageViewStore } from "@/stores/voiceStageViewStore";

// Placeholders give audio-only people a tile; a shared screen takes the stage in either view.
export const VoiceStage = () => {
  const view = useVoiceStageViewStore((s) => s.view);
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
      {!share && view === "grid" && <VoiceStageGrid tracks={cameras} />}
      {!share && view === "speaker" && <VoiceSpeakerLayout tracks={cameras} />}
    </>
  );
};
