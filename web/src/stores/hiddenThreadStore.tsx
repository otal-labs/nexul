import { create } from "zustand";
import { persist } from "zustand/middleware";

export type HiddenThreadStore = {
  // Thread conversation ids removed from the sidebar, per workspace id; the threads themselves are untouched.
  hidden: Record<string, string[]>;
  toggleHidden: (workspaceId: string, conversationId: string) => void;
};

export const useHiddenThreadStore = create<HiddenThreadStore>()(
  persist(
    (set) => ({
      hidden: {},
      toggleHidden: (workspaceId, conversationId) =>
        set((s) => {
          const current = s.hidden[workspaceId] ?? [];
          const next = current.includes(conversationId) ? current.filter((id) => id !== conversationId) : [...current, conversationId];
          return { hidden: { ...s.hidden, [workspaceId]: next } };
        }),
    }),
    { name: "sidebar-hidden-threads", partialize: (s) => ({ hidden: s.hidden }) },
  ),
);
