import { isTrailActive, type TrailState, segmentTranscript, type TranscriptSegment } from "@nexul/client-core/trail";

import type { Message } from "@/models/Chat";
import { parseQuestionMessage } from "@/models/Question";
import { type ActivityEntry, type Trail, type TrailQuestion } from "@/models/Trail";

export type TrailTurn = Extract<TranscriptSegment<ActivityEntry>, { kind: "turn" }>;

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
  bySystemBody: Record<string, TrailBlock>;
  live: TrailBlock | null;
}

// One run's own placements, before any other run of the thread is merged in; appendedReply marks a reply block made only
// from the turns after its last segment, which joins an earlier run's block for the same reply instead of replacing it.
interface Placed {
  blocks: ThreadTrailBlocks;
  appendedReply: boolean;
}

const placeRun = (run: ThreadRun): Placed => {
  const out: ThreadTrailBlocks = { byReplyMessageId: {}, byQuestionRequestId: {}, bySystemBody: {}, live: null };
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
    if (segment.kind === "note" && turns.length > 0) out.bySystemBody[segment.text] = { trail, turns: take() };
  }
  if (isTrailActive(run.state) && run.state !== "waiting") {
    out.live = { trail, turns };
    return { blocks: out, appendedReply: false };
  }
  // A transcript that ends on a turn (legacy rows, or a reply the harness never echoed as a step) still led to the reply.
  if (turns.length === 0 || trail.reply_message_id === "") return { blocks: out, appendedReply: false };
  const placed = out.byReplyMessageId[trail.reply_message_id];
  out.byReplyMessageId[trail.reply_message_id] = { trail, turns: [...(placed?.turns ?? []), ...turns] };
  return { blocks: out, appendedReply: placed === undefined };
};

// The last placement of each fetched trail, so a frame for one run hands every other run's message the block it had.
const placements = new WeakMap<Trail, { run: ThreadRun; placed: Placed }>();

const placeRunOnce = (run: ThreadRun): Placed => {
  const last = placements.get(run.trail);
  if (last && last.run.steps === run.steps && last.run.state === run.state && last.run.question === run.question) return last.placed;
  const placed = placeRun(run);
  placements.set(run.trail, { run, placed });
  return placed;
};

export const threadTrailBlocks = (runs: ThreadRun[]): ThreadTrailBlocks => {
  const out: ThreadTrailBlocks = { byReplyMessageId: {}, byQuestionRequestId: {}, bySystemBody: {}, live: null };
  for (const run of runs) {
    const { blocks, appendedReply } = placeRunOnce(run);
    Object.assign(out.byQuestionRequestId, blocks.byQuestionRequestId);
    Object.assign(out.bySystemBody, blocks.bySystemBody);
    for (const [id, block] of Object.entries(blocks.byReplyMessageId)) {
      const earlier = out.byReplyMessageId[id];
      out.byReplyMessageId[id] = appendedReply && earlier ? { trail: block.trail, turns: [...earlier.turns, ...block.turns] } : block;
    }
    if (blocks.live) out.live = blocks.live;
  }
  return out;
};

// The block a message carries, if a run led to it: a system message by its text, an Agent question by its request id,
// any other Agent message as the reply a run ended on.
export const trailBlockFor = (message: Message, blocks: ThreadTrailBlocks): TrailBlock | undefined => {
  if (message.author_kind === "system") return blocks.bySystemBody[message.body];
  if (message.author_kind !== "agent") return undefined;
  const question = parseQuestionMessage(message.body);
  if (question !== null) return blocks.byQuestionRequestId[question.request_id];
  return blocks.byReplyMessageId[message.id];
};
