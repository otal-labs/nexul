import { create } from "zustand";
import { createJSONStorage, persist } from "zustand/middleware";

export interface ViewAsTarget {
  workspaceId: string;
  workspaceName: string;
  userId: string;
  name: string;
  avatarUrl: string;
}

export type ViewAsStore = {
  target: ViewAsTarget | null;
  start: (target: ViewAsTarget) => void;
  exit: () => void;
};

// Session storage keeps the mode through a reload in this tab and never carries it into another one.
export const useViewAsStore = create<ViewAsStore>()(
  persist(
    (set) => ({
      target: null,
      start: (target) => set({ target }),
      exit: () => set({ target: null }),
    }),
    {
      name: "view-as",
      storage: createJSONStorage(() => sessionStorage),
      partialize: (state) => ({ target: state.target }),
    },
  ),
);

export const useIsViewingAs = () => useViewAsStore((s) => s.target !== null);

// A personal route answers 404 in a view, so the queries that read one stay off rather than repeat the request.
export const notViewing = () => useViewAsStore.getState().target === null;
