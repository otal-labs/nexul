import { create } from "zustand";
import { persist } from "zustand/middleware";

export type VoiceStageView = "grid" | "speaker";

export type VoiceStageViewStore = {
  view: VoiceStageView;
  toggle: () => void;
};

export const useVoiceStageViewStore = create<VoiceStageViewStore>()(
  persist(
    (set) => ({
      view: "grid",
      toggle: () => set((s) => ({ view: s.view === "grid" ? "speaker" : "grid" })),
    }),
    { name: "voice-stage-view", partialize: (s) => ({ view: s.view }) },
  ),
);
