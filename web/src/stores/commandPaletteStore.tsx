import { create } from "zustand";

export type CommandPaletteStore = {
  open: boolean;
  // Closed by a pick rather than dismissed, so focus stays with what the pick opened.
  picked: boolean;
  // Opened or dismissed from the keyboard, so it shows and goes without motion.
  keyed: boolean;
  setOpen: (open: boolean, keyed?: boolean) => void;
  toggle: () => void;
  pick: () => void;
};

// Shared so the shortcut, the sidebar's trigger and the palette agree on one open state.
export const useCommandPaletteStore = create<CommandPaletteStore>((set) => ({
  open: false,
  picked: false,
  keyed: false,
  setOpen: (open, keyed = false) => set({ open, picked: false, keyed }),
  toggle: () => set((s) => ({ open: !s.open, picked: false, keyed: true })),
  pick: () => set({ open: false, picked: true }),
}));
