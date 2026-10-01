import { create } from "zustand";

import { mergeLiveStep, type ActivityEntry } from "@/models/Trail";

const MAX_STEPS = 200;

// Ephemeral, never persisted: each setup turn's steps as computer.setup_turn_activity frames arrive.
export type SetupActivityStore = {
  steps: Record<string, ActivityEntry[]>;
  push: (turnId: string, step: ActivityEntry) => void;
};

// A later frame for the same tool call replaces its step, so a call's start and finish read as one row.
export const useSetupActivityStore = create<SetupActivityStore>((set) => ({
  steps: {},
  push: (turnId, step) => set((s) => ({ steps: { ...s.steps, [turnId]: mergeLiveStep(s.steps[turnId] ?? [], step).slice(-MAX_STEPS) } })),
}));
