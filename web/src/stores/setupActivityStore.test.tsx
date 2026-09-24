import { beforeEach, describe, expect, it } from "vitest";

import { useSetupActivityStore } from "@/stores/setupActivityStore";

describe("setupActivityStore", () => {
  beforeEach(() => useSetupActivityStore.setState({ lines: {} }));

  it("keeps each turn's lines apart and only the newest forty", () => {
    const { push } = useSetupActivityStore.getState();
    for (let i = 0; i < 45; i++) push("t1", `step ${i}`);
    push("t2", "other");

    const { lines } = useSetupActivityStore.getState();
    expect(lines.t1).toHaveLength(40);
    expect(lines.t1?.[0]).toBe("step 5");
    expect(lines.t2).toEqual(["other"]);
  });
});
