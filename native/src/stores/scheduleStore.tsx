import { create } from "zustand";

export type ScheduleStore = {
  // How many times each conversation's composer text went to a schedule; the composer remounts empty on each.
  handedOff: Record<string, number>;
  handOff: (conversationId: string) => void;
};

// Not persisted: a relaunch opens on a plain composer anyway.
export const useScheduleStore = create<ScheduleStore>((set) => ({
  handedOff: {},
  handOff: (conversationId) => set((s) => ({ handedOff: { ...s.handedOff, [conversationId]: (s.handedOff[conversationId] ?? 0) + 1 } })),
}));
