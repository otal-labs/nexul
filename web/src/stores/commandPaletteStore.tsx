import { create } from "zustand";

export type CommandPaletteStore = {
  open: boolean;
  setOpen: (open: boolean) => void;
  toggle: () => void;
};

// Shared so the shortcut, the sidebar's trigger and the palette agree on one open state.
export const useCommandPaletteStore = create<CommandPaletteStore>((set) => ({
  open: false,
  setOpen: (open) => set({ open }),
  toggle: () => set((s) => ({ open: !s.open })),
}));
