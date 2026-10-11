import { create } from "zustand";

export type PermissionPanelMode = "rules" | "check";

export type PermissionPanelStore = {
  mode: PermissionPanelMode;
  // The person whose rules are open, and the person Check access reads.
  target: string | null;
  checked: string | null;
  // People listed by the add control before they hold any rule here.
  added: string[];
  reset: () => void;
  setMode: (mode: PermissionPanelMode) => void;
  pick: (target: string | null) => void;
  check: (person: string) => void;
  add: (person: string) => void;
  drop: (person: string) => void;
};

// The panel's Rules | Check access sits in the dialog's header and the view in its body, two trees that share this.
export const usePermissionPanelStore = create<PermissionPanelStore>((set) => ({
  mode: "rules",
  target: null,
  checked: null,
  added: [],
  reset: () => set({ mode: "rules", target: null, checked: null, added: [] }),
  setMode: (mode) => set((s) => ({ mode, checked: s.checked ?? s.target })),
  pick: (target) => set({ target }),
  check: (checked) => set({ checked }),
  add: (person) => set((s) => ({ added: [...s.added, person], target: person, mode: "rules" })),
  drop: (person) => set((s) => ({ added: s.added.filter((p) => p !== person), target: null })),
}));
