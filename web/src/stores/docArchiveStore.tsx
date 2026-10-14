import { create } from "zustand";
import { persist } from "zustand/middleware";

export type DocArchiveStore = {
  showArchived: boolean;
  setShowArchived: (showArchived: boolean) => void;
};

export const useDocArchiveStore = create<DocArchiveStore>()(
  persist(
    (set) => ({
      showArchived: false,
      setShowArchived: (showArchived) => set({ showArchived }),
    }),
    { name: "doc-archive", partialize: (s) => ({ showArchived: s.showArchived }) },
  ),
);
