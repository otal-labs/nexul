import { create } from "zustand";
import { persist } from "zustand/middleware";

export type BoardStore = {
  // Per project id, lane keys (category names) matching useSwimlanes' merge-by-name lanes; arrays, not Sets, so they JSON-persist.
  collapsedLanes: Record<string, string[]>;
  toggleLane: (projectId: string, key: string) => void;
  // Done is folded to a strip until a browser opens it: finished work is counted, not read, on a board kept open all day.
  doneOpen: boolean;
  toggleDone: () => void;
};

export const useBoardStore = create<BoardStore>()(
  persist(
    (set) => ({
      collapsedLanes: {},
      doneOpen: false,
      toggleDone: () => set((state) => ({ doneOpen: !state.doneOpen })),
      toggleLane: (projectId, key) =>
        set((state) => {
          const current = state.collapsedLanes[projectId] ?? [];
          const next = current.includes(key) ? current.filter((k) => k !== key) : [...current, key];
          return { collapsedLanes: { ...state.collapsedLanes, [projectId]: next } };
        }),
    }),
    { name: "board", partialize: (s) => ({ collapsedLanes: s.collapsedLanes, doneOpen: s.doneOpen }) },
  ),
);
