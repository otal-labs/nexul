import { create } from "zustand";
import { persist } from "zustand/middleware";

import type { DocSortField } from "@/models/Doc";

export type DocSortStore = {
  sortBy: DocSortField;
  setSortBy: (sortBy: DocSortField) => void;
};

export const useDocSortStore = create<DocSortStore>()(
  persist(
    (set) => ({
      sortBy: "created_at",
      setSortBy: (sortBy) => set({ sortBy }),
    }),
    { name: "doc-sort", partialize: (s) => ({ sortBy: s.sortBy }) },
  ),
);
