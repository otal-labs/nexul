import { create } from "zustand";

import type { OptionSetting } from "@/models/Pairing";

// The Set up step's unsaved edits for one computer; each field left out shows what the computer has saved.
export interface SetupDraft {
  included?: Record<string, boolean>;
  models?: Record<string, string>;
  options?: Record<string, OptionSetting[]>;
  folder?: string;
}

export type SetupDraftStore = {
  drafts: Record<string, SetupDraft>;
  edit: (computerId: string, change: (draft: SetupDraft) => SetupDraft) => void;
  // Cancel, or a save that made the draft what the computer has.
  discard: (computerId: string) => void;
};

// Not persisted: an unsaved edit lasts until the dialog closes, shared by the step and the dialog's Done button.
export const useSetupDraftStore = create<SetupDraftStore>((set) => ({
  drafts: {},
  edit: (computerId, change) => set((s) => ({ drafts: { ...s.drafts, [computerId]: change(s.drafts[computerId] ?? {}) } })),
  discard: (computerId) =>
    set((s) => ({ drafts: Object.fromEntries(Object.entries(s.drafts).filter(([id]) => id !== computerId)) })),
}));
