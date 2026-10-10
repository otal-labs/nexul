import { create } from "zustand";
import { persist } from "zustand/middleware";

export type UnfinishedProjectStore = {
  projectId: string | null;
  start: (projectId: string) => void;
  dismiss: () => void;
};

// localStorage, not sessionStorage: installing the GitHub App returns in a new tab, and the way back must be there.
export const useUnfinishedProjectStore = create<UnfinishedProjectStore>()(
  persist(
    (set) => ({
      projectId: null,
      start: (projectId) => set({ projectId }),
      dismiss: () => set({ projectId: null }),
    }),
    { name: "unfinished-project" },
  ),
);
