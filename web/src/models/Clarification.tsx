import type { ChecklistRow } from "@/models/QuestionChecklist";
import type { TrailState } from "@/models/Trail";

export interface QuestionOption {
  label: string;
  description?: string;
}

// Mirrors docs.ClarificationQuestion: no picks, no text, and not skipped is pending.
export interface ClarificationQuestion {
  id: string;
  doc_id: string;
  round: number;
  position: number;
  question: string;
  why: string;
  options: QuestionOption[];
  multi_select: boolean;
  selected: string[];
  text: string;
  skipped: boolean;
  answered_by?: string;
  answered_at?: string;
}

// Mirrors docs.ClarificationRound; no_gaps_at reaches only people who may close the clarification.
export interface ClarificationRound {
  doc_id: string;
  round: number;
  started_by: string;
  trail_id: string;
  started_at: string;
  running: boolean;
  anything_else: string;
  anything_else_by?: string;
  anything_else_at?: string;
  anything_else_reply: string;
  no_gaps_at?: string;
  closed_by?: string;
  closed_at?: string;
  questions: ClarificationQuestion[];
}

// Mirrors docs.Clarification as the viewer sees it; can_close is docs:write plus plays:run on Clarify via AI.
export interface Clarification {
  rounds: ClarificationRound[];
  running: boolean;
  closed: boolean;
  can_close: boolean;
}

export const isPending = (q: ClarificationQuestion) => q.selected.length === 0 && q.text === "" && !q.skipped;

// The newest round and how many of its questions nobody has answered or skipped; undefined when there is no round.
export const unansweredInNewestRound = (c: Clarification) => {
  const newest = c.rounds.at(-1);
  if (!newest) return undefined;
  return { round: newest.round, count: newest.questions.filter(isPending).length };
};

export interface ClarificationAnswerInput {
  selected: string[];
  text: string;
  skipped: boolean;
}

export const questionRow = (q: ClarificationQuestion): ChecklistRow => ({
  key: q.id,
  item: { id: q.id, text: q.question, options: q.options, multi_select: q.multi_select },
  why: q.why,
  answer: q,
});

// The rounds that asked something; a running round and a no-gaps round hold no questions.
export const askedRounds = (c: Clarification): ClarificationRound[] => c.rounds.filter((r) => r.questions.length > 0);

// Questions nobody has answered or skipped, the tab's count; a closed clarification waits on nobody.
export const waitingCount = (c: Clarification): number =>
  c.closed ? 0 : c.rounds.flatMap((r) => r.questions).filter(isPending).length;

export const answeredCount = (c: Clarification): number =>
  c.rounds.flatMap((r) => r.questions).filter((q) => !isPending(q) && !q.skipped).length;

// The newest round that found no gaps and wrote the doc, seen only by people who may close.
export const noGapsRound = (c: Clarification): ClarificationRound | undefined =>
  c.rounds.filter((r) => r.no_gaps_at !== undefined).at(-1);

// Answers saved after the newest no-gaps round wrote the doc, which another round picks up.
export const answersChangedSinceWritten = (c: Clarification): number => {
  const written = noGapsRound(c)?.no_gaps_at;
  if (!written) return 0;
  return c.rounds
    .flatMap((r) => r.questions)
    .filter((q) => q.answered_at !== undefined && Date.parse(q.answered_at) > Date.parse(written)).length;
};

export type ClarificationPhase = "none" | "running" | "closed" | "noGaps" | "waiting" | "answered";

export const clarificationPhase = (c: Clarification): ClarificationPhase => {
  if (c.running) return "running";
  if (c.closed) return "closed";
  if (c.rounds.at(-1)?.no_gaps_at !== undefined) return "noGaps";
  if (waitingCount(c) > 0) return "waiting";
  if (askedRounds(c).length > 0) return "answered";
  return "none";
};

const plural = (n: number, noun: string): string => `${n} ${noun}${n === 1 ? "" : "s"}`;

export interface ClarificationStatus {
  icon: TrailState | null;
  label: string;
  detail: string;
}

// The clarification's state as one line; dev is someone who may close it, who alone sees the run and the no-gaps verdict.
export const clarificationStatus = (c: Clarification, dev: boolean, locked: boolean): ClarificationStatus => {
  const phase = clarificationPhase(c);
  const asked = askedRounds(c);
  const rounds = plural(asked.length, "round");
  const newest = c.rounds.at(-1)?.round ?? 0;
  if (phase === "running" && dev) return { icon: "running", label: `Round ${newest} running`, detail: locked ? "The doc is locked until it ends" : "" };
  if (phase === "running") return { icon: null, label: "More questions are on the way", detail: "" };
  if (phase === "closed" && dev) return { icon: "done", label: "Closed", detail: `${rounds} · ${answeredCount(c)} answered` };
  if (phase === "closed") return { icon: "done", label: "All answered", detail: rounds };
  if (phase === "noGaps" && dev) return { icon: "done", label: "No gaps left", detail: "The doc now holds every answer" };
  if (phase === "waiting") {
    const round = asked.filter((r) => r.questions.some(isPending)).at(-1)?.round ?? newest;
    return { icon: "waiting", label: `${plural(waitingCount(c), "question")} waiting`, detail: `Round ${round}` };
  }
  if (phase === "answered" && dev) return { icon: "done", label: `Round ${asked.at(-1)?.round ?? newest} answered`, detail: "" };
  if (phase === "none") return { icon: null, label: "No questions yet", detail: "" };
  return { icon: "done", label: "All answered", detail: "Thanks, nothing is waiting on you." };
};
