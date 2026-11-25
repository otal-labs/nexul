import { stepLabel, type AgentStep } from "@nexul/client-core/agentSteps";
import { answerText, type AnswerValue, type QuestionItem } from "@nexul/client-core/question";

// Mirrors internal/plays TrailState: starting precedes the harness accepting, waiting is a turn stopped on a
// question to the user, the last three are terminal.
export const TRAIL_STATES = ["starting", "running", "waiting", "done", "failed", "interrupted"] as const;
export type TrailState = (typeof TRAIL_STATES)[number];

export const isTrailActive = (state: TrailState): boolean => state === "starting" || state === "running" || state === "waiting";

const STATE_LABELS: Record<TrailState, string> = {
  starting: "Starting…",
  running: "Running…",
  waiting: "Waiting for your answer",
  done: "Done",
  failed: "Failed",
  interrupted: "Interrupted",
};

// The row's summary stays one line even for a multi-line or long harness error; a run's own page shows it whole.
const SUMMARY_ERROR_MAX_LENGTH = 80;

const truncateSummaryError = (text: string): string => {
  const firstLine = text.split("\n")[0] ?? "";
  if (firstLine.length <= SUMMARY_ERROR_MAX_LENGTH) return firstLine;
  return `${firstLine.slice(0, SUMMARY_ERROR_MAX_LENGTH)}…`;
};

// One line for a trail row: the latest step while it runs, the reason once it ended badly.
export const trailSummary = (state: TrailState, lastError: string, activity: AgentStep | null | undefined): string => {
  if (state === "running" && activity && activity.kind !== "user_message") return stepLabel(activity);
  if ((state === "failed" || state === "interrupted") && lastError !== "") {
    return `${STATE_LABELS[state]} · ${truncateSummaryError(lastError)}`;
  }
  return STATE_LABELS[state];
};

const lastIndexOfCall = (activity: AgentStep[], callId: string): number => {
  for (let i = activity.length - 1; i >= 0; i--) {
    if (activity[i]?.call_id === callId) return i;
  }
  return -1;
};

// The live frame carries only the newest step; it replaces the step sharing its call id, else joins the end.
export const mergeLiveStep = <T extends AgentStep>(activity: T[], live: T | null): T[] => {
  if (live === null) return activity;
  if (live.call_id) {
    const i = lastIndexOfCall(activity, live.call_id);
    if (i !== -1) return activity.map((e, j) => (j === i ? live : e));
  }
  const last = activity[activity.length - 1];
  if (last && last.at === live.at && last.summary === live.summary) return activity;
  return [...activity, live];
};

// Folds the steps a run pushed while the page watched onto the persisted list: a call id replaces its earlier
// entry (a result never steps back to its call), anything else is skipped when the list already has it.
export const mergeLiveSteps = <T extends AgentStep>(activity: T[], live: T[]): T[] =>
  live.reduce<T[]>((acc, step) => {
    if (step.call_id) {
      const i = lastIndexOfCall(acc, step.call_id);
      if (i === -1) return [...acc, step];
      if (acc[i]?.kind === "tool_result" && step.kind === "tool_call") return acc;
      return acc.map((e, j) => (j === i ? step : e));
    }
    if (acc.some((e) => e.at === step.at && e.summary === step.summary)) return acc;
    return [...acc, step];
  }, activity);

// What a transcript reads of the run itself and of the question it stopped on.
export interface TranscriptTrail {
  play_label: string;
  custom_instructions: string;
  started_at: string;
  ended_at: string | null;
}

export interface TranscriptQuestion {
  questions: QuestionItem[];
  answer?: { answers: Record<string, AnswerValue> };
}

// One block of the transcript as a conversation: the starter's bubbles, the Agent's turn groups, the question
// card where it was asked, the final reply as prose, and the runner's notes as muted lines between them.
export type TranscriptSegment<T extends AgentStep> =
  // via names the harness a message was written in, null for one written in Nexul.
  | { kind: "user"; body: string; at: string | null; via: string | null }
  // from is the run's start for the first turn; until is when the turn ended when that is later than its last row.
  | { kind: "turn"; entries: T[]; running: boolean; from: string | null; until: string | null }
  | { kind: "question" }
  | { kind: "reply"; entry: T }
  | { kind: "note"; text: string };

const startedBody = (trail: TranscriptTrail): string => {
  if (trail.custom_instructions === "") return `Started ${trail.play_label}`;
  return `Started ${trail.play_label}\n\n${trail.custom_instructions}`;
};

// Mirrors harness.QuestionAnswer.Summary, the message the answer was posted as: one line for one question, a bullet per question otherwise.
const answerBody = (question: TranscriptQuestion): string => {
  const answers = question.answer?.answers ?? {};
  if (question.questions.length <= 1) {
    const first = Object.values(answers)[0];
    return first ? `Answered: ${answerText(first)}` : "Answered.";
  }
  return ["Answered:", ...question.questions.map((item) => `- ${item.text}: ${answerText(answers[item.id])}`)].join("\n");
};

const isRunningState = (state: TrailState): boolean => isTrailActive(state) && state !== "waiting";

const lastIndexOfKind = (steps: AgentStep[], kind: string): number => {
  for (let i = steps.length - 1; i >= 0; i--) {
    if (steps[i]?.kind === kind) return i;
  }
  return -1;
};

// Splits the steps into turns: a question (the latest one, the only one stored with its answer), a runner's note, and a
// message the starter wrote in the harness each close the current turn, and a turn that closes on a text step hands that text over as prose; while the run is
// live the closing text stays in the turn until the harness moves on.
export const segmentTranscript = <T extends AgentStep>(
  trail: TranscriptTrail,
  steps: T[],
  state: TrailState,
  question: TranscriptQuestion | null,
): TranscriptSegment<T>[] => {
  const out: TranscriptSegment<T>[] = [{ kind: "user", body: startedBody(trail), at: trail.started_at, via: null }];
  let turn: T[] = [];
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
    if (question.answer) out.push({ kind: "user", body: answerBody(question), at: null, via: null });
  };
  const questionAt = question ? lastIndexOfKind(steps, "question") : -1;

  steps.forEach((entry, i) => {
    if (entry.kind === "note") {
      flushTurn(true);
      out.push({ kind: "note", text: entry.summary });
      return;
    }
    if (entry.kind === "user_message") {
      flushTurn(true);
      out.push({ kind: "user", body: entry.detail || entry.summary, at: entry.at, via: entry.tool ?? null });
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
