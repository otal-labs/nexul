import { create } from "zustand";
import { persist } from "zustand/middleware";

export type SidebarStore = {
  workspaceNavOpen: boolean;
  toggleWorkspaceNav: () => void;
  threadsNavOpen: boolean;
  toggleThreadsNav: () => void;
};

export const useSidebarStore = create<SidebarStore>()(
  persist(
    (set) => ({
      workspaceNavOpen: false,
      toggleWorkspaceNav: () => set((s) => ({ workspaceNavOpen: !s.workspaceNavOpen })),
      threadsNavOpen: true,
      toggleThreadsNav: () => set((s) => ({ threadsNavOpen: !s.threadsNavOpen })),
    }),
    { name: "sidebar", partialize: (s) => ({ workspaceNavOpen: s.workspaceNavOpen, threadsNavOpen: s.threadsNavOpen }) },
  ),
);
