import { create } from "zustand";

import { mergeHandoff, type Handoff } from "@/models/Handoff";
import { mergeLiveStep, type ActivityEntry, type ActivityKind } from "@/models/Trail";

// Ephemeral, never persisted — the real message lands via chat.message.created and clears this.
export interface AgentStreamFrame {
  messageId: string;
  text: string;
  streaming: boolean;
  // Latest step's summary ("Read main.go started"); empty once text follows it.
  activity: string;
  activityKind?: ActivityKind;
  activityTool?: string;
  activityCallId?: string;
  // First-frame arrival time, preserved across replaces — feeds the "Working for Xs" counter.
  startedAt: number;
  // Every hand-off the turn's frames carried so far, merged by id across replaces.
  handoffs: Handoff[];
  // The turn so far as trail steps: each piece of text written between tool calls, and each tool call, its result replacing it.
  steps: ActivityEntry[];
  // Where the open text step starts in the current message's text; a tool call moves it to the text's end.
  textFrom: number;
}

type StreamUpdate = Omit<AgentStreamFrame, "startedAt" | "activity" | "handoffs" | "steps" | "textFrom"> & {
  activity?: string;
  handoff?: Handoff;
};

export type AgentStreamStore = {
  streams: Record<string, AgentStreamFrame>;
  // The ids of the messages whose Agent turn waits in the harness's queue behind another turn.
  queued: Record<string, true>;
  setStream: (conversationId: string, frame: StreamUpdate) => void;
  clearStream: (conversationId: string) => void;
  setQueued: (messageId: string, queued: boolean) => void;
};

const nextSteps = (previous: AgentStreamFrame | undefined, frame: StreamUpdate): Pick<AgentStreamFrame, "steps" | "textFrom"> => {
  const steps = previous?.steps ?? [];
  const textFrom = previous?.messageId === frame.messageId && previous.textFrom <= frame.text.length ? previous.textFrom : 0;
  const at = new Date().toISOString();
  if (frame.activityKind === "tool_call" || frame.activityKind === "tool_result") {
    const step: ActivityEntry = {
      kind: frame.activityKind,
      ...(frame.activityCallId && { call_id: frame.activityCallId }),
      ...(frame.activityTool && { tool: frame.activityTool }),
      summary: frame.activity ?? "",
      at,
    };
    return { steps: mergeLiveStep(steps, step), textFrom: frame.activityKind === "tool_call" ? frame.text.length : textFrom };
  }
  const piece = frame.text.slice(textFrom).trim();
  if (piece === "") return { steps, textFrom };
  return { steps: mergeLiveStep(steps, { kind: "text", call_id: `text:${frame.messageId}:${textFrom}`, summary: piece, detail: piece, at }), textFrom };
};

export const useAgentStreamStore = create<AgentStreamStore>((set) => ({
  streams: {},
  queued: {},
  setStream: (conversationId, update) =>
    set((s) => {
      const { handoff, ...frame } = update;
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
            ...nextSteps(previous, update),
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
  setQueued: (messageId, queued) =>
    set((s) => {
      if (queued === (messageId in s.queued)) return s;
      if (queued) return { queued: { ...s.queued, [messageId]: true } };
      const rest = { ...s.queued };
      delete rest[messageId];
      return { queued: rest };
    }),
}));
