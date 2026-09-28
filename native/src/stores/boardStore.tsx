import { create } from "zustand";
import { createJSONStorage, persist } from "zustand/middleware";

import { kvStateStorage } from "@/lib/storage";

export type BoardStore = {
  selectedProjectId: string | null;
  selectProject: (id: string) => void;
};

// Remembers the last project the Board tab showed; the project picker is how it changes (ticket 11).
export const useBoardStore = create<BoardStore>()(
  persist(
    (set) => ({
      selectedProjectId: null,
      selectProject: (id) => set({ selectedProjectId: id }),
    }),
    { name: "board", storage: createJSONStorage(() => kvStateStorage) },
  ),
);
