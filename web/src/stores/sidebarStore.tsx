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
      // At 768px an open sidebar leaves the page too little width, so it starts as the icon rail below 1024px.
      collapsed: window.innerWidth < 1024,
      toggleCollapsed: () => set((s) => ({ collapsed: !s.collapsed })),
      workspaceNavOpen: false,
      toggleWorkspaceNav: () => set((s) => ({ workspaceNavOpen: !s.workspaceNavOpen })),
      threadsNavOpen: true,
      toggleThreadsNav: () => set((s) => ({ threadsNavOpen: !s.threadsNavOpen })),
    }),
    { name: "sidebar", partialize: (s) => ({ workspaceNavOpen: s.workspaceNavOpen, threadsNavOpen: s.threadsNavOpen }) },
  ),
);
