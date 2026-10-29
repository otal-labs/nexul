import { create } from "zustand";

export type InterviewSourceFocusStore = {
  sourceId: string | null;
  setSourceId: (sourceId: string | null) => void;
};

// The source a draft's chip is hovered or focused on, so its row in the sources rail can light up beside it.
export const useInterviewSourceFocusStore = create<InterviewSourceFocusStore>((set) => ({
  sourceId: null,
  setSourceId: (sourceId) => set({ sourceId }),
}));
