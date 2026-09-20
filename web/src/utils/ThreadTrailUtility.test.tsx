import { describe, expect, it } from "vitest";

import type { Message } from "@/models/Chat";
import type { ActivityEntry, Trail, TrailQuestion } from "@/models/Trail";
import { threadTrailBlocks, trailBlockFor, type ThreadRun } from "@/utils/ThreadTrailUtility";

const at = (sec: number) => `2026-09-18T10:00:${String(sec).padStart(2, "0")}Z`;
const read: ActivityEntry = { kind: "tool_result", call_id: "c-1", tool: "Read", summary: '{"file_path":"handler.go"}', at: at(2) };
const asked: ActivityEntry = { kind: "question", call_id: "c-2", tool: "AskUserQuestion", summary: "Proceed?", at: at(4) };
const edit: ActivityEntry = { kind: "tool_result", call_id: "c-3", tool: "Edit", summary: "/src/handler.go", at: at(7) };
const reply: ActivityEntry = { kind: "text", summary: "Opened the PR.", detail: "Opened the PR.", at: at(9) };
const note: ActivityEntry = { kind: "note", summary: "Run stopped by onik.", at: at(9) };

const question: TrailQuestion = { request_id: "req-1", questions: [{ id: "q1", text: "Proceed?", options: [] }], asked_at: at(4), answer: { answers: { q1: { selected: ["Yes"] } } } };

const trail = (overrides: Partial<Trail>): Trail =>
  ({ id: "tr-1", play_label: "Fix with AI", conversation_id: "c-1", custom_instructions: "", started_at: at(0), ended_at: at(9), state: "done", reply_message_id: "reply-1", activity: [], ...overrides }) as Trail;

const run = (t: Trail, extra: Partial<ThreadRun> = {}): ThreadRun => ({ trail: t, steps: t.activity, state: t.state, question: t.question ?? null, ...extra });

const message = (overrides: Partial<Message>): Message => ({ id: "m", conversation_id: "c-1", author_id: "u", author_kind: "user", body: "", mentions: null, created_at: at(1), updated_at: at(1), ...overrides });

describe("threadTrailBlocks", () => {
  it("places a finished run's turns above the reply it ended on", () => {
    const t = trail({ activity: [read, edit, reply] });
    const blocks = threadTrailBlocks([run(t)]);
    expect(Object.keys(blocks.byReplyMessageId)).toEqual(["reply-1"]);
    expect(blocks.byReplyMessageId["reply-1"]?.turns.map((turn) => turn.entries)).toEqual([[read, edit]]);
    expect(blocks.live).toBeNull();
  });

  it("a run whose steps are legacy rows, with no reply step to split off, still places them above its reply", () => {
    const legacy: ActivityEntry[] = [{ kind: "other", summary: "Read main.go", at: "" }, { kind: "other", summary: "Bash go test", at: "" }];
    const blocks = threadTrailBlocks([run(trail({ activity: legacy }))]);
    expect(blocks.byReplyMessageId["reply-1"]?.turns.map((turn) => turn.entries)).toEqual([legacy]);
  });

  it("splits the turns at the question: the first above the question, the rest above the reply", () => {
    const t = trail({ activity: [read, asked, edit, reply], question });
    const blocks = threadTrailBlocks([run(t)]);
    expect(blocks.byQuestionRequestId["req-1"]?.turns.map((turn) => turn.entries)).toEqual([[read]]);
    expect(blocks.byReplyMessageId["reply-1"]?.turns.map((turn) => turn.entries)).toEqual([[edit]]);
  });

  it("a waiting run keeps its question block even before any step, and is not live", () => {
    const unanswered: TrailQuestion = { request_id: question.request_id, questions: question.questions, asked_at: question.asked_at };
    const t = trail({ state: "waiting", ended_at: null, reply_message_id: "", activity: [], question: unanswered });
    const blocks = threadTrailBlocks([run(t)]);
    expect(blocks.byQuestionRequestId["req-1"]?.turns).toEqual([]);
    expect(blocks.live).toBeNull();
  });

  it("a running run's open turn is live, with its steps so far", () => {
    const t = trail({ state: "running", ended_at: null, reply_message_id: "", activity: [read] });
    const blocks = threadTrailBlocks([run(t)]);
    expect(blocks.live?.trail).toBe(t);
    expect(blocks.live?.turns).toHaveLength(1);
    expect(blocks.live?.turns[0]).toMatchObject({ running: true, entries: [read] });
  });

  it("a run stopped mid-turn leaves its turns, and any unposted reply text, above the runner's note", () => {
    const t = trail({ state: "interrupted", reply_message_id: "", activity: [read, reply, note] });
    const blocks = threadTrailBlocks([run(t)]);
    expect(blocks.byReplyMessageId).toEqual({});
    expect(blocks.byNoteBody["Run stopped by onik."]?.turns.map((turn) => turn.entries)).toEqual([[read], [reply]]);
  });

  it("a note after the reply carries no turns", () => {
    const t = trail({ activity: [read, reply, { ...note, summary: "Ticket is already in done; not moving it back" }] });
    const blocks = threadTrailBlocks([run(t)]);
    expect(blocks.byNoteBody).toEqual({});
    expect(blocks.byReplyMessageId["reply-1"]?.turns).toHaveLength(1);
  });
});

describe("trailBlockFor", () => {
  const t = trail({ activity: [read, asked, edit, reply, note], question });
  const blocks = threadTrailBlocks([run(t)]);
  const questionBody = "```nexul-question\n" + JSON.stringify({ request_id: "req-1", questions: [] }) + "\n```";

  it("finds the reply by its id, the question by its request id, the note by its text, and nothing for a person", () => {
    expect(trailBlockFor(message({ id: "reply-1", author_kind: "agent", body: "Opened the PR." }), blocks)?.turns.map((turn) => turn.entries)).toEqual([[edit]]);
    expect(trailBlockFor(message({ author_kind: "agent", body: questionBody }), blocks)?.turns.map((turn) => turn.entries)).toEqual([[read]]);
    expect(trailBlockFor(message({ id: "other", author_kind: "agent", body: "hi" }), blocks)).toBeUndefined();
    expect(trailBlockFor(message({ author_kind: "system", body: "Run stopped by onik." }), blocks)).toBeUndefined();
    expect(trailBlockFor(message({ body: "Started Fix with AI" }), blocks)).toBeUndefined();
  });
});
