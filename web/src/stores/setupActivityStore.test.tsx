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

  it("updates a tool call's step in place instead of adding its finish as a second line", () => {
    const { push } = useSetupActivityStore.getState();
    push("t1", "Ran command started", "call-1");
    push("t1", "Wrote the config");
    push("t1", "Ran command", "call-1");
    push("t1", "Ran command started", "call-2");

    expect(lines("t1")).toEqual(["Ran command", "Wrote the config", "Ran command started"]);
  });
});
