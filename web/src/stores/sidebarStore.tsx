import { create } from "zustand";
import { persist } from "zustand/middleware";

export type SidebarStore = {
  // The icon rail; it is not kept, since the width decides it on load.
  collapsed: boolean;
  toggleCollapsed: () => void;
  workspaceNavOpen: boolean;
  toggleWorkspaceNav: () => void;
  voiceNavOpen: boolean;
  toggleVoiceNav: () => void;
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
      voiceNavOpen: false,
      toggleVoiceNav: () => set((s) => ({ voiceNavOpen: !s.voiceNavOpen })),
      threadsNavOpen: false,
      toggleThreadsNav: () => set((s) => ({ threadsNavOpen: !s.threadsNavOpen })),
    }),
    {
      name: "sidebar",
      // Version 2 folds Voice channels by default; the Workspace and Threads choices a browser made carry over.
      version: 2,
      migrate: (persisted, version) => {
        const kept = persisted as Partial<SidebarStore> | undefined;
        if (version < 1) return { workspaceNavOpen: true, threadsNavOpen: false, voiceNavOpen: false };
        return { workspaceNavOpen: kept?.workspaceNavOpen ?? true, threadsNavOpen: kept?.threadsNavOpen ?? false, voiceNavOpen: false };
      },
      partialize: (s) => ({ workspaceNavOpen: s.workspaceNavOpen, voiceNavOpen: s.voiceNavOpen, threadsNavOpen: s.threadsNavOpen }),
    },
  ),
);
