import { create } from "zustand";

export type DocsProjectStore = {
  selectedProjectId: string | null;
  setSelectedProjectId: (id: string) => void;
};

// Scoped to the Docs list; Board keeps its own pick in boardStore.
export const useDocsProjectStore = create<DocsProjectStore>((set) => ({
  selectedProjectId: null,
  setSelectedProjectId: (id) => set({ selectedProjectId: id }),
}));
