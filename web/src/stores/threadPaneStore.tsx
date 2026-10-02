import { create } from "zustand";
import { persist } from "zustand/middleware";

export const THREAD_PANE_MIN = 288;
export const THREAD_PANE_MAX = 640;

export const clampThreadPaneWidth = (width: number) =>
  Math.min(THREAD_PANE_MAX, Math.max(THREAD_PANE_MIN, Math.round(width)));

// null until the first drag: the column keeps its share of the page until someone picks a width.
export type ThreadPaneStore = {
  width: number | null;
  setWidth: (width: number) => void;
  reset: () => void;
};

export const useThreadPaneStore = create<ThreadPaneStore>()(
  persist(
    (set) => ({
      width: null,
      setWidth: (width) => set({ width: clampThreadPaneWidth(width) }),
      reset: () => set({ width: null }),
    }),
    {
      name: "thread-pane",
      partialize: (s) => ({ width: s.width }),
      merge: (stored, current) => {
        const width = (stored as Partial<ThreadPaneStore>)?.width;
        return { ...current, width: typeof width === "number" ? clampThreadPaneWidth(width) : null };
      },
    },
  ),
);
