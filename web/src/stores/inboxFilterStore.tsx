import { create } from "zustand";
import { persist } from "zustand/middleware";

export type InboxFilter = "unread" | "all";

export type InboxFilterStore = {
  filter: InboxFilter;
  setFilter: (filter: InboxFilter) => void;
};

export const useInboxFilterStore = create<InboxFilterStore>()(
  persist(
    (set) => ({
      filter: "all",
      setFilter: (filter) => set({ filter }),
    }),
    { name: "inbox-filter", partialize: (s) => ({ filter: s.filter }) },
  ),
);
