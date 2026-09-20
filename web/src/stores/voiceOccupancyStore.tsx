import { create } from "zustand";

import type { VoiceOccupancy, VoiceOccupant } from "@/models/Voice";

// Ephemeral, never persisted — seeded once, then kept live by frames carrying the full replacement list.
export type VoiceOccupancyStore = {
  occupancy: VoiceOccupancy;
  hydrate: (occupancy: VoiceOccupancy) => void;
  setChannel: (conversationId: string, occupants: VoiceOccupant[]) => void;
};

export const useVoiceOccupancyStore = create<VoiceOccupancyStore>((set) => ({
  occupancy: {},
  hydrate: (occupancy) => set({ occupancy }),
  setChannel: (conversationId, occupants) =>
    set((s) => {
      // An empty room is absent, never a zero-length entry (mirrors internal/voice/occupancy.go's store).
      if (occupants.length === 0) {
        if (!(conversationId in s.occupancy)) return s;
        const rest = { ...s.occupancy };
        delete rest[conversationId];
        return { occupancy: rest };
      }
      return { occupancy: { ...s.occupancy, [conversationId]: occupants } };
    }),
}));
