import { create } from "zustand";
import { persist } from "zustand/middleware";

export const THREAD_PANE_MIN = 288;
export const THREAD_PANE_MAX = 640;

export const clampThreadPaneWidth = (width: number) =>
  Math.min(THREAD_PANE_MAX, Math.max(THREAD_PANE_MIN, Math.round(width)));

// Keyed by ticket id; a ticket without an entry keeps the column's share of the page until someone picks a width.
export type ThreadPaneStore = {
  widths: Record<string, number>;
  setWidth: (ticketId: string, width: number) => void;
  reset: (ticketId: string) => void;
};

const readWidths = (stored: unknown): Record<string, number> => {
  if (typeof stored !== "object" || stored === null) return {};
  const entries = Object.entries(stored).filter((entry): entry is [string, number] => typeof entry[1] === "number");
  return Object.fromEntries(entries.map(([id, width]) => [id, clampThreadPaneWidth(width)]));
};

export const useThreadPaneStore = create<ThreadPaneStore>()(
  persist(
    (set) => ({
      widths: {},
      setWidth: (ticketId, width) => set((s) => ({ widths: { ...s.widths, [ticketId]: clampThreadPaneWidth(width) } })),
      reset: (ticketId) =>
        set((s) => ({ widths: Object.fromEntries(Object.entries(s.widths).filter(([id]) => id !== ticketId)) })),
    }),
    {
      name: "thread-pane",
      partialize: (s) => ({ widths: s.widths }),
      merge: (stored, current) => ({ ...current, widths: readWidths((stored as Partial<ThreadPaneStore>)?.widths) }),
    },
  ),
);
