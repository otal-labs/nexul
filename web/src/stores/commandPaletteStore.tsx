import { create } from "zustand";

export type CommandPaletteStore = {
  open: boolean;
  // Closed by a pick rather than dismissed, so focus stays with what the pick opened.
  picked: boolean;
  setOpen: (open: boolean) => void;
  toggle: () => void;
  pick: () => void;
};

// Shared so the shortcut, the sidebar's trigger and the palette agree on one open state.
export const useCommandPaletteStore = create<CommandPaletteStore>((set) => ({
  open: false,
  picked: false,
  setOpen: (open) => set({ open, picked: false }),
  toggle: () => set((s) => ({ open: !s.open, picked: false })),
  pick: () => set({ open: false, picked: true }),
}));
