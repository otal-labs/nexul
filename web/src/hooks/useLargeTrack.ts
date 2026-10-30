import { isTrackReference, useSpeakingParticipants } from "@livekit/components-react";
import type { TrackReferenceOrPlaceholder } from "@livekit/components-core";
import { useEffect, useMemo, useState } from "react";

import { stepLarge, wantedLarge, type Held, type StagePerson } from "@/lib/stageSpeaker";

// The track to show large: the speaker, held for HOLD_MS so a short noise doesn't move the view.
export const useLargeTrack = (tracks: TrackReferenceOrPlaceholder[]) => {
  const speakers = useSpeakingParticipants().map((p) => p.identity);
  const people = useMemo<StagePerson[]>(
    () => tracks.map((t) => ({ identity: t.participant.identity, camera: isTrackReference(t) && !t.publication.isMuted })),
    [tracks],
  );
  const [held, setHeld] = useState<Held | null>(null);
  const speakerKey = speakers.join();

  useEffect(() => {
    const step = stepLarge(held, people, speakerKey ? speakerKey.split(",") : [], Date.now());
    if (step.next === held) return;
    const timer = setTimeout(() => setHeld(step.next), step.wait);
    return () => clearTimeout(timer);
  }, [held, people, speakerKey]);

  const identity = wantedLarge(people, [], held?.identity);
  return tracks.find((t) => t.participant.identity === identity);
};
