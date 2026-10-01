import { create } from "zustand";
import { persist } from "zustand/middleware";

export const LIST_PANE_DEFAULT = 300;
export const LIST_PANE_MIN = 220;
export const LIST_PANE_MAX = 560;

export const clampListPaneWidth = (width: number) => Math.min(LIST_PANE_MAX, Math.max(LIST_PANE_MIN, Math.round(width)));

export type ListPaneStore = {
  width: number;
  setWidth: (width: number) => void;
  reset: () => void;
};

export const useListPaneStore = create<ListPaneStore>()(
  persist(
    (set) => ({
      width: LIST_PANE_DEFAULT,
      setWidth: (width) => set({ width: clampListPaneWidth(width) }),
      reset: () => set({ width: LIST_PANE_DEFAULT }),
    }),
    {
      name: "list-pane",
      partialize: (s) => ({ width: s.width }),
      // A hand-edited or stale stored width must not break the layout.
      merge: (stored, current) => ({ ...current, width: clampListPaneWidth((stored as Partial<ListPaneStore>)?.width ?? current.width) }),
    },
  ),
);
