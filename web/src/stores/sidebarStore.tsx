import { create } from "zustand";
import { persist } from "zustand/middleware";

export type SidebarStore = {
  // The icon rail; it is not kept, since the width decides it on load.
  collapsed: boolean;
  toggleCollapsed: () => void;
  workspaceNavOpen: boolean;
  toggleWorkspaceNav: () => void;
  threadsNavOpen: boolean;
  toggleThreadsNav: () => void;
};

export const useSidebarStore = create<SidebarStore>()(
  persist(
    (set) => ({
      // Below 1280px an open sidebar leaves a list pane and its open record too little width, so it starts as the icon rail.
      collapsed: window.innerWidth < 1280,
      toggleCollapsed: () => set((s) => ({ collapsed: !s.collapsed })),
      workspaceNavOpen: true,
      toggleWorkspaceNav: () => set((s) => ({ workspaceNavOpen: !s.workspaceNavOpen })),
      threadsNavOpen: false,
      toggleThreadsNav: () => set((s) => ({ threadsNavOpen: !s.threadsNavOpen })),
    }),
    {
      name: "sidebar",
      // Version 1 opened Workspace and folded Threads by default, so every browser starts once from the new defaults.
      version: 1,
      migrate: () => ({ workspaceNavOpen: true, threadsNavOpen: false }),
      partialize: (s) => ({ workspaceNavOpen: s.workspaceNavOpen, threadsNavOpen: s.threadsNavOpen }),
    },
  ),
);
