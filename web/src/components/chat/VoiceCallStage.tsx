import { CarouselLayout, FocusLayout, FocusLayoutContainer, GridLayout, isTrackReference, useTracks } from "@livekit/components-react";
import { Track } from "livekit-client";

import { VoiceGridCell } from "@/components/chat/VoiceGridCell";
import { VoiceTile } from "@/components/chat/VoiceTile";

// GridLayout with placeholders so audio-only participants still get a tile; screen share gets FocusLayout.
export const VoiceCallStage = () => {
  const tracks = useTracks(
    [
      { source: Track.Source.Camera, withPlaceholder: true },
      { source: Track.Source.ScreenShare, withPlaceholder: false },
    ],
    { onlySubscribed: false },
  );
  const screenShareTrack = tracks.find((t) => t.source === Track.Source.ScreenShare && isTrackReference(t));
  const rest = tracks.filter((t) => t !== screenShareTrack);

  return (
    <>
      {screenShareTrack && isTrackReference(screenShareTrack) && (
        <FocusLayoutContainer className="min-h-0 flex-1">
          <CarouselLayout tracks={rest}>
            <VoiceTile />
          </CarouselLayout>
          <FocusLayout trackRef={screenShareTrack} />
        </FocusLayoutContainer>
      )}
      {!screenShareTrack && (
        <GridLayout tracks={tracks} className="min-h-0 flex-1">
          <VoiceGridCell>
            <VoiceTile />
          </VoiceGridCell>
        </GridLayout>
      )}
    </>
  );
};
