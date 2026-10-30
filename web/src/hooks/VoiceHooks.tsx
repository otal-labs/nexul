import { useEffect, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import type { RemoteParticipant, RemoteTrackPublication, Room } from "livekit-client";

import { requires } from "@nexul/client-core/requires";

import { api } from "@/api/client";
import type { VoiceJoinToken, VoiceOccupancy, VoiceOccupant } from "@/models/Voice";
import { useVoiceOccupancyStore } from "@/stores/voiceOccupancyStore";
import type { LiveFollower } from "@/lib/live";

export const getVoiceOccupancyKey = "getVoiceOccupancy";

const useFetchVoiceOccupancy = (enabled: boolean) =>
  useQuery({
    queryKey: [getVoiceOccupancyKey],
    queryFn: async () => (await api.get<VoiceOccupancy>("/api/voice/occupancy")).data,
    enabled,
  });

// Seeds voiceOccupancyStore once, then it stays live off voice.occupancy.changed frames — no refetching after.
export const useVoiceOccupancy = (enabled = true): VoiceOccupancy => {
  const { data } = useFetchVoiceOccupancy(enabled);
  const hydrate = useVoiceOccupancyStore((s) => s.hydrate);
  const occupancy = useVoiceOccupancyStore((s) => s.occupancy);
  useEffect(() => {
    if (data) hydrate(data);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [data]);
  return occupancy;
};

type LiveKit = typeof import("livekit-client");

const NOBODY: ReadonlySet<string> = new Set();

// The lobby's one hidden listener; starting another, or leaving the lobby, disconnects this one first.
let stopListener: (() => void) | undefined;

// LiveKit sends a speaker update only for someone you are subscribed to, so the listener subscribes to each audio
// track and switches it off at once: the server then forwards no media, and video is never subscribed.
const holdAudioSilently = (livekit: LiveKit, held: Set<string>, publication: RemoteTrackPublication) => {
  if (publication.kind !== livekit.Track.Kind.Audio || held.has(publication.trackSid)) return;
  held.add(publication.trackSid);
  publication.setSubscribed(true);
  publication.setEnabled(false);
};

const connectListener = async (conversationId: string, onSpeakers: (speaking: ReadonlySet<string>) => void, isStopped: () => boolean) => {
  const [{ data }, livekit] = await Promise.all([
    requires("none", api.post<VoiceJoinToken>(`/api/voice/${conversationId}/listen`)),
    import("livekit-client"),
  ]);
  if (isStopped()) return undefined;
  livekit.setLogLevel("warn");
  const room = new livekit.Room();
  const held = new Set<string>();
  const holdAll = (participant: RemoteParticipant) =>
    participant.trackPublications.forEach((publication) => holdAudioSilently(livekit, held, publication));
  room.on(livekit.RoomEvent.ParticipantConnected, holdAll);
  room.on(livekit.RoomEvent.TrackPublished, (publication) => holdAudioSilently(livekit, held, publication));
  room.on(livekit.RoomEvent.ActiveSpeakersChanged, (speakers) => onSpeakers(new Set(speakers.map((s) => s.identity))));
  await room.connect(data.ws_url, data.token, { autoSubscribe: false });
  room.remoteParticipants.forEach(holdAll);
  return room;
};

// Who is speaking in a room you have not joined, read through a hidden listener that publishes nothing and is never
// presence (ADR 0175). A failed listen shows no rings and no error.
export const useVoiceLobbySpeakers = (conversationId: string, listening: boolean): ReadonlySet<string> => {
  const [speaking, setSpeaking] = useState(NOBODY);
  useEffect(() => {
    if (!listening) return;
    stopListener?.();
    let stopped = false;
    let room: Room | undefined;
    const stop = () => {
      stopped = true;
      room?.disconnect();
      setSpeaking(NOBODY);
    };
    stopListener = stop;
    connectListener(conversationId, setSpeaking, () => stopped)
      .then((connected) => {
        room = connected;
        if (stopped) room?.disconnect();
      })
      .catch(() => undefined);
    return () => {
      stop();
      if (stopListener === stop) stopListener = undefined;
    };
  }, [conversationId, listening]);
  return listening ? speaking : NOBODY;
};

// One voice channel's full occupant list after a change, applied wholesale.
interface OccupancyChangedPayload {
  conversation_id: string;
  occupants: VoiceOccupant[];
}

export const voiceFollower: LiveFollower = {
  "voice.occupancy.changed": (p: OccupancyChangedPayload) => useVoiceOccupancyStore.getState().setChannel(p.conversation_id, p.occupants),
};
