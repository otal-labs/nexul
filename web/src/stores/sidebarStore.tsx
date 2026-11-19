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
      workspaceNavOpen: false,
      toggleWorkspaceNav: () => set((s) => ({ workspaceNavOpen: !s.workspaceNavOpen })),
      threadsNavOpen: true,
      toggleThreadsNav: () => set((s) => ({ threadsNavOpen: !s.threadsNavOpen })),
    }),
    { name: "sidebar", partialize: (s) => ({ workspaceNavOpen: s.workspaceNavOpen, threadsNavOpen: s.threadsNavOpen }) },
  ),
);
