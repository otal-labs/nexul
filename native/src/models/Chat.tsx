import type { ChatMessage, Conversation } from "@nexul/client-core/chat";

export interface Message extends ChatMessage {
  // Only an Agent reply that handed work to other agents carries it (ADR 0116).
  handoffs?: Handoff[];
}

// Mirrors internal/harness's Handoff states; left_running is work still running in T3 Code when the reply stopped waiting.
export type HandoffState = "running" | "done" | "failed" | "interrupted" | "left_running";

// Mirrors internal/chat.HandoffStep, one step of the handed-off agent's conversation.
export interface HandoffStep {
  kind: "tool_call" | "tool_result" | "text" | "question" | "other" | "note";
  call_id?: string;
  tool?: string;
  summary: string;
  detail?: string;
  at: string;
}

// Mirrors internal/chat.Handoff: the work one helper agent did for an Agent reply.
export interface Handoff {
  id: string;
  driver: string;
  model: string;
  title: string;
  prompt: string;
  state: HandoffState;
  reply: string;
  steps: HandoffStep[];
}

const HANDOFF_STATE_LABEL: Record<HandoffState, string> = {
  running: "Running",
  done: "Done",
  failed: "Failed",
  interrupted: "Interrupted",
  left_running: "Left running",
};

const HANDOFF_STATE_DOT: Record<HandoffState, string> = {
  running: "bg-warning",
  done: "bg-success",
  failed: "bg-destructive",
  interrupted: "bg-muted-foreground",
  left_running: "bg-info",
};

// The server can be newer than the app, so a state it does not know yet still reads and gets the in-flight dot.
export const handoffStateLabel = (state: HandoffState): string => HANDOFF_STATE_LABEL[state] ?? state;

export const handoffStateDot = (state: HandoffState): string => HANDOFF_STATE_DOT[state] ?? HANDOFF_STATE_DOT.running;

export interface ConversationGroup {
  title: string;
  conversations: Conversation[];
}

// Voice needs a call UI the phone does not have, and interview and channel threads are reached from their parent on the web.
export const groupConversations = (conversations: Conversation[]): ConversationGroup[] =>
  [
    { title: "Channels", conversations: conversations.filter((c) => c.kind === "channel") },
    { title: "Direct messages", conversations: conversations.filter((c) => c.kind === "dm") },
    {
      title: "Threads",
      conversations: conversations.filter((c) => c.kind === "ticket_thread" || c.kind === "doc_thread"),
    },
  ].filter((group) => group.conversations.length > 0);
