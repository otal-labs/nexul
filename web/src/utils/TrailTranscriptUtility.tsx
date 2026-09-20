import { answerText } from "@/models/Question";
import { isCommandTool, isTrailActive, type ActivityEntry, type Trail, type TrailQuestion, type TrailState } from "@/models/Trail";

// One block of the transcript as a conversation: the starter's bubbles, the Agent's turn groups, the question
// card where it was asked, the final reply as prose, and the runner's notes as muted lines between them.
export type TranscriptSegment =
  | { kind: "user"; body: string; at: string | null }
  // from is the run's start for the first turn; until is when the turn ended when that is later than its last row.
  | { kind: "turn"; entries: ActivityEntry[]; running: boolean; from: string | null; until: string | null }
  | { kind: "question" }
  | { kind: "reply"; entry: ActivityEntry }
  | { kind: "note"; text: string };

const startedBody = (trail: Trail): string => {
  if (trail.custom_instructions === "") return `Started ${trail.play_label}`;
  return `Started ${trail.play_label}\n\n${trail.custom_instructions}`;
};

// Mirrors harness.QuestionAnswer.Summary, the message the answer was posted as: one line for one question, a bullet per question otherwise.
const answerBody = (question: TrailQuestion): string => {
  const answers = question.answer?.answers ?? {};
  if (question.questions.length <= 1) {
    const first = Object.values(answers)[0];
    return first ? `Answered: ${answerText(first)}` : "Answered.";
  }
  return ["Answered:", ...question.questions.map((item) => `- ${item.text}: ${answerText(answers[item.id])}`)].join("\n");
};

const isRunningState = (state: TrailState): boolean => isTrailActive(state) && state !== "waiting";

const lastIndexOfKind = (steps: ActivityEntry[], kind: ActivityEntry["kind"]): number => {
  for (let i = steps.length - 1; i >= 0; i--) {
    if (steps[i]?.kind === kind) return i;
  }
  return -1;
};

// Splits the steps into turns: a question (the latest one, the only one stored with its answer) and a runner's note
// each close the current turn, and a turn that closes on a text step hands that text over as prose; while the run is
// live the closing text stays in the turn until the harness moves on.
export const segmentTranscript = (trail: Trail, steps: ActivityEntry[], state: TrailState, question: TrailQuestion | null): TranscriptSegment[] => {
  const out: TranscriptSegment[] = [{ kind: "user", body: startedBody(trail), at: trail.started_at }];
  let turn: ActivityEntry[] = [];
  let from: string | null = trail.started_at;
  const flushTurn = (closing: boolean, until: string | null = null) => {
    const last = turn[turn.length - 1];
    const reply = closing && last?.kind === "text" ? turn.pop() : undefined;
    if (turn.length > 0) {
      out.push({ kind: "turn", entries: turn, running: false, from, until: until ?? reply?.at ?? null });
      from = null;
    }
    if (reply) out.push({ kind: "reply", entry: reply });
    turn = [];
  };
  const pushQuestion = () => {
    if (!question) return;
    out.push({ kind: "question" });
    if (question.answer) out.push({ kind: "user", body: answerBody(question), at: null });
  };
  const questionAt = question ? lastIndexOfKind(steps, "question") : -1;

  steps.forEach((entry, i) => {
    if (entry.kind === "note") {
      flushTurn(true);
      out.push({ kind: "note", text: entry.summary });
      return;
    }
    if (i === questionAt) {
      flushTurn(true);
      pushQuestion();
      return;
    }
    turn.push(entry);
  });
  if (questionAt === -1 && question) {
    flushTurn(true);
    pushQuestion();
  }

  const over = !isTrailActive(state);
  flushTurn(over, over ? trail.ended_at : null);

  if (!isRunningState(state)) return out;
  const tail = out[out.length - 1];
  if (tail?.kind === "turn") {
    tail.running = true;
    return out;
  }
  out.push({ kind: "turn", entries: [], running: true, from, until: null });
  return out;
};

export interface TurnCounts {
  tools: number;
  commands: number;
  updates: number;
}

// Tool rows, command rows, and the Agent's sentences between them; a call and its result are one row.
export const turnCounts = (entries: ActivityEntry[]): TurnCounts => {
  const counts: TurnCounts = { tools: 0, commands: 0, updates: 0 };
  for (const entry of entries) {
    if (entry.kind === "text") {
      counts.updates += 1;
      continue;
    }
    if (isCommandTool(entry.tool)) {
      counts.commands += 1;
      continue;
    }
    counts.tools += 1;
  }
  return counts;
};

const plural = (n: number, noun: string): string => `${n} ${noun}${n === 1 ? "" : "s"}`;

// "Used 12 tools, ran 5 commands, and received 1 update"; only the non-zero parts, in that order.
export const turnSummary = ({ tools, commands, updates }: TurnCounts): string => {
  const parts: string[] = [];
  if (tools > 0) parts.push(`Used ${plural(tools, "tool")}`);
  if (commands > 0) parts.push(`${parts.length === 0 ? "Ran" : "ran"} ${plural(commands, "command")}`);
  if (updates > 0) parts.push(`${parts.length === 0 ? "Received" : "received"} ${plural(updates, "update")}`);
  if (parts.length === 0) return "No steps yet";
  if (parts.length === 1) return parts[0]!;
  return `${parts.slice(0, -1).join(", ")}${parts.length > 2 ? "," : ""} and ${parts[parts.length - 1]}`;
};

const timestampMs = (iso: string): number | null => {
  if (iso === "" || iso.startsWith("0001-")) return null;
  const ms = new Date(iso).getTime();
  return Number.isNaN(ms) ? null : ms;
};

// When the turn's first step happened, for the running clock; null when no step carries a time.
export const turnStartMs = (entries: ActivityEntry[], from: string | null = null): number | null => {
  const fromMs = from === null ? null : timestampMs(from);
  if (fromMs !== null) return fromMs;
  for (const entry of entries) {
    const ms = timestampMs(entry.at);
    if (ms !== null) return ms;
  }
  return null;
};

// The run's start (from, when given) or the first step, to the turn's end (until when given, else its last step), for
// a finished turn's "Worked for"; null when the times are missing.
export const turnSpanSeconds = (entries: ActivityEntry[], until: string | null = null, from: string | null = null): number | null => {
  const start = (from === null ? null : timestampMs(from)) ?? turnStartMs(entries);
  const end = (until === null ? null : timestampMs(until)) ?? turnStartMs([...entries].reverse());
  if (start === null || end === null) return null;
  return Math.max(0, Math.round((end - start) / 1_000));
};

export const formatElapsed = (seconds: number): string => {
  const h = Math.floor(seconds / 3600);
  const m = Math.floor((seconds % 3600) / 60);
  const s = seconds % 60;
  if (h > 0) return `${h}h ${m}m`;
  if (m > 0) return `${m}m ${s}s`;
  return `${s}s`;
};

const PULL_REQUEST_URL = /https?:\/\/github\.com\/([\w.-]+\/[\w.-]+)\/pull\/(\d+)/;

// The first pull request the reply links, for the chip under the prose.
export const pullRequestLink = (text: string): { url: string; label: string } | null => {
  const match = PULL_REQUEST_URL.exec(text);
  if (!match) return null;
  return { url: match[0], label: `${match[1]}#${match[2]}` };
};
