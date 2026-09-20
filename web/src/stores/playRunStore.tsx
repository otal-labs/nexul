import { create } from "zustand";

import type { PlayType } from "@/models/Play";
import { isTrailActive, mergeLiveStep, type ActivityEntry, type RunFrame } from "@/models/Trail";

export const targetKey = (targetType: PlayType, targetId: string): string => `${targetType}:${targetId}`;

// Ephemeral, never persisted: the latest play.run frame per trail, every step the frames carried while the page
// watched (a frame holds only its newest step), plus which trail is active on each target.
export type PlayRunStore = {
  frames: Record<string, RunFrame>;
  steps: Record<string, ActivityEntry[]>;
  activeByTarget: Record<string, string>;
  applyFrame: (frame: RunFrame) => void;
};

export const usePlayRunStore = create<PlayRunStore>((set) => ({
  frames: {},
  steps: {},
  activeByTarget: {},
  applyFrame: (frame) =>
    set((s) => {
      const key = targetKey(frame.target_type, frame.target_id);
      const activeByTarget = { ...s.activeByTarget };
      if (isTrailActive(frame.state)) activeByTarget[key] = frame.trail_id;
      if (!isTrailActive(frame.state) && activeByTarget[key] === frame.trail_id) delete activeByTarget[key];
      const steps = frame.activity === null ? s.steps : { ...s.steps, [frame.trail_id]: mergeLiveStep(s.steps[frame.trail_id] ?? [], frame.activity) };
      return { frames: { ...s.frames, [frame.trail_id]: frame }, steps, activeByTarget };
    }),
}));
