import type { HandoffState } from "@nexul/client-core/handoffs";

import type { ActivityEntry } from "@/models/Trail";

// Mirrors internal/chat.Handoff: work an Agent reply handed to another agent, its steps shaped like a trail's.
export interface Handoff {
  id: string;
  driver: string;
  model: string;
  title: string;
  prompt: string;
  state: HandoffState;
  reply: string;
  steps: ActivityEntry[];
}

export const HANDOFF_STATE_LABELS: Record<HandoffState, string> = {
  running: "Running",
  done: "Done",
  failed: "Failed",
  interrupted: "Interrupted",
  left_running: "Left running",
};

// A frame carries one whole hand-off: it replaces the one sharing its id, else joins the end in the order they started.
export const mergeHandoff = (handoffs: Handoff[], next: Handoff): Handoff[] => {
  if (!handoffs.some((h) => h.id === next.id)) return [...handoffs, next];
  return handoffs.map((h) => (h.id === next.id ? next : h));
};
