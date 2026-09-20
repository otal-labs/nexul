import { beforeEach, describe, expect, it } from "vitest";

import type { ActivityEntry, RunFrame } from "@/models/Trail";
import { targetKey, usePlayRunStore } from "@/stores/playRunStore";

const step: ActivityEntry = { kind: "tool_call", call_id: "c-1", tool: "Read", summary: '{"file_path":"main.go"}', at: "2026-09-18T10:00:00Z" };

const frame = (overrides: Partial<RunFrame> = {}): RunFrame => ({
  trail_id: "tr-1",
  play_id: "play-1",
  target_type: "ticket",
  target_id: "t-1",
  state: "running",
  activity: null,
  ended_at: null,
  last_error: "",
  ...overrides,
});

describe("playRunStore", () => {
  beforeEach(() => {
    usePlayRunStore.setState({ frames: {}, steps: {}, activeByTarget: {} });
  });

  it("keeps every step the frames carried, a result replacing its call", () => {
    const result: ActivityEntry = { ...step, kind: "tool_result", at: "2026-09-18T10:00:02Z" };
    const text: ActivityEntry = { kind: "text", summary: "Done", at: "2026-09-18T10:00:03Z" };
    usePlayRunStore.getState().applyFrame(frame({ state: "starting" }));
    usePlayRunStore.getState().applyFrame(frame({ activity: step }));
    usePlayRunStore.getState().applyFrame(frame({ activity: result }));
    usePlayRunStore.getState().applyFrame(frame({ activity: text }));

    expect(usePlayRunStore.getState().steps["tr-1"]).toEqual([result, text]);
  });

  it("keeps the latest frame per trail and indexes an active trail by its target", () => {
    usePlayRunStore.getState().applyFrame(frame({ state: "starting" }));
    usePlayRunStore.getState().applyFrame(frame({ activity: step }));

    const s = usePlayRunStore.getState();
    expect(s.frames["tr-1"]).toMatchObject({ state: "running", activity: step });
    expect(s.activeByTarget[targetKey("ticket", "t-1")]).toBe("tr-1");
  });

  it("drops the target index when the trail ends, keeping the terminal frame", () => {
    usePlayRunStore.getState().applyFrame(frame());
    usePlayRunStore.getState().applyFrame(frame({ state: "done", ended_at: "2026-09-17T10:00:00Z" }));

    const s = usePlayRunStore.getState();
    expect(s.activeByTarget[targetKey("ticket", "t-1")]).toBeUndefined();
    expect(s.frames["tr-1"]?.state).toBe("done");
  });

  it("a waiting frame keeps the trail active on its target and carries the question", () => {
    const question = { request_id: "req-1", questions: [{ id: "q1", text: "Proceed?", options: [] }], asked_at: "2026-09-18T10:01:00Z" };
    usePlayRunStore.getState().applyFrame(frame());
    usePlayRunStore.getState().applyFrame(frame({ state: "waiting", question }));

    const s = usePlayRunStore.getState();
    expect(s.activeByTarget[targetKey("ticket", "t-1")]).toBe("tr-1");
    expect(s.frames["tr-1"]).toMatchObject({ state: "waiting", question });
  });

  it("a stale terminal frame for another trail never clears the current active one", () => {
    usePlayRunStore.getState().applyFrame(frame({ trail_id: "tr-2" }));
    usePlayRunStore.getState().applyFrame(frame({ trail_id: "tr-1", state: "failed", last_error: "boom" }));

    expect(usePlayRunStore.getState().activeByTarget[targetKey("ticket", "t-1")]).toBe("tr-2");
  });
});
