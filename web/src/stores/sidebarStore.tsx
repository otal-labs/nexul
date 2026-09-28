import { create } from "zustand";
import { persist } from "zustand/middleware";

export type SidebarStore = {
  workspaceNavOpen: boolean;
  toggleWorkspaceNav: () => void;
};

export const useSidebarStore = create<SidebarStore>()(
  persist(
    (set) => ({
      workspaceNavOpen: true,
      toggleWorkspaceNav: () => set((s) => ({ workspaceNavOpen: !s.workspaceNavOpen })),
    }),
    { name: "sidebar", partialize: (s) => ({ workspaceNavOpen: s.workspaceNavOpen }) },
  ),
);
