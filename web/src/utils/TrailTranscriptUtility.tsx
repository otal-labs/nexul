import { answerText } from "@/models/Question";
import { isTrailActive, type ActivityEntry, type Trail, type TrailQuestion, type TrailState } from "@/models/Trail";

// One block of the transcript as a conversation: the starter's bubbles, the Agent's turn groups, the question
// card where it was asked, the final reply as prose, and the runner's notes as muted lines between them.
export type TranscriptSegment =
  // via names the harness a message was written in, null for one written in Nexul.
  | { kind: "user"; body: string; at: string | null; via: string | null }
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

// Splits the steps into turns: a question (the latest one, the only one stored with its answer), a runner's note, and a
// message the starter wrote in the harness each close the current turn, and a turn that closes on a text step hands that text over as prose; while the run is
// live the closing text stays in the turn until the harness moves on.
export const segmentTranscript = (trail: Trail, steps: ActivityEntry[], state: TrailState, question: TrailQuestion | null): TranscriptSegment[] => {
  const out: TranscriptSegment[] = [{ kind: "user", body: startedBody(trail), at: trail.started_at, via: null }];
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

const PULL_REQUEST_URL = /https?:\/\/github\.com\/([\w.-]+\/[\w.-]+)\/pull\/(\d+)/;

// The first pull request the reply links, for the chip under the prose.
export const pullRequestLink = (text: string): { url: string; label: string } | null => {
  const match = PULL_REQUEST_URL.exec(text);
  if (!match) return null;
  return { url: match[0], label: `${match[1]}#${match[2]}` };
};
