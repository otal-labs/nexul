import { create } from "zustand";

const MAX_LINES = 40;

// Ephemeral, never persisted: each setup turn's commentary lines as computer.setup_turn_activity frames arrive.
export type SetupActivityStore = {
  lines: Record<string, string[]>;
  push: (turnId: string, line: string) => void;
};

export const useSetupActivityStore = create<SetupActivityStore>((set) => ({
  lines: {},
  push: (turnId, line) =>
    set((s) => ({ lines: { ...s.lines, [turnId]: [...(s.lines[turnId] ?? []), line].slice(-MAX_LINES) } })),
}));
