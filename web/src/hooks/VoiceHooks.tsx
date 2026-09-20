import { useEffect } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { api, errorMessage } from "@/api/client";
import { getChatConversationsKey } from "@/hooks/ChatHooks";
import type { Conversation } from "@/models/Chat";
import type { VoiceOccupancy } from "@/models/Voice";
import { useVoiceOccupancyStore } from "@/stores/voiceOccupancyStore";

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

export const useCreateVoiceChannel = (workspaceId: string) => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async (name: string) =>
      (await api.post<Conversation>("/api/chat/voice-channels", { workspace_id: workspaceId, name })).data,
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: [getChatConversationsKey, workspaceId] });
      toast.success("Voice channel created");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};
