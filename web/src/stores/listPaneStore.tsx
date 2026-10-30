import { create } from "zustand";
import { persist } from "zustand/middleware";

export const LIST_PANE_DEFAULT = 360;
export const LIST_PANE_MIN = 220;
export const LIST_PANE_MAX = 560;
const LIST_PANE_OLD_DEFAULT = 300;

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
      version: 1,
      partialize: (s) => ({ width: s.width }),
      // Every browser saved the old default on its first visit, so that width moves to the new one; a dragged width stays.
      migrate: (stored, version) => {
        const width = (stored as Partial<ListPaneStore>)?.width;
        return { width: version === 0 && width === LIST_PANE_OLD_DEFAULT ? LIST_PANE_DEFAULT : width };
      },
      // A hand-edited or stale stored width must not break the layout.
      merge: (stored, current) => ({ ...current, width: clampListPaneWidth((stored as Partial<ListPaneStore>)?.width ?? current.width) }),
    },
  ),
);
