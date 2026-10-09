import { useEffect } from "react";
import { useQuery } from "@tanstack/react-query";

import { api } from "@/api/client";
import type { VoiceOccupancy, VoiceOccupant } from "@/models/Voice";
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


// One voice channel's full occupant list after a change, applied wholesale.
interface OccupancyChangedPayload {
  conversation_id: string;
  occupants: VoiceOccupant[];
}

export const voiceFollower: LiveFollower = {
  "voice.occupancy.changed": (p: OccupancyChangedPayload) => useVoiceOccupancyStore.getState().setChannel(p.conversation_id, p.occupants),
};
