import { turnStartMs, type AgentStep } from "@nexul/client-core/agentSteps";

// Mirrors internal/harness's Handoff states; left_running is work still running when the turn stopped waiting for it.
export type HandoffState = "running" | "done" | "failed" | "interrupted" | "left_running";

// What both apps read of a hand-off to sum several up.
export interface HandoffSummary {
  state: HandoffState;
  steps: AgentStep[];
}

const STATE_WORDS: [HandoffState, string][] = [
  ["running", "working"],
  ["failed", "failed"],
  ["left_running", "left running"],
  ["interrupted", "interrupted"],
  ["done", "done"],
];

export const subAgentCount = (n: number): string => `${n} ${n === 1 ? "sub-agent" : "sub-agents"}`;

// "2 working, 1 done": how many agents stand in each state, the ones still running first.
export const handoffTally = (handoffs: HandoffSummary[]): string =>
  STATE_WORDS.map(([state, word]) => [handoffs.filter((h) => h.state === state).length, word] as const)
    .filter(([n]) => n > 0)
    .map(([n, word]) => `${n} ${word}`)
    .join(", ");

const known = (ms: number | null): ms is number => ms !== null;

// When the first agent's first step happened; null when no step carries a time.
export const handoffsStartMs = (handoffs: HandoffSummary[]): number | null => {
  const starts = handoffs.map((h) => turnStartMs(h.steps)).filter(known);
  return starts.length > 0 ? Math.min(...starts) : null;
};

// From the first agent's first step to the last agent's last one; null when the times are missing.
export const handoffsSpanSeconds = (handoffs: HandoffSummary[]): number | null => {
  const start = handoffsStartMs(handoffs);
  const ends = handoffs.map((h) => turnStartMs([...h.steps].reverse())).filter(known);
  if (start === null || ends.length === 0) return null;
  return Math.max(0, Math.round((Math.max(...ends) - start) / 1_000));
};
