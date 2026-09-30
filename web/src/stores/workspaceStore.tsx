import { create } from "zustand";
import { persist } from "zustand/middleware";

export type WorkspaceStore = {
  selectedWorkspaceId: string;
  // The selected workspace's slug, so links into it build without waiting for the workspace list.
  selectedWorkspaceSlug: string;
  selectWorkspace: (id: string, slug: string) => void;
  // Last-viewed project, so an unscoped /board visit redirects somewhere sensible.
  selectedProjectId: string;
  selectProject: (id: string) => void;
};

// Same convention as useSessionStore: the workspace list is server state, never mirrored here (F5).
export const useWorkspaceStore = create<WorkspaceStore>()(
  persist(
    (set) => ({
      selectedWorkspaceId: "",
      selectedWorkspaceSlug: "",
      selectWorkspace: (id, slug) => set({ selectedWorkspaceId: id, selectedWorkspaceSlug: slug }),
      selectedProjectId: "",
      selectProject: (id) => set({ selectedProjectId: id }),
    }),
    {
      name: "workspace",
      partialize: (state) => ({
        selectedWorkspaceId: state.selectedWorkspaceId,
        selectedWorkspaceSlug: state.selectedWorkspaceSlug,
        selectedProjectId: state.selectedProjectId,
      }),
    },
  ),
);
