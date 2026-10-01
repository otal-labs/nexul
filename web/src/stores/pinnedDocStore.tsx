import { create } from "zustand";
import { persist } from "zustand/middleware";

export type PinnedDocStore = {
  // Doc ids per workspace id, newest pin first; ids of docs that are gone are left to the list to skip.
  pinned: Record<string, string[]>;
  togglePin: (workspaceId: string, docId: string) => void;
};

export const usePinnedDocStore = create<PinnedDocStore>()(
  persist(
    (set) => ({
      pinned: {},
      togglePin: (workspaceId, docId) =>
        set((s) => {
          const current = s.pinned[workspaceId] ?? [];
          const next = current.includes(docId) ? current.filter((id) => id !== docId) : [docId, ...current];
          return { pinned: { ...s.pinned, [workspaceId]: next } };
        }),
    }),
    { name: "doc-pins", partialize: (s) => ({ pinned: s.pinned }) },
  ),
);
