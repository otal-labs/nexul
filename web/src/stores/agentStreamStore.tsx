import { create } from "zustand";

import { mergeHandoff, type Handoff } from "@/models/Handoff";
import type { ActivityKind } from "@/models/Trail";

// Ephemeral, never persisted — the real message lands via chat.message.created and clears this.
export interface AgentStreamFrame {
  messageId: string;
  text: string;
  streaming: boolean;
  // Latest step's summary ("Read main.go started"); empty once text follows it.
  activity: string;
  activityKind?: ActivityKind;
  activityTool?: string;
  // First-frame arrival time, preserved across replaces — feeds the "Working for Xs" counter.
  startedAt: number;
  // Every hand-off the turn's frames carried so far, merged by id across replaces.
  handoffs: Handoff[];
}

export type AgentStreamStore = {
  streams: Record<string, AgentStreamFrame>;
  setStream: (
    conversationId: string,
    frame: Omit<AgentStreamFrame, "startedAt" | "activity" | "handoffs"> & { activity?: string; handoff?: Handoff },
  ) => void;
  clearStream: (conversationId: string) => void;
};

export const useAgentStreamStore = create<AgentStreamStore>((set) => ({
  streams: {},
  setStream: (conversationId, { handoff, ...frame }) =>
    set((s) => {
      const previous = s.streams[conversationId];
      const handoffs = previous?.handoffs ?? [];
      return {
        streams: {
          ...s.streams,
          [conversationId]: {
            ...frame,
            activity: frame.activity ?? "",
            startedAt: previous?.startedAt ?? Date.now(),
            handoffs: handoff ? mergeHandoff(handoffs, handoff) : handoffs,
          },
        },
      };
    }),
  clearStream: (conversationId) =>
    set((s) => {
      if (!(conversationId in s.streams)) return s;
      const rest = { ...s.streams };
      delete rest[conversationId];
      return { streams: rest };
    }),
}));
