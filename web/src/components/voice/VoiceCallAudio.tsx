import { lazy, Suspense } from "react";

import { useVoiceCallStore } from "@/stores/voiceCallStore";

// Code-split like the call view: the LiveKit bundle only loads once a call connects.
const RoomAudioRenderer = lazy(() =>
  import("@livekit/components-react").then((m) => ({ default: m.RoomAudioRenderer })),
);

// Mounted once in the Layout so remote audio plays wherever the user is, not only while the channel's thread view is open.
export const VoiceCallAudio = () => {
  const room = useVoiceCallStore((s) => s.room);

  return (
    room && (
      <Suspense fallback={null}>
        <RoomAudioRenderer room={room} />
      </Suspense>
    )
  );
};
