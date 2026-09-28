import { create } from "zustand";
import { createJSONStorage, persist } from "zustand/middleware";

import { kvStateStorage } from "@/lib/storage";

export type WorkspaceStore = {
  selectedWorkspaceId: string;
  selectWorkspace: (id: string) => void;
};

export const useWorkspaceStore = create<WorkspaceStore>()(
  persist(
    (set) => ({
      selectedWorkspaceId: "",
      selectWorkspace: (id) => set({ selectedWorkspaceId: id }),
    }),
    { name: "workspace", storage: createJSONStorage(() => kvStateStorage) },
  ),
);
