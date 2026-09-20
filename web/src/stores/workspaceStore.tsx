import { create } from "zustand";
import { persist } from "zustand/middleware";

export type WorkspaceStore = {
  selectedWorkspaceId: string;
  selectWorkspace: (id: string) => void;
  // Last-viewed project, so an unscoped /board visit redirects somewhere sensible.
  selectedProjectId: string;
  selectProject: (id: string) => void;
};

// Same convention as useSessionStore: the workspace list is server state, never mirrored here (F5).
export const useWorkspaceStore = create<WorkspaceStore>()(
  persist(
    (set) => ({
      selectedWorkspaceId: "",
      selectWorkspace: (id) => set({ selectedWorkspaceId: id }),
      selectedProjectId: "",
      selectProject: (id) => set({ selectedProjectId: id }),
    }),
    {
      name: "workspace",
      partialize: (state) => ({
        selectedWorkspaceId: state.selectedWorkspaceId,
        selectedProjectId: state.selectedProjectId,
      }),
    },
  ),
);
