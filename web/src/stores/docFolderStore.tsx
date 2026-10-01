import { create } from "zustand";
import { persist } from "zustand/middleware";

export type DocFolderStore = {
  // Collapsed folder ids per project id; ids of folders that are gone are left to the list to skip.
  collapsed: Record<string, string[]>;
  toggleCollapsed: (projectId: string, folderId: string) => void;
};

export const useDocFolderStore = create<DocFolderStore>()(
  persist(
    (set) => ({
      collapsed: {},
      toggleCollapsed: (projectId, folderId) =>
        set((s) => {
          const current = s.collapsed[projectId] ?? [];
          const next = current.includes(folderId) ? current.filter((id) => id !== folderId) : [...current, folderId];
          return { collapsed: { ...s.collapsed, [projectId]: next } };
        }),
    }),
    { name: "doc-folders-collapsed", partialize: (s) => ({ collapsed: s.collapsed }) },
  ),
);
