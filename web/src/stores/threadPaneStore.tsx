import { create } from "zustand";
import { persist } from "zustand/middleware";

export const THREAD_PANE_MIN = 288;
export const THREAD_PANE_MAX = 640;

export const clampThreadPaneWidth = (width: number) =>
  Math.min(THREAD_PANE_MAX, Math.max(THREAD_PANE_MIN, Math.round(width)));

const WEEK_MS = 7 * 24 * 60 * 60 * 1000;

export type ThreadPaneWidth = { width: number; at: number };

// Keyed by ticket id; a ticket without an entry keeps the column's share of the page until someone picks a width.
export type ThreadPaneStore = {
  widths: Record<string, ThreadPaneWidth>;
  setWidth: (ticketId: string, width: number) => void;
  reset: (ticketId: string) => void;
};

const isWidth = (value: unknown): value is ThreadPaneWidth =>
  typeof value === "object" &&
  value !== null &&
  typeof (value as ThreadPaneWidth).width === "number" &&
  typeof (value as ThreadPaneWidth).at === "number";

const readWidths = (stored: unknown): Record<string, ThreadPaneWidth> => {
  if (typeof stored !== "object" || stored === null) return {};
  const entries = Object.entries(stored).filter((entry): entry is [string, ThreadPaneWidth] => isWidth(entry[1]));
  return Object.fromEntries(entries.map(([id, { width, at }]) => [id, { width: clampThreadPaneWidth(width), at }]));
};

// Pruned on every adjustment so tickets resized once and never revisited do not pile up in the browser.
const freshWidths = (widths: Record<string, ThreadPaneWidth>, exceptId: string, now: number) =>
  Object.fromEntries(Object.entries(widths).filter(([id, { at }]) => id !== exceptId && now - at < WEEK_MS));

export const useThreadPaneStore = create<ThreadPaneStore>()(
  persist(
    (set) => ({
      widths: {},
      setWidth: (ticketId, width) =>
        set((s) => {
          const now = Date.now();
          return { widths: { ...freshWidths(s.widths, ticketId, now), [ticketId]: { width: clampThreadPaneWidth(width), at: now } } };
        }),
      reset: (ticketId) => set((s) => ({ widths: freshWidths(s.widths, ticketId, Date.now()) })),
    }),
    {
      name: "thread-pane",
      partialize: (s) => ({ widths: s.widths }),
      merge: (stored, current) => ({ ...current, widths: readWidths((stored as Partial<ThreadPaneStore>)?.widths) }),
    },
  ),
);
