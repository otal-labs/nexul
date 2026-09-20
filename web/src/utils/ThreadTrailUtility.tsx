import type { Message } from "@/models/Chat";
import { parseQuestionMessage } from "@/models/Question";
import { isTrailActive, type ActivityEntry, type Trail, type TrailQuestion, type TrailState } from "@/models/Trail";
import { segmentTranscript, type TranscriptSegment } from "@/utils/TrailTranscriptUtility";

export type TrailTurn = Extract<TranscriptSegment, { kind: "turn" }>;

// The Agent's turns that led to one message of the thread, with the run they belong to.
export interface TrailBlock {
  trail: Trail;
  turns: TrailTurn[];
}

export interface ThreadRun {
  trail: Trail;
  steps: ActivityEntry[];
  state: TrailState;
  question: TrailQuestion | null;
}

// Where each run's turns sit in the thread: above the reply the run ended on, above the question it stopped on
// (keyed by the request id the question message carries), or above the runner's note that closed them (a stop, a
// failure); live is the running run's open turn for the stream bubble.
export interface ThreadTrailBlocks {
  byReplyMessageId: Record<string, TrailBlock>;
  byQuestionRequestId: Record<string, TrailBlock>;
  byNoteBody: Record<string, TrailBlock>;
  live: TrailBlock | null;
}

const placeRun = (run: ThreadRun, out: ThreadTrailBlocks) => {
  const { trail } = run;
  let turns: TrailTurn[] = [];
  const take = (): TrailTurn[] => {
    const taken = turns;
    turns = [];
    return taken;
  };
  for (const segment of segmentTranscript(trail, run.steps, run.state, run.question)) {
    if (segment.kind === "turn") turns.push(segment);
    if (segment.kind === "question" && run.question) out.byQuestionRequestId[run.question.request_id] = { trail, turns: take() };
    if (segment.kind === "reply" && trail.reply_message_id === "") {
      // The run ended before its reply was posted; the text stays with the turns so the note that follows shows it.
      turns.push({ kind: "turn", entries: [segment.entry], running: false, from: null, until: segment.entry.at });
    }
    if (segment.kind === "reply" && trail.reply_message_id !== "" && turns.length > 0) out.byReplyMessageId[trail.reply_message_id] = { trail, turns: take() };
    if (segment.kind === "note" && turns.length > 0) out.byNoteBody[segment.text] = { trail, turns: take() };
  }
  if (isTrailActive(run.state) && run.state !== "waiting") {
    out.live = { trail, turns };
    return;
  }
  // A transcript that ends on a turn (legacy rows, or a reply the harness never echoed as a step) still led to the reply.
  if (turns.length > 0 && trail.reply_message_id !== "") {
    const placed = out.byReplyMessageId[trail.reply_message_id];
    out.byReplyMessageId[trail.reply_message_id] = { trail, turns: [...(placed?.turns ?? []), ...turns] };
  }
};

export const threadTrailBlocks = (runs: ThreadRun[]): ThreadTrailBlocks => {
  const out: ThreadTrailBlocks = { byReplyMessageId: {}, byQuestionRequestId: {}, byNoteBody: {}, live: null };
  runs.forEach((run) => placeRun(run, out));
  return out;
};

// The block a message carries, if a run led to it: a system note by its text, an Agent question by its request id,
// any other Agent message as the reply a run ended on.
export const trailBlockFor = (message: Message, blocks: ThreadTrailBlocks): TrailBlock | undefined => {
  if (message.author_kind === "system") return blocks.byNoteBody[message.body];
  if (message.author_kind !== "agent") return undefined;
  const question = parseQuestionMessage(message.body);
  if (question !== null) return blocks.byQuestionRequestId[question.request_id];
  return blocks.byReplyMessageId[message.id];
};
