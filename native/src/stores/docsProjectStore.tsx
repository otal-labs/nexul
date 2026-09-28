import { create } from "zustand";

export type DocsProjectStore = {
  selectedProjectId: string | null;
  setSelectedProjectId: (id: string) => void;
};

// Scoped to the Docs list for now; Board is building its own project picker in parallel.
export const useDocsProjectStore = create<DocsProjectStore>((set) => ({
  selectedProjectId: null,
  setSelectedProjectId: (id) => set({ selectedProjectId: id }),
}));
