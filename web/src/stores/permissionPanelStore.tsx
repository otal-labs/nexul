import { createContext, useContext } from "react";
import { createStore, useStore, type StoreApi } from "zustand";

export type PermissionPanelMode = "rules" | "check";

export type PermissionPanelStore = {
  mode: PermissionPanelMode;
  // The target whose rules are open (a targetKey), and the person Check access reads.
  target: string | null;
  checked: string | null;
  // Targets listed by the add control before they hold any rule here.
  added: string[];
  setMode: (mode: PermissionPanelMode) => void;
  pick: (target: string | null) => void;
  check: (person: string) => void;
  add: (target: string) => void;
  drop: (target: string) => void;
};

// One per open dialog, so a panel opened from a link in another stacks on it without sharing its state.
export const createPermissionPanelStore = (target: string | null = null) =>
  createStore<PermissionPanelStore>((set) => ({
    mode: "rules",
    target,
    checked: null,
    added: target ? [target] : [],
    setMode: (mode) => set((s) => ({ mode, checked: s.checked ?? (s.target?.startsWith("person:") ? s.target.slice("person:".length) : null) })),
    pick: (next) => set({ target: next }),
    check: (checked) => set({ checked }),
    add: (next) => set((s) => ({ added: s.added.includes(next) ? s.added : [...s.added, next], target: next, mode: "rules" })),
    drop: (gone) => set((s) => ({ added: s.added.filter((t) => t !== gone), target: s.target === gone ? null : s.target })),
  }));

// The panel's Rules | Check access sits in the dialog's header and the view in its body, two trees that share this.
export const PermissionPanelStoreContext = createContext<StoreApi<PermissionPanelStore> | null>(null);

export const usePermissionPanelStore = <T,>(selector: (s: PermissionPanelStore) => T): T => {
  const store = useContext(PermissionPanelStoreContext);
  if (!store) throw new Error("usePermissionPanelStore must be used inside the permissions panel");
  return useStore(store, selector);
};
