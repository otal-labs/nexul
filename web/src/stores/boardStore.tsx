import { create } from "zustand";
import { persist } from "zustand/middleware";

export type BoardStore = {
  // Lane keys (category names), matching useSwimlanes' merge-by-name lanes; an array, not a Set, so it JSON-persists.
  collapsedLaneKeys: string[];
  toggleLane: (key: string) => void;
};

export const useBoardStore = create<BoardStore>()(
  persist(
    (set) => ({
      collapsedLaneKeys: [],
      toggleLane: (key) =>
        set((state) => ({
          collapsedLaneKeys: state.collapsedLaneKeys.includes(key)
            ? state.collapsedLaneKeys.filter((k) => k !== key)
            : [...state.collapsedLaneKeys, key],
        })),
    }),
    { name: "board" },
  ),
);
