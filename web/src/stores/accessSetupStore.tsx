import { create } from "zustand";
import { persist } from "zustand/middleware";

// The workspaces whose home has already sent this Owner to the one-time access setup, which it does once.
export type AccessSetupStore = {
  redirected: string[];
  markRedirected: (workspaceId: string) => void;
};

export const useAccessSetupStore = create<AccessSetupStore>()(
  persist(
    (set) => ({
      redirected: [],
      markRedirected: (workspaceId) => set((s) => ({ redirected: [...s.redirected.filter((id) => id !== workspaceId), workspaceId] })),
    }),
    { name: "access-setup", partialize: (s) => ({ redirected: s.redirected }) },
  ),
);
