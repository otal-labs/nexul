import { create } from "zustand";
import { createJSONStorage, persist } from "zustand/middleware";

import { kvStateStorage } from "@/lib/storage";

export type BoardStore = {
  selectedProjectId: string | null;
  // Per project, the statuses the viewer folded (true) or opened (false); one not listed keeps its default.
  folds: Record<string, Record<string, boolean>>;
  selectProject: (id: string) => void;
  setFolded: (projectId: string, statusId: string, folded: boolean) => void;
};

// Remembers the last project the Board tab showed and which of its sections are folded; the project picker is how the project changes.
export const useBoardStore = create<BoardStore>()(
  persist(
    (set) => ({
      selectedProjectId: null,
      folds: {},
      selectProject: (id) => set({ selectedProjectId: id }),
      setFolded: (projectId, statusId, folded) =>
        set(({ folds }) => ({ folds: { ...folds, [projectId]: { ...folds[projectId], [statusId]: folded } } })),
    }),
    { name: "board", storage: createJSONStorage(() => kvStateStorage) },
  ),
);
