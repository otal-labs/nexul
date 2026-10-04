export interface QuestionOption {
  label: string;
  description?: string;
}

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

export interface ClarificationRound {
  doc_id: string;
  round: number;
  started_by: string;
  trail_id: string;
  started_at: string;
  running: boolean;
  anything_else: string;
  anything_else_reply: string;
  no_gaps_at?: string;
  closed_at?: string;
  questions: ClarificationQuestion[];
}

// A doc's rounds of questions and answers, as GET /api/docs/{id}/clarification returns them.
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
