import { create } from "zustand";
import { persist } from "zustand/middleware";

export type InboxFolderStore = {
  // Keys of the inbox folder groups collapsed in this browser; every group starts expanded.
  collapsed: string[];
  toggleCollapsed: (key: string) => void;
};

export const useInboxFolderStore = create<InboxFolderStore>()(
  persist(
    (set) => ({
      collapsed: [],
      toggleCollapsed: (key) =>
        set((s) => ({
          collapsed: s.collapsed.includes(key) ? s.collapsed.filter((k) => k !== key) : [...s.collapsed, key],
        })),
    }),
    { name: "inbox-folders-collapsed", partialize: (s) => ({ collapsed: s.collapsed }) },
  ),
);
