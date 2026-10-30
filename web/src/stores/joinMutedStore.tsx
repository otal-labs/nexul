import { create } from "zustand";
import { persist } from "zustand/middleware";

export type JoinMutedStore = {
  joinMuted: boolean;
  toggleJoinMuted: () => void;
};

export const useJoinMutedStore = create<JoinMutedStore>()(
  persist(
    (set) => ({
      joinMuted: false,
      toggleJoinMuted: () => set((s) => ({ joinMuted: !s.joinMuted })),
    }),
    { name: "voice-join-muted", partialize: (s) => ({ joinMuted: s.joinMuted }) },
  ),
);
