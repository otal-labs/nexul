import { create } from "zustand";

export type ServerUpdateStore = {
  pendingVersion: string | null;
  setPending: (version: string) => void;
  dismiss: () => void;
};

export const useServerUpdateStore = create<ServerUpdateStore>()((set) => ({
  pendingVersion: null,
  setPending: (version) => set({ pendingVersion: version }),
  dismiss: () => set({ pendingVersion: null }),
}));
