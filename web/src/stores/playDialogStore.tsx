import { create } from "zustand";

export type PlayDialogTab = "play" | "auto";

export type PlayDialogStore = {
  tab: PlayDialogTab;
  setTab: (tab: PlayDialogTab) => void;
  // Set while an auto play's composer is open: answers whether its edits may be dropped to show another tab.
  leaveGuard: (() => Promise<boolean>) | null;
  setLeaveGuard: (guard: (() => Promise<boolean>) | null) => void;
};

// The edit dialog's tab row sits in its header and the tab's content in its body, two trees that share this one value.
export const usePlayDialogStore = create<PlayDialogStore>((set) => ({
  tab: "play",
  setTab: (tab) => set({ tab }),
  leaveGuard: null,
  setLeaveGuard: (leaveGuard) => set({ leaveGuard }),
}));
