import { beforeEach, describe, expect, it } from "vitest";

import { useSetupActivityStore } from "@/stores/setupActivityStore";

const lines = (turnId: string) => useSetupActivityStore.getState().steps[turnId]?.map((s) => s.line);

describe("setupActivityStore", () => {
  beforeEach(() => useSetupActivityStore.setState({ steps: {} }));

  it("keeps each turn's steps apart and only the newest forty", () => {
    const { push } = useSetupActivityStore.getState();
    for (let i = 0; i < 45; i++) push("t1", `step ${i}`);
    push("t2", "other");

    expect(lines("t1")).toHaveLength(40);
    expect(lines("t1")?.[0]).toBe("step 5");
    expect(lines("t2")).toEqual(["other"]);
  });

  it("updates a tool call's step in place instead of adding its finish as a second line, closing it", () => {
    const { push } = useSetupActivityStore.getState();
    push("t1", "nexul mcp add", "call-1", true);
    push("t1", "Wrote the config");
    push("t1", "nexul mcp add", "call-1");
    push("t1", "ls ~/.claude/skills", "call-2", true);

    expect(lines("t1")).toEqual(["nexul mcp add", "Wrote the config", "ls ~/.claude/skills"]);
    expect(useSetupActivityStore.getState().steps.t1?.map((s) => s.open)).toEqual([false, false, true]);
  });
});
