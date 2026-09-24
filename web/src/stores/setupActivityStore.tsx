import { create } from "zustand";

const MAX_STEPS = 40;

// One agent step; callId is the tool call it belongs to, empty for a step that is not one.
export interface SetupActivityStep {
  callId: string;
  line: string;
}

// Ephemeral, never persisted: each setup turn's steps as computer.setup_turn_activity frames arrive.
export type SetupActivityStore = {
  steps: Record<string, SetupActivityStep[]>;
  push: (turnId: string, line: string, callId?: string) => void;
};

// A later frame for the same tool call replaces its step, so a call's start and finish read as one line.
const withStep = (steps: SetupActivityStep[], step: SetupActivityStep): SetupActivityStep[] => {
  const at = step.callId ? steps.findIndex((s) => s.callId === step.callId) : -1;
  if (at >= 0) return steps.map((s, i) => (i === at ? step : s));
  return [...steps, step].slice(-MAX_STEPS);
};

export const useSetupActivityStore = create<SetupActivityStore>((set) => ({
  steps: {},
  push: (turnId, line, callId = "") =>
    set((s) => ({ steps: { ...s.steps, [turnId]: withStep(s.steps[turnId] ?? [], { callId, line }) } })),
}));
