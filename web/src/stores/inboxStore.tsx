import { create } from "zustand";

export type InboxStore = {
  selectedKey: string | null;
  select: (key: string) => void;
};

export const useInboxStore = create<InboxStore>((set) => ({
  selectedKey: null,
  select: (key) => set({ selectedKey: key }),
}));
