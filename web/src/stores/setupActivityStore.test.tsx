import { beforeEach, describe, expect, it } from "vitest";

import { useSetupActivityStore } from "@/stores/setupActivityStore";
import type { ActivityEntry } from "@/models/Trail";

const summaries = (turnId: string) => useSetupActivityStore.getState().steps[turnId]?.map((s) => s.summary);
const step = (summary: string, kind: ActivityEntry["kind"] = "text", call_id = ""): ActivityEntry => ({ kind, call_id, tool: "", summary, detail: "", at: "" });

describe("setupActivityStore", () => {
  beforeEach(() => useSetupActivityStore.setState({ steps: {} }));

  it("keeps each turn's steps apart and only the newest two hundred", () => {
    const { push } = useSetupActivityStore.getState();
    for (let i = 0; i < 205; i++) push("t1", step(`step ${i}`));
    push("t2", step("other"));

    expect(summaries("t1")).toHaveLength(200);
    expect(summaries("t1")?.[0]).toBe("step 5");
    expect(summaries("t2")).toEqual(["other"]);
  });

  it("updates a tool call's step in place instead of adding its finish as a second row", () => {
    const { push } = useSetupActivityStore.getState();
    push("t1", step("nexul mcp add", "tool_call", "call-1"));
    push("t1", step("Wrote the config"));
    push("t1", step("nexul mcp add", "tool_result", "call-1"));
    push("t1", step("ls ~/.claude/skills", "tool_call", "call-2"));

    expect(summaries("t1")).toEqual(["nexul mcp add", "Wrote the config", "ls ~/.claude/skills"]);
    expect(useSetupActivityStore.getState().steps.t1?.map((s) => s.kind)).toEqual(["tool_result", "text", "tool_call"]);
  });
});
